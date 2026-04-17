// Copyright 2018-2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

const assert = require('assert');
const URL = require('url').URL;
const {
  buildTableRowSelector,
  clearDefaultInput,
  saveDebugScreenshot,
  selectPipelineForRun,
  waitForCondition,
  waitForHashPrefix,
  waitForRunPageReady,
} = require('./test-helpers');

const pipelineName = 'literal-input-pipeline-' + Date.now();
const runName = 'literal-input-run-' + Date.now();
const selectedLiteral = 'staging';
const uiTimeout = 10000;
const runStartTimeout = 30000;

let createdRunId = '';
let pipelineUploaded = false;

async function waitForCreatedRunId() {
  const currentHash = new URL(await browser.getUrl()).hash;
  if (currentHash.startsWith('#/runs/details/')) {
    return currentHash.replace('#/runs/details/', '').split('?')[0];
  }

  const runLinkSelector = `[data-testid="run-name-link"][data-run-name="${runName}"]`;
  let runId = '';

  await $('#refreshBtn').waitForDisplayed({ timeout: uiTimeout });
  await waitForCondition(
    async () => {
      const runLink = await $(runLinkSelector);
      if (await runLink.isExisting()) {
        runId = (await runLink.getAttribute('data-run-id')) || '';
        return !!runId;
      }
      await $('#refreshBtn').click();
      return false;
    },
    {
      timeout: runStartTimeout,
      interval: 1000,
      timeoutMsg: `waited ${runStartTimeout / 1000} seconds but run ${runName} did not start`,
    },
  );

  return runId;
}

async function fetchRunById(runId) {
  return browser.execute(async (currentRunId) => {
    const response = await fetch(`/apis/v2beta1/runs/${currentRunId}`);
    const responseText = await response.text();
    let responseBody;
    try {
      responseBody = responseText ? JSON.parse(responseText) : null;
    } catch (error) {
      responseBody = responseText;
    }
    return {
      ok: response.ok,
      run: responseBody,
      status: response.status,
    };
  }, runId);
}

async function deleteCreatedRun() {
  if (!createdRunId) {
    return;
  }

  try {
    const deleteResponse = await browser.execute(async (currentRunId) => {
      const response = await fetch(`/apis/v2beta1/runs/${currentRunId}`, {
        method: 'DELETE',
      });
      return {
        ok: response.ok,
        responseText: await response.text(),
        status: response.status,
      };
    }, createdRunId);

    if (!deleteResponse.ok && deleteResponse.status !== 404) {
      throw new Error(
        `run delete returned ${deleteResponse.status}: ${deleteResponse.responseText || '(empty body)'}`,
      );
    }
  } catch (error) {
    console.log('RUN_CLEANUP_FAILED', error.message);
    try {
      await saveDebugScreenshot('run-cleanup');
    } catch (screenshotError) {
      console.log('RUN_CLEANUP_SCREENSHOT_FAILED', screenshotError.message);
    }
  }
}

async function deleteUploadedPipeline() {
  if (!pipelineUploaded) {
    return;
  }

  try {
    await $('#pipelinesBtn').waitForDisplayed({ timeout: uiTimeout });
    await $('#pipelinesBtn').click();
    await waitForHashPrefix('#/pipelines', { timeout: uiTimeout });

    await $('#tableFilterBox').waitForDisplayed({ timeout: uiTimeout });
    await $('#tableFilterBox').click();
    await clearDefaultInput();
    await browser.keys(pipelineName);

    const pipelineRowSelector = buildTableRowSelector(pipelineName);
    await waitForCondition(
      async () => (await $(pipelineRowSelector).isExisting()),
      {
        timeout: uiTimeout,
        timeoutMsg: `expected pipeline row for ${pipelineName} after filtering`,
      },
    );

    await $(pipelineRowSelector).click();
    await $('#deletePipelinesAndPipelineVersionsBtn').waitForDisplayed({ timeout: uiTimeout });
    await $('#deletePipelinesAndPipelineVersionsBtn').click();

    const dialog = await $('[role="dialog"]');
    await dialog.waitForDisplayed({ timeout: uiTimeout });
    await dialog.$('button=Delete All').click();
    await dialog.waitForDisplayed({ timeout: uiTimeout, reverse: true });
  } catch (error) {
    console.log('PIPELINE_CLEANUP_FAILED', error.message);
    try {
      await saveDebugScreenshot('pipeline-cleanup');
    } catch (screenshotError) {
      console.log('PIPELINE_CLEANUP_SCREENSHOT_FAILED', screenshotError.message);
    }
  }
}

describe('literal input parameter integration', () => {
  before(async () => {
    await browser.url('/');
  });

  after(async () => {
    await deleteCreatedRun();
    await deleteUploadedPipeline();
  });

  it('uploads the literal-input pipeline', async () => {
    await $('#createPipelineVersionBtn').waitForDisplayed({ timeout: uiTimeout });
    await $('#createPipelineVersionBtn').click();
    await waitForHashPrefix('#/pipeline_versions/new', { timeout: uiTimeout });

    await $('#localPackageBtn').click();
    const remoteFilePath = await browser.uploadFile('./literal-input.yaml');
    await $('#dropZone input[type="file"]').addValue(remoteFilePath);
    await $('#newPipelineName').click();
    await clearDefaultInput();
    await browser.keys(pipelineName);
    await $('#createNewPipelineOrVersionBtn').click();

    await waitForHashPrefix('#/pipelines/details', { timeout: uiTimeout });
    pipelineUploaded = true;
  });

  it('opens the new run page for the uploaded pipeline', async () => {
    await $('#runsBtn').click();
    await waitForHashPrefix('#/runs', { timeout: uiTimeout });

    await $('#createNewRunBtn').waitForDisplayed({ timeout: uiTimeout });
    await $('#createNewRunBtn').click();
    await waitForHashPrefix('#/runs/new', { timeout: uiTimeout });

    await selectPipelineForRun(pipelineName, { timeout: uiTimeout });
  });

  it('renders the literal parameter as a dropdown and requires a selection before start', async () => {
    const runFormVariant = await waitForRunPageReady({ timeout: runStartTimeout });
    assert.equal(
      runFormVariant.name,
      'v2',
      'compiled literal-input pipeline should open the v2 run form',
    );
    const selectors = runFormVariant.selectors;

    await $(selectors.runName).click();
    await clearDefaultInput();
    await browser.keys(runName);

    const literalSelect = await $('//*[@role="combobox" and @id="environment"]');
    await literalSelect.waitForDisplayed({ timeout: uiTimeout });
    assert.equal(
      (await literalSelect.getText()).trim(),
      'Select a value',
      'literal dropdown should start without a selected value',
    );

    const startButton = await $('#startNewRunBtn');
    await startButton.waitForDisplayed({ timeout: uiTimeout });
    assert.equal(await startButton.isEnabled(), false, 'start should stay disabled before selection');

    await literalSelect.click();
    await $('[role="listbox"]').waitForDisplayed({ timeout: uiTimeout });
    await browser.keys('qa');
    await browser.keys('Escape');
    await $('[role="listbox"]').waitForDisplayed({ timeout: uiTimeout, reverse: true });
    assert.equal(
      (await literalSelect.getText()).trim(),
      'Select a value',
      'literal dropdown should reject arbitrary typed values',
    );
    assert.equal(
      await startButton.isEnabled(),
      false,
      'start should remain disabled after an invalid typed value attempt',
    );

    await literalSelect.click();
    await $('[role="listbox"]').waitForDisplayed({ timeout: uiTimeout });
    const optionElements = await $$('[role="option"]');
    const optionTexts = [];
    for (const optionElement of optionElements) {
      optionTexts.push(await optionElement.getText());
    }
    assert.deepEqual(optionTexts, ['dev', 'staging', 'prod'], 'literal dropdown options mismatch');

    await $(`li=${selectedLiteral}`).click();
    await waitForCondition(
      async () => (await literalSelect.getText()).trim() === selectedLiteral && (await startButton.isEnabled()),
      {
        timeout: uiTimeout,
        timeoutMsg: 'literal dropdown did not preserve the selected value or enable start',
      },
    );
  });

  it('creates a run with the selected literal value', async () => {
    await $('#startNewRunBtn').click();

    await waitForCondition(
      async () => {
        const hash = new URL(await browser.getUrl()).hash;
        return hash === '#/runs' || hash.startsWith('#/runs/details/');
      },
      {
        timeout: uiTimeout,
        timeoutMsg: 'expected run creation to land on the runs list or run details page',
      },
    );

    createdRunId = await waitForCreatedRunId();
    let fetchedRun;
    await waitForCondition(
      async () => {
        fetchedRun = await fetchRunById(createdRunId);
        return fetchedRun.ok && !!fetchedRun.run.runtime_config?.parameters;
      },
      {
        timeout: runStartTimeout,
        interval: 1000,
        timeoutMsg: 'created run did not expose runtime parameters in time',
      },
    );
    assert.equal(
      fetchedRun.run.runtime_config?.parameters?.environment,
      selectedLiteral,
      'selected literal value was not propagated to the run',
    );
  });
});

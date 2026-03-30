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
  waitForCondition,
  waitForHashPrefix,
  waitForRunPageReady,
} = require('./test-helpers');

const pipelineName = 'literal-input-pipeline-' + Date.now();
const runName = 'literal-input-run-' + Date.now();
const selectedLiteral = 'staging';
const uiTimeout = 10000;
const runStartTimeout = 30000;

async function selectPipelineForRun() {
  await $('#choosePipelineBtn').waitForDisplayed({ timeout: uiTimeout });
  await $('#choosePipelineBtn').click();

  await $('#pipelineSelectorDialog').waitForDisplayed({ timeout: uiTimeout });
  const pipelineRowSelector = buildTableRowSelector(pipelineName, {
    containerXPath: '//*[@id="pipelineSelectorDialog"]',
  });

  await waitForCondition(
    async () => (await $(pipelineRowSelector).isExisting()),
    {
      timeout: uiTimeout,
      timeoutMsg: `expected pipeline row for ${pipelineName} to appear`,
    },
  );

  await $(pipelineRowSelector).click();
  await $('#usePipelineBtn').waitForEnabled({ timeout: uiTimeout });
  await $('#usePipelineBtn').click();
  await $('#pipelineSelectorDialog').waitForDisplayed({ timeout: uiTimeout, reverse: true });
}

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

describe('literal input parameter integration', () => {
  before(async () => {
    await browser.url('/');
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
  });

  it('opens the new run page for the uploaded pipeline', async () => {
    await $('#runsBtn').click();
    await waitForHashPrefix('#/runs', { timeout: uiTimeout });

    await $('#createNewRunBtn').waitForDisplayed({ timeout: uiTimeout });
    await $('#createNewRunBtn').click();
    await waitForHashPrefix('#/runs/new', { timeout: uiTimeout });

    await selectPipelineForRun();
    await waitForRunPageReady({ timeout: runStartTimeout });
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
    const literalNativeInput = await literalSelect.$(
      './following-sibling::input[contains(@class, "MuiSelect-nativeInput")]',
    );
    assert.equal(
      await literalNativeInput.getValue(),
      '',
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
      await literalNativeInput.getValue(),
      '',
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
      async () => await startButton.isEnabled(),
      {
        timeout: uiTimeout,
        timeoutMsg: 'start button did not enable after selecting a literal value',
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

    const runId = await waitForCreatedRunId();
    let fetchedRun;
    await waitForCondition(
      async () => {
        fetchedRun = await fetchRunById(runId);
        if (!fetchedRun.ok) {
          return false;
        }
        return fetchedRun.run.runtime_config?.parameters?.environment === selectedLiteral;
      },
      {
        timeout: runStartTimeout,
        interval: 1000,
        timeoutMsg: 'selected literal value did not appear in the created run',
      },
    );
    assert.equal(
      fetchedRun.run.runtime_config?.parameters?.environment,
      selectedLiteral,
      'selected literal value was not propagated to the run',
    );
  });
});

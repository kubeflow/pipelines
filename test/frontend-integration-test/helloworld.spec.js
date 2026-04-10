// Copyright 2018-2023 The Kubeflow Authors
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
  getValueFromDetailsTable,
  saveDebugScreenshot,
  waitForCondition,
  waitForGraphNodeCount,
  waitForHashPrefix,
  waitForRunPageReady,
} = require('./test-helpers');

const experimentName = 'helloworld-experiment-' + Date.now();
const experimentDescription = 'hello world experiment description';
const pipelineName = 'helloworld-pipeline-' + Date.now();
const runName = 'helloworld-' + Date.now();
const runDescription = 'test run description ' + runName;
const runWithoutExperimentName = 'helloworld-2-' + Date.now();
const runWithoutExperimentDescription =
  'test run without experiment description ' + runWithoutExperimentName;
const uiTimeout = 5000;
const runStartTimeout = 30000;
const runCompletionTimeout = 60000;
const outputParameterValue = 'Hello world in test';

async function selectPipelineForRun() {
  await $('#choosePipelineBtn').waitForDisplayed({ timeout: uiTimeout });
  await $('#choosePipelineBtn').click();

  await $('#pipelineSelectorDialog').waitForDisplayed({ timeout: uiTimeout });
  const pipelineRowSelector = buildTableRowSelector(pipelineName, {
    containerXPath: '//*[@id="pipelineSelectorDialog"]',
  });

  try {
    await waitForCondition(
      async () => (await $(pipelineRowSelector).isExisting()),
      {
        timeout: uiTimeout,
        timeoutMsg: `expected pipeline row for ${pipelineName} to appear`,
      },
    );
  } catch (error) {
    const rowCount = await browser.execute(() => document.querySelectorAll('[data-testid="table-row"]').length);
    const emptyMessage = await browser.execute(() => {
      const emptyEl = document.querySelector('.emptyMessage');
      return emptyEl ? emptyEl.textContent : null;
    });
    console.log('PIPELINE_SELECTOR_ROW_COUNT', rowCount);
    console.log('PIPELINE_SELECTOR_EMPTY_MESSAGE', emptyMessage);
    await saveDebugScreenshot('pipeline-selector');
    throw error;
  }

  await $(pipelineRowSelector).click();
  await $('#usePipelineBtn').waitForEnabled({ timeout: uiTimeout });
  await $('#usePipelineBtn').click();
  await $('#pipelineSelectorDialog').waitForDisplayed({ timeout: uiTimeout, reverse: true });
}

async function waitForRunParameterField(selector) {
  try {
    await $(selector).waitForDisplayed({ timeout: runStartTimeout });
  } catch (error) {
    await saveDebugScreenshot('run-parameter-field');
    throw error;
  }
}

async function fillRunForm({ runName, description, message }) {
  const runFormVariant = await waitForRunPageReady({
    timeout: runStartTimeout,
    timeoutMsg: 'expected a run creation form to load',
  });
  const selectors = runFormVariant.selectors;

  await $(selectors.runName).click();
  await clearDefaultInput();
  await browser.keys(runName);

  await $(selectors.description).click();
  await browser.keys(description);

  await waitForRunParameterField(selectors.message);
  await $(selectors.message).click();
  await clearDefaultInput();
  await browser.keys(message);
}

async function waitForRunLink(runNameToFind, { timeout = runStartTimeout } = {}) {
  const runLinkSelector = `[data-testid="run-name-link"][data-run-name="${runNameToFind}"]`;

  await $('#refreshBtn').waitForDisplayed({ timeout: uiTimeout });
  await waitForCondition(
    async () => {
      if (await $(runLinkSelector).isExisting()) {
        return true;
      }
      await $('#refreshBtn').click();
      return false;
    },
    {
      timeout,
      interval: 1000,
      timeoutMsg: `waited ${timeout / 1000} seconds but run ${runNameToFind} did not start`,
    },
  );

  return runLinkSelector;
}

describe('deploy helloworld sample run', () => {
  before(async () => {
    await browser.url('/');
  });

  it('open pipeline creation page', async () => {
    await $('#createPipelineVersionBtn').click();
    await waitForHashPrefix('#/pipeline_versions/new', { timeout: uiTimeout });
  });

  it('uploads the sample pipeline', async () => {
    await $('#localPackageBtn').click();
    const remoteFilePath = await browser.uploadFile('./helloworld.yaml');
    await $('#dropZone input[type="file"]').addValue(remoteFilePath);
    await $('#newPipelineName').click();
    await clearDefaultInput();
    await browser.keys(pipelineName);
    await $('#createNewPipelineOrVersionBtn').click();
    await waitForHashPrefix('#/pipelines/details', { timeout: uiTimeout });
  });

  it('shows a 4-node static graph', async () => {
    await waitForGraphNodeCount(4, { timeout: uiTimeout });
  });

  it('creates a new experiment out of this pipeline', async () => {
    await $('#newExperimentBtn').click();
    await waitForHashPrefix('#/experiments/new', { timeout: uiTimeout });

    await $('#experimentName').setValue(experimentName);
    await $('#experimentDescription').setValue(experimentDescription);
    await $('#createExperimentBtn').click();
  });

  it('creates a new run in the experiment', async () => {
    await selectPipelineForRun();

    await fillRunForm({
      description: runDescription,
      message: outputParameterValue,
      runName,
    });

    await $('#startNewRunBtn').click();
  });

  it('redirects back to experiment page', async () => {
    await waitForHashPrefix('#/experiments/details/', { timeout: uiTimeout });
  });

  it('finds the new run in the list of runs, navigates to it', async () => {
    const runLinkSelector = await waitForRunLink(runName, { timeout: runStartTimeout });
    await $(runLinkSelector).click();
  });

  it('switches to config tab', async () => {
    await $('button=Config').waitForDisplayed({ timeout: uiTimeout });
    await $('button=Config').click();
  });

  it('waits for run to finish', async () => {
    let status = '';

    try {
      await waitForCondition(
        async () => {
          status = await getValueFromDetailsTable('Status');
          return status.trim() === 'Succeeded';
        },
        {
          timeout: runCompletionTimeout,
          interval: 1000,
          timeoutMsg: `waited for ${runCompletionTimeout / 1000} seconds but run did not succeed`,
        },
      );
    } catch (error) {
      throw new Error(`${error.message}. Current status is: ${status}`);
    }
  });

  it('displays run created at date correctly', async () => {
    const date = await getValueFromDetailsTable('Created at');
    assert(
      Date.now() - new Date(date) < 10 * 60 * 1000,
      'run created date should be within the last 10 minutes',
    );
  });

  it('displays run description inputs correctly', async () => {
    const descriptionValue = await getValueFromDetailsTable('Description');
    assert.equal(descriptionValue, runDescription, 'run description is not shown correctly');
  });

  it('displays run inputs correctly', async () => {
    const paramValue = await getValueFromDetailsTable('message');
    assert.equal(paramValue, outputParameterValue, 'run message is not shown correctly');
  });

  it('switches back to graph tab', async () => {
    await $('button=Graph').click();
  });

  it('has a 4-node graph', async () => {
    await waitForGraphNodeCount(4, { timeout: uiTimeout });
  });

  it('opens the side panel when graph node is clicked', async () => {
    await $('.graphNode').click();
    await $('button=Logs').waitForDisplayed({ timeout: uiTimeout });
  });

  it('shows logs from node', async () => {
    await $('button=Logs').click();
    await $('#logViewer').waitForDisplayed({ timeout: uiTimeout });
    await waitForCondition(
      async () => {
        const logs = await $('#logViewer').getText();
        return logs.indexOf(outputParameterValue + ' from node: ') > -1;
      },
      {
        timeout: uiTimeout,
        timeoutMsg: `expected log viewer to contain ${outputParameterValue}`,
      },
    );
  });

  it('navigates to the runs page', async () => {
    await $('#runsBtn').click();
    await waitForHashPrefix('#/runs', { timeout: uiTimeout });
  });

  it('creates a new run without selecting an experiment', async () => {
    await $('#createNewRunBtn').waitForDisplayed({ timeout: uiTimeout });
    await $('#createNewRunBtn').click();

    await selectPipelineForRun();

    await fillRunForm({
      description: runWithoutExperimentDescription,
      message: outputParameterValue,
      runName: runWithoutExperimentName,
    });

    await $('#startNewRunBtn').click();
  });

  it('lands on runs list or run details', async () => {
    await waitForCondition(
      async () => {
        const hash = new URL(await browser.getUrl()).hash;
        return hash === '#/runs' || hash.startsWith('#/runs/details/');
      },
      {
        timeout: uiTimeout,
        timeoutMsg: 'expected URL hash to equal #/runs or start with #/runs/details/',
      },
    );
  });

  it('displays both runs in all runs page', async () => {
    if (new URL(await browser.getUrl()).hash.startsWith('#/runs/details/')) {
      await $('#runsBtn').click();
      await waitForHashPrefix('#/runs', { timeout: uiTimeout });
    }

    await $('#tableFilterBox').waitForDisplayed({ timeout: uiTimeout });

    const runLinkSelector = `[data-testid="run-name-link"][data-run-name="${runName}"]`;
    await $('#tableFilterBox').click();
    await clearDefaultInput();
    await browser.keys(runName);
    await waitForCondition(
      async () => (await $(runLinkSelector).isExisting()),
      {
        timeout: uiTimeout,
        timeoutMsg: `expected run named ${runName} to appear in runs list`,
      },
    );

    const runWithoutExperimentLinkSelector =
      `[data-testid="run-name-link"][data-run-name*="${runWithoutExperimentName}"]`;
    await $('#tableFilterBox').click();
    await clearDefaultInput();
    await browser.keys(runWithoutExperimentName);
    await waitForCondition(
      async () => (await $(runWithoutExperimentLinkSelector).isExisting()),
      {
        timeout: uiTimeout,
        timeoutMsg: `expected run named ${runWithoutExperimentName} to appear in runs list`,
      },
    );
  });

  it('navigates back to the experiment list', async () => {
    await $('button=Experiments').click();
    await waitForHashPrefix('#/experiments', { timeout: uiTimeout });
  });

  it('displays both experiments in the list', async () => {
    await $('[data-testid="experiment-name-link"]').waitForDisplayed({ timeout: uiTimeout });
    const rows = (await $$('[data-testid="experiment-name-link"]')).length;
    const experimentLinkSelector =
      `[data-testid="experiment-name-link"][data-experiment-name="${experimentName}"]`;
    assert(rows >= 2, `expected at least two experiments, got: ${rows}`);
    assert(
      await $(experimentLinkSelector).isExisting(),
      `expected experiment named ${experimentName} to exist`,
    );
  });

  it('filters the experiment list', async () => {
    await $('#tableFilterBox').click();
    await clearDefaultInput();
    await browser.keys(experimentName);

    const experimentLinkSelector =
      `[data-testid="experiment-name-link"][data-experiment-name="${experimentName}"]`;
    await waitForCondition(
      async () => {
        const rows = await $$('[data-testid="experiment-name-link"]');
        return rows.length === 1 && (await $(experimentLinkSelector).isExisting());
      },
      {
        timeout: uiTimeout,
        timeoutMsg: `expected only experiment ${experimentName} to remain after filtering`,
      },
    );

    const rows = (await $$('[data-testid="experiment-name-link"]')).length;
    assert(
      rows === 1,
      'there should now be one experiment in the table, instead there are: ' + rows,
    );
  });

  it('deletes the uploaded pipeline', async () => {
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

    await $('#deletePipelinesAndPipelineVersionsBtn').click();
    await $('[role="dialog"]').waitForDisplayed({ timeout: uiTimeout });
    await $('button=Delete All').click();
    await $('[role="dialog"]').waitForDisplayed({ timeout: uiTimeout, reverse: true });
  });
});

// Copyright 2018 The Kubeflow Authors
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
const {
  buildTableRowSelector,
  clearDefaultInput,
  getValueFromDetailsTable,
  isSelectorDisplayed,
  runPhase,
  saveDebugScreenshot,
  waitForCondition,
  waitForGraphNodeCount,
  waitForHashPrefix,
  waitForRunPageReady,
} = require('./test-helpers');

const experimentName = 'tensorboard-example-experiment-' + Date.now();
const pipelineName = 'tensorboard-example-pipeline-' + Date.now();
const runName = 'tensorboard-example-' + Date.now();
const uiTimeout = 10000;
const runStartTimeout = 30000;
const runCompletionTimeout = 60000;
const tensorboardLaunchTimeout = 120000;
const tensorboardAppTimeout = 180000;
const tensorboardControlTimeout = 30000;

let pipelineUploaded = false;
let runDetailsUrl = '';
let tensorboardStarted = false;

async function waitForTensorboardControls() {
  try {
    await waitForCondition(
      async () =>
        (await isSelectorDisplayed('button=Start Tensorboard')) ||
        (await isSelectorDisplayed('button=Open Tensorboard')) ||
        (await isSelectorDisplayed('button=Stop Tensorboard')),
      {
        timeout: tensorboardControlTimeout,
        timeoutMsg: 'timed out waiting for tensorboard controls to load',
      },
    );
  } catch (error) {
    await saveDebugScreenshot('tensorboard-controls');
    throw error;
  }
}

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
    await saveDebugScreenshot('pipeline-selector');
    throw error;
  }

  await $(pipelineRowSelector).click();
  await $('#usePipelineBtn').waitForEnabled({ timeout: uiTimeout });
  await $('#usePipelineBtn').click();
  await $('#pipelineSelectorDialog').waitForDisplayed({ timeout: uiTimeout, reverse: true });
}

async function openNewRunDetails() {
  const runLinkSelector = `[data-testid="run-name-link"][data-run-name="${runName}"]`;

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
      timeout: runStartTimeout,
      interval: 1000,
      timeoutMsg: `waited ${runStartTimeout / 1000} seconds but run ${runName} did not start`,
    },
  );

  await $(runLinkSelector).click();
  await waitForHashPrefix('#/runs/details/', { timeout: uiTimeout });
  runDetailsUrl = await browser.getUrl();
}

async function waitForRunToSucceed() {
  let currentStatus = '';

  try {
    await waitForCondition(
      async () => {
        currentStatus = await getValueFromDetailsTable('Status');
        return currentStatus.trim() === 'Succeeded';
      },
      {
        timeout: runCompletionTimeout,
        interval: 1000,
        timeoutMsg: 'timed out waiting for run to succeed',
      },
    );
  } catch (error) {
    throw new Error(`${error.message}. Current status is: ${currentStatus}`);
  }
}

async function openTensorboardVisualizations() {
  await $('button=Graph').waitForDisplayed({ timeout: uiTimeout });
  await $('button=Graph').click();
  await waitForGraphNodeCount(1, { timeout: uiTimeout });

  await $('.graphNode').click();
  await $('button=Visualizations').waitForDisplayed({ timeout: uiTimeout });
  await $('button=Visualizations').click();
  await waitForTensorboardControls();
}

async function waitForOpenTensorboardButton() {
  try {
    await waitForCondition(
      async () => isSelectorDisplayed('button=Open Tensorboard'),
      {
        timeout: tensorboardLaunchTimeout,
        interval: 2000,
        timeoutMsg: 'timed out waiting for Open Tensorboard button to appear',
      },
    );
  } catch (error) {
    await saveDebugScreenshot('tensorboard-launch');
    throw error;
  }
}

async function openTensorboardApp() {
  const openTensorboardButton = await $('button=Open Tensorboard');
  await openTensorboardButton.waitForDisplayed({ timeout: uiTimeout });

  const anchor = await openTensorboardButton.$('..');
  const href = await anchor.getAttribute('href');
  assert(href, 'expected Open Tensorboard button to link to a tensorboard URL');

  await browser.url(href);

  try {
    await waitForCondition(
      async () => {
        const topBar = await $('#topBar');
        if ((await topBar.isExisting()) && (await topBar.isDisplayed())) {
          return true;
        }
        await browser.refresh();
        return false;
      },
      {
        timeout: tensorboardAppTimeout,
        interval: 2000,
        timeoutMsg: 'timed out waiting for Tensorboard top bar to become visible',
      },
    );
  } catch (error) {
    await saveDebugScreenshot('tensorboard-app');
    throw error;
  }
}

async function stopTensorboardIfRunning() {
  if (!runDetailsUrl || !tensorboardStarted) {
    return;
  }

  try {
    await browser.url(runDetailsUrl);
    await openTensorboardVisualizations();

    const stopButton = await $('button=Stop Tensorboard');
    if (!(await stopButton.isExisting())) {
      return;
    }

    await stopButton.waitForDisplayed({ timeout: uiTimeout });
    await stopButton.click();

    const dialog = await $('[role="dialog"]');
    await dialog.waitForDisplayed({ timeout: uiTimeout });
    await dialog.$('button=Stop').click();

    await waitForCondition(
      async () => isSelectorDisplayed('button=Start Tensorboard'),
      {
        timeout: tensorboardControlTimeout,
        interval: 1000,
        timeoutMsg: 'timed out waiting for Tensorboard instance to stop',
      },
    );
  } catch (error) {
    console.log('TENSORBOARD_STOP_CLEANUP_FAILED', error.message);
    try {
      await saveDebugScreenshot('tensorboard-stop-cleanup');
    } catch (screenshotError) {
      console.log('TENSORBOARD_STOP_SCREENSHOT_FAILED', screenshotError.message);
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

describe('deploy tensorboard example run', () => {
  before(async () => {
    await browser.url('/');
  });

  after(async () => {
    await stopTensorboardIfRunning();
    await deleteUploadedPipeline();
  });

  it('deploys the tensorboard example and opens tensorboard', async () => {
    await runPhase('upload tensorboard pipeline', async () => {
      await $('#createPipelineVersionBtn').waitForDisplayed({ timeout: uiTimeout });
      await $('#createPipelineVersionBtn').click();
      await waitForHashPrefix('#/pipeline_versions/new', { timeout: uiTimeout });

      await $('#localPackageBtn').waitForDisplayed({ timeout: uiTimeout });
      await $('#localPackageBtn').click();
      const remoteFilePath = await browser.uploadFile('./tensorboard-example.yaml');
      await $('#dropZone input[type="file"]').addValue(remoteFilePath);

      await $('#newPipelineName').waitForDisplayed({ timeout: uiTimeout });
      await $('#newPipelineName').click();
      await clearDefaultInput();
      await browser.keys(pipelineName);
      await $('#createNewPipelineOrVersionBtn').click();

      await waitForHashPrefix('#/pipelines/details', { timeout: uiTimeout });
      pipelineUploaded = true;
      await waitForGraphNodeCount(1, { timeout: uiTimeout });
    });

    await runPhase('create experiment', async () => {
      await $('#newExperimentBtn').waitForDisplayed({ timeout: uiTimeout });
      await $('#newExperimentBtn').click();
      await waitForHashPrefix('#/experiments/new', { timeout: uiTimeout });

      await $('#experimentName').waitForDisplayed({ timeout: uiTimeout });
      await $('#experimentName').setValue(experimentName);
      await $('#createExperimentBtn').click();
    });

    await runPhase('create run', async () => {
      await $('#choosePipelineBtn').waitForDisplayed({ timeout: uiTimeout });
      await selectPipelineForRun();
      const runFormVariant = await waitForRunPageReady({
        timeout: runStartTimeout,
        timeoutMsg: 'expected a run creation form to load',
      });

      await $(runFormVariant.selectors.runName).click();
      await clearDefaultInput();
      await browser.keys(runName);
      await $('#startNewRunBtn').click();

      await waitForHashPrefix('#/experiments/details/', { timeout: uiTimeout });
      await openNewRunDetails();
    });

    await runPhase('wait for run completion', async () => {
      await $('button=Config').waitForDisplayed({ timeout: uiTimeout });
      await $('button=Config').click();
      await waitForRunToSucceed();
    });

    await runPhase('start tensorboard', async () => {
      await openTensorboardVisualizations();

      const startTensorboardButton = await $('button=Start Tensorboard');
      await startTensorboardButton.waitForDisplayed({ timeout: uiTimeout });
      await startTensorboardButton.click();
      tensorboardStarted = true;

      // "Open Tensorboard" means the pod address exists; the app can still be warming up.
      await waitForOpenTensorboardButton();
    });

    await runPhase('open tensorboard app', async () => {
      await openTensorboardApp();
      assert(await $('#topBar').isDisplayed(), 'tensorboard top bar should be visible');
    });
  });
});

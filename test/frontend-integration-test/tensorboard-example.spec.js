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
const path = require('path');
const URL = require('url').URL;

const experimentName = 'tensorboard-example-experiment-' + Date.now();
const pipelineName = 'tensorboard-example-pipeline-' + Date.now();
const runName = 'tensorboard-example-' + Date.now();
const uiTimeout = 10000;
const runStartTimeout = 30000;
const runCompletionTimeout = 60000;
const tensorboardLaunchTimeout = 120000;
const tensorboardAppTimeout = 180000;
const tensorboardControlTimeout = 30000;
const screenshotDir = process.env.FRONTEND_INTEGRATION_SCREENSHOT_DIR || '/tmp';

let pipelineUploaded = false;
let runDetailsUrl = '';
let tensorboardStarted = false;

async function waitForCondition(condition, { timeout = uiTimeout, timeoutMsg, interval } = {}) {
  const waitOptions = { timeout };
  if (timeoutMsg) {
    waitOptions.timeoutMsg = timeoutMsg;
  }
  if (interval) {
    waitOptions.interval = interval;
  }
  await browser.waitUntil(condition, waitOptions);
}

async function getValueFromDetailsTable(key) {
  // Find the span that shows the key, get its parent div (the row), then
  // get that row's inner text, and remove the key.
  const rowText = await $(`span=${key}`).$('..').getText();
  return rowText.substr(`${key}\n`.length);
}

async function clearDefaultInput() {
  await browser.keys(['Control', 'a']);
  await browser.keys('Backspace');
}

function buildTableRowSelector(containerXPath, rowLabel) {
  return `${containerXPath}//*[@data-testid="table-row"][.//*[self::a or self::span][normalize-space()="${rowLabel}"]]`;
}

async function saveDebugScreenshot(name) {
  const screenshotPath = path.join(screenshotDir, `kfp-${name}-${Date.now()}.png`);
  await browser.saveScreenshot(screenshotPath);
  console.log('DEBUG_SCREENSHOT', screenshotPath);
}

async function isDisplayed(selector) {
  const element = await $(selector);
  return (await element.isExisting()) && (await element.isDisplayed());
}

async function waitForHashPrefix(prefix, timeout = uiTimeout) {
  await waitForCondition(
    async () => new URL(await browser.getUrl()).hash.startsWith(prefix),
    {
      timeout,
      timeoutMsg: `expected URL hash to start with ${prefix}`,
    },
  );
}

async function waitForGraphNodeCount(expectedCount, timeout = uiTimeout) {
  await waitForCondition(
    async () => (await $$('.graphNode')).length === expectedCount,
    {
      timeout,
      timeoutMsg: `expected ${expectedCount} graph node(s) to be visible`,
    },
  );
  const nodes = await $$('.graphNode');
  assert(
    nodes.length === expectedCount,
    `should have a ${expectedCount}-node graph, instead has: ${nodes.length}`,
  );
}

async function waitForTensorboardControls() {
  try {
    await waitForCondition(
      async () =>
        (await isDisplayed('button=Start Tensorboard')) ||
        (await isDisplayed('button=Open Tensorboard')) ||
        (await isDisplayed('button=Stop Tensorboard')),
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
  const pipelineRowSelector = buildTableRowSelector(
    '//*[@id="pipelineSelectorDialog"]',
    pipelineName,
  );

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

async function selectPipelineVersionForRun() {
  const pipelineVersionRowsSelector = '#pipelineVersionSelectorDialog [data-testid="table-row"]';

  await $('#choosePipelineVersionBtn').waitForDisplayed({ timeout: uiTimeout });
  await $('#choosePipelineVersionBtn').click();
  await $('#pipelineVersionSelectorDialog').waitForDisplayed({ timeout: uiTimeout });

  await waitForCondition(
    async () => (await $$(pipelineVersionRowsSelector)).length > 0,
    {
      timeout: uiTimeout,
      timeoutMsg: 'expected at least one pipeline version row to appear',
    },
  );

  const pipelineVersionRows = await $$(pipelineVersionRowsSelector);
  assert(pipelineVersionRows.length > 0, 'expected at least one pipeline version row');
  await pipelineVersionRows[0].click();
  await $('#usePipelineVersionBtn').waitForEnabled({ timeout: uiTimeout });
  await $('#usePipelineVersionBtn').click();
  await $('#pipelineVersionSelectorDialog').waitForDisplayed({
    timeout: uiTimeout,
    reverse: true,
  });
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
  await waitForHashPrefix('#/runs/details/', uiTimeout);
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
  await waitForGraphNodeCount(1, uiTimeout);

  await $('.graphNode').click();
  await $('button=Visualizations').waitForDisplayed({ timeout: uiTimeout });
  await $('button=Visualizations').click();
  await waitForTensorboardControls();
}

async function waitForOpenTensorboardButton() {
  try {
    await waitForCondition(
      async () => await isDisplayed('button=Open Tensorboard'),
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
      async () => await isDisplayed('button=Start Tensorboard'),
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
    await waitForHashPrefix('#/pipelines', uiTimeout);

    await $('#tableFilterBox').waitForDisplayed({ timeout: uiTimeout });
    await $('#tableFilterBox').click();
    await clearDefaultInput();
    await browser.keys(pipelineName);

    const pipelineRowSelector = buildTableRowSelector('', pipelineName);
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
    await $('#createPipelineVersionBtn').waitForDisplayed({ timeout: uiTimeout });
    await $('#createPipelineVersionBtn').click();
    await waitForHashPrefix('#/pipeline_versions/new', uiTimeout);

    await $('#localPackageBtn').waitForDisplayed({ timeout: uiTimeout });
    await $('#localPackageBtn').click();
    const remoteFilePath = await browser.uploadFile('./tensorboard-example.yaml');
    await $('#dropZone input[type="file"]').addValue(remoteFilePath);

    await $('#newPipelineName').waitForDisplayed({ timeout: uiTimeout });
    await $('#newPipelineName').click();
    await clearDefaultInput();
    await browser.keys(pipelineName);
    await $('#createNewPipelineOrVersionBtn').click();

    await waitForHashPrefix('#/pipelines/details', uiTimeout);
    pipelineUploaded = true;
    await waitForGraphNodeCount(1, uiTimeout);

    await $('#newExperimentBtn').waitForDisplayed({ timeout: uiTimeout });
    await $('#newExperimentBtn').click();
    await waitForHashPrefix('#/experiments/new', uiTimeout);

    await $('#experimentName').waitForDisplayed({ timeout: uiTimeout });
    await $('#experimentName').setValue(experimentName);
    await $('#createExperimentBtn').click();

    await $('#choosePipelineBtn').waitForDisplayed({ timeout: uiTimeout });
    await selectPipelineForRun();
    await selectPipelineVersionForRun();

    await $('#runNameInput').waitForDisplayed({ timeout: uiTimeout });
    await $('#runNameInput').click();
    await clearDefaultInput();
    await browser.keys(runName);
    await $('#startNewRunBtn').click();

    await waitForHashPrefix('#/experiments/details/', uiTimeout);
    await openNewRunDetails();

    await $('button=Config').waitForDisplayed({ timeout: uiTimeout });
    await $('button=Config').click();
    await waitForRunToSucceed();

    await openTensorboardVisualizations();

    const startTensorboardButton = await $('button=Start Tensorboard');
    await startTensorboardButton.waitForDisplayed({ timeout: uiTimeout });
    await startTensorboardButton.click();
    tensorboardStarted = true;

    // "Open Tensorboard" means the pod address exists; the app can still be warming up.
    await waitForOpenTensorboardButton();
    await openTensorboardApp();

    assert(await $('#topBar').isDisplayed(), 'tensorboard top bar should be visible');
  });
});

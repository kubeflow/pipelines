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
const URL = require('url').URL;

const experimentName = 'tensorboard-example-experiment-' + Date.now();
const pipelineName = 'tensorboard-example-pipeline-' + Date.now();
const runName = 'tensorboard-example-' + Date.now();
const waitTimeout = 5000;

async function getValueFromDetailsTable(key) {
  // Find the span that shows the key, get its parent div (the row), then
  // get that row's inner text, and remove the key
  const rowText = await $(`span=${key}`).$('..').getText();
  return rowText.substr(`${key}\n`.length);
}

async function clearDefaultInput() {
  await browser.keys(['Control', 'a']);
  await browser.keys('Backspace');
}

describe('deploy tensorboard example run', () => {
  before(async () => {
    await browser.url('/');
  });

  it('opens the pipeline creation page', async () => {
    await $('#createPipelineVersionBtn').click();
    await browser.waitUntil(async () => {
      return new URL(await browser.getUrl()).hash.startsWith('#/pipeline_versions/new');
    }, waitTimeout);
  });

  it('uploads the tensorboard sample pipeline', async () => {
    await $('#localPackageBtn').click();
    const remoteFilePath = await browser.uploadFile('./tensorboard-example.yaml');
    await $('#dropZone input[type="file"]').addValue(remoteFilePath);

    await $('#newPipelineName').click();
    await clearDefaultInput();
    await browser.keys(pipelineName);
    await $('#createNewPipelineOrVersionBtn').click();

    await browser.waitUntil(async () => {
      return new URL(await browser.getUrl()).hash.startsWith('#/pipelines/details');
    }, waitTimeout);
  });

  it('shows a 1-node static graph', async () => {
    const nodeSelector = '.graphNode';
    await browser.waitUntil(async () => (await $$(nodeSelector)).length === 1, waitTimeout);
    const nodes = await $$(nodeSelector);
    assert(nodes.length === 1, 'should have a 1-node graph, instead has: ' + nodes.length);
  });

  it('creates a new experiment from this pipeline', async () => {
    await $('#newExperimentBtn').click();
    await browser.waitUntil(async () => {
      return new URL(await browser.getUrl()).hash.startsWith('#/experiments/new');
    }, waitTimeout);

    await $('#experimentName').setValue(experimentName);
    await $('#createExperimentBtn').click();
  });

  it('creates a new run in the experiment', async () => {
    await $('#choosePipelineBtn').waitForDisplayed();
    await $('#choosePipelineBtn').click();

    await $('#pipelineSelectorDialog').waitForDisplayed({ timeout: waitTimeout });
    await browser.waitUntil(
      async () => (await $$('[data-testid="table-row"]')).length > 0,
      waitTimeout,
      'expected at least one pipeline row to appear',
    );
    const pipelineRows = await $$('[data-testid="table-row"]');
    assert(pipelineRows.length > 0, 'expected at least one pipeline row');
    await pipelineRows[0].click();
    await $('#usePipelineBtn').click();

    await $('#pipelineSelectorDialog').waitForDisplayed({ timeout: waitTimeout, reverse: true });

    await $('#choosePipelineVersionBtn').waitForDisplayed();
    await $('#choosePipelineVersionBtn').click();

    await $('#pipelineVersionSelectorDialog').waitForDisplayed({ timeout: waitTimeout });
    await browser.waitUntil(
      async () => (await $$('[data-testid="table-row"]')).length > 0,
      waitTimeout,
      'expected at least one pipeline version row to appear',
    );
    const pipelineVersionRows = await $$('[data-testid="table-row"]');
    assert(pipelineVersionRows.length > 0, 'expected at least one pipeline version row');
    await pipelineVersionRows[0].click();
    await $('#usePipelineVersionBtn').click();

    await $('#pipelineVersionSelectorDialog').waitForDisplayed({
      timeout: waitTimeout,
      reverse: true,
    });

    await $('#runNameInput').click();
    await clearDefaultInput();
    await browser.keys(runName);

    await $('#startNewRunBtn').click();
  });

  it('redirects back to experiment page', async () => {
    await browser.waitUntil(async () => {
      return new URL(await browser.getUrl()).hash.startsWith('#/experiments/details/');
    }, waitTimeout);
  });

  it('finds the new run in the list of runs and navigates to it', async () => {
    let attempts = 30;
    const runLinkSelector = `[data-testid="run-name-link"][data-run-name="${runName}"]`;

    while (attempts && !(await $(runLinkSelector).isExisting())) {
      await browser.pause(1000);
      await $('#refreshBtn').click();
      --attempts;
    }

    assert(attempts, 'waited for 30 seconds but run did not start.');
    await $(runLinkSelector).click();
  });

  it('switches to config tab', async () => {
    await $('button=Config').waitForDisplayed({ timeout: waitTimeout });
    await $('button=Config').click();
  });

  it('waits for run to finish', async () => {
    let status = await getValueFromDetailsTable('Status');

    let attempts = 0;
    const maxAttempts = 60;

    while (attempts < maxAttempts && status.trim() !== 'Succeeded') {
      await browser.pause(1000);
      status = await getValueFromDetailsTable('Status');
      attempts++;
    }

    assert(
      attempts < maxAttempts,
      `waited for ${maxAttempts} seconds but run did not succeed. ` + 'Current status is: ' + status,
    );
  });

  it('switches back to graph tab', async () => {
    await $('button=Graph').click();
  });

  it('has a 1-node graph', async () => {
    const nodes = (await $$('.graphNode')).length;
    assert(nodes === 1, 'should have a 1-node graph, instead has: ' + nodes);
  });

  it('opens the side panel when graph node is clicked', async () => {
    await $('.graphNode').click();
    await $('.plotCard').waitForDisplayed({ timeout: waitTimeout });
  });

  it('starts tensorboard from the plot card', async () => {
    const button = (await $$('.plotCard button'))[1];
    await button.waitForDisplayed();
    assert((await button.getText()).trim() === 'Start Tensorboard');
    await button.click();
  });

  it('waits until the button turns into Open Tensorboard', async () => {
    await browser.waitUntil(
      async () => {
        const button = (await $$('.plotCard button'))[1];
        await button.waitForDisplayed();
        return (await button.getText()).trim() === 'Open Tensorboard';
      },
      2 * 60 * 1000,
      'timed out waiting for Tensorboard app to become ready',
    );
  });

  it('opens the tensorboard app', async () => {
    const anchor = await $('.plotCard a');
    const href = await anchor.getAttribute('href');
    await browser.url(href);

    let attempts = 0;
    const maxAttempts = 60;
    while (attempts < maxAttempts && !(await $('#topBar').isExisting())) {
      await browser.pause(1000);
      await browser.refresh();
      attempts++;
    }

    assert(await $('#topBar').isDisplayed(), 'tensorboard top bar should be visible');
    await browser.back();
  });

  it('deletes the uploaded pipeline', async () => {
    await $('#pipelinesBtn').click();
    await browser.waitUntil(async () => {
      return new URL(await browser.getUrl()).hash.startsWith('#/pipelines');
    }, waitTimeout);

    await $('#tableFilterBox').waitForDisplayed();
    await $('#tableFilterBox').click();
    await clearDefaultInput();
    await browser.keys(pipelineName);

    const pipelineRowSelector =
      `//*[@data-testid="table-row"][.//a[normalize-space()="${pipelineName}"]]`;

    await browser.waitUntil(
      async () => (await $(pipelineRowSelector).isExisting()),
      waitTimeout,
      `expected pipeline row for ${pipelineName} after filtering`,
    );
    await $(pipelineRowSelector).click();

    await $('#deleteBtn').click();
    await $('[role="dialog"]').waitForDisplayed({ timeout: waitTimeout });
    await $('button=Delete').click();
    await $('[role="dialog"]').waitForDisplayed({ timeout: waitTimeout, reverse: true });
  });
});

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

const experimentName = 'helloworld-experiment-' + Date.now();
const experimentDescription = 'hello world experiment description';
const pipelineName = 'helloworld-pipeline-' + Date.now();
const runName = 'helloworld-' + Date.now();
const runDescription = 'test run description ' + runName;
const runWithoutExperimentName = 'helloworld-2-' + Date.now();
const runWithoutExperimentDescription =
  'test run without experiment description ' + runWithoutExperimentName;
const waitTimeout = 5000;
const outputParameterValue = 'Hello world in test';

async function getValueFromDetailsTable(key) {
  // Find the span that shows the key, get its parent div (the row), then
  // get that row's inner text, and remove the key
  const rowText = await $(`span=${key}`).$('..').getText();
  return rowText.substr(`${key}\n`.length);
}

async function clearDefaultInput() {
  await browser.keys(['Control', 'a'])
  await browser.keys('Backspace');
}

describe('deploy helloworld sample run', () => {
  before(async () => {
    await browser.url('/');
  });

  it('open pipeline creation page', async () => {
    await $('#createPipelineVersionBtn').click();
    await browser.waitUntil(async () => {
      return new URL(await browser.getUrl()).hash.startsWith('#/pipeline_versions/new');
    }, waitTimeout);
  });

  it('uploads the sample pipeline', async () => {
    await $('#localPackageBtn').click();
    const remoteFilePath = await browser.uploadFile('./helloworld.yaml');
    await $('#dropZone input[type="file"]').addValue(remoteFilePath);
    await $('#newPipelineName').click();
    await clearDefaultInput()
    await browser.keys(pipelineName)
    await $('#createNewPipelineOrVersionBtn').click();
    await browser.waitUntil(async () => {
      return new URL(await browser.getUrl()).hash.startsWith('#/pipelines/details');
    }, waitTimeout);
  });

  it('shows a 4-node static graph', async () => {
    const nodeSelector = '.graphNode';
    await $(nodeSelector);
    const nodes = await $$(nodeSelector);
    assert(nodes.length === 4, 'should have a 4-node graph, instead has: ' + nodes.length);
  });

  it('creates a new experiment out of this pipeline', async () => {
    await $('#newExperimentBtn').click();
    await browser.waitUntil(async () => {
      return new URL(await browser.getUrl()).hash.startsWith('#/experiments/new');
    }, waitTimeout);

    await $('#experimentName').setValue(experimentName);
    await $('#experimentDescription').setValue(experimentDescription);

    await $('#createExperimentBtn').click();
  });

  it('creates a new run in the experiment', async () => {
    await $('#choosePipelineBtn').waitForDisplayed();
    await $('#choosePipelineBtn').click();

    await $('.tableRow').waitForDisplayed();
    await $('.tableRow').click();

    await $('#usePipelineBtn').click();

    await $('#pipelineSelectorDialog').waitForDisplayed({ timeout: waitTimeout, reverse: true });

    await $('#choosePipelineVersionBtn').waitForDisplayed();
    await $('#choosePipelineVersionBtn').click();

    await $('.tableRow').waitForDisplayed();
    await $('.tableRow').click();

    await $('#usePipelineVersionBtn').click();

    await $('#pipelineVersionSelectorDialog').waitForDisplayed({
      timeout: waitTimeout,
      reverse: true,
    });

    await $('#runNameInput').click();
    await clearDefaultInput()
    await browser.keys(runName);

    await $('#descriptionInput').click();
    await browser.keys(runDescription);
    
    // the parameter name is "message" in this testing pipeline 
    await $('input#newRunPipelineParam0').click();
    await clearDefaultInput()
    await browser.keys(outputParameterValue);

    // Deploy
    await $('#startNewRunBtn').click();
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('redirects back to experiment page', async () => {
    await browser.waitUntil(async () => {
      return new URL(await browser.getUrl()).hash.startsWith('#/experiments/details/');
    }, waitTimeout);
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('finds the new run in the list of runs, navigates to it', async () => {
    let attempts = 30;

    // Wait for a reasonable amount of time until the run starts
    while (attempts && !$('.tableRow a').isExisting()) {
      await browser.pause(1000);
      await $('#refreshBtn').click();
      --attempts;
    }

    assert(attempts, 'waited for 30 seconds but run did not start.');

    assert.equal(await $$('.tableRow').length, 1, 'should only show one run');

    // Navigate to details of the deployed run by clicking its anchor element
    await browser.execute('document.querySelector(".tableRow a").click()');
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('switches to config tab', async () => {
    await $('button=Config').waitForDisplayed({ timeout: waitTimeout });
    await $('button=Config').click();
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('waits for run to finish', async () => {
    let status = await getValueFromDetailsTable('Status');

    let attempts = 0;
    const maxAttempts = 60;

    // Wait for a reasonable amount of time until the run is done
    while (attempts < maxAttempts && status.trim() !== 'Succeeded') {
      await browser.pause(1000);
      status = await getValueFromDetailsTable('Status');
      attempts++;
    }

    assert(
      attempts < maxAttempts,
      `waited for ${maxAttempts} seconds but run did not succeed. ` +
        'Current status is: ' +
        status,
    );
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('displays run created at date correctly', async () => {
    const date = await getValueFromDetailsTable('Created at');
    assert(
      Date.now() - new Date(date) < 10 * 60 * 1000,
      'run created date should be within the last 10 minutes',
    );
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('displays run description inputs correctly', async () => {
    const descriptionValue = await getValueFromDetailsTable('Description');
    assert.equal(descriptionValue, runDescription, 'run description is not shown correctly');
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('displays run inputs correctly', async () => {
    const paramValue = await getValueFromDetailsTable('message');
    assert.equal(paramValue, outputParameterValue, 'run message is not shown correctly');
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('switches back to graph tab', async () => {
    await $('button=Graph').click();
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('has a 4-node graph', async () => {
    const nodeSelector = '.graphNode';
    const nodes = await $$(nodeSelector).length;
    assert(nodes === 4, 'should have a 4-node graph, instead has: ' + nodes);
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('opens the side panel when graph node is clicked', async () => {
    await $('.graphNode').click();
    await browser.pause(1000);
    await $('button=Logs').waitForDisplayed();
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('shows logs from node', async () => {
    await $('button=Logs').click();
    await $('#logViewer').waitForDisplayed();
    await browser.waitUntil(async () => {
      const logs = await $('#logViewer').getText();
      return logs.indexOf(outputParameterValue + ' from node: ') > -1;
    }, waitTimeout);
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('navigates to the runs page', async () => {
    await $('#runsBtn').click();
    await browser.waitUntil(async () => {
      return new URL(await browser.getUrl()).hash.startsWith('#/runs');
    }, waitTimeout);
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('creates a new run without selecting an experiment', async () => {
    await $('#createNewRunBtn').waitForDisplayed();
    await $('#createNewRunBtn').click();

    await $('#choosePipelineBtn').waitForDisplayed();
    await $('#choosePipelineBtn').click();

    await $('.tableRow').waitForDisplayed();
    await $('.tableRow').click();

    await $('#usePipelineBtn').click();

    await $('#pipelineSelectorDialog').waitForDisplayed({ timeout: waitTimeout, reverse: true });

    await $('#runNameInput').click();
    await browser.keys(runWithoutExperimentName);

    await $('#descriptionInput').click();
    await browser.keys(runWithoutExperimentDescription);
    
    await $('input#newRunPipelineParam0').click();
    await clearDefaultInput()
    await browser.keys(outputParameterValue);

    // Deploy
    await $('#startNewRunBtn').click();
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('redirects back to all runs page', async () => {
    await browser.waitUntil(
      async () => {
        return new URL(await browser.getUrl()).hash === '#/runs';
      },
      waitTimeout,
      `URL was: ${new URL(await browser.getUrl())}`,
    );
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('displays both runs in all runs page', async () => {
    await $('.tableRow').waitForDisplayed();
    const rows = await $$('.tableRow').length;
    assert(rows === 2, 'there should now be two runs in the table, instead there are: ' + rows);
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('navigates back to the experiment list', async () => {
    await $('button=Experiments').click();
    await browser.waitUntil(async () => {
      return new URL(await browser.getUrl()).hash.startsWith('#/experiments');
    }, waitTimeout);
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('displays both experiments in the list', async () => {
    await $('.tableRow').waitForDisplayed();
    const rows = await $$('.tableRow').length;
    assert(
      rows === 2,
      'there should now be two experiments in the table, instead there are: ' + rows,
    );
  });

  // Skipped while https://github.com/kubeflow/pipelines/issues/10881 is not resolved
  it.skip('filters the experiment list', async () => {
    // Enter "hello" into filter bar
    await $('#tableFilterBox').click();
    await browser.keys(experimentName.substring(0, 5));
    // Wait for the list to refresh
    await browser.pause(2000);

    await $('.tableRow').waitForDisplayed();
    const rows = await $$('.tableRow').length;
    assert(
      rows === 1,
      'there should now be one experiment in the table, instead there are: ' + rows,
    );
  });

  //TODO: enable this after we change the pipeline to a unique name such that deleting this
  // pipeline will not jeopardize the concurrent basic e2e tests.
  // it('deletes the uploaded pipeline', async () => {
  //   await $('#pipelinesBtn').click();
  //
  //   await $('.tableRow').waitForDisplayed({timeout: waitTimeout});
  //   await $('.tableRow').click();
  //   await $('#deleteBtn').click();
  //   await $('.dialogButton').click();
  //   await $('.dialog').waitForDisplayed({timeout: waitTimeout, reverse:true});
  // });
});

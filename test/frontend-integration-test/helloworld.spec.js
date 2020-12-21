// Copyright 2018 Google LLC
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
const runWithoutExperimentDescription = 'test run without experiment description ' + runWithoutExperimentName;
const waitTimeout = 5000;
const outputParameterValue = 'Hello world in test'

function getValueFromDetailsTable(key) {
  // Find the span that shows the key, get its parent div (the row), then
  // get that row's inner text, and remove the key
  const row = $(`span=${key}`).$('..');
  return row.getText().substr(`${key}\n`.length);
}

describe('deploy helloworld sample run', () => {

  before(() => {
    browser.url('/');
  });

  it('open pipeline creation page', () => {
    $('#createPipelineVersionBtn').click();
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).hash.startsWith('#/pipeline_versions/new');
    }, waitTimeout);
  });

  it('uploads the sample pipeline', () => {
    $('#localPackageBtn').click();
    browser.chooseFile('#dropZone input[type="file"]', './helloworld.yaml');
    $('#newPipelineName').setValue(pipelineName);
    $('#createNewPipelineOrVersionBtn').click();
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).hash.startsWith('#/pipelines/details');
    }, waitTimeout);
  });

  it('shows a 4-node static graph', () => {
    const nodeSelector = '.graphNode';
    $(nodeSelector).waitForVisible();
    const nodes = $$(nodeSelector).length;
    assert(nodes === 4, 'should have a 4-node graph, instead has: ' + nodes);
  });

  it('creates a new experiment out of this pipeline', () => {
    $('#newExperimentBtn').click();
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).hash.startsWith('#/experiments/new');
    }, waitTimeout);

    $('#experimentName').setValue(experimentName);
    $('#experimentDescription').setValue(experimentDescription);

    $('#createExperimentBtn').click();
  });

  it('creates a new run in the experiment', () => {
    $('#choosePipelineBtn').waitForVisible();
    $('#choosePipelineBtn').click();

    $('.tableRow').waitForVisible();
    $('.tableRow').click();

    $('#usePipelineBtn').click();

    $('#pipelineSelectorDialog').waitForVisible(waitTimeout, true);

    $('#choosePipelineVersionBtn').waitForVisible();
    $('#choosePipelineVersionBtn').click();

    $('.tableRow').waitForVisible();
    $('.tableRow').click();

    $('#usePipelineVersionBtn').click();

    $('#pipelineVersionSelectorDialog').waitForVisible(waitTimeout, true);

    browser.keys('Tab');
    browser.keys(runName);

    browser.keys('Tab');
    browser.keys(runDescription);

    // Skip over "choose experiment" button
    browser.keys('Tab');
    // Skip over service account help button
    browser.keys('Tab');
    // Skip over "service account" textbox
    browser.keys('Tab');
    // Skip over "Run Type" radio button
    browser.keys('Tab');

    browser.keys('Tab');
    browser.keys(outputParameterValue);

    // Deploy
    $('#startNewRunBtn').click();
  });

  it('redirects back to experiment page', () => {
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).hash.startsWith('#/experiments/details/');
    }, waitTimeout);
  });

  it('finds the new run in the list of runs, navigates to it', () => {
    let attempts = 30;

    // Wait for a reasonable amount of time until the run starts
    while (attempts && !$('.tableRow a').isExisting()) {
      browser.pause(1000);
      $('#refreshBtn').click();
      --attempts;
    }

    assert(attempts, 'waited for 30 seconds but run did not start.');

    assert.equal($$('.tableRow').length, 1, 'should only show one run');

    // Navigate to details of the deployed run by clicking its anchor element
    browser.execute('document.querySelector(".tableRow a").click()');
  });

  it('switches to config tab', () => {
    $('button=Config').waitForVisible(waitTimeout);
    $('button=Config').click();
  });

  it('waits for run to finish', () => {
    let status = getValueFromDetailsTable('Status');

    let attempts = 0;
    const maxAttempts = 60;

    // Wait for a reasonable amount of time until the run is done
    while (attempts < maxAttempts && status.trim() !== 'Succeeded') {
      browser.pause(1000);
      status = getValueFromDetailsTable('Status');
      attempts++;
    }

    assert(attempts < maxAttempts, `waited for ${maxAttempts} seconds but run did not succeed. ` +
      'Current status is: ' + status);
  });

  it('displays run created at date correctly', () => {
    const date = getValueFromDetailsTable('Created at');
    assert(Date.now() - new Date(date) < 10 * 60 * 1000,
      'run created date should be within the last 10 minutes');
  });

  it('displays run inputs correctly', () => {
    const paramValue = getValueFromDetailsTable('message');
    assert.equal(paramValue, outputParameterValue, 'run message is not shown correctly');
  });

  it('switches back to graph tab', () => {
    $('button=Graph').click();
  });

  it('has a 4-node graph', () => {
    const nodeSelector = '.graphNode';
    const nodes = $$(nodeSelector).length;
    assert(nodes === 4, 'should have a 4-node graph, instead has: ' + nodes);
  });

  it('opens the side panel when graph node is clicked', () => {
    $('.graphNode').click();
    browser.pause(1000);
    $('button=Logs').waitForVisible();
  });

  it('shows logs from node', () => {
    $('button=Logs').click();
    $('#logViewer').waitForVisible();
    browser.waitUntil(() => {
      const logs = $('#logViewer').getText();
      return logs.indexOf(outputParameterValue + ' from node: ') > -1;
    }, waitTimeout);
  });

  it('navigates to the runs page', () => {
    $('#runsBtn').click();
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).hash.startsWith('#/runs');
    }, waitTimeout);
  });

  it('creates a new run without selecting an experiment', () => {
    $('#createNewRunBtn').waitForVisible();
    $('#createNewRunBtn').click();

    $('#choosePipelineBtn').waitForVisible();
    $('#choosePipelineBtn').click();

    $('.tableRow').waitForVisible();
    $('.tableRow').click();

    $('#usePipelineBtn').click();

    $('#pipelineSelectorDialog').waitForVisible(waitTimeout, true);

    browser.keys('Tab');
    browser.keys(runWithoutExperimentName);

    browser.keys('Tab');
    browser.keys(runWithoutExperimentDescription);

    // Skip over "choose experiment" button
    browser.keys('Tab');
    // Skip over service account help button
    browser.keys('Tab');
    // Skip over "service account" textbox
    browser.keys('Tab');
    // Skip over "Run Type" radio button
    browser.keys('Tab');

    browser.keys('Tab');
    browser.keys(outputParameterValue);

    // Deploy
    $('#startNewRunBtn').click();
  });

  it('redirects back to all runs page', () => {
    browser.waitUntil(() => {
      return (new URL(browser.getUrl())).hash === '#/runs';
    }, waitTimeout, `URL was: ${new URL(browser.getUrl())}`);
  });

  it('displays both runs in all runs page', () => {
    $('.tableRow').waitForVisible();
    const rows = $$('.tableRow').length;
    assert(rows === 2, 'there should now be two runs in the table, instead there are: ' + rows);
  });

  it('navigates back to the experiment list', () => {
    $('button=Experiments').click();
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).hash.startsWith('#/experiments');
    }, waitTimeout);
  });

  it('displays both experiments in the list', () => {
    $('.tableRow').waitForVisible();
    const rows = $$('.tableRow').length;
    assert(rows === 2, 'there should now be two experiments in the table, instead there are: ' + rows);
  });

  it('filters the experiment list', () => {
    // Enter "hello" into filter bar
    browser.click('#tableFilterBox');
    browser.keys(experimentName.substring(0, 5));
    // Wait for the list to refresh
    browser.pause(2000);

    $('.tableRow').waitForVisible();
    const rows = $$('.tableRow').length;
    assert(rows === 1, 'there should now be one experiment in the table, instead there are: ' + rows);
  });

  //TODO: enable this after we change the pipeline to a unique name such that deleting this
  // pipeline will not jeopardize the concurrent basic e2e tests.
  // it('deletes the uploaded pipeline', () => {
  //   $('#pipelinesBtn').click();
  //
  //   browser.waitForVisible('.tableRow', waitTimeout);
  //   $('.tableRow').click();
  //   $('#deleteBtn').click();
  //   $('.dialogButton').click();
  //   $('.dialog').waitForVisible(waitTimeout, true);
  // });
});

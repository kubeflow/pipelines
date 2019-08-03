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

const experimentName = 'tensorboard-example-experiment-' + Date.now();
const pipelineName = 'tensorboard-example-pipeline-' + Date.now();
const runName = 'tensorboard-example-' + Date.now();
const waitTimeout = 5000;

function getValueFromDetailsTable(key) {
  // Find the span that shows the key, get its parent div (the row), then
  // get that row's inner text, and remove the key
  const row = $(`span=${key}`).$('..');
  return row.getText().substr(`${key}\n`.length);
}

describe('deploy tensorboard example run', () => {

  before(() => {
    browser.url('/');
  });

  it('opens the pipeline upload dialog', () => {
    $('#uploadBtn').click();
    browser.waitForVisible('#uploadDialog', waitTimeout);
  });

  it('uploads the sample pipeline', () => {
    browser.chooseFile('#uploadDialog input[type="file"]', './tensorboard-example.yaml');
    const input = $('#uploadDialog #uploadFileName');
    input.clearElement();
    input.setValue(pipelineName);
    $('#confirmUploadBtn').click();
    browser.waitForVisible('#uploadDialog', waitTimeout, true);
  });

  it('opens pipeline details', () => {
    $('.tableRow a').waitForVisible(waitTimeout);
    browser.execute('document.querySelector(".tableRow a").click()');
  });

  it('shows a 1-node static graph', () => {
    const nodeSelector = '.graphNode';
    $(nodeSelector).waitForVisible();
    const nodes = $$(nodeSelector).length;
    assert(nodes === 1, 'should have a 1-node graph, instead has: ' + nodes);
  });

  it('creates a new experiment out of this pipeline', () => {
    $('#startNewExperimentBtn').click();
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).hash.startsWith('#/experiments/new');
    }, waitTimeout);

    $('#experimentName').setValue(experimentName);
    $('#createExperimentBtn').click();
  });

  it('creates a new run in the experiment', () => {
    $('#choosePipelineBtn').waitForVisible();
    $('#choosePipelineBtn').click();

    $('.tableRow').waitForVisible();
    $('.tableRow').click();

    $('#usePipelineBtn').click();

    $('#pipelineSelectorDialog').waitForVisible(waitTimeout, true);

    browser.keys('Tab');
    browser.keys(runName);

    // Deploy
    $('#createBtn').click();
  });

  it('redirects back to experiment page', () => {
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).hash.startsWith('#/experiments/details/');
    }, waitTimeout);
  });

  it('finds the new run in the list of runs, navigates to it', () => {
    $('.tableRow').waitForVisible(waitTimeout);
    assert.equal($$('.tableRow').length, 1, 'should only show one run');

    // Navigate to details of the deployed run by clicking its anchor element
    $('.tableRow a').waitForVisible(waitTimeout);
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

  it('switches back to graph tab', () => {
    $('button=Graph').click();
  });

  it('has a 1-node graph', () => {
    const nodeSelector = '.graphNode';
    const nodes = $$(nodeSelector).length;
    assert(nodes === 1, 'should have a 1-node graph, instead has: ' + nodes);
  });

  it('opens the side panel when graph node is clicked', () => {
    $('.graphNode').click();
    $('.plotCard').waitForVisible(waitTimeout);
  });

  it('shows a Tensorboard plot card, and clicks its button to start Tensorboard', () => {
    // First button is the popout button, second is the Tensoboard start button
    const button = $$('.plotCard button')[1];
    button.waitForVisible();
    assert(button.getText().trim() === 'Start Tensorboard');
    button.click();
  });

  it('waits until the button turns into Open Tensorboard', () => {
    // First button is the popout button, second is the Tensoboard open button
    browser.waitUntil(() => {
      const button = $$('.plotCard button')[1];
      button.waitForVisible();
      return button.getText().trim() === 'Open Tensorboard';
    }, 2 * 60 * 1000);
  });

  it('opens the Tensorboard app', () => {
    const anchor = $('.plotCard a');
    browser.url(anchor.getAttribute('href'));

    let attempts = 0;
    const maxAttempts = 60;

    // Wait for a reasonable amount of time until Tensorboard app shows up
    while (attempts < maxAttempts && !$('#topBar').isExisting()) {
      browser.pause(1000);
      browser.refresh();
      attempts++;
    }

    assert($('#topBar').isVisible());
    browser.back();
  });

  it('deletes the uploaded pipeline', () => {
    $('#pipelinesBtn').click();

    browser.waitForVisible('.tableRow', waitTimeout);
    $('.tableRow').click();
    $('#deleteBtn').click();
    $('.dialogButton').click();
    $('.dialog').waitForVisible(waitTimeout, true);
  });
});

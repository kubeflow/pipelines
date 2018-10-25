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

const jobName = 'helloworld-' + Date.now();
const jobDescription = 'test job description ' + jobName;
const waitTimeout = 5000;
const outputParameterValue = 'Hello world in test'

function getValueFromDetailsTable(key) {
  // Find the span that shows the key, get its parent div (the row), then
  // get that row's inner text, and remove the key
  const row = $(`span=${key}`).$('..');
  return row.getText().substr(`${key}\n`.length);
}

describe('deploy helloworld sample job', () => {

  before(() => {
    browser.url('/');
  });

  it('opens the pipeline upload dialog', () => {
    $('#uploadBtn').click();
    browser.waitForVisible('#uploadDialog', waitTimeout);
  });

  it('uploads the sample pipeline', () => {
    browser.chooseFile('#uploadDialog input', './helloworld.yaml');
    const input = $('#uploadDialog input[type="text"]');
    input.clearElement();
    input.setValue('helloworld-pipeline');
    $('#confirmUploadBtn').click();
    browser.waitForVisible('#uploadDialog', waitTimeout, true);
  });

  it('uses the uploaded pipeline to create a new job', () => {
    $('.tableRow').waitForVisible(waitTimeout);
    $('.tableRow').click();
    $('#createJobBtn').click();
  });

  it('populates job details and deploys', () => {
    // Job name field should be selected by default
    browser.keys(jobName);

    browser.keys('Tab');
    browser.keys(jobDescription);

    // Skip over job trigger, go to the first input parameter
    browser.keys('Tab');
    browser.keys('Tab');

    browser.keys(outputParameterValue);

    // Deploy
    $('#deployBtn').click();
  });

  it('redirects back to job list page', () => {
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).hash === '#/jobs';
    }, waitTimeout);
  });

  it('finds the new job in the list of jobs', () => {
    browser.waitForVisible('.tableRow', waitTimeout);
    assert.equal($$('.tableRow').length, 1, 'should only show one job');

    // Navigate to details of the deployed job, by finding the anchor element
    // with the job's name as inner text
    browser.waitForVisible('.tableRow a');
    browser.execute('document.querySelector(".tableRow a").click()');
  });

  it('switches to config tab', () => {
    browser.waitForVisible('button=Config', waitTimeout);
    $('button=Config').click();
  });

  it('displays job description correctly', () => {
    const description = getValueFromDetailsTable('Description');
    assert(description.startsWith(jobDescription), 'job description is not shown correctly');
  });

  it('displays job created at date correctly', () => {
    const date = getValueFromDetailsTable('Created at');
    assert(Date.now() - new Date(date) < 5 * 1000,
      'job created date should be within the last five seconds');
  });

  it('displays job inputs correctly', () => {
    const paramValue = getValueFromDetailsTable('message');
    assert.equal(paramValue, outputParameterValue, 'job message is not shown correctly');
  });

  it('switches to run list tab', () => {
    $('button=Runs').click();
  });

  it('schedules and lists exactly one run', (done) => {
    let attempts = 0;

    const selector = '.tableRow';
    let items = $$(selector);
    const maxAttempts = 80;

    while (attempts < maxAttempts && (!items || items.length === 0)) {
      $('#refreshBtn').click();
      browser.pause(1000);
      items = $$(selector);
      attempts++;
    }

    assert(attempts < maxAttempts, `waited for ${maxAttempts} seconds but run did not start`);
    assert(items && items.length > 0, 'only one run should show up');

    const runName = browser.getText(selector + ' div')[1];
    assert(runName.startsWith('job-helloworld'),
      'run name should start with job-helloworld');
  });

  it('opens run details', () => {
    browser.waitForVisible('.tableRow a');
    browser.execute('document.querySelector(".tableRow a").click()');

    browser.waitUntil(() => {
      return new URL(browser.getUrl()).hash.startsWith('#/runs/details/');
    }, waitTimeout);
  });

  it('waits until the whole run is complete', () => {
    const nodeSelector = '.graphNode';

    let attempts = 0;

    const maxAttempts = 30;

    while (attempts < maxAttempts && $$(nodeSelector).length < 4) {
      $('#refreshBtn').click();
      // Wait for a reasonable amount of time until the run is done
      browser.pause(1000);
      attempts++;
    }

    assert(attempts < maxAttempts, `waited for ${maxAttempts} seconds but run did not finish`);
  });

  it('deletes the job', () => {
    $('#jobsButton').click();

    browser.waitForVisible('.tableRow', waitTimeout);
    $('.tableRow').click();
    $('#deleteBtn').click();
    $('.dialogButton').click();
    $('.dialog').waitForVisible(waitTimeout, true);
  });

  it('deletes the uploaded pipeline', () => {
    $('#pipelinesButton').click();

    browser.waitForVisible('.tableRow', waitTimeout);
    $('.tableRow').click();
    $('#deleteBtn').click();
    $('.dialogButton').click();
    $('.dialog').waitForVisible(waitTimeout, true);
  });
});

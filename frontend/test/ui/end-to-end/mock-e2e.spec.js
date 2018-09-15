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
const listSelector = 'app-shell job-list item-list';
const mockJobsLength = 23;

describe('deploy new job', () => {

  before(() => {
    browser.url('/');
  });

  it('navigates to job list page', () => {
    const selector = 'app-shell side-nav #jobsBtn';

    browser.waitForVisible(selector);
    browser.click(selector);
  });

  it('starts out with all mock jobs', () => {
    browser.waitForVisible(listSelector, waitTimeout);
    const selector = 'app-shell job-list item-list #listContainer paper-item';
    browser.waitForVisible(selector, waitTimeout);

    assert.equal($$(selector).length, 20, 'should start out with a full page of 20 jobs');
  });

  it('shows the second and final page of jobs', () => {
    const selector = 'app-shell job-list item-list #nextPage';
    browser.waitForVisible(selector, waitTimeout);
    browser.click(selector);

    const listSelector = 'app-shell job-list item-list #listContainer paper-item';
    browser.waitForVisible(selector, waitTimeout);
    assert.equal($$(listSelector).length, mockJobsLength - 20,
        'second page should show the remaining jobs');
  });

  it('opens new job page', () => {
    const selector = 'app-shell job-list paper-button';

    browser.waitForVisible(selector, waitTimeout);
    browser.click(selector);
    assert.equal(new URL(browser.getUrl()).pathname, '/jobs/new');
  });

  it('uploads the hello world pipeline', () => {
    const uploadBtnSelector = 'app-shell job-new #uploadBtn';
    browser.waitForVisible(uploadBtnSelector, waitTimeout);
    browser.click(uploadBtnSelector);

    // Show the alt upload button
    browser.execute(`document.querySelector('pipeline-upload-dialog').shadowRoot
                        .querySelector('#altFileUpload').style.display=''`);

    const uploadInputSelector = 'pipeline-upload-dialog #altFileUpload';
    browser.chooseFile(uploadInputSelector, './hello-world.yaml');

    // Hide the alt upload button
    browser.execute(`document.querySelector('pipeline-upload-dialog').shadowRoot
                        .querySelector('#altFileUpload').style.display='none'`);

    // Pipeline name should default to uploaded file name
    const pipelineNameInputSelector = 'pipeline-upload-dialog #pipelineNameInput';
    assert.equal(browser.getValue(pipelineNameInputSelector), 'hello-world.yaml');

    // Clear pipeline name input
    browser.execute(`document.querySelector('pipeline-upload-dialog').shadowRoot
                        .querySelector('#pipelineNameInput').value=''`)
    assert.equal(browser.getValue(pipelineNameInputSelector), '');

    // Manually edit pipeline name
    browser.click(pipelineNameInputSelector);
    browser.keys('my-new-pipeline');
    assert.equal(browser.getValue(pipelineNameInputSelector), 'my-new-pipeline');

    // Complete the upload flow
    const finalUploadBtnSelector = 'pipeline-upload-dialog popup-dialog #button1';
    browser.click(finalUploadBtnSelector);

    const pkgIdSelector = 'app-shell job-new paper-dropdown-menu ' +
                          'paper-menu-button::paper-input paper-input-container::iron-input';
    browser.waitForValue(pkgIdSelector, waitTimeout);
  });

  it('populates job details and deploys', () => {
    browser.click('app-shell job-new #name');
    browser.keys(jobName);

    browser.keys('Tab');
    browser.keys(jobDescription);

    // Skip trigger and maximum concurrent jobs inputs
    browser.keys('Tab');
    browser.keys('Tab');

    browser.keys('Tab');
    browser.keys('param-1 value');
    browser.keys('Tab');
    browser.keys('param-2 value');
    browser.keys('Tab');
    browser.keys('output param value');

    // Hide upload success toast. It appears directly over the deploy button
    browser.execute(`document.querySelector('paper-toast').style.display='none'`);

    // Wait for toast to be hidden
    browser.waitForVisible('paper-toast', waitTimeout, true);
    browser.click('app-shell job-new #deployButton');
  });

  it('redirects back to job list page', () => {
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname === '/jobs';
    }, waitTimeout);
  });

  it('finds the new job ' + jobName + ' in the list of jobs', () => {
    // Navigate to second page, where the new job should be
    const nextPageSelector = 'app-shell job-list item-list #nextPage';
    browser.waitForVisible(nextPageSelector, waitTimeout);
    browser.click(nextPageSelector);

    browser.waitForVisible(listSelector, waitTimeout);

    const selector = 'app-shell job-list item-list #listContainer paper-item';
    browser.waitForVisible(selector, waitTimeout);
    const index = mockJobsLength - 20 + 1;
    assert.equal($$(selector).length, index, 'should have a new item added to job list');

    // Navigate to details of the deployed job
    browser.doubleClick(selector + `:nth-of-type(${index})`);
  });

  it('displays job name correctly', () => {
    const selector = 'app-shell job-details .job-name';
    browser.waitForVisible(selector, waitTimeout);
    assert(browser.getText(selector).startsWith(jobName),
        'job name is not shown correctly: ' + browser.getText(selector));
  });

  it('displays job description correctly', () => {
    const selector = 'app-shell job-details .description.value';
    browser.waitForVisible(selector, waitTimeout);
    assert.equal(browser.getText(selector), jobDescription,
        'job description is not shown correctly');
  });

  it('displays job created at date correctly', () => {
    const selector = 'app-shell job-details .created-at.value';
    browser.waitForVisible(selector, waitTimeout);
    const createdDate = Date.parse(browser.getText(selector));
    assert(Date.now() - createdDate < 5 * 1000,
        'job created date should be within the last five seconds: ' + createdDate);
  });

  it('displays job parameters correctly', () => {
    const selector = 'app-shell job-details .params-table';
    browser.waitForVisible(selector, waitTimeout);

    const paramsSelector = 'app-shell job-details .params-table';
    assert.deepEqual(
        browser.getText(paramsSelector),
        'param-1\nparam-1 value\nparam-2\nparam-2 value\noutput\noutput param value',
        'parameter values are incorrect: ' + browser.getText(paramsSelector));
  });

  it('switches to run list tab', () => {
    const selector = 'app-shell job-details paper-tab:last-child';
    browser.click(selector);
  });

  it('lists exactly one run', () => {
    const selector = 'app-shell job-details run-list item-list #listContainer paper-item';
    const runsText = browser.getText(selector);

    assert(!Array.isArray(runsText) && typeof runsText === 'string',
        'only one run should show up');
    assert(runsText.startsWith('coinflip-recursive-run-lknlfs3'), 'run name is incorrect');
  });

  it('opens run details on double click', () => {
    const selector = 'app-shell job-details run-list item-list #listContainer paper-item';

    browser.waitForVisible(selector, waitTimeout);
    browser.doubleClick(selector);

    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname.startsWith('/jobRun');
    }, waitTimeout);
  });

  it('deletes the job', () => {
    const backBtn = 'app-shell run-details #jobLink';
    browser.waitForVisible(backBtn, waitTimeout);
    browser.click(backBtn);

    const selector = 'app-shell job-details #deleteBtn';
    browser.waitForVisible(selector);
    browser.click(selector);

    // Can't find a better way to wait for dialog to appear. For some reason,
    // waitForVisible just hangs.
    browser.pause(500);
    browser.click('popup-dialog #button1');
  });

  it('redirects back to job list page', () => {
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname === '/jobs';
    }, waitTimeout);
  });

  it(`shows only ${mockJobsLength - 20} jobs on second page after deletion`, () => {
    const selector = 'app-shell job-list item-list #nextPage';
    browser.waitForVisible(selector, waitTimeout);
    browser.click(selector);

    browser.pause(500);

    const listSelector = 'app-shell job-list item-list #listContainer paper-item';
    browser.waitForVisible(selector, waitTimeout);
    assert.equal($$(listSelector).length, mockJobsLength - 20,
        'second page should show the remaining jobs, it shows: ' + $$(listSelector).length);
  });

  it('deletes the uploaded pipeline', () => {
    const selector = 'app-shell side-nav #pipelinesBtn';
    browser.click(selector);

    browser.pause(500);

    const pipelinesListSelector = 'app-shell pipeline-list item-list #listContainer paper-item';

    const allPipelineElements = $$(pipelinesListSelector);

    // Find newly uploaded pipeline and click it.
    const newPipelineElement = allPipelineElements.find(
        (e) => browser.elementIdText(e.ELEMENT).value.startsWith('my-new-pipeline'));
    browser.elementIdClick(newPipelineElement.ELEMENT);

    browser.click('app-shell pipeline-list #deleteBtn');

    // Can't find a better way to wait for dialog to appear. For some reason,
    // waitForVisible just hangs.
    browser.pause(500);

    // Confirm deletion
    browser.click('popup-dialog #button1');
  });

  it('can use hash navigation to show the jobs page directly', () => {
    browser.url('/jobs/details/7fc01714-4a13-4c05-7186-a8a72c14253b#runs');
    const selector = 'app-shell job-details run-list item-list #listContainer paper-item';
     // Should find a page full of jobs
    browser.waitForVisible(selector, waitTimeout);
    assert.equal($$(selector).length, 20, 'should show a full page of jobs ');
  });

  it('shows error if user navigates to a bad path', () => {
    browser.url('/jobz');
    const selector = 'app-shell #errorEl .error-message';
    browser.waitForVisible(selector, waitTimeout);
    assert.equal(browser.getText(selector), 'Cannot find page: /jobz');
  });

});

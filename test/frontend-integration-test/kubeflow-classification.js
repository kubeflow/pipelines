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

const jobName = 'tfjob-classification-' + Date.now();
const jobDescription = 'test job description ' + jobName;
const waitTimeout = 5000;

const outputDir = process.env.PIPELINE_OUTPUT;
//let tensorboardAddress = '';

describe('deploy tfjob sample job', () => {

  before(() => {
    browser.url('/');
  });

  it('navigates to job list page', () => {
      const selector = 'app-shell side-nav #jobsBtn';

      browser.waitForVisible(selector);
      browser.click(selector);
    });

  it('opens new job page', () => {
    const selector = 'app-shell job-list paper-button';

    browser.waitForVisible(selector, waitTimeout);
    browser.click(selector);
    assert.equal(new URL(browser.getUrl()).pathname, '/jobs/new');
  });

  it('uploads the kubeflow classfication sample pipeline', () => {
    const uploadBtnSelector = 'app-shell job-new #uploadBtn';
    browser.waitForVisible(uploadBtnSelector, waitTimeout);
    browser.click(uploadBtnSelector);

    // Show the alt upload button
    browser.execute(`document.querySelector('pipeline-upload-dialog').shadowRoot
                        .querySelector('#altFileUpload').style.display=''`);
    const uploadInputSelector = 'pipeline-upload-dialog #altFileUpload';
    browser.chooseFile(uploadInputSelector, './kubeflow-classification.yaml');

    // Hide the alt upload button
    browser.execute(`document.querySelector('pipeline-upload-dialog').shadowRoot
                        .querySelector('#altFileUpload').style.display='none'`);

    // Pipeline name should default to uploaded file name
    const pipelineNameInputSelector = 'pipeline-upload-dialog #pipelineNameInput';
    assert.equal(browser.getValue(pipelineNameInputSelector), 'kubeflow-classification.yaml');

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
    browser.waitForValue(pkgIdSelector, 5 * waitTimeout);
  });

  it('populates job details and deploys', () => {
    browser.click('app-shell job-new #name');
    browser.keys(jobName);

    browser.keys('Tab');
    browser.keys(jobDescription);

    // Skip trigger and maximum concurrent jobs inputs
    browser.keys('Tab');
    browser.keys('Tab');

    browser.keys('Tab')
    browser.keys(outputDir)

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

  it('finds the new job in the list of jobs', () => {
    const selector = 'app-shell job-list item-list #listContainer paper-item';
    browser.waitForVisible(selector, waitTimeout);
    assert.equal($$(selector).length, 1, 'should only show one job');

    // Navigate to details of the deployed job
    browser.doubleClick(selector + `:nth-of-type(1)`);
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
        'job created date should be within the last five seconds');
  });

  it('display job output directory correctly', () => {
    const selector = 'app-shell job-details .params-table.details-table::div:nth-child(1)::span.value'
    browser.waitForVisible(selector, waitTimeout);
    assert.equal(browser.getText(selector), outputDir,
        'job output directory is not shown correctly');
  });

  it('switches to run list tab', () => {
    const selector = 'app-shell job-details paper-tab:last-child';
    browser.click(selector);
  });

  it('schedules and lists exactly one run', (done) => {
    const listSelector = 'app-shell job-details run-list item-list #listContainer';
    browser.waitForVisible(listSelector, waitTimeout);

    let attempts = 0;

    const selector = 'app-shell job-details run-list item-list #listContainer paper-item';
    let items = $$(selector);
    const maxAttempts = 80;

    while (attempts < maxAttempts && (!items || items.length === 0)) {
      browser.click('app-shell job-details paper-button#refreshBtn');
      browser.pause(1000);
      items = $$(selector);
      attempts++;
    }

    assert(attempts < maxAttempts, `waited for ${maxAttempts} seconds but run did not start`);
    assert(items && items.length > 0, 'only one run should show up');

    const runsText = browser.getText(selector);
    assert(!Array.isArray(runsText) && typeof runsText === 'string',
      'only one run should show up');
    assert(runsText.startsWith('job-tfjob-classification'),
      'run name should start with job-tfjob-classification: ' + runsText);
  });

  it('opens run details on double click', () => {
    const selector = 'app-shell job-details run-list item-list #listContainer paper-item';

    browser.waitForVisible(selector, waitTimeout);
    browser.doubleClick(selector);

    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname.startsWith('/jobRun');
    }, waitTimeout);
  });

  it('waits until the whole job is complete', () => {
    const selector = 'app-shell run-details iron-icon.status-icon';
    browser.waitForVisible(selector, waitTimeout);

    let attempts = 0;

    const maxAttempts = 240;

    while (attempts < maxAttempts && browser.getAttribute(selector, 'icon') !== 'check-circle') {
      browser.click('app-shell run-details #refreshButton');
      // Wait for a reasonable amount of time until the run is done
      browser.pause(5000);
      attempts++;
    }

    assert(attempts < maxAttempts, `waited for ${maxAttempts * 5} seconds but run did not finish`);
  });

  it('generates four nodes in the runtime graph', () => {
    assert.equal(browser.elements(
        'app-shell run-details runtime-graph pipeline-graph .job-node').value.length, 4);
  });
// TODO: add the visualization tests after the frontend is updated with the plot orders.
//  it('switches to output tab', () => {
//    const selector = 'app-shell run-details #output-tab'
//    browser.click(selector)
//  });
//
//  it('creates a confusion matrix with an svg', () => {
//    const selector = 'app-shell run-details data-plot .plotTitle';
//    assert(browser.getText(selector).startsWith('Confusion Matrix from file:'));
//
//    const svgSelector = 'app-shell run-details data-plot svg';
//    assert(browser.isVisible(svgSelector));
//  });
//
//  it('creates a Start Tensorboard button', () => {
//    const selector = 'app-shell run-details data-plot:nth-of-type(3) .plotTitle';
//    assert(browser.getText(selector).startsWith('Tensorboard for logdir:'));
//
//    const buttonSelector = 'app-shell run-details data-plot:nth-of-type(3) paper-button';
//    assert(browser.isVisible(buttonSelector));
//    assert.equal(browser.getText(buttonSelector), 'Start Tensorboard');
//  });
//
//  it('starts Tensorboard when button is clicked', () => {
//    const buttonSelector = 'app-shell run-details data-plot:nth-of-type(3) paper-button';
//    browser.click(buttonSelector);
//
//    let attempts = 0;
//
//    const maxAttempts = 120;
//
//    while (attempts < maxAttempts && browser.getText(buttonSelector) !== 'Open Tensorboard') {
//      browser.pause(1000);
//      attempts++;
//    }
//
//    assert(attempts < maxAttempts, `waited for ${maxAttempts} seconds but Tensorboard did not start`);
//  });
//
//  it('generates the right Tensorboard proxy hyperlink', () => {
//    const buttonSelector = 'app-shell run-details data-plot:nth-of-type(3) a';
//    tensorboardAddress = browser.getAttribute(buttonSelector, 'href');
//    assert(tensorboardAddress.indexOf('/apis/v1alpha2/_proxy/') > -1);
//  });

  it('deletes the job', () => {
    const backBtn = 'app-shell run-details #jobLink';
    browser.waitForVisible(backBtn, waitTimeout);
    browser.click(backBtn);

    const selector = 'app-shell job-details #deleteBtn';
    browser.waitForVisible(selector);
    browser.click(selector);

    browser.pause(500);
    browser.click('popup-dialog #button1');
  });

  it('redirects back to job list page', () => {
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname === '/jobs';
    }, waitTimeout);
  });

  // it('deletes the uploaded pipeline', () => {
  //   const selector = 'app-shell side-nav #pipelinesBtn';
  //   browser.click(selector);
  //   browser.pause(500);
  //   const pipelinesListSelector = 'app-shell pipeline-list item-list #listContainer paper-item';
  //   const allPipelineElements = $$(pipelinesListSelector);
  //   // Find newly uploaded pipeline and click it.
  //   const newPipelineElement = allPipelineElements.find(
  //       (e) => browser.elementIdText(e.ELEMENT).value.startsWith('my-new-pipeline'));
  //   browser.elementIdClick(newPipelineElement.ELEMENT);
  //   browser.click('app-shell pipeline-list #deleteBtn');
  //   // Can't find a better way to wait for dialog to appear. For some reason,
  //   // waitForVisible just hangs.
  //   browser.pause(500);
  //   // Confirm deletion
  //   browser.click('popup-dialog #button1');
  // });

//  it('can visit the Tensorboard pod using the proxy link', () => {
//    browser.url(tensorboardAddress);
//    browser.waitForVisible('paper-toolbar');
//  });
});

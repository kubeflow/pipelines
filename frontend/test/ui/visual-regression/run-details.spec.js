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

function assertDiffs(results) {
  results.forEach(r => assert.ok(r.isExactSameImage));
}

describe('view run details', () => {

  before(() => {
    // This job has runs.
    browser.url('/jobs/details/7fc01714-4a13-4c05-7186-a8a72c14253b');
  });

  it('navigates to run list', () => {
    const selector = 'app-shell job-details paper-tab:last-child';
    browser.waitForVisible(selector);
    browser.click(selector);

    assertDiffs(browser.checkDocument());
  });

  it('collapses the left nav panel', () => {
    const selector = 'app-shell side-nav paper-icon-button';

    browser.waitForVisible(selector);
    browser.click(selector);

    assertDiffs(browser.checkDocument());
  });

  it('shows runtime graph on double click', () => {
    const selector = 'app-shell job-details run-list item-list paper-item';

    browser.waitForVisible(selector);
    browser.doubleClick(selector);
    assertDiffs(browser.checkDocument());
  });

  it('switches to run details', () => {
    const selector = 'app-shell run-details #info-tab';

    browser.waitForVisible(selector);
    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('clears previous run\'s graph', () => {
    // Switch back to graph tab
    const selector = 'app-shell run-details #graph-tab';

    browser.waitForVisible(selector);
    browser.click(selector);

    // Return to run list
    const backBtnSelector = 'app-shell run-details #jobLink';
    browser.waitForVisible(backBtnSelector);
    browser.click(backBtnSelector);

    const runListTabSelector = 'app-shell job-details paper-tab:last-child';
    browser.waitForVisible(runListTabSelector);
    browser.click(runListTabSelector);

    // Select a run with a different graph.
    const differentRunSelector =
        'app-shell job-details run-list item-list paper-item:nth-of-type(3)';
    browser.waitForVisible(differentRunSelector);
    browser.doubleClick(differentRunSelector);

    // Return to run list
    browser.waitForVisible(backBtnSelector);
    browser.click(backBtnSelector);
    browser.waitForVisible(runListTabSelector);
    browser.click(runListTabSelector);

    // Select the original run again.
    const originalRunSelector = 'app-shell job-details run-list item-list paper-item';
    browser.waitForVisible(originalRunSelector);
    browser.doubleClick(originalRunSelector);

    const graphSelector = 'app-shell run-details #graph-tab';
    browser.waitForVisible(graphSelector);
    browser.click(graphSelector);

    assertDiffs(browser.checkDocument());
  });

  it('opens node details upon click', () => {
    // Select a step that will show in the viewport without scrolling.
    const selector = 'app-shell run-details runtime-graph .job-node:nth-of-type(4)';

    browser.waitForVisible(selector);
    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('switches to task configuration viewer', () => {
    browser.click('app-shell run-details runtime-graph #configTab');
    assertDiffs(browser.checkDocument());
  });

  it('switches to logs viewer', () => {
    browser.click('app-shell run-details runtime-graph #logsTab');
    // Wait for the mock server to return the logs
    browser.pause(400);

    const logsSelector = 'app-shell run-details runtime-graph #logsContainer';
    browser.waitForVisible(logsSelector);
    assertDiffs(browser.checkDocument());
  });

  it('closes node details', () => {
    const selector = 'app-shell run-details runtime-graph .hide-node-details';

    browser.waitForVisible(selector);
    browser.click(selector);

    browser.pause(300);
    assertDiffs(browser.checkDocument());
  });

  it('clones run into a new job', () => {
    browser.click('app-shell run-details #cloneButton');
    assertDiffs(browser.checkDocument());
  });

  describe('error handling', () => {

    before(() => {
      // This job has runs.
      browser.url('/jobs/details/7fc01714-4a13-4c05-7186-a8a72c14253b');

      // Navigate to Run list
      const selector = 'app-shell job-details paper-tab:last-child';
      browser.waitForVisible(selector);
      browser.click(selector);

      // Collapse nav panel.
      const navExpanderSelector = 'app-shell side-nav paper-icon-button';
      browser.waitForVisible(navExpanderSelector);
      browser.click(navExpanderSelector);
    });

    it('displays error message', () => {
      // Select a run that will show an error message.
      const failedRunSelector =
          'app-shell job-details run-list item-list paper-item:nth-of-type(2)';

      browser.waitForVisible(failedRunSelector);
      browser.doubleClick(failedRunSelector);

      assertDiffs(browser.checkDocument());
    });

    it('clears previous runtime graph even if there is an error', () => {
      // Return to run list
      const backBtnSelector = 'app-shell run-details #jobLink';
      browser.waitForVisible(backBtnSelector);
      browser.click(backBtnSelector);

      // Select a run that succeeded.
      const successfulRunSelector = 'app-shell job-details run-list item-list paper-item';
      browser.waitForVisible(successfulRunSelector);
      browser.doubleClick(successfulRunSelector);

      // Return to run list
      browser.waitForVisible(backBtnSelector);
      browser.click(backBtnSelector);

      // Select a run that will show an error message.
      const failedRunSelector =
          'app-shell job-details run-list item-list paper-item:nth-of-type(2)';
      browser.waitForVisible(failedRunSelector);
      browser.doubleClick(failedRunSelector);

      const graphSelector = 'app-shell run-details #graph-tab';
      browser.waitForVisible(graphSelector);
      browser.click(graphSelector);

      assertDiffs(browser.checkDocument());
    });

    it('clears error message for subsequent successful run pages', () => {
      // Return to run list
      const backBtnSelector = 'app-shell run-details #jobLink';
      browser.waitForVisible(backBtnSelector);
      browser.click(backBtnSelector);

      // Select a run that succeeded.
      const successfulRunSelector = 'app-shell job-details run-list item-list paper-item';
      browser.waitForVisible(successfulRunSelector);
      browser.doubleClick(successfulRunSelector);

      assertDiffs(browser.checkDocument());
    });
  });
});

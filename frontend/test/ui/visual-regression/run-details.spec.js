
const fixedData = require('../../../mock-backend/fixed-data').data;
const assert = require('assert');

function assertDiffs(results) {
  results.forEach(r => assert.ok(r.isExactSameImage));
}

describe('view run details', () => {

  beforeAll(() => {
    // This job has runs.
    browser.url(`/jobs/details/${fixedData.jobs[1].id}`);
  });

  it('navigates to run list', () => {
    const selector = 'app-shell job-details paper-tab:last-child';
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

  it('switches to run outputs', () => {
    const selector = 'app-shell run-details #output-tab';

    browser.waitForVisible(selector);
    browser.click(selector);
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

    beforeAll(() => {
      // This job has runs.
      browser.url(`/jobs/details/${fixedData.jobs[1].id}`);

      // Navigate to Run list
      const selector = 'app-shell job-details paper-tab:last-child';
      browser.waitForVisible(selector);
      browser.click(selector);
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

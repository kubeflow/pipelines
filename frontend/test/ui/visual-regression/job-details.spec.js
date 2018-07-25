
const fixedData = require('../../../mock-backend/fixed-data').data;
const assert = require('assert');

function assertDiffs(results) {
  results.forEach(r => assert.ok(r.isExactSameImage));
}

describe('view job details', () => {

  beforeAll(() => {
    // This pipeline has jobs.
    browser.url(`/pipelines/details/${fixedData.pipelines[1].id}`);
  });

  it('navigates to job list', () => {
    const selector = 'app-shell pipeline-details paper-tab:last-child';
    browser.waitForVisible(selector);
    browser.click(selector);

    assertDiffs(browser.checkDocument());
  });

  it('opens job details on double click', () => {
    const selector = 'app-shell pipeline-details job-list item-list paper-item';

    browser.waitForVisible(selector);
    browser.doubleClick(selector);
    assertDiffs(browser.checkDocument());
  });

  // TODO: The order of the plots on the job output page is currently
  // non-deterministic likely due to the async calls job-details.

  it('views the job graph', () => {
    const selector = 'app-shell job-details #graph-tab';

    browser.waitForVisible(selector);
    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('clears previous job\'s graph', () => {
    // Return to job list
    const backBtnSelector = 'app-shell job-details paper-icon-button';
    browser.waitForVisible(backBtnSelector);
    browser.click(backBtnSelector);

    const jobListTabSelector = 'app-shell pipeline-details paper-tab:last-child';
    browser.waitForVisible(jobListTabSelector);
    browser.click(jobListTabSelector);

    // Select a job with a different graph.
    const differentJobSelector =
        'app-shell pipeline-details job-list item-list paper-item:nth-of-type(3)';
    browser.waitForVisible(differentJobSelector);
    browser.doubleClick(differentJobSelector);

    // Return to job list
    browser.waitForVisible(backBtnSelector);
    browser.click(backBtnSelector);
    browser.waitForVisible(jobListTabSelector);
    browser.click(jobListTabSelector);

    // Select the original job again.
    const originalJobSelector = 'app-shell pipeline-details job-list item-list paper-item';
    browser.waitForVisible(originalJobSelector);
    browser.doubleClick(originalJobSelector);

    const graphSelector = 'app-shell job-details #graph-tab';
    browser.waitForVisible(graphSelector);
    browser.click(graphSelector);

    assertDiffs(browser.checkDocument());
  });

  it('opens node details upon click', () => {
    // Select a step that will show in the viewport without scrolling.
    const selector = 'app-shell job-details job-graph .pipeline-node:nth-of-type(4)';

    browser.waitForVisible(selector);
    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('switches to logs viewer', () => {
    browser.click('app-shell job-details job-graph #logsTab');
    // Wait for the mock server to return the logs
    browser.pause(400);

    const logsSelector = 'app-shell job-details job-graph #logsContainer';
    browser.waitForVisible(logsSelector);
    assertDiffs(browser.checkDocument());
  });

  it('closes node details', () => {
    const selector = 'app-shell job-details job-graph .hide-node-details';

    browser.waitForVisible(selector);
    browser.click(selector);

    browser.pause(300);
    assertDiffs(browser.checkDocument());
  });

  describe('error handling', () => {

    beforeAll(() => {
      // This pipeline has jobs.
      browser.url(`/pipelines/details/${fixedData.pipelines[1].id}`);

      // Navigate to Job list
      const selector = 'app-shell pipeline-details paper-tab:last-child';
      browser.waitForVisible(selector);
      browser.click(selector);
    });

    it('displays error message', () => {
      // Select a job that will show an error message.
      const failedJobSelector =
          'app-shell pipeline-details job-list item-list paper-item:nth-of-type(2)';

      browser.waitForVisible(failedJobSelector);
      browser.doubleClick(failedJobSelector);

      assertDiffs(browser.checkDocument());
    });

    it('clears previous job graph even if there is an error', () => {
      // Return to job list
      const backBtnSelector = 'app-shell job-details paper-icon-button';
      browser.waitForVisible(backBtnSelector);
      browser.click(backBtnSelector);

      const jobListTabSelector = 'app-shell pipeline-details paper-tab:last-child';
      browser.waitForVisible(jobListTabSelector);
      browser.click(jobListTabSelector);

      // Select a job that succeeded.
      const successfulJobSelector = 'app-shell pipeline-details job-list item-list paper-item';
      browser.waitForVisible(successfulJobSelector);
      browser.doubleClick(successfulJobSelector);

      // Return to job list
      browser.waitForVisible(backBtnSelector);
      browser.click(backBtnSelector);
      browser.waitForVisible(jobListTabSelector);
      browser.click(jobListTabSelector);

      // Select a job that will show an error message.
      const failedJobSelector =
          'app-shell pipeline-details job-list item-list paper-item:nth-of-type(2)';
      browser.waitForVisible(failedJobSelector);
      browser.doubleClick(failedJobSelector);

      const graphSelector = 'app-shell job-details #graph-tab';
      browser.waitForVisible(graphSelector);
      browser.click(graphSelector);

      assertDiffs(browser.checkDocument());
    });

    it('clears error message for subsequent successful job pages', () => {
      // Return to job list
      const backBtnSelector = 'app-shell job-details paper-icon-button';
      browser.waitForVisible(backBtnSelector);
      browser.click(backBtnSelector);

      const jobListTabSelector = 'app-shell pipeline-details paper-tab:last-child';
      browser.waitForVisible(jobListTabSelector);
      browser.click(jobListTabSelector);

      // Select a job that succeeded.
      const successfulJobSelector = 'app-shell pipeline-details job-list item-list paper-item';
      browser.waitForVisible(successfulJobSelector);
      browser.doubleClick(successfulJobSelector);

      assertDiffs(browser.checkDocument());
    });
  });
});

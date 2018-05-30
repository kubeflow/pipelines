const assert = require('assert');

function assertDiffs(results) {
  results.forEach(r => assert.ok(r.isExactSameImage));
}

describe('view job details', () => {

  beforeAll(() => {
    browser.url('/pipelines/details/2');
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

  it('highlights a step upon hover', () => {
    // Select a step that will show in the viewport without scrolling.
    const selector = 'app-shell job-details job-graph .pipeline-node:nth-of-type(5)';

    browser.waitForVisible(selector);
    browser.moveToObject(selector, 0, 0);
    assertDiffs(browser.checkDocument());
  });
});

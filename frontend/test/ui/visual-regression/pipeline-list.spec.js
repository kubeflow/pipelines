const assert = require('assert');

function assertDiffs(results) {
  results.forEach(r => assert.ok(r.isExactSameImage));
}

describe('list pipelines', () => {

  beforeAll(() => {
    browser.url('/');
  });

  it('shows list of pipelines on first page', () => {
    const selector = 'app-shell pipeline-list';

    browser.waitForVisible(selector);
    assertDiffs(browser.checkDocument());
  });

  it('shows hover effect on pipeline', () => {
    const selector = 'app-shell pipeline-list item-list paper-item';

    browser.moveToObject(selector, 0, 0);
    assertDiffs(browser.checkDocument());
  });

  it('enables Clone when pipeline clicked', () => {
    const selector = 'app-shell pipeline-list item-list paper-item';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('disables Clone when multiple pipelines selected', () => {
    const selector = 'app-shell pipeline-list item-list paper-checkbox';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('loads next page on "next" button press', () => {
    const selector = 'app-shell pipeline-list item-list paper-button#nextPage';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('loads previous page on "previous" button press after pressing "next"', () => {
    const previousButtonselector = 'app-shell pipeline-list item-list paper-button#previousPage';
    browser.click(previousButtonselector);

    assertDiffs(browser.checkDocument());
  });

  it('resets the pipelne list after clicking into a pipeline', () => {
    // Navigate to second page
    const nextButtonselector = 'app-shell pipeline-list item-list paper-button#nextPage';
    browser.click(nextButtonselector);

    // Go to a Pipeline's details page
    const pipelineSelector = 'app-shell pipeline-list item-list paper-item';
    browser.doubleClick(pipelineSelector);

    // Return to Pipeline list page
    const backButtonSelector = 'app-shell pipeline-details .toolbar-arrow-back';
    browser.waitForVisible(backButtonSelector);
    browser.click(backButtonSelector);

    // List should be reset to first page of results
    assertDiffs(browser.checkDocument());
  });

  it('populates cloned pipeline', () => {
    // Find a pipeline with package ID of 1 so it can be cloned. The first pipeline works.
    // TODO: Explore making this more reliable
    const selector = 'app-shell pipeline-list item-list paper-item';
    browser.click(selector);

    const cloneBtnSelector = 'app-shell pipeline-list paper-button#cloneBtn';
    browser.click(cloneBtnSelector);

    assertDiffs(browser.checkDocument());
  });
});

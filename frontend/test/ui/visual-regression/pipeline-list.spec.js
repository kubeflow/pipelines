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
    const selector = 'app-shell pipeline-list item-list #listContainer paper-item';

    browser.moveToObject(selector, 0, 0);
    assertDiffs(browser.checkDocument());
  });

  it('enables Clone when pipeline clicked', () => {
    const selector = 'app-shell pipeline-list item-list #listContainer paper-item';

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
    const previousButtonSelector = 'app-shell pipeline-list item-list paper-button#previousPage';
    browser.click(previousButtonSelector);

    assertDiffs(browser.checkDocument());
  });

  it('resets the pipelne list after clicking into a pipeline', () => {
    // Navigate to second page
    const nextButtonSelector = 'app-shell pipeline-list item-list paper-button#nextPage';
    browser.click(nextButtonSelector);

    // Go to a Pipeline's details page
    const pipelineSelector = 'app-shell pipeline-list item-list #listContainer paper-item';
    browser.doubleClick(pipelineSelector);

    // Return to Pipeline list page
    const backButtonSelector = 'app-shell pipeline-details .toolbar-arrow-back';
    browser.waitForVisible(backButtonSelector);
    browser.click(backButtonSelector);

    // List should be reset to first page of results
    assertDiffs(browser.checkDocument());
  });

  it('loads additional pipelines after changing page size', () => {
    // Default is 20, but we'll change it to 50.
    const pageSizeDropdownSelector = 'app-shell pipeline-list item-list paper-dropdown-menu';
    browser.click(pageSizeDropdownSelector);

    const pageSizeSelector =
        'app-shell pipeline-list item-list paper-dropdown-menu::paper-item:nth-of-type(2)';
    browser.click(pageSizeSelector);

    assertDiffs(browser.checkDocument());
  });

  it('populates cloned pipeline', () => {
    // Find a pipeline with package ID of 1 so it can be cloned.
    // TODO: Explore making this more reliable
    const selector = 'app-shell pipeline-list item-list #listContainer paper-item';
    browser.doubleClick(selector);

    const cloneBtnSelector = 'app-shell pipeline-list paper-button#cloneBtn';
    browser.click(cloneBtnSelector);

    assertDiffs(browser.checkDocument());
  });
});

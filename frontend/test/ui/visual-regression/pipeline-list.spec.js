const assert = require('assert');

function assertDiffs(results) {
  results.forEach(r => assert.ok(r.isExactSameImage));
}

describe('list pipelines', () => {

  before(() => {
    browser.url('/');
  });

  it('expands the left nav panel', () => {
    const selector = 'app-shell side-nav paper-icon-button';

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

  it('shows hover effect on pipeline', () => {
    const selector = 'app-shell pipeline-list item-list #listContainer paper-item';

    browser.moveToObject(selector, 0, 0);
    assertDiffs(browser.checkDocument());
  });

  it('loads next page on "next" button press', () => {
    const selector = 'app-shell pipeline-list item-list #nextPage';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('loads previous page on "previous" button press after pressing "next"', () => {
    const previousButtonSelector = 'app-shell pipeline-list item-list #previousPage';
    browser.click(previousButtonSelector);

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

  it('shows confirmation when trying to delete a pipeline', () => {
    const pipelineSelector = 'app-shell pipeline-list item-list #listContainer paper-item';
    browser.click(pipelineSelector);

    const deleteBtnSelector = 'app-shell pipeline-list paper-button#deleteBtn';
    browser.waitForEnabled(deleteBtnSelector);
    browser.click(deleteBtnSelector);

    browser.waitForExist('popup-dialog');
    assertDiffs(browser.checkDocument());
  });

  it('can dismiss the confirmation dialog by canceling', () => {
    const cancelBtnSelector = 'popup-dialog paper-button:nth-of-type(2)';
    browser.click(cancelBtnSelector);
    assertDiffs(browser.checkDocument());
  });

  it('displays an error message dialog when deleting a pipeline fails', () => {
    // Second pipeline cannot be deleted.
    const pipelineSelector =
        'app-shell pipeline-list item-list #listContainer paper-item:nth-of-type(2)';
    browser.click(pipelineSelector);

    const deleteBtnSelector = 'app-shell pipeline-list paper-button#deleteBtn';
    browser.waitForEnabled(deleteBtnSelector);
    browser.click(deleteBtnSelector);

    browser.waitForExist('popup-dialog');

    const confirmDeletionSelector = 'popup-dialog paper-button';
    browser.click(confirmDeletionSelector);
    assertDiffs(browser.checkDocument());
  });

  it('can dismiss the error message dialog', () => {
    const dismissBtnSelector = 'popup-dialog paper-button';
    browser.click(dismissBtnSelector);
    assertDiffs(browser.checkDocument());
  });

  it('allows the list to be sorted. Defaults to ascending order', () => {
    // Sort by Pipeline 'Name' column (ascending)
    const columnButtonSelector =
        'app-shell pipeline-list item-list #header::div:nth-of-type(2)::paper-button';
    browser.click(columnButtonSelector);

    assertDiffs(browser.checkDocument());
  });

  it('sorts in descending order on second time a column is clicked', () => {
    // Sort by Pipeline 'Name' column (descending)
    // Sort will be descending now since it has already been clicked once in the previous test.
    const columnButtonSelector =
        'app-shell pipeline-list item-list #header::div:nth-of-type(2)::paper-button';

    browser.click(columnButtonSelector);

    // List should be reset to first page of results
    assertDiffs(browser.checkDocument());
  });

  // TODO: add clone tests

  // TODO: add pipeline details tests
});

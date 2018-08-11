const assert = require('assert');

function assertDiffs(results) {
  results.forEach(r => assert.ok(r.isExactSameImage));
}

describe('list jobs', () => {

  beforeAll(() => {
    browser.url('/');
  });

  it('shows list of jobs on first page', () => {
    const selector = 'app-shell job-list';

    browser.waitForVisible(selector);
    assertDiffs(browser.checkDocument());
  });

  it('shows hover effect on job', () => {
    const selector = 'app-shell job-list item-list #listContainer paper-item';

    browser.moveToObject(selector, 0, 0);
    assertDiffs(browser.checkDocument());
  });

  it('enables Clone when job clicked', () => {
    const selector = 'app-shell job-list item-list #listContainer paper-item';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('disables Clone when multiple jobs selected', () => {
    const selector = 'app-shell job-list item-list paper-checkbox';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('loads next page on "next" button press', () => {
    const selector = 'app-shell job-list item-list paper-button#nextPage';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('loads previous page on "previous" button press after pressing "next"', () => {
    const previousButtonSelector = 'app-shell job-list item-list paper-button#previousPage';
    browser.click(previousButtonSelector);

    assertDiffs(browser.checkDocument());
  });

  it('resets the pipelne list after clicking into a job', () => {
    // Navigate to second page
    const nextButtonSelector = 'app-shell job-list item-list paper-button#nextPage';
    browser.click(nextButtonSelector);

    // Go to a job's details page
    const jobSelector = 'app-shell job-list item-list #listContainer paper-item';
    browser.doubleClick(jobSelector);

    // Return to job list page
    const backButtonSelector = 'app-shell job-details .toolbar-arrow-back';
    browser.waitForVisible(backButtonSelector);
    browser.click(backButtonSelector);

    // List should be reset to first page of results
    assertDiffs(browser.checkDocument());
  });

  it('loads additional jobs after changing page size', () => {
    // Default is 20, but we'll change it to 50.
    const pageSizeDropdownSelector = 'app-shell job-list item-list paper-dropdown-menu';
    browser.click(pageSizeDropdownSelector);

    const pageSizeSelector =
        'app-shell job-list item-list paper-dropdown-menu::paper-item:nth-of-type(2)';
    browser.click(pageSizeSelector);

    assertDiffs(browser.checkDocument());
  });

  it('allows the list to be sorted. Defaults to ascending order', () => {
    // Sort by Pipeline ID column (ascending)
    const columnButtonSelector =
        'app-shell job-list item-list #header::div:nth-of-type(4)::paper-button';
    browser.click(columnButtonSelector);

    assertDiffs(browser.checkDocument());
  });

  it('sorts in descending order on second time a column is clicked', () => {
    // Sort by Pipeline ID column (descending)
    // Sort will be descending now since it has already been clicked once in the previous test.
    const pipelineIdColumnButtonSelector =
        'app-shell job-list item-list #header::div:nth-of-type(4)::paper-button';

    browser.click(pipelineIdColumnButtonSelector);

    // List should be reset to first page of results
    assertDiffs(browser.checkDocument());
  });

  it('allows the list to be filtered by job name', () => {
    // Open up the filter box
    const filterButtonSelector = 'app-shell job-list item-list paper-icon-button';
    browser.click(filterButtonSelector);

    const filterBoxSelector =
        'app-shell job-list item-list #headerContainer::div:nth-of-type(2)::input';
    browser.setValue(filterBoxSelector, 'can');

    assertDiffs(browser.checkDocument());
  });

  it('allows the list to be filtered and sorted', () => {
    // List is already filtered from previous test
    // Sort by job name column, click twice to invert ordering.
    const nameColumnButtonSelector =
        'app-shell job-list item-list #header::div:nth-of-type(2)::paper-button';
    browser.click(nameColumnButtonSelector);
    browser.click(nameColumnButtonSelector);

    assertDiffs(browser.checkDocument());

    // Clear filter by clicking the button again.
    const filterButtonSelector = 'app-shell job-list item-list paper-icon-button';
    browser.click(filterButtonSelector);
    // Reset sorting to default, which is ascending by created time.
    const createdAtColumnButtonSelector =
        'app-shell job-list item-list #header::div:nth-of-type(5)::paper-button';
    browser.click(createdAtColumnButtonSelector);
  });

  it('populates cloned job', () => {
    // Find a job with pipeline ID of 1 so it can be cloned.
    // TODO: Explore making this more reliable
    const selector = 'app-shell job-list item-list #listContainer paper-item';
    browser.doubleClick(selector);

    const cloneBtnSelector = 'app-shell job-list paper-button#cloneBtn';
    browser.click(cloneBtnSelector);

    assertDiffs(browser.checkDocument());
  });
});

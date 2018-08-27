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

describe('list jobs', () => {

  before(() => {
    browser.url('/');
  });

  it('shows list of jobs on first page', () => {
    const selector = 'app-shell job-list';

    browser.waitForVisible(selector);
    assertDiffs(browser.checkDocument());
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
    const selector = 'app-shell job-list item-list #nextPage';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('loads previous page on "previous" button press after pressing "next"', () => {
    const previousButtonSelector = 'app-shell job-list item-list #previousPage';
    browser.click(previousButtonSelector);

    assertDiffs(browser.checkDocument());
  });

  it('resets the pipelne list after clicking into a job', () => {
    // Navigate to second page
    const nextButtonSelector = 'app-shell job-list item-list #nextPage';
    browser.click(nextButtonSelector);

    // Go to a job's details page
    const jobSelector = 'app-shell job-list item-list #listContainer paper-item';
    browser.doubleClick(jobSelector);

    // Return to job list page
    const backButtonSelector = 'app-shell job-details #allJobsLink';
    browser.waitForVisible(backButtonSelector);
    browser.click(backButtonSelector);

    // List should be reset to first page of results
    assertDiffs(browser.checkDocument());
  });

  it('shows confirmation dialog when trying to delete a job', () => {
    // The list is now filtered to show jobs that cannot be deleted
    const selector = 'app-shell job-list item-list #listContainer paper-item';
    browser.click(selector);

    const deleteBtnSelector = 'app-shell job-list paper-button#deleteBtn';
    browser.waitForEnabled(deleteBtnSelector);
    browser.click(deleteBtnSelector);

    browser.waitForExist('popup-dialog');
    assertDiffs(browser.checkDocument());
  });

  it('can dismiss the confirmation dialog by canceling', () => {
    browser.click('popup-dialog paper-button:nth-of-type(2)');
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

    // Reset to default sort, which is created at, ascending
    const createdAtColumnButtonSelector =
        'app-shell job-list item-list #header::div:nth-of-type(5)::paper-button';
    browser.click(createdAtColumnButtonSelector);
  });

  it('populates cloned job', () => {
    // Find a job with pipeline ID of 1 so it can be cloned.
    // TODO: Explore making this more reliable
    const selector = 'app-shell job-list item-list #listContainer paper-item';
    browser.waitForVisible(selector);
    browser.click(selector);

    const cloneBtnSelector = 'app-shell job-list paper-button#cloneBtn';
    browser.waitForEnabled(cloneBtnSelector);
    browser.click(cloneBtnSelector);

    assertDiffs(browser.checkDocument());
  });
});

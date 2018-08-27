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

describe('view job details', () => {

  before(() => {
    browser.url('/');
  });

  it('opens job details on double click', () => {
    // Find a job with runs that can also be cloned. The 3rd job is one.
    // TODO: Explore making this more reliable
    const selector = 'app-shell job-list item-list #listContainer paper-item:nth-of-type(3)';

    browser.waitForVisible(selector);
    browser.doubleClick(selector);
    assertDiffs(browser.checkDocument());
  });

  it('can switch to run list tab', () => {
    const selector = 'app-shell job-details paper-tab:last-child';

    browser.click(selector);
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

  it('loads next page on "next" button press', () => {
    const nextButtonselector =
        'app-shell job-details run-list item-list #nextPage';

    browser.click(nextButtonselector);
    assertDiffs(browser.checkDocument());
  });

  it('loads previous page on "previous" button press after pressing "next"', () => {
    const previousButtonselector =
        'app-shell job-details run-list item-list #previousPage';
    browser.click(previousButtonselector);

    assertDiffs(browser.checkDocument());
  });

  it('loads additional runs after changing page size', () => {
    // Default is 20, but we'll change it to 50.
    const pageSizeDropdownSelector =
        'app-shell job-details run-list item-list paper-dropdown-menu';
    browser.click(pageSizeDropdownSelector);

    const pageSizeSelector =
        'app-shell job-details run-list item-list ' +
        'paper-dropdown-menu::paper-item:nth-of-type(2)';
    browser.click(pageSizeSelector);

    assertDiffs(browser.checkDocument());
  });

  it('allows the list to be sorted. Defaults to ascending order', () => {
    // Sort by Run Name column (ascending)
    const runNameColumnButtonSelector =
        'app-shell job-details run-list item-list #header::div:nth-of-type(2)::paper-button';
    browser.click(runNameColumnButtonSelector);

    assertDiffs(browser.checkDocument());
  });

  it('sorts in descending order on second time a column is clicked', () => {
    // Sort by Run Name column (descending)
    // Sort will be descending now since it has already been clicked once in the previous test.
    const runNameColumnButtonSelector =
        'app-shell job-details run-list item-list #header::div:nth-of-type(2)::paper-button';

    browser.click(runNameColumnButtonSelector);

    // List should be reset to first page of results
    assertDiffs(browser.checkDocument());
  });

  it('populates new job on clone', () => {
    const cloneBtnSelector = 'app-shell job-details paper-button#cloneBtn';

    browser.click(cloneBtnSelector);
    assertDiffs(browser.checkDocument());
  });
});

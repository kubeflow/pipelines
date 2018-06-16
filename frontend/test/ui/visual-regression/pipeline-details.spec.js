const assert = require('assert');

function assertDiffs(results) {
  results.forEach(r => assert.ok(r.isExactSameImage));
}

describe('view pipeline details', () => {

  beforeAll(() => {
    browser.url('/');
  });

  it('opens pipeline details on double click', () => {
    // Find a pipeline with jobs that can also be cloned. The 3rd pipeline is one.
    // TODO: Explore making this more reliable
    const selector = 'app-shell pipeline-list item-list paper-item:nth-of-type(3)';

    browser.waitForVisible(selector);
    browser.doubleClick(selector);
    assertDiffs(browser.checkDocument());
  });

  it('can switch to run list tab', () => {
    const selector = 'app-shell pipeline-details paper-tab:last-child';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('loads next page on "next" button press', () => {
    const nextButtonselector = 'app-shell pipeline-details job-list item-list paper-button#nextPage';

    browser.click(nextButtonselector);
    assertDiffs(browser.checkDocument());
  });

  it('loads previous page on "previous" button press after pressing "next"', () => {
    const previousButtonselector = 'app-shell pipeline-details job-list item-list paper-button#previousPage';
    browser.click(previousButtonselector);

    assertDiffs(browser.checkDocument());
  });

  it('populates new pipeline on clone', () => {
    const selector = 'app-shell pipeline-details paper-button';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });
});

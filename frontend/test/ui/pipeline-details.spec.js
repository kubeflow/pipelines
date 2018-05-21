const assert = require('assert');

function assertDiffs(results) {
  results.forEach(r => assert.ok(r.isExactSameImage));
}

describe('view pipeline details', () => {

  beforeAll(() => {
    browser.url('/');
  });

  it('opens pipeline details on double click', () => {
    // Find a pipeline with jobs that can also be cloned. The 6th pipeline is one.
    // TODO: Explore making this more reliable
    const selector = 'app-shell pipeline-list item-list paper-item:nth-of-type(6)';

    browser.waitForVisible(selector);
    browser.doubleClick(selector);
    assertDiffs(browser.checkDocument());
  });

  it('can switch to run list tab', () => {
    const selector = 'app-shell pipeline-details paper-tab:last-child';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('populates new pipeline on clone', () => {
    const selector = 'app-shell pipeline-details paper-button';

    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });
});

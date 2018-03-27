const assert = require('assert');

function assertDiffs(results) {
  results.forEach(r => assert.ok(r.isExactSameImage));
}

describe('create new pipeline', () => {

  beforeAll(() => {
    browser.url('/');
  });

  it('opens new pipeline page', () => {
    const selector = 'app-shell pipeline-list paper-button';

    browser.waitForVisible(selector);
    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('shows package list', () => {
    const selector = 'app-shell pipeline-new paper-dropdown-menu';

    browser.waitForVisible(selector);
    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('selects first package from list', () => {
    const selector = 'app-shell pipeline-new paper-item';

    browser.waitForVisible(selector);
    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('populates pipeline details', () => {
    const selector = 'app-shell pipeline-new paper-item';

    // Skip the upload package button
    browser.keys('Tab');

    browser.keys('Tab');
    browser.keys('test pipeline name');

    browser.keys('Tab');
    browser.keys('test pipeline description');

    browser.keys('Tab');
    browser.keys('test x value');

    browser.keys('Tab');
    browser.keys('test y value. this can be very long, and should not wrap around.');

    browser.keys('Tab');
    browser.keys('test output value');

    assertDiffs(browser.checkDocument());
  });

  // TODO: mock POST for deploying the pipeline is not supported yet.
});

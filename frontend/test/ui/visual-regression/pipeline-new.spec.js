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

  it('validates pipeline input', () => {
    const selector = 'app-shell pipeline-new #description';

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

  it('enables deploy button', () => {
    const selector = 'app-shell pipeline-new .scrollable-padded';
    browser.keys('PageDown');

    assertDiffs(browser.checkDocument());
  });

  // TODO: add visual regression tests for various ways to schedule a pipeline.

  it('resets the page when the user navigates away', () => {
    // Select a package
    const backSelector = 'app-shell pipeline-new paper-icon-button';
    browser.waitForVisible(backSelector);
    browser.click(backSelector);

    const newPipelineSelector = 'app-shell pipeline-list paper-button#newBtn'
    browser.waitForVisible(newPipelineSelector);
    browser.click(newPipelineSelector);

    assertDiffs(browser.checkDocument());
  });

  // TODO: Add test for cloning a pipeline with no params after a selecting a package with params.

  // TODO: Add test for test pipeline deployment failure by making a package that always fails when
  // used in pipeline deployment.

  // TODO: Add test for error reset. Try to deploy package that always fails, take a screenshot of
  // the error, navigate away then back, and make sure the error is gone.

  // TODO: mock POST for deploying the pipeline is not supported yet.
});

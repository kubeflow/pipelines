const assert = require('assert');

function assertDiffs(results) {
  results.forEach(r => assert.ok(r.isExactSameImage));
}

describe('create new job', () => {

  beforeAll(() => {
    browser.url('/');
  });

  it('opens new job page', () => {
    const selector = 'app-shell job-list paper-button';

    browser.waitForVisible(selector);
    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('validates job input', () => {
    const selector = 'app-shell job-new #description';

    browser.waitForVisible(selector);
    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('shows pipeline list', () => {
    const selector = 'app-shell job-new paper-dropdown-menu';

    browser.waitForVisible(selector);
    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('selects first pipeline from list', () => {
    const selector = 'app-shell job-new paper-item';

    browser.waitForVisible(selector);
    browser.click(selector);
    assertDiffs(browser.checkDocument());
  });

  it('populates job details', () => {
    // Skip the upload pipeline button
    browser.keys('Tab');

    browser.keys('Tab');
    browser.keys('test job name');

    browser.keys('Tab');
    browser.keys('test job description');

    browser.keys('Tab');
    browser.keys('test x value');

    browser.keys('Tab');
    browser.keys('test y value. this can be very long, and should not wrap around.');

    browser.keys('Tab');
    browser.keys('test output value');

    assertDiffs(browser.checkDocument());
  });

  it('enables deploy button', () => {
    const selector = 'app-shell job-new .scrollable-padded';
    browser.keys('PageDown');

    assertDiffs(browser.checkDocument());
  });

  // TODO: add visual regression tests for various ways to schedule a job.

  it('resets the page when the user navigates away', () => {
    // Select a pipeline
    const backSelector = 'app-shell job-new paper-icon-button';
    browser.waitForVisible(backSelector);
    browser.click(backSelector);

    const newJobSelector = 'app-shell job-list paper-button#newBtn'
    browser.waitForVisible(newJobSelector);
    browser.click(newJobSelector);

    assertDiffs(browser.checkDocument());
  });

  // TODO: Add test for cloning a job with no params after a selecting a pipeline with params.

  // TODO: Add test for test job deployment failure by making a pipeline that always fails when
  // used in job deployment.

  // TODO: Add test for error reset. Try to deploy pipeline that always fails, take a screenshot of
  // the error, navigate away then back, and make sure the error is gone.

  // TODO: mock POST for deploying the job is not supported yet.
});

const assert = require('assert');

/**
 * This file navigates to the component tests page, and waits until the test
 * status is printed into a special div on the document.
 */
describe('component tests', () => {
  it('starts component tests', () => {
    browser.url('/_componentTests');
    browser.waitForText('#testFailures', 60000);

    const failures = browser.getText('#testFailures');
    const failuresText =
        JSON.parse(failures).map(f => '- ' + f.title + ':\n  ' + f.error + '\n').join('\n');
    assert(failures === '[]',
        'component tests failed with the following errors:\n' + failuresText);
  });
});

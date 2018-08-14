const assert = require('assert');
const URL = require('url').URL;

const jobName = 'helloworld-' + Date.now();
const jobDescription = 'test job description ' + jobName;
const waitTimeout = 5000;

describe('deploy new job', () => {

  before(() => {
    browser.url('/');
  });

  it('starts out with no jobs', () => {
    const selector = 'app-shell job-list item-list #listContainer';
    browser.waitForVisible(selector, waitTimeout);

    assert.equal(browser.getText(selector), 'No jobs found. Click New Job to start.',
        'initial job list is not empty');
  });

  it('opens new job page', () => {
    const selector = 'app-shell job-list paper-button';

    browser.waitForVisible(selector, waitTimeout);
    browser.click(selector);
    assert.equal(new URL(browser.getUrl()).pathname, '/jobs/new');
  });

  it('uploads the hello world pipeline', () => {
    const selector = 'app-shell job-new #deployButton';
    browser.waitForVisible(selector, waitTimeout);

    // Show the alt upload button
    browser.execute(`document.querySelector('app-shell').shadowRoot
                        .querySelector('job-new').shadowRoot
                        .querySelector('#altFileUpload').style.display=''`);
    const uploadSelector = 'app-shell job-new #altFileUpload';
    browser.waitForVisible(uploadSelector, waitTimeout);
    browser.chooseFile(uploadSelector, './hello-world.yaml');

    // Hide the alt upload button
    browser.execute(`document.querySelector('app-shell').shadowRoot
                        .querySelector('job-new').shadowRoot
                        .querySelector('#altFileUpload').style.display='none'`);
    const pkgIdSelector = 'app-shell job-new paper-dropdown-menu ' +
                          'paper-menu-button::paper-input paper-input-container::iron-input';
    browser.waitForValue(pkgIdSelector, 5 * waitTimeout);
  });

  it('populates job details and deploys', () => {
    browser.click('app-shell job-new #name');
    browser.keys(jobName);

    browser.keys('Tab');
    browser.keys(jobDescription);

    browser.click('app-shell job-new #deployButton');
  });

  it('redirects back to job list page', () => {
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname === '/jobs';
    }, waitTimeout);
  });

  it('finds the new job in the list of jobs', () => {
    const selector = 'app-shell job-list item-list #listContainer paper-item';
    browser.waitForVisible(selector, waitTimeout);
    assert.equal($$(selector).length, 1, 'should only show one job');

    // Navigate to details of the deployed job
    browser.doubleClick(selector + `:nth-of-type(1)`);
  });

  it('displays job name correctly', () => {
    const selector = 'app-shell job-details .job-name';
    browser.waitForVisible(selector, waitTimeout);
    assert(browser.getText(selector).startsWith(jobName),
        'job name is not shown correctly: ' + browser.getText(selector));
  });

  it('displays job description correctly', () => {
    const selector = 'app-shell job-details .description.value';
    browser.waitForVisible(selector, waitTimeout);
    assert.equal(browser.getText(selector), jobDescription,
        'job description is not shown correctly');
  });

  it('displays job created at date correctly', () => {
    const selector = 'app-shell job-details .created-at.value';
    browser.waitForVisible(selector, waitTimeout);
    const createdDate = Date.parse(browser.getText(selector));
    assert(Date.now() - createdDate < 5 * 1000,
        'job created date should be within the last five seconds');
  });

  it('switches to run list tab', () => {
    const selector = 'app-shell job-details paper-tab:last-child';
    browser.click(selector);
  });

  it('schedules and lists exactly one run', (done) => {
    const listSelector = 'app-shell job-details run-list item-list #listContainer';
    browser.waitForVisible(listSelector, waitTimeout);

    let attempts = 0;

    const selector = 'app-shell job-details run-list item-list #listContainer paper-item';
    let items = $$(selector);
    const maxAttempts = 80;

    while (attempts < maxAttempts && (!items || items.length === 0)) {
      browser.click('app-shell job-details paper-button#refreshBtn');
      browser.pause(1000);
      items = $$(selector);
      attempts++;
    }

    assert(attempts < maxAttempts, `waited for ${maxAttempts} seconds but run did not start`);
    assert(items && items.length > 0, 'only one run should show up');

    const runsText = browser.getText(selector);
    assert(!Array.isArray(runsText) && typeof runsText === 'string',
      'only one run should show up');
    assert(runsText.startsWith('job-helloworld'),
      'run name should start with job-helloworld: ' + runsText);
  });

  it('opens run details on double click', () => {
    const selector = 'app-shell job-details run-list item-list #listContainer paper-item';

    browser.waitForVisible(selector, waitTimeout);
    browser.doubleClick(selector);

    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname.startsWith('/jobRun');
    }, waitTimeout);
  });

  it('deletes the job', () => {
    const backBtn = 'app-shell run-details #jobLink';
    browser.waitForVisible(backBtn, waitTimeout);
    browser.click(backBtn);

    const selector = 'app-shell job-details #deleteBtn';
    browser.waitForVisible(selector);
    browser.click(selector);

    browser.pause(500);
    browser.click('popup-dialog paper-button');
  });

  it('redirects back to job list page', () => {
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname === '/jobs';
    }, waitTimeout);
  });

  it('shows an empty list of jobs after deletion', () => {
    const selector = 'app-shell job-list item-list #listContainer';
    browser.waitForVisible(selector, waitTimeout);

    assert.equal(browser.getText(selector), 'No jobs found. Click New Job to start.',
        'final job list is not empty');
  });

});

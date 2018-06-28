const assert = require('assert');
const URL = require('url').URL;

const pipelineName = 'helloworld-' + Date.now();
const pipelineDescription = 'test pipeline description ' + pipelineName;
const waitTimeout = 5000;

describe('deploy new pipeline', () => {

  beforeAll(() => {
    browser.url('/');
  });

  it('starts out with no pipelines', () => {
    const selector = 'app-shell pipeline-list item-list #listContainer';
    browser.waitForVisible(selector, waitTimeout);

    assert.equal(browser.getText(selector), '', 'initial pipeline list is not empty');
  });

  it('opens new pipeline page', () => {
    const selector = 'app-shell pipeline-list paper-button';

    browser.waitForVisible(selector, waitTimeout);
    browser.click(selector);
    assert.equal(new URL(browser.getUrl()).pathname, '/pipelines/new');
  });

  it('uploads the hello world package', () => {
    const selector = 'app-shell pipeline-new #deployButton';
    browser.waitForVisible(selector, waitTimeout);

    // Show the alt upload button
    browser.execute(`document.querySelector('app-shell').shadowRoot
                        .querySelector('pipeline-new').shadowRoot
                        .querySelector('#altFileUpload').style.display=''`);
    const uploadSelector = 'app-shell pipeline-new #altFileUpload';
    browser.waitForVisible(uploadSelector, waitTimeout);
    browser.chooseFile(uploadSelector, './hello-world.yaml');

    // Hide the alt upload button
    browser.execute(`document.querySelector('app-shell').shadowRoot
                        .querySelector('pipeline-new').shadowRoot
                        .querySelector('#altFileUpload').style.display='none'`);
    const pkgIdSelector = 'app-shell pipeline-new paper-dropdown-menu ' +
                          'paper-menu-button::paper-input paper-input-container::iron-input';
    browser.waitForValue(pkgIdSelector, waitTimeout);
  });

  it('populates pipeline details and deploys', () => {
    browser.click('app-shell pipeline-new #name');
    browser.keys(pipelineName);

    browser.keys('Tab');
    browser.keys(pipelineDescription);

    browser.click('app-shell pipeline-new #deployButton');
  });

  it('redirects back to pipeline list page', () => {
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname === '/pipelines';
    }, waitTimeout);
  });

  it('finds the new pipeline ' + pipelineName + ' in the list of pipelines', () => {
    const selector = 'app-shell pipeline-list item-list paper-item';
    browser.waitForVisible(selector, waitTimeout);
    assert.equal($$(selector).length, 1, 'should only show one pipeline');

    // Navigate to details of the deployed pipeline
    browser.doubleClick(selector + `:nth-of-type(1)`);
  });

  it('displays pipeline name correctly', () => {
    const selector = 'app-shell pipeline-details .pipeline-name';
    browser.waitForVisible(selector, waitTimeout);
    assert(browser.getText(selector).startsWith(pipelineName),
        'pipeline name is not shown correctly: ' + browser.getText(selector));
  });

  it('displays pipeline description correctly', () => {
    const selector = 'app-shell pipeline-details .description.value';
    browser.waitForVisible(selector, waitTimeout);
    assert.equal(browser.getText(selector), pipelineDescription,
        'pipeline description is not shown correctly');
  });

  it('displays pipeline created at date correctly', () => {
    const selector = 'app-shell pipeline-details .created-at.value';
    browser.waitForVisible(selector, waitTimeout);
    const createdDate = Date.parse(browser.getText(selector));
    assert(Date.now() - createdDate < 5 * 1000,
        'pipeline created date should be within the last five seconds');
  });

  it('switches to run list tab', () => {
    const selector = 'app-shell pipeline-details paper-tab:last-child';
    browser.click(selector);
  });

  it('lists exactly one job', () => {
    const selector = 'app-shell pipeline-details job-list item-list paper-item';
    const jobsText = browser.getText(selector);

    assert(!Array.isArray(jobsText) && typeof jobsText === 'string',
        'only one job should show up');
    assert(jobsText.startsWith('hello-world'), 'job name should start with hello-world');
  });

  it('opens job details on double click', () => {
    const selector = 'app-shell pipeline-details job-list item-list paper-item';

    browser.waitForVisible(selector, waitTimeout);
    browser.doubleClick(selector);

    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname.startsWith('/pipelineJob');
    }, waitTimeout);
  });

  it('deletes the pipeline', () => {
    const backBtn = 'app-shell job-details .toolbar-arrow-back';
    browser.waitForVisible(backBtn, waitTimeout);
    browser.click(backBtn);

    const selector = 'app-shell pipeline-details #deleteBtn';
    browser.waitForVisible(selector);
    browser.click(selector);
  });

  it('redirects back to pipeline list page', () => {
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname === '/pipelines';
    }, waitTimeout);
  });

  it('shows an empty list of pipelines after deletion', () => {
    const selector = 'app-shell pipeline-list item-list #listContainer';
    browser.waitForVisible(selector, waitTimeout);

    assert.equal(browser.getText(selector), '', 'initial pipeline list is not empty');
  });

});

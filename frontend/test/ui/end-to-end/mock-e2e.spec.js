const assert = require('assert');
const URL = require('url').URL;
const fixedData = require('../../../mock-backend/fixed-data').data;

const pipelineName = 'helloworld-' + Date.now();
const pipelineDescription = 'test pipeline description ' + pipelineName;
const waitTimeout = 5000;
const listSelector = 'app-shell pipeline-list item-list';
const mockPipelinesLength = fixedData.pipelines.length;

describe('deploy new pipeline', () => {

  beforeAll(() => {
    browser.url('/');
  });

  it('starts out with all mock pipelines', () => {
    browser.waitForVisible(listSelector, waitTimeout);
    const selector = 'app-shell pipeline-list item-list paper-item';
    browser.waitForVisible(selector, waitTimeout);

    assert.equal($$(selector).length, 20, 'should start out with a full page of 20 pipelines');
  });

  it('shows the second and final page of pipelines', () => {
    const selector = 'app-shell pipeline-list item-list #nextPage';
    browser.waitForVisible(selector, waitTimeout);
    browser.click(selector);

    const listSelector = 'app-shell pipeline-list item-list paper-item';
    browser.waitForVisible(selector, waitTimeout);
    assert.equal($$(listSelector).length, mockPipelinesLength - 20,
        'second page should show the remaining pipelines');
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
    browser.keys(pipelineName);

    browser.keys('Tab');
    browser.keys(pipelineDescription);

    browser.keys('Tab');
    browser.keys('x param value');
    browser.keys('Tab');
    browser.keys('y param value');
    browser.keys('Tab');
    browser.keys('output param value');

    browser.click('app-shell pipeline-new #deployButton');
  });

  it('redirects back to pipeline list page', () => {
    browser.waitUntil(() => {
      return new URL(browser.getUrl()).pathname === '/pipelines';
    }, waitTimeout);
  });

  it('finds the new pipeline ' + pipelineName + ' in the list of pipelines', () => {
    // Navigate to second page, where the new pipeline should be
    const nextPageSelector = 'app-shell pipeline-list item-list #nextPage';
    browser.waitForVisible(nextPageSelector, waitTimeout);
    browser.click(nextPageSelector);

    browser.waitForVisible(listSelector, waitTimeout);

    const selector = 'app-shell pipeline-list item-list paper-item';
    browser.waitForVisible(selector, waitTimeout);
    const index = mockPipelinesLength - 20 + 1;
    assert.equal($$(selector).length, index, 'should have a new item added to pipeline list');

    // Navigate to details of the deployed pipeline
    browser.doubleClick(selector + `:nth-of-type(${index})`);
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
    const now = new Date();
    assert(Date.now() - createdDate < 5 * 1000,
        'pipeline created date should be within the last five seconds: ' + createdDate);
  });

  it('displays pipeline parameters correctly', () => {
    const selector = 'app-shell pipeline-details .params-table';
    browser.waitForVisible(selector, waitTimeout);

    const paramsSelector = 'app-shell pipeline-details .params-table';
    assert.deepEqual(browser.getText(paramsSelector),
                     'x\nx param value\ny\ny param value\noutput\noutput param value',
                     'parameter values are incorrect');
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
    assert(jobsText.startsWith(fixedData.jobs[0].job.name), 'job name is incorrect');
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

  it(`shows only ${mockPipelinesLength - 20} pipelines on second page after deletion`, () => {
    const selector = 'app-shell pipeline-list item-list #nextPage';
    browser.waitForVisible(selector, waitTimeout);
    browser.click(selector);

    const listSelector = 'app-shell pipeline-list item-list paper-item';
    browser.waitForVisible(selector, waitTimeout);
    assert.equal($$(listSelector).length, mockPipelinesLength - 20,
        'second page should show the remaining pipelines');
  });

});

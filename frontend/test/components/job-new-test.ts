import * as sinon from 'sinon';
import * as assert from '../../node_modules/assert/assert';
import * as APIs from '../../src/lib/apis';

import { Job } from '../../src/api/job';
import { Parameter } from '../../src/api/parameter';
import { Pipeline} from '../../src/api/pipeline';
import { JobNew } from '../../src/components/job-new/job-new';
import { JobSchedule } from '../../src/components/job-schedule/job-schedule';
import { resetFixture } from './test-utils';

import * as fixedData from '../../mock-backend/fixed-data';

const pipelines = fixedData.namedPipelines;

const TEST_TAG = 'job-new';

let fixture: JobNew;
let listPipelinesStub: sinon.SinonStub;

async function _resetFixture(): Promise<void> {
  return resetFixture('job-new', null, (f: JobNew) => {
    fixture = f;
    return f.load('', {});
  });
}

describe('job-new', () => {

  before(() => {
    listPipelinesStub = sinon.stub(APIs, 'listPipelines');
    listPipelinesStub.returns({ pipelines: fixedData.data.pipelines });
  });

  /** Unset all fields and dropdowns */
  beforeEach(async () => {
    await _resetFixture();
  });

  it('starts out with deploy button disabled', () => {
    assert(fixture.deployButton.disabled, 'Deploy button should initially be disabled.');
  });

  it('starts out with no pipeline selected', () => {
    assert.strictEqual(fixture.listBox.selected, -1,
        'No pipeline should initially be selected.');
  });

  it('starts out with Job name field empty', () => {
    assert.strictEqual(fixture.nameInput.value, '',
        'Job name field should initially be empty.');
  });

  it('starts out with Job description field empty', () => {
    assert.strictEqual(fixture.descriptionInput.value, '',
        'Job description field should initially be empty.');
  });

  it('starts out with no parameter fields displayed', () => {
    assert(!fixture.$.jobParameters.querySelector('#parametersTitle'),
        'Parameters title should not exist before pipeline is selected');
    assert.strictEqual(fixture.$.jobParameters.querySelectorAll('paper-input').length, 0,
        'Pipeline parameters should not exist before pipeline is selected');
  });

  it('pipeline selector should contain all pipelines', () => {
    assert.deepStrictEqual(
        fixture.listBox.items.map((item) => item.pipelineId),
        fixture.pipelines.map((pkg) => pkg.id));
  });

  it('displays parameter fields after pipeline is selected', async () => {
    listPipelinesStub.returns({ pipelines: [pipelines.examplePipeline] });
    await _resetFixture();

    fixture.listBox.select(0);
    Polymer.flush();

    assert(fixture.$.jobParameters.querySelector('#parametersTitle'),
        'Pipeline parameters should exist after pipeline is selected');
    Array.from(fixture.$.jobParameters.querySelectorAll('paper-input')).forEach((input, i) =>
        assert.strictEqual(input.label, pipelines.examplePipeline.parameters[i].name));
  });

  it('updates the displayed parameters when pipeline selection changes', async () => {
    listPipelinesStub.returns(
        { pipelines: [pipelines.examplePipeline, pipelines.examplePipeline2] });
    await _resetFixture();

    fixture.listBox.select(0);
    Polymer.flush();
    const oldParamNames = JSON.stringify(
        Array.from(fixture.$.jobParameters.querySelectorAll('paper-input'))
            .map((input) => input.label));

    fixture.listBox.select(1);
    Polymer.flush();
    const newParamNames = JSON.stringify(
        Array.from(fixture.$.jobParameters.querySelectorAll('paper-input'))
            .map((input) => input.label));

    assert.notStrictEqual(oldParamNames, newParamNames);
  });

  it('clears the displayed parameters when pipeline with no params is selected', async () => {
    listPipelinesStub.returns(
        { pipelines: [pipelines.examplePipeline, pipelines.noParamsPipeline] });
    await _resetFixture();

    fixture.listBox.select(0);
    Polymer.flush();
    assert(fixture.$.jobParameters.querySelectorAll('paper-input').length > 0,
        'This selected pipeline should have at least 1 parameter.');

    fixture.listBox.select(1);
    Polymer.flush();
    assert(_paramsNotVisible());
  });

  it('clears displayed parameters when pipeline with undefined params is selected', async () => {
    listPipelinesStub.returns(
        { pipelines: [pipelines.examplePipeline, pipelines.undefinedParamsPipeline] });
    await _resetFixture();

    fixture.listBox.select(0);
    Polymer.flush();
    assert(fixture.$.jobParameters.querySelectorAll('paper-input').length > 0,
        'This selected pipeline should have at least 1 parameter.');

    fixture.listBox.select(1);
    Polymer.flush();
    assert(_paramsNotVisible());
  });

  it('enables deploy button if pipeline selected and Job name is not empty', () => {
    fixture.listBox.select(0);
    Polymer.flush();
    fixture.nameInput.value = 'Some Job Name';

    assert(!fixture.deployButton.disabled,
        'Deploy button should be enabled if pipeline selected and Job name is not empty.');
  });

  it('disables deploy button if no pipeline is selected', () => {
    // Ensure that deploy button is enabled before testing lack of selected pipeline.
    fixture.listBox.select(0);
    Polymer.flush();
    fixture.nameInput.value = 'Some Job Name';
    assert(!fixture.deployButton.disabled, 'Deploy button should be enabled before real test.');

    fixture.listBox.select(-1);
    Polymer.flush();

    assert(fixture.deployButton.disabled,
        'Deploy button should be disabled if no pipeline selected.');
  });

  it('disables deploy button if Job name is empty', () => {
    // Ensure that deploy button is enabled before testing lack of name.
    fixture.listBox.select(0);
    Polymer.flush();
    fixture.nameInput.value = 'Some Job Name';
    assert(!fixture.deployButton.disabled,
        'Deploy button should be enabled before method under test.');

    fixture.nameInput.value = '';

    assert(fixture.deployButton.disabled,
        'Deploy button should be disabled if Job name is empty.');
  });

  it('constructs and passes the correct Job on deploy', () => {
    const deployJobStub = sinon.stub(APIs, 'newJob');
    deployJobStub.returns({});

    fixture.listBox.select(0);
    Polymer.flush();
    fixture.nameInput.value = 'Some Job Name';
    fixture.descriptionInput.value = 'The Job description';
    const parameterInputs = fixture.$.jobParameters.querySelectorAll('paper-input');
    parameterInputs.forEach((input, index) => (input as PaperInputElement).value = index + '');
    fixture.schedule.maxConcurrentRunsInput.value = '50';
    // Include default cron schedule
    fixture.schedule.scheduleTypeListbox.select(2);
    Polymer.flush();

    fixture.deployButton.click();

    assert(deployJobStub.calledOnce, 'Apis.newJob() should only be called once.');
    const actualJob = deployJobStub.firstCall.args[0] as Job;
    assert.strictEqual(actualJob.name, fixture.nameInput.value);
    assert.strictEqual(actualJob.description, fixture.descriptionInput.value);
    assert.strictEqual(actualJob.max_concurrency, fixture.schedule.maxConcurrentRuns);

    // TODO: mock time and test format.
    assert.strictEqual(actualJob.pipeline_id, (fixture.listBox.selectedItem as any).pipelineId);
    assert.strictEqual(actualJob.parameters.length, parameterInputs.length);
    for (let i = 0; i < parameterInputs.length; i++) {
      assert.strictEqual(
          actualJob.parameters[i].value, (parameterInputs[i] as PaperInputElement).value);
    }
    // "0 0 * * * ?" is the default schedule. It means "run every hour".
    assert.strictEqual(actualJob.trigger.cronExpression, '0 0 * * * ?');

    deployJobStub.restore();
  });

  describe('cloning', () => {

    it('starts out with a Pipeline pipeline selected', async () => {
      listPipelinesStub.returns({ pipelines: [pipelines.noParamsPipeline] });

      await fixture.load('', {}, { pipelineId: pipelines.noParamsPipeline.id, parameters: [] });
      Polymer.flush();

      assert.strictEqual(
          (fixture.listBox.selectedItem as any).pipelineId,
          pipelines.noParamsPipeline.id,
          'Pipeline should initially be selected.');
      assert(_paramsNotVisible());
    });

    it('starts out with empty Job name', async () => {
      listPipelinesStub.returns({ pipelines: [pipelines.noParamsPipeline] });

      await fixture.load('', {}, { pipelineId: pipelines.noParamsPipeline.id, parameters: [] });
      Polymer.flush();

      assert.strictEqual(fixture.nameInput.value, '',
          'Job name field should initially be empty.');
    });

    it('starts out with Job parameters matching cloned Job', async () => {
      listPipelinesStub.returns({ pipelines: [pipelines.examplePipeline2] });

      await fixture.load('', {}, {
        parameters: [
          { name: 'project', value: 'a-project-name' },
          { name: 'workers', value: '10' },
          { name: 'rounds', value: '5' },
          { name: 'output', value: 'some/output/path' }
        ],
        pipelineId: pipelines.examplePipeline2.id,
      });
      Polymer.flush();

      const params = Array.from(fixture.$.jobParameters.querySelectorAll('paper-input'))
          .map((param) => (param as PaperInputElement).value);
      assert.strictEqual(params[0], 'a-project-name');
      assert.strictEqual(params[1], '10');
      assert.strictEqual(params[2], '5');
      assert.strictEqual(params[3], 'some/output/path');
      Array.from(fixture.$.jobParameters.querySelectorAll('paper-input')).forEach((input, i) =>
            assert.strictEqual(input.label, pipelines.examplePipeline2.parameters[i].name));
    });
  });

  after(() => {
    document.body.removeChild(fixture);
    listPipelinesStub.restore();
  });
});

function _paramsNotVisible(): boolean {
  const params = fixture.$.jobParameters.querySelectorAll('paper-input');
  return params.length === 0 || Array.from(params)
    .map((el) => _isNotVisible(el))
    .reduce((prev, cur) => prev && cur);
}

function _isNotVisible(el: HTMLElement): boolean {
  return !el || el.hidden || el.style.visibility === 'hidden' || el.style.display === 'none';
}

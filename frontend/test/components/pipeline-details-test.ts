import * as sinon from 'sinon';
import * as fixedData from '../../mock-backend/fixed-data';
import * as assert from '../../node_modules/assert/assert';
import * as Apis from '../../src/lib/apis';
import * as Utils from '../../src/lib/utils';

import { ListJobsResponse } from '../../src/api/list_jobs_response';
import { Pipeline } from '../../src/api/pipeline';
import { JobList } from '../../src/components/job-list/job-list';
import { PageError } from '../../src/components/page-error/page-error';
import { PipelineDetails } from '../../src/components/pipeline-details/pipeline-details';
import { RouteEvent } from '../../src/model/events';
import { dialogStub, isVisible, notificationStub, resetFixture } from './test-utils';

let fixture: PipelineDetails;
let getPipelineStub: sinon.SinonStub;
let getJobsStub: sinon.SinonStub;
let deletePipelineStub: sinon.SinonStub;
let enablePipelinesStub: sinon.SinonStub;
let disablePipelinesStub: sinon.SinonStub;

async function _resetFixture(): Promise<void> {
  return resetFixture('pipeline-details', null, (f: PipelineDetails) => {
    fixture = f;
    f.load('1000');
  });
}

const testPipeline: Pipeline = {
  created_at: new Date().toISOString(),
  description: 'test pipeline description',
  enabled: false,
  enabled_at: '',
  id: 1000,
  name: 'test pipeline name',
  package_id: 2000,
  parameters: [],
  schedule: '',
};

describe('pipeline-details', () => {

  before(() => {
    getPipelineStub = sinon.stub(Apis, 'getPipeline');
    getPipelineStub.returns(testPipeline);

    const getJobsResponse: ListJobsResponse = {
      jobs: fixedData.data.jobs.map((j) => j.job),
      nextPageToken: '',
    };
    getJobsStub = sinon.stub(Apis, 'getJobs');
    getJobsStub.returns(getJobsResponse);
  });

  beforeEach(async () => {
    await _resetFixture();
  });

  it('shows the basic details of the pipeline without schedule', () => {
    const packageIdDiv = fixture.shadowRoot.querySelector('.package-id.value') as HTMLDivElement;
    assert(isVisible(packageIdDiv), 'cannot find package id div');
    assert.strictEqual(packageIdDiv.innerText, testPipeline.package_id.toString(),
        'displayed package id does not match test data');

    const descriptionDiv = fixture.shadowRoot.querySelector('.description.value') as HTMLDivElement;
    assert(isVisible(descriptionDiv), 'cannot find description div');
    assert.strictEqual(descriptionDiv.innerText, testPipeline.description,
        'displayed description does not match test data');

    const createdAtDiv = fixture.shadowRoot.querySelector('.created-at.value') as HTMLDivElement;
    assert(isVisible(createdAtDiv), 'cannot find createdAt div');
    assert.strictEqual(createdAtDiv.innerText, Utils.formatDateString(testPipeline.created_at),
        'displayed createdAt does not match test data');

    const scheduleDiv = fixture.shadowRoot.querySelector('.schedule.value') as HTMLDivElement;
    assert(!isVisible(scheduleDiv), 'should not show schedule div');

    const enabledDiv = fixture.shadowRoot.querySelector('.enabled.value') as HTMLDivElement;
    assert(!isVisible(enabledDiv), 'should not show enabled div');

    const paramsTable = fixture.shadowRoot.querySelector('.params-table') as HTMLDivElement;
    assert(!isVisible(paramsTable),
        'should not show params table for test pipeline with no params');
  });

  it('shows schedule and enabled info if a schedule is specified', async () => {
    testPipeline.schedule = 'some cron tab here';
    testPipeline.enabled = true;
    await _resetFixture();

    const scheduleDiv = fixture.shadowRoot.querySelector('.schedule.value') as HTMLDivElement;
    assert(isVisible(scheduleDiv), 'should find schedule div');
    assert.strictEqual(scheduleDiv.innerText, testPipeline.schedule,
        'displayed schedule does not match test data');

    const enabledDiv = fixture.shadowRoot.querySelector('.enabled.value') as HTMLDivElement;
    assert(isVisible(enabledDiv), 'should find enabled div');
    assert.strictEqual(enabledDiv.innerText,
        Utils.enabledDisplayString(testPipeline.schedule, testPipeline.enabled),
        'displayed enabled does not match test data');
    testPipeline.schedule = '';
  });

  it('shows parameters table if there are parameters', async () => {
    testPipeline.parameters = [
      { name: 'param1', value: 'value1' },
      { name: 'param2', value: 'value2' },
    ];
    await _resetFixture();

    const paramsTable = fixture.shadowRoot.querySelector('.params-table') as HTMLDivElement;
    assert(isVisible(paramsTable), 'should show params table');
    const paramRows = paramsTable.querySelectorAll('div');
    assert.strictEqual(paramRows.length, 2, 'there should be two rows of parameters');
    paramRows.forEach((row, i) => {
      const key = row.querySelector('.key') as HTMLDivElement;
      const value = row.querySelector('.value') as HTMLDivElement;
      assert.strictEqual(key.innerText, testPipeline.parameters[i].name);
      assert.strictEqual(value.innerText, testPipeline.parameters[i].value);
    });
  });

  it('shows the list of jobs upon switching to Jobs tab', () => {
    const jobList = fixture.shadowRoot.querySelector('job-list') as JobList;
    assert(!isVisible(jobList), 'should not show jobs div by default');

    fixture.tabs.select(1);
    Polymer.flush();

    assert(isVisible(jobList), 'should not show jobs div by default');
    assert.deepStrictEqual(jobList.jobsMetadata, fixedData.data.jobs.map((j) => j.job),
        'jost list does not match test data');
  });

  it('refreshes the list of jobs', (done) => {
    fixture.tabs.select(1);
    const jobList = fixture.shadowRoot.querySelector('job-list') as JobList;

    assert.deepStrictEqual(jobList.jobsMetadata, fixedData.data.jobs.map((j) => j.job),
        'jost list does not match test data');

    getJobsStub.returns({ nextPageToken: '', pipelines: [fixedData.data.pipelines[0]] });
    fixture.refreshButton.click();

    getJobsStub.returns({
      jobs: [fixedData.data.jobs.map((j) => j.job)[0]],
      nextPageToken: '',
    });
    fixture.refreshButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert.strictEqual(jobList.jobsMetadata.length, 1);
      assert.deepStrictEqual(jobList.jobsMetadata[0], fixedData.data.jobs[0].job);
      done();
    });
  });

  it('clones the pipeline', (done) => {
    const listener = (e: RouteEvent) => {
      assert.strictEqual(e.detail.path, '/pipelines/new');
      assert.deepStrictEqual(e.detail.data, {
        packageId: testPipeline.package_id,
        parameters: testPipeline.parameters,
      }, 'parameters should be passed when cloning the pipeline');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);
    fixture.cloneButton.click();
  });

  it('deletes selected pipeline, shows success notification', (done) => {
    deletePipelineStub = sinon.stub(Apis, 'deletePipeline');
    deletePipelineStub.returns('ok');

    fixture.deleteButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(deletePipelineStub.calledWith(testPipeline.id));
      assert(notificationStub.calledWith(`Successfully deleted Pipeline: "${testPipeline.name}"`),
          'success notification should be created with pipeline name');
      deletePipelineStub.restore();
      done();
    });
  });

  it('should set status of enable/disable buttons according to pipeline schedule', async () => {
    testPipeline.schedule = '';
    await _resetFixture();
    assert(fixture.enableButton.disabled, 'both enable and disable buttons should be disabled');
    assert(fixture.disableButton.disabled, 'both enable and disable buttons should be disabled');

    testPipeline.schedule = 'some crontab';
    testPipeline.enabled = true;
    await _resetFixture();
    assert(fixture.enableButton.disabled, 'enable button should be disabled');
    assert(!fixture.disableButton.disabled, 'disable button should be enabled');

    testPipeline.enabled = false;
    await _resetFixture();
    assert(!fixture.enableButton.disabled, 'enable button should be enabled');
    assert(fixture.disableButton.disabled, 'disable button should be disabled');
  });

  it('disables selected pipeline, shows success notification', (done) => {
    disablePipelinesStub = sinon.stub(Apis, 'disablePipeline');
    disablePipelinesStub.returns('ok');

    fixture.disableButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(disablePipelinesStub.calledWith(testPipeline.id));
      assert(notificationStub.calledWith('Pipeline disabled'),
          'success notification should be created');
      disablePipelinesStub.restore();
      done();
    });
  });

  it('enables selected pipeline, shows success notification', (done) => {
    enablePipelinesStub = sinon.stub(Apis, 'enablePipeline');
    enablePipelinesStub.returns('ok');

    fixture.enableButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(enablePipelinesStub.calledWith(testPipeline.id));
      assert(notificationStub.calledWith('Pipeline enabled'),
          'success notification should be created');
      enablePipelinesStub.restore();
      done();
    });
  });

  describe('error handling', () => {

    it('shows page load error when failing to get pipeline details', async () => {
      getPipelineStub.throws('cannot get pipeline, bad stuff happened');
      await _resetFixture();
      assert.equal(fixture.pipeline, null, 'should not have loaded a pipeline');
      const errorEl = fixture.$.pageErrorElement as PageError;
      assert.equal(errorEl.error,
          'There was an error while loading details for pipeline ' + testPipeline.id,
          'should show pipeline load error');
      getPipelineStub.restore();
      getPipelineStub = sinon.stub(Apis, 'getPipeline');
      getPipelineStub.returns(testPipeline);
    });

    it('shows error dialog when failing to delete pipeline', (done) => {
      deletePipelineStub = sinon.stub(Apis, 'deletePipeline');
      deletePipelineStub.throws('bad stuff happened while deleting');

      _resetFixture()
        .then(() => {
          fixture.deleteButton.click();

          Polymer.Async.idlePeriod.run(() => {
            assert(dialogStub.calledWith('Failed to delete Pipeline'),
                'error dialog should show with delete failure message');
            deletePipelineStub.restore();
            done();
          });
        });
    });

    it('shows error dialog when failing to disable pipeline', (done) => {
      disablePipelinesStub = sinon.stub(Apis, 'disablePipeline');
      disablePipelinesStub.throws('bad stuff happened while disabling');
      _resetFixture()
        .then(() => {
          fixture.disableButton.click();

          Polymer.Async.idlePeriod.run(() => {
            assert(dialogStub.calledWith(
                'Error disabling pipeline: bad stuff happened while disabling'),
                'error dialog should show with disable failure message');
            disablePipelinesStub.restore();
            done();
          });
        });
    });

    it('shows error dialog when failing to enable pipeline', (done) => {
      enablePipelinesStub = sinon.stub(Apis, 'enablePipeline');
      enablePipelinesStub.throws('bad stuff happened while enabling');
      _resetFixture()
        .then(() => {
          fixture.enableButton.click();

          Polymer.Async.idlePeriod.run(() => {
            assert(dialogStub.calledWith(
                'Error enabling pipeline: bad stuff happened while enabling'),
                'error dialog should show with disable failure message');
            enablePipelinesStub.restore();
            done();
          });
        });
    });
  });

  after(() => {
    document.body.removeChild(fixture);
  });

});

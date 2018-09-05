// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import '../../src/components/job-details/job-details';

import * as sinon from 'sinon';
// @ts-ignore No module declaration at this time.
import * as fixedData from '../../mock-backend/fixed-data';
import * as Apis from '../../src/lib/apis';
import * as Utils from '../../src/lib/utils';

import { assert } from 'chai';
import { apiJob, apiTrigger } from '../../src/api/job';
import { apiListRunsResponse, apiRunDetail } from '../../src/api/run';
import { JobDetails } from '../../src/components/job-details/job-details';
import { PageError } from '../../src/components/page-error/page-error';
import { DialogResult } from '../../src/components/popup-dialog/popup-dialog';
import { RunList } from '../../src/components/run-list/run-list';
import { RouteEvent } from '../../src/model/events';
import { dialogStub, isVisible, notificationStub, resetFixture } from './test-utils';

let fixture: JobDetails;
let getJobStub: sinon.SinonStub;
let listRunsStub: sinon.SinonStub;
let deleteJobStub: sinon.SinonStub;
let enableJobsStub: sinon.SinonStub;
let disableJobsStub: sinon.SinonStub;

async function _resetFixture(): Promise<void> {
  location.hash = '';
  return resetFixture('job-details', undefined, async (f: JobDetails) => {
    fixture = f;
    await f.load('test-job-id');
  });
}

const testCronTrigger: apiTrigger = {
  cron_schedule: {
    cron: '0 * * * ? *',
    end_time: new Date(Math.round(Date.now() / 1000) + 5 * 24 * 60 * 60),
    start_time: new Date(),
  }
};

const testJob: apiJob = {
  created_at: new Date(),
  description: 'test job description',
  enabled: false,
  id: 'test-job-id',
  max_concurrency: '10',
  name: 'test job name',
  parameters: [],
  pipeline_id: '2000',
  status: '',
  trigger: undefined,
  updated_at: new Date(),
};

describe('job-details', () => {

  before(() => {
    getJobStub = sinon.stub(Apis, 'getJob');
    getJobStub.returns(testJob);

    const listRunsResponse: apiListRunsResponse = {
      next_page_token: '',
      runs: fixedData.data.runs.map((r: apiRunDetail) => r.run!),
    };
    listRunsStub = sinon.stub(Apis, 'listRuns');
    listRunsStub.returns(listRunsResponse);
  });

  beforeEach(async () => {
    await _resetFixture();
  });

  it('shows the basic details of the job without schedule', () => {
    const pipelineIdDiv = fixture.shadowRoot!.querySelector('.pipeline-id.value') as HTMLDivElement;
    assert(isVisible(pipelineIdDiv), 'cannot find pipeline id div');
    assert.strictEqual(pipelineIdDiv.innerText, testJob.pipeline_id,
        'displayed pipeline id does not match test data');

    const descriptionDiv =
        fixture.shadowRoot!.querySelector('.description.value') as HTMLDivElement;
    assert(isVisible(descriptionDiv), 'cannot find description div');
    assert.strictEqual(descriptionDiv.innerText, testJob.description,
        'displayed description does not match test data');

    const createdAtDiv = fixture.shadowRoot!.querySelector('.created-at.value') as HTMLDivElement;
    assert(isVisible(createdAtDiv), 'cannot find createdAt div');
    assert.strictEqual(
        createdAtDiv.innerText,
        Utils.formatDateString(testJob.created_at),
        'displayed createdAt does not match test data');

    const scheduleDiv = fixture.shadowRoot!.querySelector('.schedule.value') as HTMLDivElement;
    assert(!isVisible(scheduleDiv), 'should not show schedule div');

    const enabledDiv = fixture.shadowRoot!.querySelector('.enabled.value') as HTMLDivElement;
    assert(!isVisible(enabledDiv), 'should not show enabled div');

    const paramsTable = fixture.shadowRoot!.querySelector('.params-table') as HTMLDivElement;
    assert(!isVisible(paramsTable),
        'should not show params table for test job with no params');
  });

  it('shows schedule and enabled info if a schedule is specified', async () => {
    testJob.trigger = testCronTrigger;
    testJob.enabled = true;
    await _resetFixture();

    const scheduleDiv = fixture.shadowRoot!.querySelector('.schedule.value') as HTMLDivElement;
    assert(isVisible(scheduleDiv), 'should find schedule div');
    assert.strictEqual(scheduleDiv.innerText, testJob.trigger.cron_schedule!.cron,
        'displayed schedule does not match test data');

    const enabledDiv = fixture.shadowRoot!.querySelector('.enabled.value') as HTMLDivElement;
    assert(isVisible(enabledDiv), 'should find enabled div');
    assert.strictEqual(enabledDiv.innerText,
        Utils.enabledDisplayString(testJob.trigger, testJob.enabled),
        'displayed enabled does not match test data');
  });

  it('shows parameters table if there are parameters', async () => {
    testJob.parameters = [
      { name: 'param1', value: 'value1' },
      { name: 'param2', value: 'value2' },
    ];
    await _resetFixture();

    const paramsTable = fixture.shadowRoot!.querySelector('.params-table') as HTMLDivElement;
    assert(isVisible(paramsTable), 'should show params table');
    const paramRows = paramsTable.querySelectorAll('div');
    assert.strictEqual(paramRows.length, 2, 'there should be two rows of parameters');
    paramRows.forEach((row, i) => {
      const key = row.querySelector('.key') as HTMLDivElement;
      const value = row.querySelector('.value') as HTMLDivElement;
      assert.strictEqual(key.innerText, testJob.parameters![i].name);
      assert.strictEqual(value.innerText, testJob.parameters![i].value);
    });
  });

  it('shows the list of runs upon switching to Runs tab', () => {
    const runList = fixture.shadowRoot!.querySelector('run-list') as RunList;
    assert(!isVisible(runList), 'should not show runs div by default');

    fixture.tabs.select(1);
    Polymer.flush();

    assert(isVisible(runList), 'should now show runs div');
    assert.deepStrictEqual(runList.runsMetadata,
        fixedData.data.runs.map((r: apiRunDetail) => r.run),
        'jost list does not match test data');
  });

  it('refreshes the list of runs', (done) => {
    fixture.tabs.select(1);
    const runList = fixture.shadowRoot!.querySelector('run-list') as RunList;

    assert.deepStrictEqual(runList.runsMetadata,
        fixedData.data.runs.map((r: apiRunDetail) => r.run),
        'jost list does not match test data');

    listRunsStub.returns({ nextPageToken: '', jobs: [fixedData.data.jobs[0]] });
    fixture.refreshButton.click();

    listRunsStub.returns({
      nextPageToken: '',
      runs: [fixedData.data.runs.map((r: apiRunDetail) => r.run)[0]],
    });
    fixture.refreshButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert.strictEqual(runList.runsMetadata.length, 1);
      assert.deepStrictEqual(runList.runsMetadata[0], fixedData.data.runs[0].run);
      done();
    });
  });

  it('clones the job', (done) => {
    const listener = (e: Event) => {
      const detail = (e as RouteEvent).detail;
      assert.strictEqual(detail.path, '/jobs/new');
      assert.deepStrictEqual(detail.data, {
        parameters: testJob.parameters,
        pipelineId: testJob.pipeline_id,
      }, 'parameters should be passed when cloning the job');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);
    fixture.cloneButton.click();
  });

  it('deletes selected job, shows success notification', (done) => {
    deleteJobStub = sinon.stub(Apis, 'deleteJob');
    deleteJobStub.returns('ok');

    dialogStub.returns(DialogResult.BUTTON1);
    fixture.deleteButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(dialogStub.calledOnce, 'dialog should be called once for deletion confirmation');
      assert(deleteJobStub.calledWith(testJob.id),
          'delete job should be called with the test job metadata');
      assert(notificationStub.calledWith(`Successfully deleted Job: "${testJob.name}"`),
          'success notification should be created with job name');
      deleteJobStub.restore();
      done();
    });
  });

  it('should set status of enable/disable buttons according to job schedule', async () => {
    testJob.trigger = undefined;
    await _resetFixture();
    assert(fixture.enableButton.disabled, 'both enable and disable buttons should be disabled');
    assert(fixture.disableButton.disabled, 'both enable and disable buttons should be disabled');

    testJob.trigger = testCronTrigger;
    testJob.enabled = true;
    await _resetFixture();
    assert(fixture.enableButton.disabled, 'enable button should be disabled');
    assert(!fixture.disableButton.disabled, 'disable button should be enabled');

    testJob.enabled = false;
    await _resetFixture();
    assert(!fixture.enableButton.disabled, 'enable button should be enabled');
    assert(fixture.disableButton.disabled, 'disable button should be disabled');
  });

  it('disables selected job, shows success notification', (done) => {
    disableJobsStub = sinon.stub(Apis, 'disableJob');
    disableJobsStub.returns('ok');

    fixture.disableButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(disableJobsStub.calledWith(testJob.id));
      assert(notificationStub.calledWith('Job disabled'),
          'success notification should be created');
      disableJobsStub.restore();
      done();
    });
  });

  it('enables selected job, shows success notification', (done) => {
    enableJobsStub = sinon.stub(Apis, 'enableJob');
    enableJobsStub.returns('ok');

    fixture.enableButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(enableJobsStub.calledWith(testJob.id));
      assert(notificationStub.calledWith('Job enabled'),
          'success notification should be created');
      enableJobsStub.restore();
      done();
    });
  });

  describe('error handling', () => {

    it('shows page load error when failing to get job details', async () => {
      getJobStub.throws('cannot get job, bad stuff happened');
      await _resetFixture();
      assert.equal(fixture.job, null, 'should not have loaded a job');
      const errorEl = fixture.$.pageErrorElement as PageError;
      assert.equal(errorEl.error,
          'There was an error while loading details for job ' + testJob.id,
          'should show job load error');
      getJobStub.restore();
      getJobStub = sinon.stub(Apis, 'getJob');
      getJobStub.returns(testJob);
    });

    it('shows error dialog when failing to delete job', (done) => {
      deleteJobStub = sinon.stub(Apis, 'deleteJob');
      deleteJobStub.throws('bad stuff happened while deleting');

      dialogStub.reset();
      dialogStub.returns(DialogResult.DISMISS);
      dialogStub.onFirstCall().returns(DialogResult.BUTTON1);

      _resetFixture()
        .then(() => {
          fixture.deleteButton.click();

          Polymer.Async.idlePeriod.run(() => {
            assert(dialogStub.calledTwice,
                'dialog should be called twice: once to confirm deletion, another for error');
            assert(dialogStub.secondCall.calledWith('Failed to delete Job'),
                'error dialog should show with delete failure message');
            deleteJobStub.restore();
            done();
          });
        });
    });

    it('shows error dialog when failing to disable job', (done) => {
      disableJobsStub = sinon.stub(Apis, 'disableJob');
      disableJobsStub.throws('bad stuff happened while disabling');
      dialogStub.returns(DialogResult.DISMISS);
      _resetFixture()
        .then(() => {
          fixture.disableButton.click();

          Polymer.Async.idlePeriod.run(() => {
            assert(dialogStub.calledWith(
                'Error disabling job: bad stuff happened while disabling'),
                'error dialog should show with disable failure message');
            disableJobsStub.restore();
            done();
          });
        });
    });

    it('shows error dialog when failing to enable job', (done) => {
      enableJobsStub = sinon.stub(Apis, 'enableJob');
      enableJobsStub.throws('bad stuff happened while enabling');
      dialogStub.returns(DialogResult.DISMISS);
      _resetFixture()
        .then(() => {
          fixture.enableButton.click();

          Polymer.Async.idlePeriod.run(() => {
            assert(dialogStub.calledWith(
                'Error enabling job: bad stuff happened while enabling'),
                'error dialog should show with disable failure message');
            enableJobsStub.restore();
            done();
          });
        });
    });
  });

  after(() => {
    getJobStub.restore();
    listRunsStub.restore();
    document.body.removeChild(fixture);
  });

});

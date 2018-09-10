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

import '../../src/components/job-list/job-list';

import * as sinon from 'sinon';
import * as Apis from '../../src/lib/apis';

import { assert } from 'chai';
import { apiListJobsResponse } from '../../src/api/job';
import { JobList } from '../../src/components/job-list/job-list';
import { PageError } from '../../src/components/page-error/page-error';
import { DialogResult } from '../../src/components/popup-dialog/popup-dialog';
import { RouteEvent } from '../../src/model/events';
import { dialogStub, notificationStub, resetFixture, waitUntil } from './test-utils';

import * as fixedData from '../../mock-backend/fixed-data';

let fixture: JobList;
let deleteJobStub: sinon.SinonStub;
let listJobsStub: sinon.SinonStub;
let listRunsStub: sinon.SinonStub;

const allJobsResponse: apiListJobsResponse = {};
allJobsResponse.next_page_token = '';
allJobsResponse.jobs = fixedData.data.jobs;

async function _resetFixture(): Promise<void> {
  return resetFixture('job-list', undefined, (f: JobList) => {
    fixture = f;
    return f.load();
  });
}

describe('job-list', () => {

  before(() => {
    listJobsStub = sinon.stub(Apis, 'listJobs');
    listJobsStub.returns(allJobsResponse);
  });

  beforeEach(async () => {
    location.hash = '';
    await _resetFixture();
    await waitUntil(() => !!fixture.jobsItemList.rows.length, 2000);
  });

  it('shows the list of jobs', () => {
    assert.deepStrictEqual(
        fixture.jobs.map((job) => job.id),
        fixedData.data.jobs.map((job: any) => job.id));
  });

  it('shows last 5 runs status', async () => {
    listRunsStub = sinon.stub(Apis, 'listRuns');
    const successRun = fixedData.data.runs[0];
    successRun.run!.status = 'Succeeded';
    const failureRun = fixedData.data.runs[1];
    failureRun.run!.status = 'Failed';
    listRunsStub.returns({ runs: [] });
    listRunsStub.onFirstCall().returns({ runs: [successRun.run, failureRun.run] });

    await _resetFixture();
    await waitUntil(() => fixture.jobsItemList.rows[0].columns[1] === 'Succeeded:Failed', 5000);
    assert.strictEqual(fixture.jobsItemList.rows[1].columns[1], '');
    listRunsStub.restore();
  });

  it('shows list of all runs', async () => {
    listRunsStub = sinon.stub(Apis, 'listRuns');
    listRunsStub.returns({ runs: fixedData.data.runs.map((r) => r.run!) });
    (fixture.tabs.children[1] as PaperTabElement).click();
    await waitUntil(() =>
        fixture.runsItemList.runsMetadata.length === fixedData.data.runs.length, 5000);

    assert.deepStrictEqual(fixture.runsItemList.runsMetadata[0], fixedData.data.runs[0].run);
    listRunsStub.restore();
  });

  it('displays the job name as a link to the job\'s details page', () => {
    const link = fixture.jobsItemList.getCellElement(1, 1).children[0];
    assert.strictEqual(link.tagName, 'A');
    assert.strictEqual(link.getAttribute('href'), `/jobs/details/${fixture.jobs[0].id}`);
  });

  it('refreshes the list of jobs', (done) => {
    assert.deepStrictEqual(
        fixture.jobs.map((job) => job.id),
        fixedData.data.jobs.map((job: any) => job.id));

    listJobsStub.returns({ nextPageToken: '', jobs: [fixedData.data.jobs[0]] });
    fixture.refreshButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert.strictEqual(fixture.jobs.length, 1);
      assert.deepStrictEqual(fixture.jobs[0], fixedData.data.jobs[0]);
      done();
    });
  });

  it('navigates to job details page on double click', (done) => {
    fixture.jobsItemList._selectItemByDisplayIndex(0);
    const index = fixture.jobsItemList.selectedIndices[0];
    const id = fixedData.data.jobs[index].id;
    const listener = (e: Event) => {
      const detail = (e as RouteEvent).detail;
      assert.strictEqual(detail.path, '/jobs/details/' + id);
      assert.deepStrictEqual(detail.data, undefined,
          'no parameters should be passed when opening the job details');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);

    const firstItem = fixture.jobsItemList.shadowRoot!.querySelector('paper-item');
    firstItem!.dispatchEvent(new MouseEvent('dblclick'));
  });

  it('navigates to new job page with cloned job data', (done) => {
    fixture.jobsItemList._selectItemByRealIndex(0);

    const listener = (e: Event) => {
      const detail = (e as RouteEvent).detail;
      assert.strictEqual(detail.path, '/jobs/new');
      assert.deepStrictEqual(detail.data, {
        parameters: fixedData.data.jobs[0].parameters,
        pipelineId: fixedData.data.jobs[0].pipeline_id,
      }, 'parameters should be passed when cloning the job');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);
    fixture.cloneButton.click();
  });

  it('navigates to new job page when New Job button clicked', (done) => {
    const listener = (e: Event) => {
      const detail = (e as RouteEvent).detail;
      assert.strictEqual(detail.path, '/jobs/new');
      assert.deepStrictEqual(detail.data, undefined,
          'no parameters should be passed when creating a new job');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);
    fixture.newButton.click();
  });

  it('deletes selected job, shows success notification', (done) => {
    deleteJobStub = sinon.stub(Apis, 'deleteJob');
    dialogStub.reset();
    dialogStub.returns(DialogResult.BUTTON1);
    deleteJobStub.returns('ok');

    fixture.jobsItemList._selectItemByRealIndex(0);
    fixture.deleteButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(dialogStub.calledOnce, 'dialog should show only once to confirm deletion');
      assert(deleteJobStub.calledWith(fixedData.data.jobs[0].id),
          'delete job API should be called with the selected pipelin id');
      assert(notificationStub.calledWith('Successfully deleted 1 Jobs!'),
          'success notification should be created');
      deleteJobStub.restore();
      done();
    });
  });

  describe('error handling', () => {
    it('shows the list of jobs', async () => {
      listJobsStub.throws('cannot get list, bad stuff happened');
      await _resetFixture();
      fixture.load();
      assert.equal(fixture.jobs.length, 0, 'should not show any jobs');
      const errorEl = fixture.$.pageErrorElement as PageError;
      assert.equal(errorEl.error, 'There was an error while loading the job list',
          'should show job load error');
      listJobsStub.restore();
    });

    it('shows error dialog when failing to delete selected job', () => {
      deleteJobStub = sinon.stub(Apis, 'deleteJob');
      deleteJobStub.throws('bad stuff happened while deleting');
      listJobsStub = sinon.stub(Apis, 'listJobs');
      listJobsStub.returns(allJobsResponse);
      _resetFixture()
        .then(() => fixture.load())
        .then(() => {
          fixture.jobsItemList._selectItemByRealIndex(0);
          fixture.deleteButton.click();

          Polymer.Async.idlePeriod.run(() => {
            assert(dialogStub.calledWith(
                'Failed to delete 1 Jobs',
                'Deleting Job: "' + fixedData.data.jobs[0].name +
                    '" failed with error: "bad stuff happened while deleting"'
            ), 'error dialog should show with delete failure message');
            deleteJobStub.restore();
            listJobsStub.restore();
          });
        });
    });

  });

  describe('sanitizing HTML', () => {
    before(() => {
      const mockJob = fixedData.data.jobs[0];
      mockJob.name = '<script>alert("surprise!")</script>';
      listJobsStub.returns({ jobs: [mockJob] });
    });

    it('sanitizes user data before inlining as HTML', () => {
      const link = fixture.jobsItemList.getCellElement(1, 1).children[0];
      assert(link.innerHTML.trim(), '&lt;script&gt;alert("surprise!")&lt;/script&gt;');
    });

    after(() => {
      listJobsStub.returns(allJobsResponse);
    });
  });

  after(() => {
    document.body.removeChild(fixture);
  });

});

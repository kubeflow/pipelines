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

import '../../src/components/run-details/run-details';

import * as sinon from 'sinon';
import * as fixedData from '../../mock-backend/fixed-data';
import coinflipRun from '../../mock-backend/mock-coinflip-runtime';
import * as Apis from '../../src/lib/apis';
import * as Utils from '../../src/lib/utils';

import { assert } from 'chai';
import { Job } from '../../src/api/job';
import { Run } from '../../src/api/run';
import { RunDetails } from '../../src/components/run-details/run-details';
import { RuntimeGraph } from '../../src/components/runtime-graph/runtime-graph';
import { RouteEvent } from '../../src/model/events';
import * as testUtils from './test-utils';

let fixture: RunDetails;
let getJobStub: sinon.SinonStub;
let getRunStub: sinon.SinonStub;
let graphRefreshStub: sinon.SinonStub;

const mockRun: Run = {
  run: {
    created_at: 'test-created-at',
    id: 'test-id',
    name: 'test-run',
    namespace: 'test-namespace',
    scheduled_at: 'test-scheduled-at',
    status: 'RUNNING',
  },
  workflow: JSON.stringify(coinflipRun),
};

async function _resetFixture(): Promise<void> {
  return testUtils.resetFixture('run-details', undefined, (f: RunDetails) => {
    fixture = f;
    return f.load('', { runId: 'test-run', jobId: '1000' });
  });
}

const testRun = fixedData.data.runs[0];
const testWorkflow = JSON.parse(testRun.workflow);

const testJob = Job.buildFromObject({
  created_at: new Date().toISOString(),
  description: 'test job description',
  enabled: false,
  id: '1000',
  max_concurrency: 10,
  name: 'test job name',
  parameters: [],
  pipeline_id: 2000,
  status: '',
  trigger: null,
  updated_at: new Date().toISOString(),
});

describe('run-details', () => {

  beforeEach(async () => {
    getRunStub = sinon.stub(Apis, 'getRun');
    getRunStub.returns(testRun);

    graphRefreshStub = sinon.stub(RuntimeGraph.prototype, 'refresh');

    getJobStub = sinon.stub(Apis, 'getJob');
    getJobStub.returns(testJob);
    await _resetFixture();
    fixture.tabs.select(1);
  });

  afterEach(() => {
    getJobStub.restore();
    getRunStub.restore();
    graphRefreshStub.restore();
  });

  it('shows the basic details of the run without schedule', () => {
    assert(!testUtils.isVisible(fixture.runtimeGraph), 'should not show runtime graph');

    const statusDiv = fixture.shadowRoot!.querySelector('.status.value') as HTMLDivElement;
    assert(testUtils.isVisible(statusDiv), 'cannot find status div');
    assert.strictEqual(statusDiv.innerText, testWorkflow.status.phase,
        'displayed status does not match test data');

    const createdAtDivDiv =
        fixture.shadowRoot!.querySelector('.created-at.value') as HTMLDivElement;
    assert(testUtils.isVisible(createdAtDivDiv), 'cannot find createdAt div');
    assert.strictEqual(createdAtDivDiv.innerText,
        Utils.formatDateString(testWorkflow.metadata.creationTimestamp),
        'displayed createdAt does not match test data');

    const startedAtDiv = fixture.shadowRoot!.querySelector('.started-at.value') as HTMLDivElement;
    assert(testUtils.isVisible(startedAtDiv), 'cannot find startedAt div');
    assert.strictEqual(startedAtDiv.innerText,
        Utils.formatDateString(testWorkflow.status.startedAt),
        'displayed startedAt does not match test data');

    const finishedAtDiv = fixture.shadowRoot!.querySelector('.finished-at.value') as HTMLDivElement;
    assert(testUtils.isVisible(finishedAtDiv), 'cannot find finishedAt div');
    assert.strictEqual(finishedAtDiv.innerText,
        Utils.formatDateString(testWorkflow.status.finishedAt),
        'displayed finishedAt does not match test data');

    const durationDiv = fixture.shadowRoot!.querySelector('.duration.value') as HTMLDivElement;
    assert(testUtils.isVisible(durationDiv), 'cannot find duration div');
    assert.strictEqual(durationDiv.innerText,
        Utils.getRunTime(testWorkflow.status.startedAt, testWorkflow.status.finishedAt,
            testWorkflow.status.phase as any),
        'displayed duration does not match test data');
  });

  it('shows parameters table with substituted run parameters if there are any', async () => {
    testJob.parameters = [
      { name: 'param1', value: 'value1' },
      { name: 'param2', value: 'value2[[placeholder]]' },
    ];
    getJobStub.restore();
    getJobStub = sinon.stub(Apis, 'getJob');
    getJobStub.returns(testJob);
    await _resetFixture();
    fixture.tabs.select(1);
    const workflow = JSON.parse(mockRun.workflow) as any;
    workflow.spec.arguments!.parameters = testJob.parameters;
    workflow.spec.arguments!.parameters![1].value = 'value2withplaceholder';
    fixture.workflow = workflow;
    Polymer.flush();

    const paramsTable = fixture.shadowRoot!.querySelector('.params-table') as HTMLDivElement;
    assert(testUtils.isVisible(paramsTable), 'should show params table');
    const paramRows = paramsTable.querySelectorAll('div');
    assert.strictEqual(paramRows.length, 2, 'there should be two rows of parameters');
    paramRows.forEach((row, i) => {
      const key = row.querySelector('.key') as HTMLDivElement;
      const value = row.querySelector('.value') as HTMLDivElement;
      assert.strictEqual(key.innerText, fixture.workflow!.spec.arguments!.parameters![i].name);
      assert.strictEqual(value.innerText, fixture.workflow!.spec.arguments!.parameters![i].value);
    });
  });

  it('clones the run into a new job', (done) => {
    const workflow = JSON.parse(mockRun.workflow) as any;
    const params = [{ name: 'param1', value: 'value2withplaceholder' }];
    workflow.spec.arguments!.parameters = params;
    fixture.workflow = workflow;
    Polymer.flush();

    const listener = (e: Event) => {
      const detail = (e as RouteEvent).detail;
      assert.strictEqual(detail.path, '/jobs/new');
      assert.deepStrictEqual(detail.data, {
        parameters: params,
        pipelineId: testJob.pipeline_id,
      }, 'parameters should be passed when cloning the job');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);
    fixture.cloneButton.click();
  });

  describe('Runtime graph', () => {

    beforeEach(() => {
      fixture.tabs.select(0);
    });

    it('switches to the runtime graph upon switching its page', () => {
      assert(testUtils.isVisible(fixture.runtimeGraph), 'should now show runtime graph');
    });

    it('passes the runtime graph object to the runtime-graph component', async () => {
      await _resetFixture();
      assert.deepStrictEqual(graphRefreshStub.lastCall.args[0].metadata,
          JSON.parse(testRun.workflow).metadata);
    });

  });

  afterEach(() => {
    getRunStub.restore();
    getJobStub.restore();
  });

  after(() => {
    document.body.removeChild(fixture);
  });

});

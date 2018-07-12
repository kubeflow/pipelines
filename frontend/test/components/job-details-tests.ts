import * as sinon from 'sinon';
import * as fixedData from '../../mock-backend/fixed-data';
import * as assert from '../../node_modules/assert/assert';
import * as Apis from '../../src/lib/apis';
import * as Utils from '../../src/lib/utils';

import { Job } from '../../src/api/job';
import { Pipeline } from '../../src/api/pipeline';
import { DataPlot } from '../../src/components/data-plotter/data-plot';
import { JobDetails } from '../../src/components/job-details/job-details';
import { JobGraph } from '../../src/components/job-graph/job-graph';
import { PageError } from '../../src/components/page-error/page-error';
import { NODE_PHASE, NodePhase } from '../../src/model/argo_template';
import { RouteEvent } from '../../src/model/events';
import { OutputMetadata, PlotType } from '../../src/model/output_metadata';
import * as testUtils from './test-utils';

let fixture: JobDetails;
let getPipelineStub: sinon.SinonStub;
let getJobStub: sinon.SinonStub;

const mockTemplate = `
  spec:
    entrypoint: entrypoint-name
    templates:
    - name: entrypoint-name
      steps:
      - - name: step1
          arguments:
            parameters:
            - name: output
              value: output/path1
        - name: step2
          arguments:
            parameters:
            - name: output
              value: output/path2
`;

async function _resetFixture(): Promise<void> {
  return testUtils.resetFixture('job-details', null, (f: JobDetails) => {
    fixture = f;
    return f.load('', { jobId: 'test-job', pipelineId: '1000' });
  });
}

const testJob = fixedData.data.jobs[0];
const testWorkflow = JSON.parse(testJob.workflow);

const testPipeline: Pipeline = {
  created_at: new Date().toISOString(),
  description: 'test pipeline description',
  enabled: false,
  id: '1000',
  max_concurrency: 10,
  name: 'test pipeline name',
  package_id: 2000,
  parameters: [],
  status: '',
  trigger: null,
  updated_at: new Date().toISOString(),
};

describe('job-details', () => {

  beforeEach(async () => {
    getJobStub = sinon.stub(Apis, 'getJob');
    getJobStub.returns(testJob);

    getPipelineStub = sinon.stub(Apis, 'getPipeline');
    getPipelineStub.returns(testPipeline);
    await _resetFixture();
  });

  it('shows the basic details of the job without schedule', () => {
    assert(!testUtils.isVisible(fixture.outputList), 'should not show output list div');
    assert(!testUtils.isVisible(fixture.jobGraph), 'should not show job graph');

    const statusDiv = fixture.shadowRoot.querySelector('.status.value') as HTMLDivElement;
    assert(testUtils.isVisible(statusDiv), 'cannot find status div');
    assert.strictEqual(statusDiv.innerText, testWorkflow.status.phase,
        'displayed status does not match test data');

    const createdAtDivDiv = fixture.shadowRoot.querySelector('.created-at.value') as HTMLDivElement;
    assert(testUtils.isVisible(createdAtDivDiv), 'cannot find createdAt div');
    assert.strictEqual(createdAtDivDiv.innerText,
        Utils.formatDateString(testWorkflow.metadata.creationTimestamp),
        'displayed createdAt does not match test data');

    const startedAtDiv = fixture.shadowRoot.querySelector('.started-at.value') as HTMLDivElement;
    assert(testUtils.isVisible(startedAtDiv), 'cannot find startedAt div');
    assert.strictEqual(startedAtDiv.innerText,
        Utils.formatDateString(testWorkflow.status.startedAt),
        'displayed startedAt does not match test data');

    const finishedAtDiv = fixture.shadowRoot.querySelector('.finished-at.value') as HTMLDivElement;
    assert(testUtils.isVisible(finishedAtDiv), 'cannot find finishedAt div');
    assert.strictEqual(finishedAtDiv.innerText,
        Utils.formatDateString(testWorkflow.status.finishedAt),
        'displayed finishedAt does not match test data');

    const durationDiv = fixture.shadowRoot.querySelector('.duration.value') as HTMLDivElement;
    assert(testUtils.isVisible(durationDiv), 'cannot find duration div');
    assert.strictEqual(durationDiv.innerText,
        Utils.getRunTime(testWorkflow.status.startedAt, testWorkflow.status.finishedAt,
            testWorkflow.status.phase as NodePhase),
        'displayed duration does not match test data');
  });

  it('shows parameters table if there are parameters', async () => {
    testPipeline.parameters = [
      { name: 'param1', value: 'value1' },
      { name: 'param2', value: 'value2' },
    ];
    getPipelineStub.restore();
    getPipelineStub = sinon.stub(Apis, 'getPipeline');
    getPipelineStub.returns(testPipeline);
    await _resetFixture();

    const paramsTable = fixture.shadowRoot.querySelector('.params-table') as HTMLDivElement;
    assert(testUtils.isVisible(paramsTable), 'should show params table');
    const paramRows = paramsTable.querySelectorAll('div');
    assert.strictEqual(paramRows.length, 2, 'there should be two rows of parameters');
    paramRows.forEach((row, i) => {
      const key = row.querySelector('.key') as HTMLDivElement;
      const value = row.querySelector('.value') as HTMLDivElement;
      assert.strictEqual(key.innerText, fixture.pipeline.parameters[i].name);
      assert.strictEqual(value.innerText, fixture.pipeline.parameters[i].value);
    });
  });

  describe('Output list', () => {

    let listFilesStub: sinon.SinonStub;
    let readFileStub: sinon.SinonStub;
    let templatesStub: sinon.SinonStub;

    const metadata1: OutputMetadata = {
      outputs: [{
        source: 'test/confusion/matrix/path',
        type: PlotType.CONFUSION_MATRIX,
      }],
    };

    const metadata2: OutputMetadata = {
      outputs: [{
        source: 'test/roc/curve/path',
        type: PlotType.ROC,
      }],
    };

    before(() => {
      testUtils.stubTag('data-plot', 'div');
    });

    beforeEach(() => {
      testPipeline.parameters = [
        { name: 'param1', value: 'value1' },
        { name: 'param2', value: 'value2' },
        { name: 'output', value: 'gs://test/base/output/path' },
      ];

      listFilesStub = sinon.stub(Apis, 'listFiles');
      readFileStub = sinon.stub(Apis, 'readFile');
      templatesStub = sinon.stub(Apis, 'getPackageTemplate');

      listFilesStub.returns(['gs://test/bucket/path/metadata.json']);
      readFileStub.onFirstCall().returns(JSON.stringify(metadata1));
      readFileStub.onSecondCall().returns(JSON.stringify(metadata2));
      getPipelineStub.returns(testPipeline);
      templatesStub.returns({ template: mockTemplate });

      fixture.tabs.select(1);
    });

    after(() => {
      testUtils.restoreTag('data-plot');
    });

    afterEach(() => {
      listFilesStub.restore();
      readFileStub.restore();
      templatesStub.restore();
    });

    it('switches to the list of outputs upon switching to the outputs tab', async () => {
      assert(testUtils.isVisible(fixture.outputList), 'should now show output list');
    });

    it('shows an error and no outputs if the YAML is bad', async () => {
      templatesStub.returns({ template: `bad yaml` });
      await _resetFixture();
      assert.strictEqual(fixture.outputList.innerText.trim(), '',
          'no outputs should be rendered if YAML is bad');
      assert.deepStrictEqual(fixture.outputPlots, [],
          'should not have any output plots if YAML is bad');
      const errorEl = fixture.$.pageErrorElement as PageError;
      assert.deepStrictEqual(errorEl.error,
          'There was an error while parsing this job\'s YAML template',
          'should show YAML load error');
    });

    it('shows the list of outputs if good YAML is specified', async () => {
      templatesStub.returns({ template: mockTemplate });
      await _resetFixture();
      fixture.tabs.select(1);

      assert.strictEqual(fixture.outputPlots.length, 2);
      const plots: DataPlot[] = fixture.plotContainer.querySelectorAll('div') as any;
      assert.strictEqual(plots.length, 2);
      assert.strictEqual(plots[0].plotMetadata.type, PlotType.CONFUSION_MATRIX);
      assert.strictEqual(plots[1].plotMetadata.type, PlotType.ROC);
    });

  });

  describe('Job graph', () => {

    beforeEach(() => {
      fixture.tabs.select(2);
    });

    it('switches to the job graph upon switching its page', () => {
      assert(testUtils.isVisible(fixture.jobGraph), 'should now show job graph');
    });

    it('passes the job graph object to the job-graph component', async () => {
      const jobGraphRefreshStub = sinon.spy(JobGraph.prototype, 'refresh');
      await _resetFixture();
      assert.deepStrictEqual(jobGraphRefreshStub.lastCall.args[0].metadata,
          JSON.parse(testJob.workflow).metadata);
    });

  });

  afterEach(() => {
    getJobStub.restore();
    getPipelineStub.restore();
  });

  after(() => {
    document.body.removeChild(fixture);
  });

});

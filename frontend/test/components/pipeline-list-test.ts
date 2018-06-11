/// <reference path="../../bower_components/paper-button/paper-button.d.ts" />
/// <reference path="../../bower_components/paper-item/paper-item.d.ts" />
/// <reference path="../../node_modules/@types/mocha/index.d.ts" />

import * as sinon from 'sinon';
import * as assert from '../../node_modules/assert/assert';
import * as Apis from '../../src/lib/apis';
import * as Utils from '../../src/lib/utils';

import { ListPipelinesResponse } from '../../src/api/list_pipelines_response';
import { PageError } from '../../src/components/page-error/page-error';
import { PipelineList } from '../../src/components/pipeline-list/pipeline-list';
import { EventName, RouteEvent } from '../../src/model/events';

import * as fixedData from '../../mock-backend/fixed-data';
import { ItemListElement } from '../../src/components/item-list/item-list';
import { deletePipeline } from '../../src/lib/apis';

const packages = fixedData.namedPackages;

const TEST_TAG = 'pipeline-list';

let fixture: PipelineList;
let deletePipelinesStub: sinon.SinonStub;
let getPipelinesStub: sinon.SinonStub;
let notificationStub: sinon.SinonSpy;
let dialogStub: sinon.SinonSpy;

let pipelinesItemList: ItemListElement;
let cloneButton: PaperButtonElement;
let newPipelineButton: PaperButtonElement;
let deletePipelineButton: PaperButtonElement;

const allPipelinesResponse: ListPipelinesResponse = {
  nextPageToken: '',
  pipelines: fixedData.data.pipelines,
};

// TODO: Refactor the common parts of the resetFixture functions into a util file.
async function resetFixture(): Promise<void> {
  const old = document.querySelector(TEST_TAG);
  if (old) {
    document.body.removeChild(old);
  }
  document.body.appendChild(document.createElement(TEST_TAG));
  fixture = document.querySelector(TEST_TAG) as PipelineList;
  await fixture.load('');

  Polymer.flush();

  pipelinesItemList = fixture.$.pipelinesItemList as ItemListElement;
  cloneButton = fixture.$.cloneBtn as PaperButtonElement;
  newPipelineButton = fixture.$.newBtn as PaperButtonElement;
  deletePipelineButton = fixture.$.deleteBtn as PaperButtonElement;
}

describe('pipeline-list', () => {

  before(() => {
    getPipelinesStub = sinon.stub(Apis, 'getPipelines');
    getPipelinesStub.returns(allPipelinesResponse);

    notificationStub = sinon.stub(Utils, 'showNotification');
    dialogStub = sinon.stub(Utils, 'showDialog');
  });

  beforeEach(async () => {
    await resetFixture();
  });

  it('shows the list of pipelines', () => {
    assert.deepStrictEqual(
        fixture.pipelines.map((pipeline) => pipeline.id),
        fixedData.data.pipelines.map((pipeline) => pipeline.id));
  });

  it('navigates to pipeline details page on double click', (done) => {
    pipelinesItemList._selectItemByDisplayIndex(0);
    const index = pipelinesItemList.selectedIndices[0];
    const id = fixedData.data.pipelines[index].id;
    const listener = (e: RouteEvent) => {
      assert.strictEqual(e.detail.path, '/pipelines/details/' + id);
      assert.deepStrictEqual(e.detail.data, undefined,
          'no parameters should be passed when opening the pipeline details');
      document.removeEventListener(EventName.ROUTE, listener);
      done();
    };
    document.addEventListener(EventName.ROUTE, listener);

    const firstItem = pipelinesItemList.shadowRoot.querySelector('paper-item') as PaperItemElement;
    firstItem.dispatchEvent(new MouseEvent('dblclick'));
  });

  it('navigates to new pipeline page with cloned pipeline data', (done) => {
    pipelinesItemList._selectItemByRealIndex(0);

    const listener = (e: RouteEvent) => {
      assert.strictEqual(e.detail.path, '/pipelines/new');
      assert.deepStrictEqual(e.detail.data, {
        packageId: fixedData.data.pipelines[0].package_id,
        parameters: fixedData.data.pipelines[0].parameters,
      }, 'parameters should be passed when cloning the pipeline');
      document.removeEventListener(EventName.ROUTE, listener);
      done();
    };
    document.addEventListener(EventName.ROUTE, listener);
    cloneButton.click();
  });

  it('navigates to new pipeline page when New Pipeline button clicked', (done) => {
    const listener = (e: RouteEvent) => {
      assert.strictEqual(e.detail.path, '/pipelines/new');
      assert.deepStrictEqual(e.detail.data, undefined,
          'no parameters should be passed when creating a new pipeline');
      document.removeEventListener(EventName.ROUTE, listener);
      done();
    };
    document.addEventListener(EventName.ROUTE, listener);
    newPipelineButton.click();
  });

  it('deletes selected pipeline, shows success notification', (done) => {
    deletePipelinesStub = sinon.stub(Apis, 'deletePipeline');
    deletePipelinesStub.returns('ok');

    pipelinesItemList._selectItemByRealIndex(0);
    deletePipelineButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(deletePipelinesStub.calledWith(fixedData.data.pipelines[0].id),
          'delete pipeline API should be called with the selected pipelin id');
      assert(notificationStub.calledWith('Successfully deleted 1 Pipelines!'),
          'success notification should be created');
      deletePipelinesStub.restore();
      done();
    });
  });

  describe('error handling', () => {
    it('shows the list of pipelines', async () => {
      getPipelinesStub.throws('cannot get list, bad stuff happened');
      await resetFixture();
      assert.equal(fixture.pipelines.length, 0, 'should not show any pipelines');
      const errorEl = fixture.$.pageErrorElement as PageError;
      assert.equal(errorEl.error, 'There was an error while loading the pipeline list.',
          'should show pipeline load error');
      getPipelinesStub.restore();
    });

    it('shows error dialog when failing to delete selected pipeline', () => {
      deletePipelinesStub = sinon.stub(Apis, 'deletePipeline');
      deletePipelinesStub.throws('bad stuff happened while deleting');
      getPipelinesStub = sinon.stub(Apis, 'getPipelines');
      getPipelinesStub.returns(allPipelinesResponse);
      resetFixture()
        .then(() => {
          pipelinesItemList._selectItemByRealIndex(0);
          deletePipelineButton.click();

          Polymer.Async.idlePeriod.run(() => {
            assert(dialogStub.calledWith(
                'Failed to delete 1 Pipelines',
                'Deleting Pipeline: "' + fixedData.data.pipelines[0].name +
                    '" failed with error: "bad stuff happened while deleting"'
            ), 'error dialog show should with delete failure message');
            deletePipelinesStub.restore();
            getPipelinesStub.restore();
          });
        });
    });

  });

  after(() => {
    document.body.removeChild(fixture);
  });

});

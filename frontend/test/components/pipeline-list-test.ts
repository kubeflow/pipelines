import * as sinon from 'sinon';
import * as assert from '../../node_modules/assert/assert';
import * as Apis from '../../src/lib/apis';

import { ListPipelinesResponse } from '../../src/api/list_pipelines_response';
import { Pipeline } from '../../src/api/pipeline';
import { PageError } from '../../src/components/page-error/page-error';
import { PipelineList } from '../../src/components/pipeline-list/pipeline-list';
import { RouteEvent } from '../../src/model/events';
import { dialogStub, notificationStub, resetFixture } from './test-utils';

import * as fixedData from '../../mock-backend/fixed-data';
import { ItemListElement } from '../../src/components/item-list/item-list';
import { DialogResult } from '../../src/components/popup-dialog/popup-dialog';

const packages = fixedData.namedPackages;

let fixture: PipelineList;
let deletePipelinesStub: sinon.SinonStub;
let getPipelinesStub: sinon.SinonStub;

const allPipelinesResponse = new ListPipelinesResponse();
allPipelinesResponse.next_page_token = '';
allPipelinesResponse.pipelines =
    fixedData.data.pipelines.map((p: any) => Pipeline.buildFromObject(p));

async function _resetFixture(): Promise<void> {
  return resetFixture('pipeline-list', null, (f: PipelineList) => {
    fixture = f;
    return f.load('');
  });
}

describe('pipeline-list', () => {

  before(() => {
    getPipelinesStub = sinon.stub(Apis, 'getPipelines');
    getPipelinesStub.returns(allPipelinesResponse);
  });

  beforeEach(async () => {
    await _resetFixture();
  });

  it('shows the list of pipelines', () => {
    assert.deepStrictEqual(
        fixture.pipelines.map((pipeline) => pipeline.id),
        fixedData.data.pipelines.map((pipeline) => pipeline.id));
  });

  it('refreshes the list of pipeline', (done) => {
    assert.deepStrictEqual(
        fixture.pipelines.map((pipeline) => pipeline.id),
        fixedData.data.pipelines.map((pipeline) => pipeline.id));

    getPipelinesStub.returns({ nextPageToken: '', pipelines: [fixedData.data.pipelines[0]] });
    fixture.refreshButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert.strictEqual(fixture.pipelines.length, 1);
      assert.deepStrictEqual(fixture.pipelines[0], fixedData.data.pipelines[0]);
      done();
    });
  });

  it('navigates to pipeline details page on double click', (done) => {
    fixture.itemList._selectItemByDisplayIndex(0);
    const index = fixture.itemList.selectedIndices[0];
    const id = fixedData.data.pipelines[index].id;
    const listener = (e: RouteEvent) => {
      assert.strictEqual(e.detail.path, '/pipelines/details/' + id);
      assert.deepStrictEqual(e.detail.data, undefined,
          'no parameters should be passed when opening the pipeline details');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);

    const firstItem = fixture.itemList.shadowRoot.querySelector('paper-item') as PaperItemElement;
    firstItem.dispatchEvent(new MouseEvent('dblclick'));
  });

  it('navigates to new pipeline page with cloned pipeline data', (done) => {
    fixture.itemList._selectItemByRealIndex(0);

    const listener = (e: RouteEvent) => {
      assert.strictEqual(e.detail.path, '/pipelines/new');
      assert.deepStrictEqual(e.detail.data, {
        packageId: fixedData.data.pipelines[0].package_id,
        parameters: fixedData.data.pipelines[0].parameters,
      }, 'parameters should be passed when cloning the pipeline');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);
    fixture.cloneButton.click();
  });

  it('navigates to new pipeline page when New Pipeline button clicked', (done) => {
    const listener = (e: RouteEvent) => {
      assert.strictEqual(e.detail.path, '/pipelines/new');
      assert.deepStrictEqual(e.detail.data, undefined,
          'no parameters should be passed when creating a new pipeline');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);
    fixture.newButton.click();
  });

  it('deletes selected pipeline, shows success notification', (done) => {
    deletePipelinesStub = sinon.stub(Apis, 'deletePipeline');
    dialogStub.reset();
    dialogStub.returns(DialogResult.BUTTON1);
    deletePipelinesStub.returns('ok');

    fixture.itemList._selectItemByRealIndex(0);
    fixture.deleteButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(dialogStub.calledOnce, 'dialog should show only once to confirm deletion');
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
      await _resetFixture();
      assert.equal(fixture.pipelines.length, 0, 'should not show any pipelines');
      const errorEl = fixture.$.pageErrorElement as PageError;
      assert.equal(errorEl.error, 'There was an error while loading the pipeline list',
          'should show pipeline load error');
      getPipelinesStub.restore();
    });

    it('shows error dialog when failing to delete selected pipeline', () => {
      deletePipelinesStub = sinon.stub(Apis, 'deletePipeline');
      deletePipelinesStub.throws('bad stuff happened while deleting');
      getPipelinesStub = sinon.stub(Apis, 'getPipelines');
      getPipelinesStub.returns(allPipelinesResponse);
      _resetFixture()
        .then(() => {
          fixture.itemList._selectItemByRealIndex(0);
          fixture.deleteButton.click();

          Polymer.Async.idlePeriod.run(() => {
            assert(dialogStub.calledWith(
                'Failed to delete 1 Pipelines',
                'Deleting Pipeline: "' + fixedData.data.pipelines[0].name +
                    '" failed with error: "bad stuff happened while deleting"'
            ), 'error dialog should show with delete failure message');
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

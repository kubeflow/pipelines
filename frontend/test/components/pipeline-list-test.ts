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

import '../../src/components/pipeline-list/pipeline-list';

import * as sinon from 'sinon';
import * as Apis from '../../src/lib/apis';

import { assert } from 'chai';
import { apiListPipelinesResponse, apiPipeline } from '../../src/api/pipeline';
import { PageError } from '../../src/components/page-error/page-error';
import { PipelineList } from '../../src/components/pipeline-list/pipeline-list';
import {
  PipelineUploadDialogResult
} from '../../src/components/pipeline-upload-dialog/pipeline-upload-dialog';
import { DialogResult } from '../../src/components/popup-dialog/popup-dialog';
import { RouteEvent } from '../../src/model/events';
import {
  dialogStub, notificationStub, resetFixture, showPipelineUploadDialog, waitUntil
} from './test-utils';

import * as fixedData from '../../mock-backend/fixed-data';

let fixture: PipelineList;
let deletePipelineStub: sinon.SinonStub;
let listPipelinesStub: sinon.SinonStub;

const allPipelinesResponse: apiListPipelinesResponse = {};
allPipelinesResponse.next_page_token = '';
allPipelinesResponse.pipelines = fixedData.data.pipelines;

const testPipeline: apiPipeline = {
  created_at: new Date(),
  description: 'a dummy pipeline to be used in tests',
  id: 'test-pipeline-id',
  name: 'test-pipeline',
  parameters: [],
};

async function _resetFixture(): Promise<void> {
  return resetFixture('pipeline-list', undefined, (f: PipelineList) => {
    fixture = f;
    return f.load();
  });
}

describe('pipeline-list', () => {

  before(() => {
    listPipelinesStub = sinon.stub(Apis, 'listPipelines');
    listPipelinesStub.returns(allPipelinesResponse);
  });

  beforeEach(async () => {
    location.hash = '';
    await _resetFixture();
    await waitUntil(() => !!fixture.itemList.rows.length, 2000);
  });

  it('shows the list of pipelines', () => {
    assert.deepStrictEqual(
        fixture.pipelines.map((pipeline) => pipeline.id),
        fixedData.data.pipelines.map((pipeline: any) => pipeline.id));
  });

  it('displays the pipeline name as a link to the pipeline\'s details page', () => {
    const link = fixture.itemList.getCellElement(1, 1).children[0];
    assert.strictEqual(link.tagName, 'A');
    assert.strictEqual(link.getAttribute('href'), `/pipelines/details/${fixture.pipelines[0].id}`);
  });

  it('refreshes the list of pipelines', (done) => {
    assert.deepStrictEqual(
        fixture.pipelines.map((pipeline) => pipeline.id),
        fixedData.data.pipelines.map((pipeline: any) => pipeline.id));

    listPipelinesStub.returns({ nextPageToken: '', pipelines: [fixedData.data.pipelines[0]] });
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
    const listener = (e: Event) => {
      const detail = (e as RouteEvent).detail;
      assert.strictEqual(detail.path, '/pipelines/details/' + id);
      assert.deepStrictEqual(detail.data, undefined,
          'no parameters should be passed when opening the pipeline details');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);

    const firstItem = fixture.itemList.shadowRoot!.querySelector('paper-item');
    firstItem!.dispatchEvent(new MouseEvent('dblclick'));
  });

  it('navigates to new job page with selected pipeline', (done) => {
    fixture.itemList._selectItemByRealIndex(0);

    const listener = (e: Event) => {
      const detail = (e as RouteEvent).detail;
      assert.strictEqual(detail.path, '/jobs/new');
      assert.deepStrictEqual(detail.data, {
        pipelineId: fixedData.data.pipelines[0].id,
      }, 'ID should be passed when creating a new Job');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);
    fixture.createJobButton.click();
  });

  it('uploads a new pipeline, shows success notification', (done) => {
    showPipelineUploadDialog.reset();
    // TODO: add component tests for the upload dialog.
    showPipelineUploadDialog.returns(
        new Promise((resolve) => resolve({
          buttonPressed: DialogResult.BUTTON1,
          pipeline: testPipeline,
        } as PipelineUploadDialogResult)));

    fixture.uploadButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(showPipelineUploadDialog.calledOnce, 'dialog should show only once');
      assert(notificationStub.calledWith(`Successfully uploaded pipeline: ${testPipeline.name}`),
          'success notification should be created');
      done();
    });
  });

  it('deletes selected pipeline, shows success notification', (done) => {
    deletePipelineStub = sinon.stub(Apis, 'deletePipeline');
    dialogStub.reset();
    dialogStub.returns(DialogResult.BUTTON1);
    deletePipelineStub.returns('ok');

    fixture.itemList._selectItemByRealIndex(0);
    fixture.deleteButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(dialogStub.calledOnce, 'dialog should show only once to confirm deletion');
      assert(deletePipelineStub.calledWith(fixedData.data.pipelines[0].id),
          'delete pipeline API should be called with the selected pipeline id');
      assert(notificationStub.calledWith('Successfully deleted 1 Pipelines!'),
          'success notification should be created');
      deletePipelineStub.restore();
      done();
    });
  });

  describe('error handling', () => {
    it('shows the list of pipelines', async () => {
      listPipelinesStub.throws(new Error('cannot get list, bad stuff happened'));
      await _resetFixture();
      fixture.load();
      assert.equal(fixture.pipelines.length, 0, 'should not show any pipelines');
      const errorEl = fixture.$.pageErrorElement as PageError;
      assert.equal(errorEl.error, 'There was an error while loading the pipeline list',
          'should show pipeline load error');
      assert.equal(errorEl.details, 'cannot get list, bad stuff happened');
      listPipelinesStub.restore();
    });

    it('shows error dialog when failing to delete selected pipeline', (done) => {
      deletePipelineStub = sinon.stub(Apis, 'deletePipeline');
      deletePipelineStub.throws(new Error('bad stuff happened while deleting'));
      listPipelinesStub = sinon.stub(Apis, 'listPipelines');
      listPipelinesStub.returns(allPipelinesResponse);
      _resetFixture()
        .then(() => fixture.load())
        .then(() => {
          fixture.itemList._selectItemByRealIndex(0);
          fixture.deleteButton.click();

          Polymer.Async.idlePeriod.run(() => {
            assert(dialogStub.calledWith(
                'Failed to delete 1 Pipelines',
                'Deleting Pipeline: "' + fixedData.data.pipelines[0].name +
                    '" failed with error: "Error: bad stuff happened while deleting"'
            ), 'error dialog should show with delete failure message');
            deletePipelineStub.restore();
            listPipelinesStub.restore();
            done();
          });
        });
    });

  });

  describe('sanitizing HTML', () => {
    before(() => {
      const mockPipeline = fixedData.data.pipelines[0];
      mockPipeline.name = '<script>alert("surprise!")</script>';
      listPipelinesStub.returns({ pipelines: [mockPipeline] });
    });

    it('sanitizes user data before inlining as HTML', () => {
      const link = fixture.itemList.getCellElement(1, 1).children[0];
      assert(link.innerHTML.trim(), '&lt;script&gt;alert("surprise!")&lt;/script&gt;');
    });

    after(() => {
      listPipelinesStub.returns(allPipelinesResponse);
    });
  });

  after(() => {
    deletePipelineStub.restore();
    listPipelinesStub.restore();
    dialogStub.reset();
    document.body.removeChild(fixture);
  });

});

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

import '../../src/components/pipeline-details/pipeline-details';

import * as sinon from 'sinon';
import * as Apis from '../../src/lib/apis';

import { assert } from 'chai';
import { apiGetTemplateResponse, apiPipeline } from '../../src/api/pipeline';
import { PageError } from '../../src/components/page-error/page-error';
import { PipelineDetails } from '../../src/components/pipeline-details/pipeline-details';
import { DialogResult } from '../../src/components/popup-dialog/popup-dialog';
import { RouteEvent } from '../../src/model/events';
import { dialogStub, notificationStub, resetFixture } from './test-utils';

let fixture: PipelineDetails;
let getPipelineStub: sinon.SinonStub;
let getPipelineTemplateStub: sinon.SinonStub;
let deletePipelineStub: sinon.SinonStub;

async function _resetFixture(): Promise<void> {
  location.hash = '';
  return resetFixture('pipeline-details', undefined, async (f: PipelineDetails) => {
    fixture = f;
    await f.load('test-pipeline-id');
  });
}

const testPipeline: apiPipeline = {
  created_at: new Date(),
  description: 'test pipeline description',
  id: 'test-pipeline-id',
  name: 'test pipeline name',
  parameters: [],
};

const testPipelineTemplate: apiGetTemplateResponse = {
  template:
      'apiVersion: argoproj.io/v1alpha1\
      kind: Workflow\
      metadata:\
        generateName: hello-world-\
      spec:\
        entrypoint: whalesay\
        templates:\
        - name: whalesay\
          container:\
            image: docker/whalesay:latest\
            command: [cowsay]\
            args: ["hello world"]'
};

describe('pipeline-details', () => {

  before(() => {
    getPipelineStub = sinon.stub(Apis, 'getPipeline');
    getPipelineStub.returns(testPipeline);

    getPipelineTemplateStub = sinon.stub(Apis, 'getPipelineTemplate');
    getPipelineTemplateStub.returns(testPipelineTemplate);
  });

  beforeEach(async () => {
    await _resetFixture();
  });

  it('creates a job from the pipeline', (done) => {
    const listener = (e: Event) => {
      const detail = (e as RouteEvent).detail;
      assert.strictEqual(detail.path, '/jobs/new');
      assert.deepStrictEqual(
          detail.data,
          { pipelineId: testPipeline.id },
          'Pipeline ID should be passed creating a new job');
      document.removeEventListener(RouteEvent.name, listener);
      done();
    };
    document.addEventListener(RouteEvent.name, listener);
    fixture.createJobButton.click();
  });

  it('deletes selected pipeline, shows success notification', (done) => {
    deletePipelineStub = sinon.stub(Apis, 'deletePipeline');
    deletePipelineStub.returns('ok');

    dialogStub.reset();
    dialogStub.returns(DialogResult.BUTTON1);
    fixture.deleteButton.click();

    Polymer.Async.idlePeriod.run(() => {
      assert(dialogStub.calledOnce, 'dialog should be called once for deletion confirmation');
      assert(deletePipelineStub.calledWith(testPipeline.id),
          'delete pipeline should be called with the test pipeline id');
      assert(
          notificationStub.calledWith(`Successfully deleted Pipeline: "${testPipeline.name}"`),
          'success notification should be created with pipeline name');
      deletePipelineStub.restore();
      done();
    });
  });

  describe('error handling', () => {

    it('shows page load error when failing to get pipeline details', async () => {
      getPipelineStub.throws(new Error('cannot get pipeline, bad stuff happened'));
      await _resetFixture();
      assert.equal(fixture.pipeline, null, 'should not have loaded a pipeline');
      const errorEl = fixture.$.pageErrorElement as PageError;
      assert.equal(errorEl.error,
          'There was an error while loading details for Pipeline ' + testPipeline.id,
          'should show pipeline load error');
      assert.equal(errorEl.details, 'cannot get pipeline, bad stuff happened');
      getPipelineStub.restore();
      getPipelineStub = sinon.stub(Apis, 'getPipeline');
      getPipelineStub.returns(testPipeline);
    });

    it('shows page load error when failing to get the pipeline\'s template', async () => {
      getPipelineTemplateStub.throws(new Error('cannot get pipeline template, bad stuff happened'));
      await _resetFixture();
      assert.equal(fixture.pipelineTemplate, null, 'should not have loaded a pipeline template');
      const errorEl = fixture.$.pageErrorElement as PageError;
      assert.equal(errorEl.error,
          'There was an error while loading details for Pipeline ' + testPipeline.id,
          'should show pipeline load error');
      assert.equal(errorEl.details, 'cannot get pipeline template, bad stuff happened');
      getPipelineTemplateStub.restore();
      getPipelineTemplateStub = sinon.stub(Apis, 'getPipelineTemplate');
      getPipelineTemplateStub.returns(testPipelineTemplate);
    });

    it('shows error dialog when failing to delete pipeline', (done) => {
      deletePipelineStub = sinon.stub(Apis, 'deletePipeline');
      deletePipelineStub.throws(new Error('bad stuff happened while deleting'));

      dialogStub.reset();
      dialogStub.returns(DialogResult.DISMISS);
      dialogStub.onFirstCall().returns(DialogResult.BUTTON1);

      _resetFixture()
        .then(() => {
          fixture.deleteButton.click();

          Polymer.Async.idlePeriod.run(() => {
            assert(dialogStub.calledTwice,
                'dialog should be called twice: once to confirm deletion, another for error');
            assert(dialogStub.secondCall.calledWith('Failed to delete Pipeline'),
                'error dialog should show with delete failure message');
            deletePipelineStub.restore();
            done();
          });
        });
    });
  });

  after(() => {
    getPipelineStub.restore();
    getPipelineTemplateStub.restore();
    deletePipelineStub.restore();
    dialogStub.reset();
    document.body.removeChild(fixture);
  });

});

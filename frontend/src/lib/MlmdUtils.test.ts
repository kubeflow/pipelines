/**
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {
  Api,
  Context,
  GetContextByTypeAndNameRequest,
  GetContextByTypeAndNameResponse,
} from '@kubeflow/frontend';
import { expectWarnings, testBestPractices } from 'src/TestUtils';
import { Workflow, WorkflowSpec, WorkflowStatus } from 'third_party/argo-ui/argo_template';
import { getRunContext } from './MlmdUtils';

testBestPractices();

const WORKFLOW_NAME = 'run-st448';
const WORKFLOW_EMPTY: Workflow = {
  metadata: {
    name: WORKFLOW_NAME,
  },
  // there are many unrelated fields here, omit them
  spec: {} as WorkflowSpec,
  status: {} as WorkflowStatus,
};

const V2_CONTEXT = new Context();
V2_CONTEXT.setName(WORKFLOW_NAME);
V2_CONTEXT.setType('kfp.PipelineRun');

const TFX_CONTEXT = new Context();
TFX_CONTEXT.setName('run.run-st448');
TFX_CONTEXT.setType('run');

const V1_CONTEXT = new Context();
V1_CONTEXT.setName(WORKFLOW_NAME);
V1_CONTEXT.setType('KfpRun');

describe('MlmdUtils', () => {
  describe('getRunContext', () => {
    it('gets KFP v2 context', async () => {
      mockGetContextByTypeAndName([V2_CONTEXT]);
      const context = await getRunContext({
        ...WORKFLOW_EMPTY,
        metadata: {
          ...WORKFLOW_EMPTY.metadata,
          annotations: { 'pipelines.kubeflow.org/v2_pipeline': 'true' },
        },
      });
      expect(context).toEqual(V2_CONTEXT);
    });

    it('gets TFX context', async () => {
      mockGetContextByTypeAndName([TFX_CONTEXT, V1_CONTEXT]);
      const context = await getRunContext(WORKFLOW_EMPTY);
      expect(context).toEqual(TFX_CONTEXT);
    });

    it('gets KFP v1 context', async () => {
      const verify = expectWarnings();
      mockGetContextByTypeAndName([V1_CONTEXT]);
      const context = await getRunContext(WORKFLOW_EMPTY);
      expect(context).toEqual(V1_CONTEXT);
      verify();
    });

    it('throws error when not found', async () => {
      const verify = expectWarnings();
      mockGetContextByTypeAndName([]);
      await expect(getRunContext(WORKFLOW_EMPTY)).rejects.toThrow();
      verify();
    });
  });
});

function mockGetContextByTypeAndName(contexts: Context[]) {
  const getContextByTypeAndNameSpy = jest.spyOn(
    Api.getInstance().metadataStoreService,
    'getContextByTypeAndName',
  );
  getContextByTypeAndNameSpy.mockImplementation((req: GetContextByTypeAndNameRequest) => {
    const response = new GetContextByTypeAndNameResponse();
    const found = contexts.find(
      context =>
        context.getType() === req.getTypeName() && context.getName() === req.getContextName(),
    );
    response.setContext(found);
    return response;
  });
}

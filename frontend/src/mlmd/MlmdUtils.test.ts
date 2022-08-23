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

import { Api } from 'src/mlmd/library';
import {
  EXECUTION_KEY_CACHED_EXECUTION_ID,
  filterLinkedArtifactsByType,
  getArtifactName,
  getArtifactTypeName,
  getArtifactNameFromEvent,
  getContextByExecution,
  getRunContext,
  getArtifactsFromContext,
  getEventsByExecutions,
} from 'src/mlmd/MlmdUtils';
import { expectWarnings, testBestPractices } from 'src/TestUtils';
import {
  Artifact,
  ArtifactType,
  Context,
  ContextType,
  Event,
  Execution,
  GetContextByTypeAndNameRequest,
  GetContextByTypeAndNameResponse,
} from 'src/third_party/mlmd';
import {
  GetArtifactsByContextRequest,
  GetArtifactsByContextResponse,
  GetContextsByExecutionRequest,
  GetContextsByExecutionResponse,
  GetContextTypeRequest,
  GetContextTypeResponse,
  GetEventsByExecutionIDsRequest,
  GetEventsByExecutionIDsResponse,
} from 'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_service_pb';
import { Workflow, WorkflowSpec, WorkflowStatus } from 'third_party/argo-ui/argo_template';

testBestPractices();

const WORKFLOW_NAME = 'run-st448';
const RUN_ID = 'abcdefghijk';
const WORKFLOW_EMPTY: Workflow = {
  metadata: {
    name: WORKFLOW_NAME,
  },
  // there are many unrelated fields here, omit them
  spec: {} as WorkflowSpec,
  status: {} as WorkflowStatus,
};

const V2_CONTEXT = new Context();
V2_CONTEXT.setName(RUN_ID);
V2_CONTEXT.setType('system.PipelineRun');

const TFX_CONTEXT = new Context();
TFX_CONTEXT.setName('run-st448');
TFX_CONTEXT.setType('pipeline_run');

const V1_CONTEXT = new Context();
V1_CONTEXT.setName(WORKFLOW_NAME);
V1_CONTEXT.setType('KfpRun');

describe('MlmdUtils', () => {
  describe('getContextByExecution', () => {
    it('Found matching context', async () => {
      const context1 = new Context().setId(1).setTypeId(10);
      const context2 = new Context().setId(2).setTypeId(20);
      const contextType = new ContextType().setName('type').setId(20);
      mockGetContextType(contextType);
      mockGetContextsByExecution([context1, context2]);

      const execution = new Execution().setId(3);
      const context = await getContextByExecution(execution, 'type');
      expect(context).toEqual(context2);
    });

    it('No matching context', async () => {
      const context1 = new Context().setId(1).setTypeId(10);
      const context2 = new Context().setId(2).setTypeId(20);
      const contextType = new ContextType().setName('type').setId(30);
      mockGetContextType(contextType);
      mockGetContextsByExecution([context1, context2]);

      const execution = new Execution().setId(3);
      const context = await getContextByExecution(execution, 'type');

      expect(context).toBeUndefined();
    });

    it('More than 1 matching context', async () => {
      const context1 = new Context().setId(1).setTypeId(20);
      const context2 = new Context().setId(2).setTypeId(20);
      const contextType = new ContextType().setName('type').setId(20);
      mockGetContextType(contextType);
      mockGetContextsByExecution([context1, context2]);

      const execution = new Execution().setId(3);
      const context = await getContextByExecution(execution, 'type');

      // Although more than 1 matching context, this case is unexpected and result is omitted.
      expect(context).toBeUndefined();
    });
  });

  describe('getRunContext', () => {
    it('gets KFP v2 context', async () => {
      mockGetContextByTypeAndName([V2_CONTEXT]);
      const context = await getRunContext(
        {
          ...WORKFLOW_EMPTY,
          metadata: {
            ...WORKFLOW_EMPTY.metadata,
            annotations: { 'pipelines.kubeflow.org/v2_pipeline': 'true' },
          },
        },
        RUN_ID,
      );
      expect(context).toEqual(V2_CONTEXT);
    });

    it('gets TFX context', async () => {
      mockGetContextByTypeAndName([TFX_CONTEXT, V1_CONTEXT]);
      const context = await getRunContext(WORKFLOW_EMPTY, RUN_ID);
      expect(context).toEqual(TFX_CONTEXT);
    });

    it('gets KFP v1 context', async () => {
      const verify = expectWarnings();
      mockGetContextByTypeAndName([V1_CONTEXT]);
      const context = await getRunContext(WORKFLOW_EMPTY, RUN_ID);
      expect(context).toEqual(V1_CONTEXT);
      verify();
    });

    it('throws error when not found', async () => {
      const verify = expectWarnings();
      mockGetContextByTypeAndName([]);
      await expect(getRunContext(WORKFLOW_EMPTY, RUN_ID)).rejects.toThrow();
      verify();
    });
  });

  describe('getArtifactName', () => {
    it('get the first key of steps list', () => {
      const path = new Event.Path();
      path.getStepsList().push(new Event.Path.Step().setKey('key1'));
      path.getStepsList().push(new Event.Path.Step().setKey('key2'));
      const event = new Event();
      event.setPath(path);
      const linkedArtifact = { event: event, artifact: new Artifact() };
      expect(getArtifactName(linkedArtifact)).toEqual('key1');
    });
  });

  describe('getArtifactTypeName', () => {
    it('return artifactTypeName[]', () => {
      const artifactTypes = [
        new ArtifactType().setId(1).setName('metrics'),
        new ArtifactType().setId(2).setName('markdown'),
      ];
      const artifact1 = new Artifact();
      artifact1.setTypeId(1);
      const artifact2 = new Artifact();
      artifact2.setTypeId(2);
      const linkedArtifacts = [
        { event: new Event(), artifact: artifact1 },
        { event: new Event(), artifact: artifact2 },
      ];
      expect(getArtifactTypeName(artifactTypes, linkedArtifacts)).toEqual(['metrics', 'markdown']);
    });
  });

  describe('getArtifactNameFromEvent', () => {
    it('get the first key of steps list', () => {
      const path = new Event.Path();
      path.getStepsList().push(new Event.Path.Step().setKey('key1'));
      path.getStepsList().push(new Event.Path.Step().setKey('key2'));
      const event = new Event();
      event.setPath(path);
      expect(getArtifactNameFromEvent(event)).toEqual('key1');
    });
  });

  describe('getActifactsFromContext', () => {
    it('returns list of artifacts', async () => {
      const context = new Context();
      context.setId(2);
      const artifacts = [new Artifact().setId(10), new Artifact().setId(20)];
      mockGetArtifactsByContext(context, artifacts);
      const artifactResult = await getArtifactsFromContext(context);
      expect(artifactResult).toEqual(artifacts);
    });
  });

  describe('getEventsByExecutions', () => {
    it('returns list of events', async () => {
      const executions = [new Execution().setId(1), new Execution().setId(2)];
      const events = [new Event().setExecutionId(1), new Event().setExecutionId(2)];

      mockGetEventsByExecutions(executions, events);
      const eventsResult = await getEventsByExecutions(executions);
      expect(eventsResult).toEqual(events);
    });
  });

  describe('filterLinkedArtifactsByType', () => {
    it('filter input artifacts', () => {
      const artifactTypeName = 'INPUT';
      const artifactTypes = [
        new ArtifactType().setId(1).setName('INPUT'),
        new ArtifactType().setId(2).setName('OUTPUT'),
      ];
      const inputArtifact = { artifact: new Artifact().setTypeId(1), event: new Event() };
      const outputArtifact = { artifact: new Artifact().setTypeId(2), event: new Event() };
      const artifacts = [inputArtifact, outputArtifact];
      expect(filterLinkedArtifactsByType(artifactTypeName, artifactTypes, artifacts)).toEqual([
        inputArtifact,
      ]);
    });

    it('filter output artifacts', () => {
      const artifactTypeName = 'OUTPUT';
      const artifactTypes = [
        new ArtifactType().setId(1).setName('INPUT'),
        new ArtifactType().setId(2).setName('OUTPUT'),
      ];
      const inputArtifact = { artifact: new Artifact().setTypeId(1), event: new Event() };
      const outputArtifact = { artifact: new Artifact().setTypeId(2), event: new Event() };
      const artifacts = [inputArtifact, outputArtifact];
      expect(filterLinkedArtifactsByType(artifactTypeName, artifactTypes, artifacts)).toEqual([
        outputArtifact,
      ]);
    });
  });
});

function mockGetContextType(contextType: ContextType) {
  jest
    .spyOn(Api.getInstance().metadataStoreService, 'getContextType')
    .mockImplementation((req: GetContextTypeRequest) => {
      const response = new GetContextTypeResponse();
      response.setContextType(contextType);
      return response;
    });
}

function mockGetContextsByExecution(contexts: Context[]) {
  jest
    .spyOn(Api.getInstance().metadataStoreService, 'getContextsByExecution')
    .mockImplementation((req: GetContextsByExecutionRequest) => {
      const response = new GetContextsByExecutionResponse();
      response.setContextsList(contexts);
      return response;
    });
}

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

function mockGetArtifactsByContext(context: Context, artifacts: Artifact[]) {
  jest
    .spyOn(Api.getInstance().metadataStoreService, 'getArtifactsByContext')
    .mockImplementation((req: GetArtifactsByContextRequest) => {
      const response = new GetArtifactsByContextResponse();
      if (req.getContextId() === context.getId()) {
        response.setArtifactsList(artifacts);
      }
      return response;
    });
}

function mockGetEventsByExecutions(executions: Execution[], events: Event[]) {
  jest
    .spyOn(Api.getInstance().metadataStoreService, 'getEventsByExecutionIDs')
    .mockImplementation((req: GetEventsByExecutionIDsRequest) => {
      const response = new GetEventsByExecutionIDsResponse();
      const executionIds = executions.map(e => e.getId());
      if (req.getExecutionIdsList().every((val, index) => val === executionIds[index])) {
        response.setEventsList(events);
      }
      return response;
    });
}

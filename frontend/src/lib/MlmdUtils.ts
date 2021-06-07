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
  Artifact,
  ArtifactType,
  Context,
  Event,
  Execution,
  ExecutionCustomProperties,
  ExecutionProperties,
  GetArtifactsByIDRequest,
  GetArtifactsByIDResponse,
  GetArtifactTypesRequest,
  GetArtifactTypesResponse,
  GetContextByTypeAndNameRequest,
  GetEventsByExecutionIDsRequest,
  GetEventsByExecutionIDsResponse,
  GetExecutionsByContextRequest,
  getResourceProperty,
  getResourcePropertyViaFallBack,
} from '@kubeflow/frontend';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import { Workflow } from 'third_party/argo-ui/argo_template';
import { logger } from './Utils';
import { isV2Pipeline } from './v2/WorkflowUtils';

async function getContext({ type, name }: { type: string; name: string }): Promise<Context> {
  if (type === '') {
    throw new Error('Failed to getContext: type is empty.');
  }
  if (name === '') {
    throw new Error('Failed to getContext: name is empty.');
  }
  const request = new GetContextByTypeAndNameRequest();
  request.setTypeName(type);
  request.setContextName(name);
  try {
    const res = await Api.getInstance().metadataStoreService.getContextByTypeAndName(request);
    const context = res.getContext();
    if (context == null) {
      throw new Error('Cannot find specified context');
    }
    return context;
  } catch (err) {
    err.message = `Cannot find context with ${JSON.stringify(request.toObject())}: ` + err.message;
    throw err;
  }
}

/**
 * @throws error when network error, or not found
 */
async function getTfxRunContext(argoWorkflowName: string): Promise<Context> {
  // argoPodName has the general form "pipelineName-workflowId-executionId".
  // All components of a pipeline within a single run will have the same
  // "pipelineName-workflowId" prefix.
  const pipelineName = argoWorkflowName
    .split('-')
    .slice(0, -1)
    .join('_');
  const runID = argoWorkflowName;
  // An example run context name is parameterized_tfx_oss.parameterized-tfx-oss-4rq5v.
  const tfxRunContextName = `${pipelineName}.${runID}`;
  return await getContext({ name: tfxRunContextName, type: 'run' });
}

/**
 * @throws error when network error, or not found
 */
async function getKfpRunContext(argoWorkflowName: string): Promise<Context> {
  return await getContext({ name: argoWorkflowName, type: 'KfpRun' });
}

async function getKfpV2RunContext(argoWorkflowName: string): Promise<Context> {
  return await getContext({ name: argoWorkflowName, type: 'kfp.PipelineRun' });
}

export async function getRunContext(workflow: Workflow): Promise<Context> {
  const workflowName = workflow?.metadata?.name || '';
  if (isV2Pipeline(workflow)) {
    return await getKfpV2RunContext(workflowName);
  }
  try {
    return await getTfxRunContext(workflowName);
  } catch (err) {
    logger.warn(`Cannot find tfx run context (this is expected for non tfx runs)`, err);
    return await getKfpRunContext(workflowName);
  }
}

/**
 * @throws error when network error
 */
export async function getExecutionsFromContext(context: Context): Promise<Execution[]> {
  const request = new GetExecutionsByContextRequest();
  request.setContextId(context.getId());
  try {
    const res = await Api.getInstance().metadataStoreService.getExecutionsByContext(request);
    const list = res.getExecutionsList();
    if (list == null) {
      throw new Error('response.getExecutionsList() is empty');
    }
    return list;
  } catch (err) {
    err.message =
      `Cannot find executions by context ${context.getId()} with name ${context.getName()}: ` +
      err.message;
    throw err;
  }
}

export enum KfpExecutionProperties {
  KFP_POD_NAME = 'kfp_pod_name',
}

const EXECUTION_PROPERTY_REPOS = [ExecutionProperties, ExecutionCustomProperties];

export const ExecutionHelpers = {
  getWorkspace(execution: Execution): string | number | undefined {
    return (
      getResourcePropertyViaFallBack(execution, EXECUTION_PROPERTY_REPOS, ['RUN_ID']) ||
      getStringProperty(execution, ExecutionCustomProperties.WORKSPACE, true) ||
      getStringProperty(execution, ExecutionProperties.PIPELINE_NAME) ||
      undefined
    );
  },
  getName(execution: Execution): string | number | undefined {
    return (
      // TODO(Bobgy): move task_name to a const when ExecutionCustomProperties are moved back to this repo.
      getStringProperty(execution, 'task_name', true) ||
      getStringProperty(execution, ExecutionProperties.NAME) ||
      getStringProperty(execution, ExecutionProperties.COMPONENT_ID) ||
      getStringProperty(execution, ExecutionCustomProperties.TASK_ID, true) ||
      undefined
    );
  },
  getState(execution: Execution): string | number | undefined {
    return getStringProperty(execution, ExecutionProperties.STATE) || undefined;
  },
  getKfpPod(execution: Execution): string | number | undefined {
    return (
      getStringProperty(execution, KfpExecutionProperties.KFP_POD_NAME) ||
      getStringProperty(execution, KfpExecutionProperties.KFP_POD_NAME, true) ||
      undefined
    );
  },
};

function getStringProperty(
  resource: Artifact | Execution,
  propertyName: string,
  fromCustomProperties = false,
): string | undefined {
  const value = getResourceProperty(resource, propertyName, fromCustomProperties);
  return getStringValue(value);
}

function getStringValue(value?: string | number | Struct | null): string | undefined {
  if (typeof value != 'string') {
    return undefined;
  }
  return value;
}

/**
 * @throws error when network error or invalid data
 */
export async function getOutputArtifactsInExecution(execution: Execution): Promise<Artifact[]> {
  const executionId = execution.getId();
  if (!executionId) {
    throw new Error('Execution must have an ID');
  }

  const request = new GetEventsByExecutionIDsRequest();
  request.addExecutionIds(executionId);
  let res: GetEventsByExecutionIDsResponse;
  try {
    res = await Api.getInstance().metadataStoreService.getEventsByExecutionIDs(request);
  } catch (err) {
    err.message = 'Failed to getExecutionsByExecutionIDs: ' + err.message;
    throw err;
  }

  const outputArtifactIds = res
    .getEventsList()
    .filter(event => event.getType() === Event.Type.OUTPUT && event.getArtifactId())
    .map(event => event.getArtifactId());

  const artifactsRequest = new GetArtifactsByIDRequest();
  artifactsRequest.setArtifactIdsList(outputArtifactIds);
  let artifactsRes: GetArtifactsByIDResponse;
  try {
    artifactsRes = await Api.getInstance().metadataStoreService.getArtifactsByID(artifactsRequest);
  } catch (artifactsErr) {
    artifactsErr.message = 'Failed to getArtifactsByID: ' + artifactsErr.message;
    throw artifactsErr;
  }

  return artifactsRes.getArtifactsList();
}

export async function getArtifactTypes(): Promise<ArtifactType[]> {
  const request = new GetArtifactTypesRequest();
  let res: GetArtifactTypesResponse;
  try {
    res = await Api.getInstance().metadataStoreService.getArtifactTypes(request);
  } catch (err) {
    err.message = 'Failed to getArtifactTypes: ' + err.message;
    throw err;
  }
  return res.getArtifactTypesList();
}

export function filterArtifactsByType(
  artifactTypeName: string,
  artifactTypes: ArtifactType[],
  artifacts: Artifact[],
): Artifact[] {
  const artifactTypeIds = artifactTypes
    .filter(artifactType => artifactType.getName() === artifactTypeName)
    .map(artifactType => artifactType.getId());
  return artifacts.filter(artifact => artifactTypeIds.includes(artifact.getTypeId()));
}

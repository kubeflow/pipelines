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

import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import { getResourceStateText, ResourceType } from 'src/components/ResourceInfo';
import { logger } from 'src/lib/Utils';
import { isV2Pipeline } from 'src/lib/v2/WorkflowUtils';
import {
  Api,
  ArtifactCustomProperties,
  ArtifactProperties,
  ExecutionCustomProperties,
  ExecutionProperties,
  getResourceProperty,
  getResourcePropertyViaFallBack,
} from 'src/mlmd/library';
import {
  Artifact,
  ArtifactType,
  Context,
  ContextType,
  Event,
  Execution,
  GetArtifactsByIDRequest,
  GetArtifactsByIDResponse,
  GetArtifactTypesRequest,
  GetArtifactTypesResponse,
  GetContextByTypeAndNameRequest,
  GetEventsByExecutionIDsRequest,
  GetEventsByExecutionIDsResponse,
  GetExecutionsByContextRequest,
} from 'src/third_party/mlmd';
import {
  GetArtifactsByContextRequest,
  GetContextsByExecutionRequest,
  GetContextsByExecutionResponse,
  GetContextTypeRequest,
  GetContextTypeResponse,
} from 'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_service_pb';
import { Workflow } from 'src/third_party/mlmd/argo_template';

export const KFP_V2_RUN_CONTEXT_TYPE = 'system.PipelineRun';
export const EXECUTION_KEY_CACHED_EXECUTION_ID = 'cached_execution_id';

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
  // Context: https://github.com/kubeflow/pipelines/issues/6138
  // Require TFX version to be >= 1.2.0.
  return await getContext({ name: argoWorkflowName, type: 'pipeline_run' });
}

/**
 * @throws error when network error, or not found
 */
async function getKfpRunContext(argoWorkflowName: string): Promise<Context> {
  return await getContext({ name: argoWorkflowName, type: 'KfpRun' });
}

export async function getKfpV2RunContext(runID: string): Promise<Context> {
  return await getContext({ name: runID, type: KFP_V2_RUN_CONTEXT_TYPE });
}

export async function getRunContext(workflow: Workflow, runID: string): Promise<Context> {
  const workflowName = workflow?.metadata?.name || '';
  if (isV2Pipeline(workflow)) {
    return await getKfpV2RunContext(runID);
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
  // kfp_pod_name is kept for backward compatibility.
  // KFP v1 and TFX logs kfp_pod_name property, but KFP v2 logs pod_name.
  KFP_POD_NAME = 'kfp_pod_name',
  POD_NAME = 'pod_name',
  DISPLAY_NAME = 'display_name',
  TASK_NAME = 'task_name',
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
  getName(execution: Execution): string {
    return `${getStringProperty(execution, KfpExecutionProperties.DISPLAY_NAME, true) ||
      getStringProperty(execution, KfpExecutionProperties.TASK_NAME, true) ||
      getStringProperty(execution, ExecutionProperties.NAME) ||
      getStringProperty(execution, ExecutionProperties.COMPONENT_ID) ||
      getStringProperty(execution, ExecutionCustomProperties.TASK_ID, true) ||
      // TFX 1.2.0 executions do not have any of the above, adding pod name as a fallback name
      getStringProperty(execution, KfpExecutionProperties.KFP_POD_NAME, true) ||
      '(No name)'}`;
  },
  getState(execution: Execution): string | number | undefined {
    return getResourceStateText({
      resourceType: ResourceType.EXECUTION,
      resource: execution,
      typeName: 'Execution',
    });
  },
  getKfpPod(execution: Execution): string | undefined {
    return (
      getStringProperty(execution, KfpExecutionProperties.POD_NAME, true) ||
      getStringProperty(execution, KfpExecutionProperties.KFP_POD_NAME) ||
      getStringProperty(execution, KfpExecutionProperties.KFP_POD_NAME, true) ||
      undefined
    );
  },
};

export enum KfpArtifactProperties {
  DISPLAY_NAME = 'display_name',
}

export const ArtifactHelpers = {
  getName(a: Artifact): string {
    const name =
      getResourceProperty(a, KfpArtifactProperties.DISPLAY_NAME, true) ||
      getResourceProperty(a, ArtifactProperties.NAME) ||
      getResourceProperty(a, ArtifactCustomProperties.NAME, true) ||
      '(No name)';
    return `${name}`;
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

export async function getEventByExecution(execution: Execution): Promise<Event[]> {
  const executionId = execution.getId();
  if (!executionId) {
    throw new Error('Execution must have an ID');
  }

  const request = new GetEventsByExecutionIDsRequest().addExecutionIds(executionId);
  let response: GetEventsByExecutionIDsResponse;
  try {
    response = await Api.getInstance().metadataStoreService.getEventsByExecutionIDs(request);
  } catch (err) {
    err.message = 'Failed to getEventsByExecutionIDs: ' + err.message;
    throw err;
  }
  return response.getEventsList();
}

export async function getContextByExecution(
  execution: Execution,
  contextTypeName: string,
): Promise<Context | undefined> {
  const contexts = await getContextsByExecution(execution);
  const contextType = await getContextType(contextTypeName);
  const result = contexts.filter(c => contextType && c.getTypeId() === contextType.getId());
  if (result.length === 0) {
    console.warn('No context is found for type name: ' + contextTypeName);
    return;
  }
  if (result.length > 1) {
    console.warn('Found more than one context for type name: ' + contextTypeName);
    console.warn('Contexts: ');
    contexts.forEach(c => console.warn(c));
    return;
  }

  return result[0];
}

async function getContextsByExecution(execution: Execution): Promise<Context[]> {
  const executionId = execution.getId();
  if (!executionId) {
    throw new Error('Execution must have an ID');
  }

  const request = new GetContextsByExecutionRequest().setExecutionId(executionId);
  let response: GetContextsByExecutionResponse;
  try {
    response = await Api.getInstance().metadataStoreService.getContextsByExecution(request);
  } catch (err) {
    err.message = 'Failed to getContextsByExecution: ' + err.message;
    throw err;
  }
  return response.getContextsList();
}

async function getContextType(contextTypeName: string): Promise<ContextType | undefined> {
  const request = new GetContextTypeRequest();
  if (contextTypeName) {
    request.setTypeName(contextTypeName);
  }
  let response: GetContextTypeResponse;
  try {
    response = await Api.getInstance().metadataStoreService.getContextType(request);
  } catch (err) {
    err.message = 'Failed to getContextType: ' + err.message;
    throw err;
  }
  return response.getContextType();
}

// An artifact which has associated event.
// You can retrieve artifact name from event.path.steps[0].key
export interface LinkedArtifact {
  event: Event;
  artifact: Artifact;
}

export function getStoreSessionInfoFromArtifact(artifact: LinkedArtifact): string | undefined {
  return artifact.artifact
    .getCustomPropertiesMap()
    .get('store_session_info')
    ?.getStringValue();
}

export async function getLinkedArtifactsByEvents(events: Event[]): Promise<LinkedArtifact[]> {
  const artifactIds = events
    .filter(event => event.getArtifactId())
    .map(event => event.getArtifactId());

  const artifactsRequest = new GetArtifactsByIDRequest().setArtifactIdsList(artifactIds);
  let artifactsRes: GetArtifactsByIDResponse;
  try {
    artifactsRes = await Api.getInstance().metadataStoreService.getArtifactsByID(artifactsRequest);
  } catch (artifactsErr) {
    artifactsErr.message = 'Failed to getArtifactsByID: ' + artifactsErr.message;
    throw artifactsErr;
  }

  const artifactMap = new Map();
  for (const [, artifactEntry] of Object.entries(artifactsRes.getArtifactsList())) {
    artifactMap.set(artifactEntry.getId(), artifactEntry);
  }

  return events.map(event => {
    const artifact = artifactMap.get(event.getArtifactId());
    return { event: event, artifact: artifact };
  });
}

export async function getLinkedArtifactsByExecution(
  execution: Execution,
): Promise<LinkedArtifact[]> {
  const event = await getEventByExecution(execution);
  return getLinkedArtifactsByEvents(event);
}

export function filterEventWithInputArtifact(linkedArtifact: LinkedArtifact[]) {
  return linkedArtifact.filter(obj => obj.event.getType() === Event.Type.INPUT);
}

export function filterEventWithOutputArtifact(linkedArtifact: LinkedArtifact[]) {
  return linkedArtifact.filter(obj => obj.event.getType() === Event.Type.OUTPUT);
}

/**
 * @throws error when network error or invalid data
 */
export async function getOutputArtifactsInExecution(execution: Execution): Promise<Artifact[]> {
  const linkedArtifacts = await getLinkedArtifactsByExecution(execution);
  return filterEventWithOutputArtifact(linkedArtifacts).map(
    linkedArtifact => linkedArtifact.artifact,
  );
}
export async function getOutputLinkedArtifactsInExecution(
  execution: Execution,
): Promise<LinkedArtifact[]> {
  const linkedArtifacts = await getLinkedArtifactsByExecution(execution);
  return filterEventWithOutputArtifact(linkedArtifacts);
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

export function filterLinkedArtifactsByType(
  artifactTypeName: string,
  artifactTypes: ArtifactType[],
  artifacts: LinkedArtifact[],
): LinkedArtifact[] {
  const artifactTypeIds = artifactTypes
    .filter(artifactType => artifactType.getName() === artifactTypeName)
    .map(artifactType => artifactType.getId());
  return artifacts.filter(x => artifactTypeIds.includes(x.artifact.getTypeId()));
}

export function getExecutionDisplayName(execution: Execution): string | undefined {
  return execution
    .getCustomPropertiesMap()
    .get('display_name')
    ?.getStringValue();
}

export function getArtifactName(linkedArtifact: LinkedArtifact): string | undefined {
  return getArtifactNameFromEvent(linkedArtifact.event);
}

export function getArtifactTypeName(
  artifactTypes: ArtifactType[],
  linkedArtifacts: LinkedArtifact[],
): string[] {
  let artifactTypeNames: string[] = [];
  if (artifactTypes && linkedArtifacts) {
    Object.values(linkedArtifacts).forEach(linkedArtifact => {
      const artifactType = artifactTypes.filter(
        aType => aType.getId() === linkedArtifact.artifact.getTypeId(),
      );
      artifactTypeNames.push(
        artifactType.length === 1 && artifactType[0].getName() ? artifactType[0].getName() : '-',
      );
    });
  }
  return artifactTypeNames;
}

export function getArtifactNameFromEvent(event: Event): string | undefined {
  return event
    .getPath()
    ?.getStepsList()[0]
    .getKey();
}

export async function getArtifactsFromContext(context: Context): Promise<Artifact[]> {
  const request = new GetArtifactsByContextRequest();
  request.setContextId(context.getId());
  try {
    const res = await Api.getInstance().metadataStoreService.getArtifactsByContext(request);
    const list = res.getArtifactsList();
    if (list == null) {
      throw new Error('response.getExecutionsList() is empty');
    }
    // Display name of artifact exists in getCustomPropertiesMap().get('display_name').getStringValue().
    // Note that the actual artifact name is in Event which generates this artifact.
    return list;
  } catch (err) {
    err.message =
      `Cannot find executions by context ${context.getId()} with name ${context.getName()}: ` +
      err.message;
    throw err;
  }
}

export async function getEventsByExecutions(executions: Execution[] | undefined): Promise<Event[]> {
  if (!executions) {
    return [];
  }
  const request = new GetEventsByExecutionIDsRequest();
  for (let exec of executions) {
    const execId = exec.getId();
    if (!execId) {
      throw new Error('Execution must have an ID');
    }
    request.addExecutionIds(execId);
  }
  let response: GetEventsByExecutionIDsResponse;
  try {
    response = await Api.getInstance().metadataStoreService.getEventsByExecutionIDs(request);
  } catch (err) {
    err.message = 'Failed to getEventsByExecutionIDs: ' + err.message;
    throw err;
  }
  return response.getEventsList();
}

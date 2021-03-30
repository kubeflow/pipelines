/**
 * Copyright 2021 Google LLC
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
  Execution,
  ExecutionCustomProperties,
  getResourcePropertyViaFallBack,
  ExecutionProperties,
  getResourceProperty,
} from '@kubeflow/frontend';
import {
  GetContextByTypeAndNameRequest,
  GetExecutionsByContextRequest,
} from '@kubeflow/frontend/src/mlmd/generated/ml_metadata/proto/metadata_store_service_pb';
import { Workflow } from 'third_party/argo-ui/argo_template';
import { logger } from './Utils';

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
  if (workflow?.metadata?.annotations?.['pipelines.kubeflow.org/v2_pipeline'] === 'true') {
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
      getResourceProperty(execution, ExecutionCustomProperties.WORKSPACE, true) ||
      getResourceProperty(execution, ExecutionProperties.PIPELINE_NAME) ||
      undefined
    );
  },
  getName(execution: Execution): string | number | undefined {
    return (
      getResourceProperty(execution, ExecutionProperties.NAME) ||
      getResourceProperty(execution, ExecutionProperties.COMPONENT_ID) ||
      getResourceProperty(execution, ExecutionCustomProperties.TASK_ID, true) ||
      undefined
    );
  },
  getState(execution: Execution): string | number | undefined {
    return getResourceProperty(execution, ExecutionProperties.STATE) || undefined;
  },
  getKfpPod(execution: Execution): string | number | undefined {
    return (
      getResourceProperty(execution, KfpExecutionProperties.KFP_POD_NAME) ||
      getResourceProperty(execution, KfpExecutionProperties.KFP_POD_NAME, true) ||
      undefined
    );
  },
};

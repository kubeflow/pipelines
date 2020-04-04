import {
  Context,
  Api,
  Execution,
  getResourceProperty,
  ExecutionProperties,
  ExecutionCustomProperties,
} from '@kubeflow/frontend';
import {
  GetContextByTypeAndNameRequest,
  GetExecutionsByContextRequest,
} from '@kubeflow/frontend/src/mlmd/generated/ml_metadata/proto/metadata_store_service_pb';

/**
 * @throws error when network error
 */
export async function getTfxRunContext(argoWorkflowName: string): Promise<Context> {
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

  const request = new GetContextByTypeAndNameRequest();
  request.setTypeName('run');
  request.setContextName(tfxRunContextName);
  try {
    const res = await Api.getInstance().metadataStoreService.getContextByTypeAndName(request);
    const context = res.getContext();
    if (context == null) {
      throw new Error('response.getContext() is empty');
    }
    return context;
  } catch (err) {
    err.message = `Cannot find run context with name "${tfxRunContextName}": ` + err.message;
    throw err;
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

export const ExecutionHelpers = {
  getPipeline(execution: Execution): string | number | undefined {
    return (
      getResourceProperty(execution, ExecutionProperties.PIPELINE_NAME) ||
      getResourceProperty(execution, ExecutionCustomProperties.WORKSPACE, true) ||
      getResourceProperty(execution, ExecutionCustomProperties.RUN_ID, true) ||
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
};

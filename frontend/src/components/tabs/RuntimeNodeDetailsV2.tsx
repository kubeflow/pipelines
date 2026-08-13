/*
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Button } from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import {
  InputOutputsIOArtifact,
  PipelineTaskTaskPodType,
  PipelineTaskTaskState,
  V2beta1PipelineTask,
} from 'src/apisv2beta1/run';
import MD2Tabs from 'src/atoms/MD2Tabs';
import ArtifactPreview from 'src/components/ArtifactPreview';
import { buildRuntimeArtifactRows } from 'src/components/RuntimeArtifactRows';
import Banner from 'src/components/Banner';
import DetailsTable from 'src/components/DetailsTable';
import LogViewer from 'src/components/LogViewer';
import { RuntimeInputOutputTab } from 'src/components/tabs/RuntimeInputOutputTab';
import { RuntimeMetricsVisualizations } from 'src/components/viewers/RuntimeMetricsVisualizations';
import { commonCss, padding } from 'src/Css';
import {
  KubernetesExecutorConfig,
  PvcMount,
} from 'src/generated/platform_spec/kubernetes_platform';
import { PlatformDeploymentConfig } from 'src/generated/pipeline_spec/pipeline_spec';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis } from 'src/lib/Apis';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { readArtifactFile } from 'src/lib/v2/ArtifactFileUtils';
import { getComponentSpec } from 'src/lib/v2/NodeUtils';
import {
  EXECUTOR_LOGS_ARTIFACT_KEY,
  flattenArtifactGroups,
  getArtifactDisplayName,
  getArtifactTypeName,
  getOutputArtifactByName,
  isTaskFinished,
} from 'src/lib/v2/RuntimeArtifactUtils';
import { NodeRuntimeInfo } from 'src/lib/v2/DynamicFlow';
import { getTaskKeyFromNodeKey, NodeTypeNames, PipelineFlowElement } from 'src/lib/v2/StaticFlow';
import { convertYamlToPlatformSpec, convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';

export const LOGS_DETAILS = 'logs_details';
export const LOGS_BANNER_MESSAGE = 'logs_banner_message';
export const LOGS_BANNER_ADDITIONAL_INFO = 'logs_banner_additional_info';
export const K8S_PLATFORM_KEY = 'kubernetes';

const NODE_INFO_UNKNOWN = (
  <div className='relative flex flex-col h-screen'>
    <div className='absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2'>
      Unable to retrieve node info.
    </div>
  </div>
);

const NODE_STATE_UNAVAILABLE = (
  <div className='relative flex flex-col h-screen'>
    <div className='absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2'>
      Content is not available yet.
    </div>
  </div>
);

interface RuntimeNodeDetailsV2Props {
  layers: string[];
  onLayerChange: (layers: string[]) => void;
  pipelineJobString?: string;
  runId?: string;
  element?: PipelineFlowElement | null;
  elementRuntimeInfo?: NodeRuntimeInfo | null;
  namespace?: string;
}

export function RuntimeNodeDetailsV2({
  layers,
  onLayerChange,
  pipelineJobString,
  runId,
  element,
  elementRuntimeInfo,
  namespace,
}: RuntimeNodeDetailsV2Props) {
  if (!element) {
    return NODE_INFO_UNKNOWN;
  }
  if (element.type === NodeTypeNames.EXECUTION) {
    return (
      <TaskNodeDetail
        pipelineJobString={pipelineJobString}
        runId={runId}
        element={element}
        task={elementRuntimeInfo?.task}
        layers={layers}
        namespace={namespace}
      />
    );
  }
  if (element.type === NodeTypeNames.ARTIFACT) {
    return (
      <ArtifactNodeDetail
        task={elementRuntimeInfo?.task}
        artifactGroup={elementRuntimeInfo?.artifactGroup}
        namespace={namespace}
      />
    );
  }
  if (element.type === NodeTypeNames.SUB_DAG) {
    return (
      <SubDAGNodeDetail
        element={element}
        task={elementRuntimeInfo?.task}
        layers={layers}
        onLayerChange={onLayerChange}
        namespace={namespace}
      />
    );
  }
  return NODE_INFO_UNKNOWN;
}

interface TaskNodeDetailProps {
  pipelineJobString?: string;
  runId?: string;
  element?: PipelineFlowElement | null;
  task?: V2beta1PipelineTask;
  layers: string[];
  namespace?: string;
}

function TaskNodeDetail({
  pipelineJobString,
  runId,
  element,
  task,
  layers,
  namespace,
}: TaskNodeDetailProps) {
  const [selectedTab, setSelectedTab] = useState(0);
  const {
    data: logsInfo,
    isError: logsQueryFailed,
    error: logsQueryError,
  } = useQuery<Map<string, string>, Error>({
    queryKey: queryKeys.taskLogs(task?.task_id, task?.state, namespace),
    queryFn: () => {
      if (!task) {
        throw new Error('No task is found.');
      }
      return getLogsInfo(task, runId, namespace);
    },
    enabled: !!task && selectedTab === 2,
  });

  const logsDetails = logsInfo?.get(LOGS_DETAILS);
  const logsBannerMessage =
    logsInfo?.get(LOGS_BANNER_MESSAGE) ||
    (logsQueryFailed ? 'Failed to retrieve pod logs.' : undefined);
  const logsBannerAdditionalInfo =
    logsInfo?.get(LOGS_BANNER_ADDITIONAL_INFO) || logsQueryError?.message;

  return (
    <div className={commonCss.page}>
      <MD2Tabs
        tabs={['Input/Output', 'Task Details', 'Logs']}
        selectedTab={selectedTab}
        onSwitch={setSelectedTab}
      />
      <div className={commonCss.page}>
        {selectedTab === 0 &&
          (task ? (
            <RuntimeInputOutputTab task={task} namespace={namespace} />
          ) : (
            NODE_STATE_UNAVAILABLE
          ))}
        {selectedTab === 1 && (
          <div className={padding(20)}>
            <DetailsTable title='Task Details' fields={getTaskDetailsFields(element, task)} />
            <DetailsTable
              title='Volume Mounts'
              fields={getNodeVolumeMounts(layers, pipelineJobString, element)}
            />
          </div>
        )}
        {selectedTab === 2 && (
          <div className={commonCss.page}>
            {logsBannerMessage ? (
              <Banner message={logsBannerMessage} additionalInfo={logsBannerAdditionalInfo} />
            ) : (
              <div className={commonCss.pageOverflowHidden} data-testid='logs-view-window'>
                <LogViewer logLines={(logsDetails || '').split(/[\r\n]+/)} />
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export function getTaskDetailsFields(
  element?: PipelineFlowElement | null,
  task?: V2beta1PipelineTask,
): Array<KeyValue<string>> {
  if (!element) {
    return [];
  }
  const details: Array<KeyValue<string>> = [['Task ID', task?.task_id || element.id || '-']];
  if (!task) {
    return details;
  }
  details.push(['Task name', task.display_name || task.name || '-']);
  details.push(['Status', formatTaskState(task.state)]);
  details.push(['Created At', formatDateString(task.create_time)]);
  details.push(['Finished At', isTaskFinished(task.state) ? formatDateString(task.end_time) : '-']);
  if (task.status_metadata?.message) {
    details.push(['Message', task.status_metadata.message]);
  }
  const podNames = (task.pods || [])
    .map((pod) => pod.name)
    .filter(Boolean)
    .join(', ');
  if (podNames) {
    details.push(['Pods', podNames]);
  }
  return details;
}

function formatTaskState(state?: PipelineTaskTaskState): string {
  if (!state || state === PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED) {
    return 'Unknown';
  }
  return state.charAt(0) + state.slice(1).toLowerCase();
}

function getNodeVolumeMounts(
  layers: string[],
  pipelineJobString?: string,
  element?: PipelineFlowElement | null,
): Array<KeyValue<string>> {
  if (!pipelineJobString || !element) {
    return [];
  }
  const taskKey = getTaskKeyFromNodeKey(element.id);
  const pipelineSpec = convertYamlToV2PipelineSpec(pipelineJobString);
  const componentSpec = getComponentSpec(pipelineSpec, layers, taskKey);
  const platformSpec = convertYamlToPlatformSpec(pipelineJobString);
  if (!platformSpec || !platformSpec.platforms[K8S_PLATFORM_KEY]) {
    return [];
  }
  const deploymentSpec = PlatformDeploymentConfig.fromJSON(
    platformSpec.platforms[K8S_PLATFORM_KEY].deploymentSpec,
  );
  const matchedExecutor = Object.entries(deploymentSpec.executors).find(
    ([executorName]) => executorName === componentSpec?.executorLabel,
  );
  if (!matchedExecutor) {
    return [];
  }
  const executor = KubernetesExecutorConfig.fromJSON(matchedExecutor[1]);
  return Object.values(executor.pvcMount)
    .map((mount) => PvcMount.fromJSON(mount))
    .map((mount) => [mount.mountPath, mount.taskOutputParameter?.producerTask]);
}

export async function getLogsInfo(
  task: V2beta1PipelineTask,
  runId?: string,
  namespace?: string,
): Promise<Map<string, string>> {
  const logsInfo = new Map<string, string>();
  if (task.state === PipelineTaskTaskState.CACHED) {
    logsInfo.set(LOGS_DETAILS, 'This step output is taken from cache.');
    return logsInfo;
  }

  // Driver pods have different logs from the task executor. If no executor pod is available,
  // use the executor-logs artifact fallback rather than presenting driver output as task logs.
  const pod = task.pods?.find((candidate) => candidate.type === PipelineTaskTaskPodType.EXECUTOR);
  const createdAt = (task.create_time || new Date()).toISOString().split('T')[0];
  let podLogsError: unknown;
  if (runId && pod?.name) {
    try {
      logsInfo.set(
        LOGS_DETAILS,
        await Apis.getPodLogs(runId, pod.name, namespace || '', createdAt),
      );
      return logsInfo;
    } catch (error) {
      podLogsError = error;
    }
  } else {
    podLogsError = new Error('Task pod information is not available.');
  }

  const executorLogsArtifact = getOutputArtifactByName(task, EXECUTOR_LOGS_ARTIFACT_KEY);
  let artifactLogsError: unknown;
  if (executorLogsArtifact?.uri) {
    try {
      logsInfo.set(LOGS_DETAILS, await readArtifactFile(executorLogsArtifact, namespace));
      return logsInfo;
    } catch (error) {
      artifactLogsError = error;
    }
  }

  const podErrorMessage = await errorToMessage(podLogsError);
  logsInfo.set(
    LOGS_BANNER_MESSAGE,
    artifactLogsError ? 'Failed to retrieve task logs.' : 'Failed to retrieve pod logs.',
  );
  logsInfo.set(
    LOGS_BANNER_ADDITIONAL_INFO,
    artifactLogsError
      ? `Pod logs error: ${podErrorMessage}\nExecutor logs artifact error: ${await errorToMessage(
          artifactLogsError,
        )}`
      : `Error response: ${podErrorMessage}`,
  );
  return logsInfo;
}

interface ArtifactNodeDetailProps {
  task?: V2beta1PipelineTask;
  artifactGroup?: InputOutputsIOArtifact;
  namespace?: string;
}

function ArtifactNodeDetail({ task, artifactGroup, namespace }: ArtifactNodeDetailProps) {
  const [selectedTab, setSelectedTab] = useState(0);
  const artifacts = artifactGroup?.artifacts || [];
  if (!task || !artifactGroup || !artifacts.length) {
    return NODE_STATE_UNAVAILABLE;
  }
  return (
    <div className={commonCss.page}>
      <MD2Tabs
        tabs={['Artifact Info', 'Visualization']}
        selectedTab={selectedTab}
        onSwitch={setSelectedTab}
      />
      <div className={padding(20)}>
        {selectedTab === 0 && (
          <ArtifactInfo task={task} artifactGroup={artifactGroup} namespace={namespace} />
        )}
        {selectedTab === 1 && (
          <RuntimeMetricsVisualizations
            artifacts={artifacts}
            artifactKey={artifactGroup.artifact_key}
            namespace={namespace}
          />
        )}
      </div>
    </div>
  );
}

function ArtifactInfo({
  task,
  artifactGroup,
  namespace,
}: Required<Pick<ArtifactNodeDetailProps, 'task' | 'artifactGroup'>> &
  Pick<ArtifactNodeDetailProps, 'namespace'>) {
  const artifactEntries = flattenArtifactGroups([artifactGroup]);
  const uriRows = buildRuntimeArtifactRows([artifactGroup]);
  const firstArtifact = artifactEntries[0].artifact;
  const artifactInfo: Array<KeyValue<string>> = [
    ['Upstream Task Name', task.display_name || task.name || '-'],
    ['Artifact Name', getArtifactDisplayName(firstArtifact, artifactGroup.artifact_key)],
    ['Artifact Type', getArtifactTypeName(firstArtifact)],
    ['Created At', formatDateString(firstArtifact.created_at)],
  ];
  if (artifactEntries.length > 1) {
    artifactInfo.push(['Artifact Count', String(artifactEntries.length)]);
  }
  return (
    <div>
      <h3>{getArtifactDisplayName(firstArtifact, artifactGroup.artifact_key)}</h3>
      <DetailsTable title='Artifact Info' fields={artifactInfo} />
      <DetailsTable
        title='Artifact URI'
        fields={uriRows}
        valueComponent={ArtifactPreview}
        valueComponentProps={{ namespace }}
      />
    </div>
  );
}

interface SubDAGNodeDetailProps {
  element: PipelineFlowElement;
  task?: V2beta1PipelineTask;
  layers: string[];
  onLayerChange: (layers: string[]) => void;
  namespace?: string;
}

function SubDAGNodeDetail({
  element,
  task,
  layers,
  onLayerChange,
  namespace,
}: SubDAGNodeDetailProps) {
  const [selectedTab, setSelectedTab] = useState(0);
  const taskKey = getTaskKeyFromNodeKey(element.id);
  return (
    <div className={commonCss.page}>
      <div className={padding(20, 'blr')}>
        <Button variant='contained' onClick={() => onLayerChange([...layers, taskKey])}>
          Open Sub-DAG
        </Button>
      </div>
      <MD2Tabs
        tabs={['Input/Output', 'Task Details']}
        selectedTab={selectedTab}
        onSwitch={setSelectedTab}
      />
      <div className={commonCss.page}>
        {selectedTab === 0 &&
          (task ? (
            <RuntimeInputOutputTab task={task} namespace={namespace} />
          ) : (
            NODE_STATE_UNAVAILABLE
          ))}
        {selectedTab === 1 && (
          <div className={padding(20)}>
            <DetailsTable title='Task Details' fields={getTaskDetailsFields(element, task)} />
          </div>
        )}
      </div>
    </div>
  );
}

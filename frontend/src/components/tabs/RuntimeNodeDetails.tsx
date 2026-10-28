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

import { useQuery } from '@tanstack/react-query';
import { useEffect, useId, useMemo, useRef, useState } from 'react';
import { Check, Copy, ChevronDown, ChevronRight } from 'lucide-react';
import './RuntimeNodeDetails.css';
import {
  InputOutputsIOArtifact,
  PipelineTaskTaskPod,
  PipelineTaskTaskPodType,
  PipelineTaskTaskState,
  V2PipelineTask,
  V2Artifact,
} from 'src/apisv2/run';
import { InspectionTabs } from '../inspection/InspectionTabs';
import { buildRuntimeArtifactRows, RuntimeArtifactValue } from 'src/components/RuntimeArtifactRows';
import { InspectionNotice as Banner } from '../inspection/InspectionNotice';
import { InspectionFields as DetailsTable } from '../inspection/InspectionFields';
import { Button } from '../ui/button';
import LogViewer, { LOG_ROW_HEIGHT } from 'src/components/LogViewer';
import { RuntimeInputOutputTab } from 'src/components/tabs/RuntimeInputOutputTab';
import { RuntimeMetricsVisualizations } from 'src/components/viewers/RuntimeMetricsVisualizations';
import {
  KubernetesExecutorConfig,
  PvcMount,
} from 'src/generated/platform_spec/kubernetes_platform';
import { PlatformDeploymentConfig } from 'src/generated/pipeline_spec/pipeline_spec';
import { queryKeys } from 'src/hooks/queryKeys';
import { Apis } from 'src/lib/Apis';
import { KeyValue } from 'src/lib/DetailsTableTypes';
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { readArtifactFile } from 'src/lib/v2/ArtifactFileUtils';
import { getComponentSpec } from 'src/lib/v2/NodeUtils';
import {
  EXECUTOR_LOGS_ARTIFACT_KEY,
  flattenArtifactGroups,
  getArtifactDisplayName,
  getArtifactTypeName,
  getOutputArtifactByName,
  getExecutorLogsRetryIndex,
  isTaskFinished,
} from 'src/lib/v2/RuntimeArtifactUtils';
import { NodeRuntimeInfo } from 'src/lib/v2/DynamicFlow';
import { getTaskDisplayName, getTaskLifecycleMessage } from 'src/lib/v2/RunTaskUtils';
import { getTaskKeyFromNodeKey, NodeTypeNames } from 'src/lib/v2/StaticFlow';
import type { PipelineFlowElement } from 'src/lib/v2/FlowTypes';
import { convertYamlToPlatformSpec, convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';

export const LOGS_DETAILS = 'logs_details';
export const LOGS_BANNER_MESSAGE = 'logs_banner_message';
export const LOGS_BANNER_ADDITIONAL_INFO = 'logs_banner_additional_info';
export const K8S_PLATFORM_KEY = 'kubernetes';

export const LIFECYCLE_WARNING_MESSAGE =
  "Kubernetes reported a problem with this task's pod. The task has not failed yet and may recover. Select Details to see the raw message.";
export const LIFECYCLE_ENDED_MESSAGE =
  "Kubernetes reported a problem with this task's pod before the run ended. Select Details to see the raw message.";
export const LIFECYCLE_FAILURE_MESSAGE =
  'The task failed after Kubernetes reported a problem with its pod. Select Details to see the raw message.';

function LifecycleBanner({
  task,
  sourceFinished,
}: {
  task?: V2PipelineTask;
  sourceFinished?: boolean;
}) {
  const lifecycleMessage = getTaskLifecycleMessage(task);
  if (!lifecycleMessage) {
    return null;
  }
  const failed = task?.state === PipelineTaskTaskState.FAILED;
  return (
    <Banner
      mode={failed ? 'error' : 'warning'}
      message={
        failed
          ? LIFECYCLE_FAILURE_MESSAGE
          : sourceFinished
            ? LIFECYCLE_ENDED_MESSAGE
            : LIFECYCLE_WARNING_MESSAGE
      }
      additionalInfo={lifecycleMessage}
    />
  );
}

const DRIVER_LOGS_UNAVAILABLE: ReadonlyMap<string, string> = new Map([
  [LOGS_BANNER_MESSAGE, 'Driver logs are not available yet.'],
]);

const NODE_INFO_UNKNOWN = (
  <p className='kfp-inspection-empty' role='status'>
    Unable to retrieve node info.
  </p>
);

const NODE_STATE_UNAVAILABLE = (
  <p className='kfp-inspection-empty' role='status'>
    Content is not available yet.
  </p>
);

interface RuntimeNodeDetailsProps {
  layers: string[];
  pipelineJobString?: string;
  runId?: string;
  element?: PipelineFlowElement | null;
  elementRuntimeInfo?: NodeRuntimeInfo | null;
  namespace?: string;
  namespacePending?: boolean;
  sourceFinished?: boolean;
  selectedTaskTab?: number;
  onTaskTabChange?: (tab: number) => void;
}

export function RuntimeNodeDetails({
  layers,
  pipelineJobString,
  runId,
  element,
  elementRuntimeInfo,
  namespace,
  namespacePending = false,
  sourceFinished,
  selectedTaskTab,
  onTaskTabChange,
}: RuntimeNodeDetailsProps) {
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
        namespacePending={namespacePending}
        sourceFinished={sourceFinished}
        selectedTaskTab={selectedTaskTab}
        onTaskTabChange={onTaskTabChange}
      />
    );
  }
  if (element.type === NodeTypeNames.ARTIFACT) {
    return (
      <ArtifactNodeDetail
        task={elementRuntimeInfo?.task}
        artifactGroup={elementRuntimeInfo?.artifactGroup}
        namespace={namespace}
        namespacePending={namespacePending}
        sourceFinished={sourceFinished}
      />
    );
  }
  if (element.type === NodeTypeNames.SUB_DAG) {
    return (
      <SubDAGNodeDetail
        element={element}
        task={elementRuntimeInfo?.task}
        namespace={namespace}
        namespacePending={namespacePending}
        sourceFinished={sourceFinished}
      />
    );
  }
  return NODE_INFO_UNKNOWN;
}

interface TaskNodeDetailProps {
  selectedTaskTab?: number;
  onTaskTabChange?: (tab: number) => void;
  pipelineJobString?: string;
  runId?: string;
  element?: PipelineFlowElement | null;
  task?: V2PipelineTask;
  layers: string[];
  namespace?: string;
  namespacePending?: boolean;
  sourceFinished?: boolean;
}

function getLatestTaskPod(
  task: V2PipelineTask | undefined,
  type: PipelineTaskTaskPodType,
): PipelineTaskTaskPod | undefined {
  const pods = task?.pods || [];
  for (let index = pods.length - 1; index >= 0; index--) {
    if (pods[index].type === type) {
      return pods[index];
    }
  }
  return undefined;
}

interface TaskLogSource {
  key: string;
  label: string;
  artifact?: V2Artifact;
  pod?: PipelineTaskTaskPod;
}

export function getTaskLogSources(
  task: V2PipelineTask | undefined,
  type: PipelineTaskTaskPodType = PipelineTaskTaskPodType.EXECUTOR,
): TaskLogSource[] {
  const key =
    type === PipelineTaskTaskPodType.EXECUTOR ? EXECUTOR_LOGS_ARTIFACT_KEY : 'driver-logs';
  const seen = new Set<string>();
  const artifacts = flattenArtifactGroups(task?.outputs?.artifacts)
    .filter(({ artifact, artifactKey }) => {
      if (!artifact.uri || (artifactKey !== key && artifact.name !== key) || seen.has(artifact.uri))
        return false;
      seen.add(artifact.uri);
      return true;
    })
    .map(({ artifact }) => artifact)
    .sort(
      (a, b) =>
        (getExecutorLogsRetryIndex(b) ?? -1) - (getExecutorLogsRetryIndex(a) ?? -1) ||
        (b.created_at?.getTime() || 0) - (a.created_at?.getTime() || 0),
    );
  const sources: TaskLogSource[] = artifacts.map((artifact) => {
    const retry =
      type === PipelineTaskTaskPodType.EXECUTOR ? getExecutorLogsRetryIndex(artifact) : undefined;
    const identity = artifact.artifact_id || artifact.uri!;
    // Retry indices are local to an execution, not a lifetime counter across manual retries.
    const ambiguous =
      retry !== undefined &&
      artifacts.filter((other) => getExecutorLogsRetryIndex(other) === retry).length > 1;
    return {
      key: `artifact:${identity}`,
      label:
        retry === undefined
          ? `Archived · ${artifact.created_at ? formatDateString(artifact.created_at) : identity}`
          : `Attempt ${retry + 1} · archived${ambiguous ? ` · ${identity}` : ''}`,
      artifact,
    };
  });
  const pods = task?.pods?.filter((pod) => pod.type === type && pod.name) || [];
  // The API does not associate pods with artifact attempt indices. Keep pod identities explicit.
  for (const pod of pods.slice(0, -1)) {
    const key = `pod:${pod.uid || pod.name}`;
    if (!sources.some((source) => source.key === key))
      sources.push({ key, label: `Pod · ${pod.name}`, pod });
  }
  return sources;
}

// Archives are the canonical history when present. The API does not identify which pod
// belongs to which archive, so listing both would offer duplicate copies of attempts.
function historicalLogSources(sources: TaskLogSource[], latestUri?: string): TaskLogSource[] {
  const archives = sources.filter((source) => source.artifact);
  return archives.length
    ? archives.filter((source) => source.artifact?.uri !== latestUri)
    : sources.filter((source) => source.pod);
}

function HistoricalLogSection({
  source,
  task,
  runId,
  namespace,
  namespacePending,
  driver,
}: {
  source: TaskLogSource;
  task: V2PipelineTask;
  runId?: string;
  namespace?: string;
  namespacePending: boolean;
  driver: boolean;
}) {
  const [expanded, setExpanded] = useState(false);
  const contentId = useId();
  const kind = driver ? 'Driver' : 'Component';
  const label = `${kind} · ${source.label.replace(' · archived', '')}`;
  const load = () =>
    driver
      ? getTaskDriverLogsInfo(task, runId, namespace, source)
      : getLogsInfo(task, runId, namespace, source);
  const { data, isError, error, refetch } = useQuery<Map<string, string>, Error>({
    queryKey: driver
      ? queryKeys.driverLogs(
          task.task_id,
          task.state,
          namespace,
          source.key + source.artifact?.uri,
          true,
        )
      : queryKeys.taskLogs(
          task.task_id,
          task.state,
          namespace,
          source.key + source.artifact?.uri,
          true,
        ),
    queryFn: load,
    enabled: expanded && !namespacePending,
  });
  return (
    <section
      aria-label={`${label} logs`}
      className={driver ? 'kfp-node-logs-driver' : 'kfp-node-logs-task'}
    >
      <div className='kfp-node-logs-heading'>
        <button
          type='button'
          className='kfp-node-logs-toggle'
          aria-expanded={expanded}
          aria-controls={contentId}
          onClick={() => setExpanded(!expanded)}
        >
          {expanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
          {label}
        </button>
        <CopyLogsButton
          label={`${label} logs`}
          text={data?.get(LOGS_DETAILS)}
          disabled={namespacePending}
          load={load}
        />
      </div>
      {expanded && (
        <div id={contentId}>
          {driver && (
            <p className='kfp-node-logs-description'>
              Pipeline orchestration logs from the KFP driver.
            </p>
          )}
          {namespacePending ? (
            <p role='status'>Loading experiment namespace…</p>
          ) : !data && !isError ? (
            <p role='status'>Loading {kind.toLowerCase()} logs…</p>
          ) : (
            <LogsTabPanel
              label={`${label} logs`}
              details={data?.get(LOGS_DETAILS)}
              bannerMessage={
                data?.get(LOGS_BANNER_MESSAGE) ||
                (isError ? `Failed to retrieve ${kind.toLowerCase()} logs.` : undefined)
              }
              bannerAdditionalInfo={data?.get(LOGS_BANNER_ADDITIONAL_INFO) || error?.message}
              bannerMode={isError || data?.has(LOGS_BANNER_ADDITIONAL_INFO) ? 'error' : 'info'}
              onRefresh={() => void refetch()}
            />
          )}
        </div>
      )}
    </section>
  );
}

function TaskNodeDetail({
  pipelineJobString,
  runId,
  element,
  task,
  layers,
  namespace,
  namespacePending = false,
  sourceFinished,
  selectedTaskTab,
  onTaskTabChange,
}: TaskNodeDetailProps) {
  const [localTab, setLocalTab] = useState(0);
  const requestedTab = selectedTaskTab ?? localTab;
  const selectedTab = requestedTab === 3 ? 2 : requestedTab;
  const driverContentId = useId();
  const taskContentId = useId();
  const [taskChoice, setTaskChoice] = useState<{ taskId?: string; expanded: boolean }>();
  const taskExpanded =
    taskChoice && taskChoice.taskId === task?.task_id ? taskChoice.expanded : true;
  const logSources = getTaskLogSources(task);
  const driverSources = getTaskLogSources(task, PipelineTaskTaskPodType.DRIVER);
  const [driverChoice, setDriverChoice] = useState<{ taskId?: string; expanded: boolean }>();
  const setSelectedTab = onTaskTabChange ?? setLocalTab;
  const executorPod = getLatestTaskPod(task, PipelineTaskTaskPodType.EXECUTOR);
  const driverPod = getLatestTaskPod(task, PipelineTaskTaskPodType.DRIVER);
  const executorLogsArtifact = task
    ? getOutputArtifactByName(task, EXECUTOR_LOGS_ARTIFACT_KEY)
    : undefined;
  // A running pod can be a newer attempt than every archive. Do not relabel or hide
  // the previous archive as its output; archives cannot be correlated to pods by the API.
  const liveSource: TaskLogSource | undefined =
    executorPod?.name && task && !isTaskFinished(task.state)
      ? { key: `pod:${executorPod.uid || executorPod.name}`, label: 'Latest', pod: executorPod }
      : undefined;
  const logsSourceIdentity = [
    executorPod?.name,
    executorLogsArtifact?.artifact_id,
    executorLogsArtifact?.uri,
  ].join(':');
  const {
    data: logsInfo,
    isError: logsQueryFailed,
    error: logsQueryError,
  } = useQuery<Map<string, string>, Error>({
    queryKey: queryKeys.taskLogs(
      task?.task_id,
      task?.state,
      namespace,
      logsSourceIdentity,
      sourceFinished,
    ),
    queryFn: () => {
      if (!task) {
        throw new Error('No task is found.');
      }
      return getLogsInfo(task, runId, namespace, liveSource);
    },
    // Experiment metadata may still be resolving the namespace used by pod and artifact reads.
    enabled: !!task && selectedTab === 2 && taskExpanded && !namespacePending,
    // Preserve output only while refreshing the same pod. A newly discovered pod is a
    // different attempt and must never temporarily display the previous attempt's output.
    placeholderData: (previousLogs, previousQuery) => {
      const previousTaskId = (previousQuery?.queryKey[1] as { taskId?: string } | undefined)
        ?.taskId;
      const previousSource = (previousQuery?.queryKey[1] as { sourceIdentity?: string } | undefined)
        ?.sourceIdentity;
      return previousSource?.split(':')[0] === (executorPod?.name || '') &&
        task?.task_id &&
        previousTaskId === task.task_id
        ? previousLogs
        : undefined;
    },
    // Live logs and transient "not available yet" responses must recover while the task runs.
    refetchInterval:
      taskExpanded && task && !sourceFinished && !isTaskFinished(task.state) ? 10000 : false,
  });

  const logsDetails = logsInfo?.get(LOGS_DETAILS);
  const logsBannerMessage =
    logsInfo?.get(LOGS_BANNER_MESSAGE) ||
    (logsQueryFailed ? 'Failed to retrieve component logs.' : undefined);
  const logsBannerAdditionalInfo =
    logsInfo?.get(LOGS_BANNER_ADDITIONAL_INFO) || logsQueryError?.message;
  const taskLogsUnavailable =
    (!executorPod?.name && !executorLogsArtifact?.uri) ||
    logsQueryFailed ||
    (logsInfo !== undefined && !logsDetails?.trim());
  const driverExpanded =
    driverChoice?.taskId === task?.task_id && driverChoice !== undefined
      ? driverChoice.expanded
      : taskLogsUnavailable || requestedTab === 3;
  const driverLogUri = getDriverLogUri(task) || driverSources[0]?.artifact?.uri;
  const componentHistory = historicalLogSources(
    logSources,
    liveSource ? undefined : executorLogsArtifact?.uri,
  );
  const driverHistory = historicalLogSources(driverSources, driverLogUri);
  const latestRetry =
    !liveSource && executorLogsArtifact
      ? getExecutorLogsRetryIndex(executorLogsArtifact)
      : undefined;
  const componentTitle = componentHistory.length
    ? `Component · ${latestRetry !== undefined ? `Attempt ${latestRetry + 1} · ` : ''}Latest`
    : 'Component';
  const hasDriverSource = !!driverLogUri || !!(runId && driverPod?.name);
  const { data: driverLogsInfo, refetch: refetchDriverLogs } = useQuery({
    queryKey: queryKeys.driverLogs(
      task?.task_id,
      task?.state,
      namespace,
      [driverLogUri, driverPod?.name, runId].join(':'),
      sourceFinished,
    ),
    queryFn: () => getTaskDriverLogsInfo(task!, runId, namespace),
    enabled: !!task && hasDriverSource && selectedTab === 2 && driverExpanded && !namespacePending,
    refetchInterval: task && !sourceFinished && !isTaskFinished(task.state) ? 10000 : false,
  });
  const driverLogsState = hasDriverSource ? driverLogsInfo : DRIVER_LOGS_UNAVAILABLE;

  return (
    <div className='kfp-inspection-column'>
      <LifecycleBanner task={task} sourceFinished={sourceFinished} />
      <InspectionTabs
        tabs={['Input/Output', 'Task Details', 'Logs']}
        selectedTab={selectedTab}
        onSwitch={setSelectedTab}
        ariaLabel='Task inspection'
      >
        <div className='kfp-inspection-column'>
          {selectedTab === 0 &&
            (task ? (
              <RuntimeInputOutputTab
                task={task}
                namespace={namespace}
                namespacePending={namespacePending}
              />
            ) : (
              NODE_STATE_UNAVAILABLE
            ))}
          {selectedTab === 1 && (
            <div className='kfp-inspection-scroll'>
              <RuntimeTaskDetails element={element} task={task} />
              <TaskPodsDetails pods={task?.pods} />
              <TaskVolumeMountsDetails
                element={element}
                layers={layers}
                pipelineJobString={pipelineJobString}
              />
            </div>
          )}
          {selectedTab === 2 && (
            <div className='kfp-node-logs'>
              <section aria-label='Driver logs' className='kfp-node-logs-driver'>
                <div className='kfp-node-logs-heading'>
                  <button
                    type='button'
                    className='kfp-node-logs-toggle'
                    aria-expanded={driverExpanded}
                    aria-controls={driverContentId}
                    onClick={() =>
                      setDriverChoice({ taskId: task?.task_id, expanded: !driverExpanded })
                    }
                  >
                    {driverExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                    Driver
                  </button>
                  <CopyLogsButton
                    key={`driver-${task?.task_id}-${runId}-${namespace}-${driverLogUri}-${driverPod?.name}`}
                    label='Driver logs'
                    text={driverLogsState?.get(LOGS_DETAILS)}
                    disabled={namespacePending || !task}
                    load={() => getTaskDriverLogsInfo(task!, runId, namespace)}
                  />
                </div>
                {driverExpanded && (
                  <div id={driverContentId}>
                    <p className='kfp-node-logs-description'>
                      Pipeline orchestration logs from the KFP driver.
                    </p>
                    {namespacePending ? (
                      <p role='status'>Loading experiment namespace…</p>
                    ) : !driverLogsState ? (
                      <p role='status'>Loading driver logs…</p>
                    ) : (
                      <LogsTabPanel
                        label='Driver logs'
                        details={driverLogsState.get(LOGS_DETAILS)}
                        bannerMessage={driverLogsState.get(LOGS_BANNER_MESSAGE)}
                        bannerAdditionalInfo={driverLogsState.get(LOGS_BANNER_ADDITIONAL_INFO)}
                        bannerMode={
                          driverLogsState.has(LOGS_BANNER_ADDITIONAL_INFO) ? 'error' : 'info'
                        }
                        onRefresh={hasDriverSource ? () => void refetchDriverLogs() : undefined}
                      />
                    )}
                  </div>
                )}
              </section>
              {task &&
                driverHistory.map((source) => (
                  <HistoricalLogSection
                    key={`${task.task_id}-${source.key}`}
                    source={source}
                    task={task}
                    runId={runId}
                    namespace={namespace}
                    namespacePending={namespacePending}
                    driver
                  />
                ))}
              <section aria-label='Component logs' className='kfp-node-logs-task'>
                <div className='kfp-node-logs-heading'>
                  <button
                    type='button'
                    className='kfp-node-logs-toggle'
                    aria-expanded={taskExpanded}
                    aria-controls={taskContentId}
                    onClick={() =>
                      setTaskChoice({ taskId: task?.task_id, expanded: !taskExpanded })
                    }
                  >
                    {taskExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                    {componentTitle}
                  </button>
                  <CopyLogsButton
                    key={`task-${task?.task_id}-${runId}-${namespace}-${logsSourceIdentity}`}
                    label='Component logs'
                    text={logsDetails}
                    disabled={namespacePending || !task}
                    load={() => getLogsInfo(task!, runId, namespace, liveSource)}
                  />
                </div>
                {taskExpanded && (
                  <div id={taskContentId}>
                    {namespacePending ? (
                      <p role='status'>Loading experiment namespace…</p>
                    ) : !task ? (
                      <p role='status'>Component logs are not available yet.</p>
                    ) : !logsInfo && !logsQueryFailed ? (
                      <p role='status'>Loading component logs…</p>
                    ) : (
                      <LogsTabPanel
                        label='Component logs'
                        details={logsDetails}
                        bannerMessage={logsBannerMessage}
                        bannerAdditionalInfo={logsBannerAdditionalInfo}
                        bannerMode={logsBannerAdditionalInfo ? 'error' : 'info'}
                      />
                    )}
                  </div>
                )}
              </section>
              {task &&
                componentHistory.map((source) => (
                  <HistoricalLogSection
                    key={`${task.task_id}-${source.key}`}
                    source={source}
                    task={task}
                    runId={runId}
                    namespace={namespace}
                    namespacePending={namespacePending}
                    driver={false}
                  />
                ))}
            </div>
          )}
        </div>
      </InspectionTabs>
    </div>
  );
}

export function getTaskDetailsFields(
  element?: PipelineFlowElement | null,
  task?: V2PipelineTask,
): Array<KeyValue<string>> {
  if (!element) {
    return [];
  }
  const details: Array<KeyValue<string>> = [['Task ID', task?.task_id || element.id || '-']];
  if (!task) {
    return details;
  }
  details.push(['Task name', getTaskDisplayName(task, '-')]);
  details.push(['Task type', task.type || '-']);
  details.push(['Status', formatTaskState(task.state)]);
  details.push(['Created At', formatDateString(task.create_time)]);
  details.push(['Finished At', isTaskFinished(task.state) ? formatDateString(task.end_time) : '-']);
  if (task.parent_task_id) {
    details.push(['Parent task ID', task.parent_task_id]);
  }
  if (task.scope_path) {
    details.push(['Scope path', task.scope_path]);
  }
  if (task.cache_fingerprint) {
    details.push(['Cache fingerprint', task.cache_fingerprint]);
  }
  if (task.type_attributes && Object.keys(task.type_attributes).length) {
    details.push(['Type attributes', JSON.stringify(task.type_attributes)]);
  }
  // Successful retries can retain the failed attempt's status metadata. Errors remain
  // available alongside their original state and timestamp in state history.
  if (
    task.status_metadata?.message &&
    task.state !== PipelineTaskTaskState.SUCCEEDED &&
    task.state !== PipelineTaskTaskState.CACHED
  ) {
    details.push(['Message', task.status_metadata.message]);
  }
  const stateHistory = (task.state_history || [])
    .map((status) => {
      const value = `${formatTaskState(status.state)} · ${formatDateString(status.update_time)}`;
      return status.error?.message ? `${value} · ${status.error.message}` : value;
    })
    .join('\n');
  if (stateHistory) {
    details.push(['State history', stateHistory]);
  }
  return details;
}

function TaskPodsDetails({ pods = [] }: { pods?: PipelineTaskTaskPod[] }) {
  if (!pods.length) return null;
  // The API preserves pod order but exposes no attempt number. Keep that order
  // without inferring retry numbers from potentially missing driver/executor pods.
  return (
    <details className='kfp-inspection-pods'>
      <summary>Pods ({pods.length})</summary>
      {pods.map((pod, index) => (
        <DetailsTable
          key={pod.uid || `${pod.type}-${pod.name}-${index}`}
          title={`Pod ${index + 1} · ${pod.type || 'Unknown'}`}
          fields={[
            ['Name', pod.name || '-'],
            ['UID', pod.uid || '-'],
          ]}
        />
      ))}
    </details>
  );
}

function RuntimeTaskDetails({
  element,
  task,
}: {
  element?: PipelineFlowElement | null;
  task?: V2PipelineTask;
}) {
  return (
    <>
      <DetailsTable
        title='Task Details'
        fields={getTaskDetailsFields(element, task).filter(([name]) => name !== 'State history')}
      />
      {!!task?.state_history?.length && (
        <table className='kfp-inspection-history'>
          <caption>State history</caption>
          <thead>
            <tr>
              <th style={{ textAlign: 'left', padding: 8 }}>State</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Updated at</th>
            </tr>
          </thead>
          <tbody>
            {task.state_history.map((status, index) => (
              <tr key={index}>
                <td style={{ verticalAlign: 'top', padding: 8 }}>
                  {formatTaskState(status.state)}
                  {status.error?.message && (
                    <div style={{ overflowWrap: 'anywhere' }}>{status.error.message}</div>
                  )}
                </td>
                <td
                  style={{ verticalAlign: 'top', padding: 8, fontVariantNumeric: 'tabular-nums' }}
                >
                  {formatDateString(status.update_time)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );
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

interface TaskVolumeMountsDetailsProps {
  element?: PipelineFlowElement | null;
  layers: string[];
  pipelineJobString?: string;
}

function TaskVolumeMountsDetails({
  element,
  layers,
  pipelineJobString,
}: TaskVolumeMountsDetailsProps) {
  const fields = useMemo(
    () => getNodeVolumeMounts(layers, pipelineJobString, element),
    [element, layers, pipelineJobString],
  );
  return <DetailsTable title='Volume Mounts' fields={fields} />;
}

function CopyLogsButton({
  label,
  text,
  disabled,
  load,
}: {
  label: string;
  text?: string;
  disabled?: boolean;
  load: () => Promise<Map<string, string> | undefined>;
}) {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');
  const [copied, setCopied] = useState(false);
  const active = useRef(true);
  // External synchronization: invalidate pending clipboard work when its source is unmounted.
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  return (
    <>
      <button
        type='button'
        className='kfp-node-logs-copy'
        aria-label={`Copy ${label.toLowerCase()}`}
        aria-busy={busy}
        title={message || `Copy ${label.toLowerCase()}`}
        disabled={disabled || busy}
        onClick={async () => {
          setBusy(true);
          setCopied(false);
          setMessage(`Loading ${label.toLowerCase()}…`);
          let failureMessage = 'Could not copy. Select the log text and copy it manually.';
          const getText = async () => {
            let loaded: Map<string, string> | undefined;
            try {
              loaded = text === undefined ? await load() : undefined;
            } catch {
              failureMessage = `Could not load ${label.toLowerCase()}. Try copying again.`;
              throw new Error(failureMessage);
            }
            if (!active.current) throw new Error('Log source changed.');
            const output = text ?? loaded?.get(LOGS_DETAILS);
            if (!output) {
              failureMessage =
                loaded?.get(LOGS_BANNER_ADDITIONAL_INFO) ||
                loaded?.get(LOGS_BANNER_MESSAGE) ||
                `${label} are not available yet.`;
              throw new Error(failureMessage);
            }
            return output;
          };
          try {
            if (typeof ClipboardItem !== 'undefined' && navigator.clipboard?.write) {
              // Safari needs write() during the click, before the network request resolves.
              const content = getText().then(
                (output) => new Blob([output], { type: 'text/plain' }),
              );
              // A synchronous clipboard denial may leave the content promise unconsumed.
              void content.catch(() => {});
              await navigator.clipboard.write([new ClipboardItem({ 'text/plain': content })]);
            } else if (text) {
              await navigator.clipboard.writeText(text);
            } else {
              const output = await getText();
              if (!active.current) return;
              await navigator.clipboard.writeText(output);
            }
            if (active.current) {
              setCopied(true);
              setMessage(`${label} copied.`);
            }
          } catch {
            if (active.current) setMessage(failureMessage);
          } finally {
            if (active.current) setBusy(false);
          }
        }}
      >
        {copied ? <Check size={16} /> : <Copy size={16} />}
      </button>
      <span className='sr-only' role='status'>
        {message}
      </span>
    </>
  );
}

interface LogsTabPanelProps {
  label: string;
  details?: string;
  bannerMessage?: string;
  bannerAdditionalInfo?: string;
  bannerMode: 'info' | 'error';
  onRefresh?: () => void;
}

function LogsTabPanel({
  label,
  details,
  bannerMessage,
  bannerAdditionalInfo,
  bannerMode,
  onRefresh,
}: LogsTabPanelProps) {
  const logLines = details?.split(/[\r\n]+/) ?? [];
  // Reserve space for a horizontal scrollbar without stretching short log output.
  const contentHeight = logLines.length * LOG_ROW_HEIGHT + 16;
  return (
    <div className='kfp-inspection-column'>
      {bannerMessage && (
        <Banner message={bannerMessage} additionalInfo={bannerAdditionalInfo} mode={bannerMode} />
      )}
      {bannerMessage && onRefresh && <Button onClick={onRefresh}>Refresh</Button>}
      {details !== undefined && (
        <div
          className='kfp-inspection-log-window'
          data-testid={label === 'Driver logs' ? 'driver-logs-view-window' : 'logs-view-window'}
          style={{ height: contentHeight }}
        >
          <LogViewer label={label} logLines={logLines} />
        </div>
      )}
    </div>
  );
}

/** Location of the driver (system) log artifact recorded by the centralized driver, if any. */
export function getDriverLogUri(task?: V2PipelineTask): string | undefined {
  const driverLogUri = task?.status_metadata?.custom_properties?.driver_logs_uri;
  return typeof driverLogUri === 'string' && driverLogUri ? driverLogUri : undefined;
}

// Uses the same keys as getLogsInfo so both log sections render through LogsTabPanel.
export async function getDriverLogsInfo(
  driverLogUri: string,
  namespace?: string,
): Promise<Map<string, string>> {
  const logsInfo = new Map<string, string>();
  try {
    const details = await readArtifactFile({ uri: driverLogUri }, namespace);
    if (details.trim()) {
      logsInfo.set(LOGS_DETAILS, details);
    } else {
      logsInfo.set(LOGS_BANNER_MESSAGE, 'Driver logs are not available yet.');
    }
  } catch (error) {
    logsInfo.set(LOGS_BANNER_MESSAGE, 'Failed to retrieve driver logs.');
    logsInfo.set(LOGS_BANNER_ADDITIONAL_INFO, `Error response: ${await errorToMessage(error)}`);
  }
  return logsInfo;
}

/** Keep orchestration output out of the task's executor stream. */
export async function getTaskDriverLogsInfo(
  task: V2PipelineTask,
  runId?: string,
  namespace?: string,
  source?: TaskLogSource,
): Promise<Map<string, string>> {
  const uri = source
    ? source.artifact?.uri
    : getDriverLogUri(task) ||
      getTaskLogSources(task, PipelineTaskTaskPodType.DRIVER)[0]?.artifact?.uri;
  const artifactResult = uri ? await getDriverLogsInfo(uri, namespace) : undefined;
  if (artifactResult?.has(LOGS_DETAILS)) return artifactResult;
  const pod = source ? source.pod : getLatestTaskPod(task, PipelineTaskTaskPodType.DRIVER);
  if (!runId || !pod?.name) return artifactResult ?? new Map(DRIVER_LOGS_UNAVAILABLE);
  try {
    const createdAt =
      task.create_time && !Number.isNaN(task.create_time.getTime())
        ? task.create_time.toISOString().split('T')[0]
        : new Date().toISOString().split('T')[0];
    const details = await Apis.getPodLogs(runId, pod.name, namespace || '', createdAt);
    return details.trim()
      ? new Map([[LOGS_DETAILS, details]])
      : (artifactResult ?? new Map(DRIVER_LOGS_UNAVAILABLE));
  } catch (error) {
    return new Map([
      [LOGS_BANNER_MESSAGE, 'Failed to retrieve driver logs.'],
      [
        LOGS_BANNER_ADDITIONAL_INFO,
        [
          artifactResult?.get(LOGS_BANNER_ADDITIONAL_INFO),
          `Driver pod logs error: ${await errorToMessage(error)}`,
        ]
          .filter(Boolean)
          .join('\n'),
      ],
    ]);
  }
}

export async function getLogsInfo(
  task: V2PipelineTask,
  runId?: string,
  namespace?: string,
  source?: TaskLogSource,
): Promise<Map<string, string>> {
  const logsInfo = new Map<string, string>();
  if (!source && task.state === PipelineTaskTaskState.CACHED) {
    logsInfo.set(LOGS_DETAILS, 'This step output is taken from cache.');
    return logsInfo;
  }

  const executorPod = source
    ? source.pod
    : getLatestTaskPod(task, PipelineTaskTaskPodType.EXECUTOR);
  const taskCreatedAt = task.create_time;
  const createdAt =
    taskCreatedAt && !Number.isNaN(taskCreatedAt.getTime())
      ? taskCreatedAt.toISOString().split('T')[0]
      : new Date().toISOString().split('T')[0];
  let podLogsError: unknown;
  if (runId && executorPod?.name) {
    try {
      const details = await Apis.getPodLogs(runId, executorPod.name, namespace || '', createdAt);
      if (details.trim()) {
        logsInfo.set(LOGS_DETAILS, details);
        return logsInfo;
      }
    } catch (error) {
      podLogsError = error;
    }
  }

  const executorLogsArtifact = source
    ? source.artifact
    : getOutputArtifactByName(task, EXECUTOR_LOGS_ARTIFACT_KEY);
  let artifactLogsError: unknown;
  if (executorLogsArtifact?.uri) {
    try {
      const details = await readArtifactFile(executorLogsArtifact, namespace);
      if (details.trim()) {
        logsInfo.set(LOGS_DETAILS, details);
        return logsInfo;
      }
    } catch (error) {
      artifactLogsError = error;
    }
  }

  if (!podLogsError && !artifactLogsError) {
    logsInfo.set(LOGS_BANNER_MESSAGE, 'Component logs are not available yet.');
    return logsInfo;
  }
  const podErrorMessage = podLogsError
    ? await errorToMessage(podLogsError)
    : 'No executor pod logs available.';
  logsInfo.set(
    LOGS_BANNER_MESSAGE,
    artifactLogsError ? 'Failed to retrieve component logs.' : 'Failed to retrieve pod logs.',
  );
  const additionalInfo = [`Pod logs error: ${podErrorMessage}`];
  if (artifactLogsError) {
    additionalInfo.push(`Executor logs artifact error: ${await errorToMessage(artifactLogsError)}`);
  }
  logsInfo.set(
    LOGS_BANNER_ADDITIONAL_INFO,
    artifactLogsError ? additionalInfo.join('\n') : `Error response: ${podErrorMessage}`,
  );
  return logsInfo;
}

interface ArtifactNodeDetailProps {
  task?: V2PipelineTask;
  artifactGroup?: InputOutputsIOArtifact;
  namespace?: string;
  namespacePending?: boolean;
  sourceFinished?: boolean;
}

function ArtifactNodeDetail({
  task,
  artifactGroup,
  namespace,
  namespacePending = false,
  sourceFinished,
}: ArtifactNodeDetailProps) {
  const [selectedTab, setSelectedTab] = useState(0);
  const [hasOpenedVisualization, setHasOpenedVisualization] = useState(false);
  const artifacts = artifactGroup?.artifacts || [];
  if (!task || !artifactGroup || !artifacts.length) {
    return NODE_STATE_UNAVAILABLE;
  }
  return (
    <div className='kfp-inspection-column'>
      <InspectionTabs
        tabs={['Artifact Info', 'Visualization']}
        selectedTab={selectedTab}
        onSwitch={(tab) => {
          setSelectedTab(tab);
          if (tab === 1) {
            setHasOpenedVisualization(true);
          }
        }}
        ariaLabel='Artifact inspection'
      >
        <div className='kfp-inspection-scroll'>
          <div hidden={selectedTab !== 0}>
            <ArtifactInfo
              task={task}
              artifactGroup={artifactGroup}
              namespace={namespace}
              namespacePending={namespacePending}
            />
          </div>
          {(selectedTab === 1 || hasOpenedVisualization) && (
            <div hidden={selectedTab !== 1} className='kfp-inspection-legacy'>
              {namespacePending ? (
                <p role='status'>Loading experiment namespace…</p>
              ) : (
                <RuntimeMetricsVisualizations
                  active={selectedTab === 1}
                  artifacts={artifacts}
                  artifactKey={artifactGroup.artifact_key}
                  namespace={namespace}
                  sourceFinished={sourceFinished || isTaskFinished(task.state)}
                />
              )}
            </div>
          )}
        </div>
      </InspectionTabs>
    </div>
  );
}

function ArtifactInfo({
  task,
  artifactGroup,
  namespace,
  namespacePending,
}: Required<Pick<ArtifactNodeDetailProps, 'task' | 'artifactGroup'>> &
  Pick<ArtifactNodeDetailProps, 'namespace' | 'namespacePending'>) {
  const artifactEntries = flattenArtifactGroups([artifactGroup]);
  const uriRows = buildRuntimeArtifactRows([artifactGroup]);
  const firstArtifact = artifactEntries[0].artifact;
  const artifactInfo: Array<KeyValue<string>> = [
    ['Upstream Task Name', getTaskDisplayName(task, '-')],
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
        title='Artifact Value'
        fields={uriRows}
        valueComponent={RuntimeArtifactValue}
        valueComponentProps={{ namespace, namespacePending }}
      />
    </div>
  );
}

interface SubDAGNodeDetailProps {
  element: PipelineFlowElement;
  task?: V2PipelineTask;
  namespace?: string;
  namespacePending?: boolean;
  sourceFinished?: boolean;
}

function SubDAGNodeDetail({
  element,
  task,
  namespace,
  namespacePending = false,
  sourceFinished,
}: SubDAGNodeDetailProps) {
  const [selectedTab, setSelectedTab] = useState(0);
  return (
    <div className='kfp-inspection-column'>
      <LifecycleBanner task={task} sourceFinished={sourceFinished} />
      <InspectionTabs
        tabs={['Input/Output', 'Task Details']}
        selectedTab={selectedTab}
        onSwitch={setSelectedTab}
        ariaLabel='Sub-DAG inspection'
      >
        <div className='kfp-inspection-column'>
          {selectedTab === 0 &&
            (task ? (
              <RuntimeInputOutputTab
                task={task}
                namespace={namespace}
                namespacePending={namespacePending}
              />
            ) : (
              NODE_STATE_UNAVAILABLE
            ))}
          {selectedTab === 1 && (
            <div className='kfp-inspection-scroll'>
              <RuntimeTaskDetails element={element} task={task} />
              <TaskPodsDetails pods={task?.pods} />
            </div>
          )}
        </div>
      </InspectionTabs>
    </div>
  );
}

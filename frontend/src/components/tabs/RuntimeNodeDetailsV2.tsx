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

import { Button } from '@material-ui/core';
import * as React from 'react';
import { useState } from 'react';
import { FlowElement } from 'react-flow-renderer';
// import { ComponentSpec, PipelineSpec } from 'src/generated/pipeline_spec';
import {
  KubernetesExecutorConfig,
  PvcMount,
} from 'src/generated/platform_spec/kubernetes_platform';
import { useQuery } from 'react-query';
import MD2Tabs from 'src/atoms/MD2Tabs';
import { commonCss, padding } from 'src/Css';
import { Apis } from 'src/lib/Apis';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { errorToMessage } from 'src/lib/Utils';
import { getTaskKeyFromNodeKey, NodeTypeNames } from 'src/lib/v2/StaticFlow';
import {
  EXECUTION_KEY_CACHED_EXECUTION_ID,
  getArtifactTypeName,
  getArtifactTypes,
  KfpExecutionProperties,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { NodeMlmdInfo } from 'src/pages/RunDetailsV2';
import { ArtifactType, Execution } from 'src/third_party/mlmd';
import ArtifactPreview from 'src/components/ArtifactPreview';
import Banner from 'src/components/Banner';
import DetailsTable from 'src/components/DetailsTable';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import LogViewer from 'src/components/LogViewer';
import { getResourceStateText, ResourceType } from 'src/components/ResourceInfo';
import { MetricsVisualizations } from 'src/components/viewers/MetricsVisualizations';
import { ArtifactTitle } from 'src/components/tabs/ArtifactTitle';
import InputOutputTab, {
  getArtifactParamList,
  ParamList,
} from 'src/components/tabs/InputOutputTab';
import { convertYamlToPlatformSpec, convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import { PlatformDeploymentConfig } from 'src/generated/pipeline_spec/pipeline_spec';
import { getComponentSpec } from 'src/lib/v2/NodeUtils';

export const LOGS_DETAILS = 'logs_details';
export const LOGS_BANNER_MESSAGE = 'logs_banner_message';
export const LOGS_BANNER_ADDITIONAL_INFO = 'logs_banner_additional_info';
export const K8S_PLATFORM_KEY = 'kubernetes';

// Pod status/events related interfaces
interface PodStatusInfo {
  phase?: string;
  message?: string;
  reason?: string;
  podIP?: string;
  nodeName?: string;
  containerStatuses?: ContainerStatusInfo[];
  cached?: boolean;
  cachedAt?: number;
}

interface ContainerStatusInfo {
  name: string;
  ready: boolean;
  restartCount: number;
  state?: string;
  reason?: string;
  message?: string;
  exitCode?: number;
}

interface PodEventInfo {
  type: string;
  reason: string;
  message: string;
  firstTimestamp?: string;
  lastTimestamp?: string;
  count: number;
  source?: string;
  cached?: boolean;
}

interface PodStateSnapshot {
  timestamp: number;
  phase?: string;
  reason?: string;
  message?: string;
  containerStatuses?: ContainerStatusInfo[];
  isError: boolean;
}

interface PodStatusResult {
  status: PodStatusInfo | null;
  events: PodEventInfo[];
  stateHistory: PodStateSnapshot[];
  cached?: boolean;
  cachedAt?: number;
}

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
  element?: FlowElement<FlowElementDataBase> | null;
  elementMlmdInfo?: NodeMlmdInfo | null;
  namespace: string | undefined;
}

export function RuntimeNodeDetailsV2({
  layers,
  onLayerChange,
  pipelineJobString,
  runId,
  element,
  elementMlmdInfo,
  namespace,
}: RuntimeNodeDetailsV2Props) {
  if (!element) {
    return NODE_INFO_UNKNOWN;
  }

  return (() => {
    if (NodeTypeNames.EXECUTION === element.type) {
      return (
        <TaskNodeDetail
          pipelineJobString={pipelineJobString}
          runId={runId}
          element={element}
          execution={elementMlmdInfo?.execution}
          layers={layers}
          namespace={namespace}
        ></TaskNodeDetail>
      );
    } else if (NodeTypeNames.ARTIFACT === element.type) {
      return (
        <ArtifactNodeDetail
          execution={elementMlmdInfo?.execution}
          linkedArtifact={elementMlmdInfo?.linkedArtifact}
          namespace={namespace}
        />
      );
    } else if (NodeTypeNames.SUB_DAG === element.type) {
      return (
        <SubDAGNodeDetail
          element={element}
          execution={elementMlmdInfo?.execution}
          layers={layers}
          onLayerChange={onLayerChange}
          namespace={namespace}
        />
      );
    }
    return NODE_INFO_UNKNOWN;
  })();
}

interface TaskNodeDetailProps {
  pipelineJobString?: string;
  runId?: string;
  element?: FlowElement<FlowElementDataBase> | null;
  execution?: Execution;
  layers: string[];
  namespace: string | undefined;
}

function TaskNodeDetail({
  pipelineJobString,
  runId,
  element,
  execution,
  layers,
  namespace,
}: TaskNodeDetailProps) {
  const { data: logsInfo } = useQuery<Map<string, string>, Error>(
    [execution],
    async () => {
      if (!execution) {
        throw new Error('No execution is found.');
      }
      return await getLogsInfo(execution, runId);
    },
    { enabled: !!execution },
  );

  // Fetch pod status and events
  // Note: We enable this query even without execution to catch early failures like ImagePullBackOff
  // where the pod exists but MLMD execution hasn't been created yet
  console.log('[TaskNodeDetail] Pod status query config:', {
    runId,
    namespace,
    elementId: element?.id,
    executionId: execution?.getId(),
    enabled: !!(runId && namespace),
  });

  const { data: podStatusInfo, error: podStatusError } = useQuery<
    PodStatusResult,
    Error
  >(
    ['podStatus', execution?.getId(), runId, element?.id, namespace],
    async () => {
      console.log('[TaskNodeDetail] Fetching pod status...');
      return await getPodStatusAndEvents(execution, namespace, runId, element);
    },
    { enabled: !!(runId && namespace), refetchInterval: 5000 }, // Refresh every 5 seconds for running pods
  );

  const logsDetails = logsInfo?.get(LOGS_DETAILS);
  const logsBannerMessage = logsInfo?.get(LOGS_BANNER_MESSAGE);
  const logsBannerAdditionalInfo = logsInfo?.get(LOGS_BANNER_ADDITIONAL_INFO);

  const [selectedTab, setSelectedTab] = useState(0);

  return (
    <div className={commonCss.page}>
      <MD2Tabs
        tabs={['Input/Output', 'Task Details', 'Logs', 'Pod Status']}
        selectedTab={selectedTab}
        onSwitch={tab => setSelectedTab(tab)}
      />
      <div className={commonCss.page}>
        {/* Input/Output tab */}
        {selectedTab === 0 &&
          (() => {
            if (execution) {
              return <InputOutputTab execution={execution} namespace={namespace} />;
            }
            return NODE_STATE_UNAVAILABLE;
          })()}

        {/* Task Details tab */}
        {selectedTab === 1 && (
          <div className={padding(20)}>
            <DetailsTable title='Task Details' fields={getTaskDetailsFields(element, execution)} />
            <DetailsTable
              title='Volume Mounts'
              fields={getNodeVolumeMounts(layers, pipelineJobString, element)}
            />
          </div>
        )}
        {/* Logs tab */}
        {selectedTab === 2 && (
          <div className={commonCss.page}>
            {logsBannerMessage && (
              <React.Fragment>
                <Banner message={logsBannerMessage} additionalInfo={logsBannerAdditionalInfo} />
              </React.Fragment>
            )}
            {!logsBannerMessage && (
              <div className={commonCss.pageOverflowHidden} data-testid={'logs-view-window'}>
                <LogViewer logLines={(logsDetails || '').split(/[\r\n]+/)} />
              </div>
            )}
          </div>
        )}
        {/* Pod Status tab */}
        {selectedTab === 3 && (
          <div className={padding(20)}>
            <PodStatusTab
              podStatus={podStatusInfo?.status || null}
              podEvents={podStatusInfo?.events || []}
              stateHistory={podStatusInfo?.stateHistory || []}
              error={podStatusError}
              cached={podStatusInfo?.cached}
              cachedAt={podStatusInfo?.cachedAt}
            />
          </div>
        )}
      </div>
    </div>
  );
}

function getTaskDetailsFields(
  element?: FlowElement<FlowElementDataBase> | null,
  execution?: Execution,
): Array<KeyValue<string>> {
  const details: Array<KeyValue<string>> = [];
  if (element) {
    details.push(['Task ID', element.id || '-']);
    if (execution) {
      // Static execution info.
      details.push([
        'Task name',
        execution
          .getCustomPropertiesMap()
          .get('display_name')
          ?.getStringValue() || '-',
      ]);

      // Runtime execution info.
      const stateText = getResourceStateText({
        resourceType: ResourceType.EXECUTION,
        resource: execution,
        typeName: 'Execution',
      });
      details.push(['Status', stateText || '-']);

      const createdAt = new Date(execution.getCreateTimeSinceEpoch()).toString();
      details.push(['Created At', createdAt]);

      const lastUpdatedTime = execution.getLastUpdateTimeSinceEpoch();
      let finishedAt = '-';
      if (
        lastUpdatedTime &&
        (execution.getLastKnownState() === Execution.State.COMPLETE ||
          execution.getLastKnownState() === Execution.State.FAILED ||
          execution.getLastKnownState() === Execution.State.CACHED ||
          execution.getLastKnownState() === Execution.State.CANCELED)
      ) {
        finishedAt = new Date(lastUpdatedTime).toString();
      }
      details.push(['Finished At', finishedAt]);
    }
  }

  return details;
}

function getNodeVolumeMounts(
  layers: string[],
  pipelineJobString?: string,
  element?: FlowElement<FlowElementDataBase> | null,
): Array<KeyValue<string>> {
  if (!pipelineJobString || !element) {
    return [];
  }

  const taskKey = getTaskKeyFromNodeKey(element.id);
  const pipelineSpec = convertYamlToV2PipelineSpec(pipelineJobString);
  const componentSpec = getComponentSpec(pipelineSpec, layers, taskKey);
  const platformSpec = convertYamlToPlatformSpec(pipelineJobString);

  // Currently support kubernetes platform
  if (!platformSpec || !platformSpec.platforms[K8S_PLATFORM_KEY]) {
    return [];
  }

  const k8sDeploymentSpec = PlatformDeploymentConfig.fromJSON(
    platformSpec.platforms[K8S_PLATFORM_KEY].deploymentSpec,
  );
  const matchedExecutorObj = Object.entries(k8sDeploymentSpec.executors).find(
    ([executorName]) => executorName === componentSpec?.executorLabel,
  );

  let volumeMounts: Array<KeyValue<string>> = [];
  if (matchedExecutorObj) {
    const executor = KubernetesExecutorConfig.fromJSON(matchedExecutorObj[1]);
    const pvcMounts = Object.values(executor.pvcMount).map(pvcm => PvcMount.fromJSON(pvcm));
    volumeMounts = pvcMounts.map(pvcm => [pvcm.mountPath, pvcm.taskOutputParameter?.producerTask]);
  }

  return volumeMounts;
}

// Error reasons that indicate a failed/problematic state
const ERROR_REASONS = [
  'ErrImagePull',
  'ImagePullBackOff',
  'CrashLoopBackOff',
  'CreateContainerConfigError',
  'InvalidImageName',
  'CreateContainerError',
  'RunContainerError',
  'PreCreateHookError',
  'PostStartHookError',
  'ContainerCannotRun',
  'OOMKilled',
  'Error',
];

// Derives the effective pod state based on container statuses
// This is important because pod phase might be "Pending" while containers are in error state
function deriveEffectiveState(
  phase?: string,
  containerStatuses?: ContainerStatusInfo[],
  initContainerStatuses?: ContainerStatusInfo[],
): { effectivePhase: string; errorReason?: string; errorMessage?: string } {
  // Check init containers first (they run before regular containers)
  if (initContainerStatuses && initContainerStatuses.length > 0) {
    for (const cs of initContainerStatuses) {
      if (cs.reason && ERROR_REASONS.includes(cs.reason)) {
        return {
          effectivePhase: 'Failed',
          errorReason: cs.reason,
          errorMessage: cs.message,
        };
      }
      // Check for terminated with non-zero exit code
      if (cs.state === 'terminated' && cs.exitCode !== undefined && cs.exitCode !== 0) {
        return {
          effectivePhase: 'Failed',
          errorReason: cs.reason || `Exit code ${cs.exitCode}`,
          errorMessage: cs.message,
        };
      }
    }
  }

  // Check regular containers
  if (containerStatuses && containerStatuses.length > 0) {
    for (const cs of containerStatuses) {
      if (cs.reason && ERROR_REASONS.includes(cs.reason)) {
        return {
          effectivePhase: 'Failed',
          errorReason: cs.reason,
          errorMessage: cs.message,
        };
      }
      // Check for terminated with non-zero exit code
      if (cs.state === 'terminated' && cs.exitCode !== undefined && cs.exitCode !== 0) {
        return {
          effectivePhase: 'Failed',
          errorReason: cs.reason || `Exit code ${cs.exitCode}`,
          errorMessage: cs.message,
        };
      }
    }
  }

  return { effectivePhase: phase || 'Unknown' };
}

// Fetch pod status and events from Kubernetes API
// Note: execution is optional to support early failures where MLMD execution doesn't exist yet
async function getPodStatusAndEvents(
  execution: Execution | undefined,
  namespace?: string,
  runId?: string,
  element?: FlowElement<FlowElementDataBase> | null,
): Promise<PodStatusResult> {
  console.log('[getPodStatusAndEvents] Called with:', { namespace, runId, elementId: element?.id, hasExecution: !!execution });

  let podName = '';
  let podNameSpace = namespace || '';

  // Try to get pod info from MLMD execution custom properties
  if (execution) {
    const customPropertiesMap = execution.getCustomPropertiesMap();
    podName = customPropertiesMap.get(KfpExecutionProperties.POD_NAME)?.getStringValue() || '';
    podNameSpace = customPropertiesMap.get('namespace')?.getStringValue() || namespace || '';
    console.log('[getPodStatusAndEvents] From MLMD execution:', { podName, podNameSpace });
  }

  // If pod name is not in MLMD, try to find it by run ID label
  // This handles early failures like ImagePullBackOff where the driver never ran
  if (!podName && runId && podNameSpace) {
    console.log('[getPodStatusAndEvents] No pod name from MLMD, trying to find by run ID label...');
    try {
      const taskName = element?.id ? getTaskKeyFromNodeKey(element.id) : undefined;
      console.log('[getPodStatusAndEvents] Calling API getPodsByRunId:', { runId, podNameSpace, taskName });
      const pods = await Apis.getPodsByRunId(runId, podNameSpace, taskName);
      console.log('[getPodStatusAndEvents] API response pods:', pods?.length, 'pods found');
      if (pods && pods.length > 0) {
        // Helper to check if a pod has container errors
        const hasContainerError = (pod: any): boolean => {
          const containerStatuses = pod.status?.containerStatuses || [];
          const initContainerStatuses = pod.status?.initContainerStatuses || [];
          const allStatuses = [...containerStatuses, ...initContainerStatuses];
          return allStatuses.some((cs: any) => {
            const waitingReason = cs.state?.waiting?.reason;
            const terminatedReason = cs.state?.terminated?.reason;
            return (
              ERROR_REASONS.includes(waitingReason) ||
              ERROR_REASONS.includes(terminatedReason) ||
              (cs.state?.terminated?.exitCode !== undefined && cs.state?.terminated?.exitCode !== 0)
            );
          });
        };

        // Helper to check if pod is an impl/launcher pod (runs user container)
        const isImplPod = (pod: any): boolean => {
          const name = pod.metadata?.name || '';
          return name.includes('-impl-') || name.includes('-launcher-') || name.includes('-user-');
        };

        // Helper to check if pod is a driver pod
        const isDriverPod = (pod: any): boolean => {
          const name = pod.metadata?.name || '';
          return name.includes('-driver-') || name.includes('-dag-driver-');
        };

        // Score pods to prioritize: error state > impl pods > non-driver pods
        const scorePod = (pod: any): number => {
          let score = 0;
          if (hasContainerError(pod)) score += 100; // Highest priority: pods with errors
          if (isImplPod(pod)) score += 50; // Prefer impl pods
          if (!isDriverPod(pod)) score += 10; // De-prioritize driver pods
          // Prefer pods that are not in Succeeded phase
          if (pod.status?.phase !== 'Succeeded') score += 25;
          return score;
        };

        // Sort pods by score (highest first)
        const sortedPods = [...pods].sort((a: any, b: any) => scorePod(b) - scorePod(a));
        console.log('[getPodStatusAndEvents] Pod scores:', sortedPods.map((p: any) => ({
          name: p.metadata?.name,
          phase: p.status?.phase,
          score: scorePod(p),
          hasError: hasContainerError(p),
          isImpl: isImplPod(p),
        })));

        let matchingPod = sortedPods[0]; // Default to highest scored pod

        // If task name provided, try to find a better match within top-scored pods
        if (taskName && sortedPods.length > 1) {
          const found = sortedPods.find((p: any) => {
            const labels = p.metadata?.labels || {};
            const podTaskLabel = labels['pipelines.kubeflow.org/task_name'] || '';
            const pName = p.metadata?.name || '';
            return (
              // Exact match (backward compatible)
              podTaskLabel === taskName ||
              // Path-based match: label ends with ".taskName" (e.g., "root.for-loop-1.process-item")
              podTaskLabel.endsWith('.' + taskName) ||
              // Fallback to component label
              labels['component'] === taskName ||
              // Fallback to pod name pattern
              pName.includes(taskName.replace(/_/g, '-')) ||
              pName.includes(taskName)
            );
          });
          if (found) {
            matchingPod = found;
          }
        }
        podName = (matchingPod as any).metadata?.name || '';
        console.log('[getPodStatusAndEvents] Selected pod:', podName);
      }
    } catch (err) {
      // Failed to get pods - this might be expected if no pods exist yet
    }
  }

  if (!podName || !podNameSpace) {
    return { status: null, events: [], stateHistory: [], cached: false };
  }

  // Resolve taskName for cache association
  const resolvedTaskName = element?.id ? getTaskKeyFromNodeKey(element.id) : undefined;

  let status: PodStatusInfo | null = null;
  let events: PodEventInfo[] = [];
  let stateHistory: PodStateSnapshot[] = [];
  let cached = false;
  let cachedAt: number | undefined;

  // Helper function to parse container status
  const parseContainerStatus = (cs: any): ContainerStatusInfo => ({
    name: cs.name,
    ready: cs.ready,
    restartCount: cs.restartCount,
    state: cs.state ? Object.keys(cs.state)[0] : undefined,
    reason: cs.state?.waiting?.reason || cs.state?.terminated?.reason,
    message: cs.state?.waiting?.message || cs.state?.terminated?.message,
    exitCode: cs.state?.terminated?.exitCode,
  });

  // Try to get pod info
  try {
    const podInfo = await Apis.getPodInfo(podName, podNameSpace, runId, resolvedTaskName);
    if (podInfo && podInfo.status) {
      const podStatus = podInfo.status as any;

      // Check if this is cached data
      if ((podInfo as any)._cached) {
        cached = true;
        cachedAt = (podInfo as any)._cachedAt;
      }

      // Capture state history from server
      if ((podInfo as any)._stateHistory) {
        stateHistory = (podInfo as any)._stateHistory;
      }

      // Parse both regular and init container statuses
      const containerStatuses = (podStatus.containerStatuses || []).map(parseContainerStatus);
      const initContainerStatuses = (podStatus.initContainerStatuses || []).map(parseContainerStatus);

      // Derive effective state based on container statuses
      const { effectivePhase, errorReason, errorMessage } = deriveEffectiveState(
        podStatus.phase,
        containerStatuses,
        initContainerStatuses,
      );

      status = {
        phase: effectivePhase,
        message: errorMessage || podStatus.message,
        reason: errorReason || podStatus.reason,
        podIP: podStatus.podIP,
        nodeName: (podInfo.spec as any)?.nodeName,
        containerStatuses: [...initContainerStatuses, ...containerStatuses],
        cached,
        cachedAt,
      };
    }
  } catch (err) {
    // Pod may not exist anymore, that's okay - we'll show events if available
    console.log('Failed to get pod info:', err);
  }

  // Try to get pod events
  try {
    const eventList = await Apis.getPodEvents(podName, podNameSpace, runId, resolvedTaskName);
    if (eventList && eventList.items) {
      // Check if events are cached
      if ((eventList as any)._cached) {
        cached = true;
        cachedAt = cachedAt || (eventList as any)._cachedAt;
      }

      // Capture state history from events response (if not already from pod info)
      if ((eventList as any)._stateHistory && stateHistory.length === 0) {
        stateHistory = (eventList as any)._stateHistory;
      }

      events = (eventList.items as any[]).map((event: any) => ({
        type: event.type,
        reason: event.reason,
        message: event.message,
        firstTimestamp: event.firstTimestamp,
        lastTimestamp: event.lastTimestamp,
        count: event.count || 1,
        source: event.source?.component,
        cached: (event as any)._cached,
      }));
      // Sort by lastTimestamp descending (most recent first)
      events.sort((a, b) => {
        const timeA = a.lastTimestamp ? new Date(a.lastTimestamp).getTime() : 0;
        const timeB = b.lastTimestamp ? new Date(b.lastTimestamp).getTime() : 0;
        return timeB - timeA;
      });
    }
  } catch (err) {
    // Events may not be available, that's okay
    console.log('Failed to get pod events:', err);
  }

  return { status, events, stateHistory, cached, cachedAt };
}

// Component to display pod status, lifecycle history, and events
interface PodStatusTabProps {
  podStatus: PodStatusInfo | null;
  podEvents: PodEventInfo[];
  stateHistory: PodStateSnapshot[];
  error?: Error | null;
  cached?: boolean;
  cachedAt?: number;
}

// Color helpers
const getPhaseColor = (phase?: string) => {
  switch (phase) {
    case 'Running': return '#4caf50';
    case 'Succeeded': return '#2196f3';
    case 'Failed': return '#f44336';
    case 'Pending': return '#ff9800';
    default: return '#9e9e9e';
  }
};

const getPhaseBackgroundColor = (phase?: string) => {
  switch (phase) {
    case 'Failed': return '#ffebee';
    case 'Running': return '#e8f5e9';
    case 'Succeeded': return '#e3f2fd';
    case 'Pending': return '#fff3e0';
    default: return '#fafafa';
  }
};

const getContainerStatusColor = (cs: ContainerStatusInfo) => {
  if (cs.reason && ERROR_REASONS.includes(cs.reason)) return '#f44336';
  if (cs.state === 'terminated' && cs.exitCode !== undefined && cs.exitCode !== 0) return '#f44336';
  if (cs.state === 'terminated' && cs.exitCode === 0) return '#2196f3';
  if (cs.state === 'terminated') return '#e57373';
  if (cs.state === 'running' && cs.ready) return '#4caf50';
  if (cs.state === 'waiting') return '#ff9800';
  return '#9e9e9e';
};

const getContainerBackgroundColor = (cs: ContainerStatusInfo) => {
  if (cs.reason && ERROR_REASONS.includes(cs.reason)) return '#ffebee';
  if (cs.state === 'terminated' && cs.exitCode !== undefined && cs.exitCode !== 0) return '#ffebee';
  return '#fff';
};

function PodStatusTab({ podStatus, podEvents, stateHistory, error, cached, cachedAt }: PodStatusTabProps) {
  if (error) {
    return (
      <Banner
        message='Failed to retrieve pod status'
        additionalInfo={error.message}
        mode='error'
      />
    );
  }

  const formatTime = (timestamp?: number) => {
    if (!timestamp) return '';
    return new Date(timestamp).toLocaleString();
  };

  // Summarize a state snapshot into a short label
  const getStateLabel = (snapshot: PodStateSnapshot): string => {
    const phase = snapshot.phase || 'Unknown';
    if (snapshot.reason) {
      return `${phase} (${snapshot.reason})`;
    }
    return phase;
  };

  // Get container summary for a snapshot (only main container, filter out Argo-internal init/wait)
  const getContainerSummary = (snapshot: PodStateSnapshot): string[] => {
    if (!snapshot.containerStatuses) return [];
    return snapshot.containerStatuses
      .filter(cs => cs.name === 'main')
      .map(cs => {
        const parts: string[] = [];
        if (cs.state) parts.push(cs.state);
        if (cs.reason) parts.push(`(${cs.reason})`);
        if (cs.exitCode !== undefined) {
          parts.push(cs.exitCode === 0 ? 'exit:0' : `exit:${cs.exitCode}`);
        }
        return parts.join(' ');
      });
  };

  return (
    <div>
      {/* Cached Data Indicator */}
      {cached && (
        <div
          style={{
            backgroundColor: '#e3f2fd',
            border: '1px solid #90caf9',
            borderRadius: '4px',
            padding: '12px 16px',
            marginBottom: '16px',
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <div>
            <strong style={{ color: '#1565c0' }}>Cached Data</strong>
            <div style={{ fontSize: '12px', color: '#666', marginTop: '2px' }}>
              Pod status and events cached from a previous query.
              {cachedAt && ` Last updated: ${formatTime(cachedAt)}`}
            </div>
          </div>
        </div>
      )}

      {/* Current Pod Status */}
      <div style={{ marginBottom: '24px' }}>
        <h3 style={{ marginBottom: '12px', fontSize: '16px', fontWeight: 500 }}>Current Status</h3>
        {podStatus ? (
          <div
            style={{
              border: `1px solid ${podStatus.phase === 'Failed' ? '#f44336' : '#e0e0e0'}`,
              borderRadius: '4px',
              padding: '16px',
              backgroundColor: getPhaseBackgroundColor(podStatus.phase),
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: '12px' }}>
              <span
                style={{
                  display: 'inline-block',
                  width: '12px',
                  height: '12px',
                  borderRadius: '50%',
                  backgroundColor: getPhaseColor(podStatus.phase),
                  marginRight: '8px',
                }}
              />
              <strong style={{ fontSize: '14px', color: podStatus.phase === 'Failed' ? '#c62828' : 'inherit' }}>
                Phase: {podStatus.phase || 'Unknown'}
              </strong>
            </div>

            {podStatus.phase === 'Failed' && podStatus.reason && (
              <div
                style={{
                  backgroundColor: '#f44336',
                  color: '#fff',
                  padding: '12px',
                  borderRadius: '4px',
                  marginBottom: '12px',
                }}
              >
                <strong>Error: {podStatus.reason}</strong>
                {podStatus.message && (
                  <div style={{ marginTop: '4px', fontSize: '13px' }}>{podStatus.message}</div>
                )}
              </div>
            )}

            {podStatus.reason && podStatus.phase !== 'Failed' && (
              <div style={{ marginBottom: '8px' }}><strong>Reason:</strong> {podStatus.reason}</div>
            )}
            {podStatus.message && podStatus.phase !== 'Failed' && (
              <div style={{ marginBottom: '8px' }}><strong>Message:</strong> {podStatus.message}</div>
            )}
            {podStatus.nodeName && (
              <div style={{ marginBottom: '8px' }}><strong>Node:</strong> {podStatus.nodeName}</div>
            )}
            {podStatus.podIP && (
              <div style={{ marginBottom: '8px' }}><strong>Pod IP:</strong> {podStatus.podIP}</div>
            )}

            {/* Container Status - show only main container (filter out Argo-internal init/wait) */}
            {podStatus.containerStatuses && podStatus.containerStatuses.filter(cs => cs.name === 'main').length > 0 && (
              <div style={{ marginTop: '16px' }}>
                <strong>Container:</strong>
                {podStatus.containerStatuses.filter(cs => cs.name === 'main').map((cs, idx) => (
                  <div
                    key={idx}
                    style={{
                      marginTop: '8px',
                      padding: '12px',
                      backgroundColor: getContainerBackgroundColor(cs),
                      border: `1px solid ${cs.reason && ERROR_REASONS.includes(cs.reason) ? '#f44336' : '#e0e0e0'}`,
                      borderRadius: '4px',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center' }}>
                      <span
                        style={{
                          display: 'inline-block',
                          width: '8px',
                          height: '8px',
                          borderRadius: '50%',
                          backgroundColor: getContainerStatusColor(cs),
                          marginRight: '8px',
                        }}
                      />
                      <strong>{cs.name}</strong>
                      <span style={{ marginLeft: '8px', color: '#666' }}>{cs.state || 'unknown'}</span>
                      <span
                        style={{
                          marginLeft: '4px',
                          color: cs.state === 'terminated'
                            ? (cs.exitCode === 0 ? '#2196f3' : '#c62828')
                            : '#666',
                          fontWeight: cs.state === 'terminated' && cs.exitCode !== 0 ? 500 : 'normal',
                        }}
                      >
                        {cs.state === 'terminated'
                          ? (cs.exitCode === 0 ? '(Completed)' : '(Failed)')
                          : (cs.ready ? '(Ready)' : '(Not Ready)')}
                      </span>
                    </div>
                    {cs.restartCount > 0 && (
                      <div style={{ color: '#ff9800', marginTop: '4px' }}>Restarts: {cs.restartCount}</div>
                    )}
                    {cs.reason && (
                      <div style={{
                        marginTop: '4px',
                        color: ERROR_REASONS.includes(cs.reason) ? '#c62828' : '#333',
                        fontWeight: ERROR_REASONS.includes(cs.reason) ? 500 : 'normal',
                      }}>
                        Reason: {cs.reason}
                      </div>
                    )}
                    {cs.message && (
                      <div style={{ color: '#666', fontSize: '12px', marginTop: '4px', wordBreak: 'break-word' }}>
                        {cs.message}
                      </div>
                    )}
                    {cs.exitCode !== undefined && (
                      <div style={{
                        color: cs.exitCode !== 0 ? '#f44336' : '#4caf50',
                        marginTop: '4px',
                        fontWeight: cs.exitCode !== 0 ? 500 : 'normal',
                      }}>
                        Exit Code: {cs.exitCode}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        ) : (
          <div style={{ color: '#666', fontStyle: 'italic' }}>
            Pod status not available. The pod may have been deleted.
          </div>
        )}
      </div>

      {/* Pod Lifecycle Timeline */}
      {stateHistory.length > 0 && (
        <div style={{ marginBottom: '24px' }}>
          <h3 style={{ marginBottom: '12px', fontSize: '16px', fontWeight: 500 }}>
            Pod Lifecycle ({stateHistory.length} state changes)
          </h3>
          <div style={{ position: 'relative', paddingLeft: '24px' }}>
            {/* Vertical timeline line */}
            <div
              style={{
                position: 'absolute',
                left: '7px',
                top: '4px',
                bottom: '4px',
                width: '2px',
                backgroundColor: '#e0e0e0',
              }}
            />
            {stateHistory.map((snapshot, idx) => {
              const isLast = idx === stateHistory.length - 1;
              const containerSummary = getContainerSummary(snapshot);
              return (
                <div key={idx} style={{ position: 'relative', marginBottom: isLast ? 0 : '16px' }}>
                  {/* Timeline dot */}
                  <div
                    style={{
                      position: 'absolute',
                      left: '-20px',
                      top: '3px',
                      width: '12px',
                      height: '12px',
                      borderRadius: '50%',
                      backgroundColor: snapshot.isError ? '#f44336' : getPhaseColor(snapshot.phase),
                      border: '2px solid #fff',
                      boxShadow: '0 0 0 1px ' + (snapshot.isError ? '#f44336' : getPhaseColor(snapshot.phase)),
                    }}
                  />
                  <div
                    style={{
                      padding: '8px 12px',
                      borderRadius: '4px',
                      backgroundColor: snapshot.isError ? '#ffebee' : (isLast ? getPhaseBackgroundColor(snapshot.phase) : '#fafafa'),
                      border: `1px solid ${snapshot.isError ? '#ef9a9a' : '#e0e0e0'}`,
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <span style={{
                        fontWeight: 500,
                        color: snapshot.isError ? '#c62828' : '#333',
                        fontSize: '13px',
                      }}>
                        {getStateLabel(snapshot)}
                      </span>
                      <span style={{ color: '#999', fontSize: '11px' }}>
                        {formatTime(snapshot.timestamp)}
                      </span>
                    </div>
                    {snapshot.message && (
                      <div style={{ color: '#666', fontSize: '12px', marginTop: '4px', wordBreak: 'break-word' }}>
                        {snapshot.message}
                      </div>
                    )}
                    {containerSummary.length > 0 && (
                      <div style={{ marginTop: '4px' }}>
                        {containerSummary.map((cs, csIdx) => (
                          <div key={csIdx} style={{ color: '#666', fontSize: '11px' }}>
                            {cs}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Pod Events */}
      <div>
        <h3 style={{ marginBottom: '12px', fontSize: '16px', fontWeight: 500 }}>
          Pod Events {podEvents.length > 0 && `(${podEvents.length})`}
        </h3>
        {podEvents.length > 0 ? (
          <div style={{ border: '1px solid #e0e0e0', borderRadius: '4px', overflow: 'hidden' }}>
            {podEvents.map((event, idx) => (
              <div
                key={idx}
                style={{
                  padding: '12px 16px',
                  borderBottom: idx < podEvents.length - 1 ? '1px solid #e0e0e0' : 'none',
                  backgroundColor: event.type === 'Warning' ? '#fff3e0' : '#fff',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}>
                  <span>
                    <span
                      style={{
                        color: event.type === 'Warning' ? '#ff9800' : '#4caf50',
                        fontWeight: 500,
                        marginRight: '8px',
                      }}
                    >
                      [{event.type}]
                    </span>
                    <strong>{event.reason}</strong>
                    {event.count > 1 && (
                      <span style={{ color: '#666', marginLeft: '8px' }}>(x{event.count})</span>
                    )}
                  </span>
                  <span style={{ color: '#666', fontSize: '12px' }}>
                    {event.lastTimestamp ? new Date(event.lastTimestamp).toLocaleString() : '-'}
                  </span>
                </div>
                <div style={{ color: '#333', fontSize: '13px' }}>{event.message}</div>
                {event.source && (
                  <div style={{ color: '#999', fontSize: '11px', marginTop: '4px' }}>
                    Source: {event.source}
                  </div>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div style={{ color: '#666', fontStyle: 'italic' }}>
            No events available. Events may have expired or the pod may have been deleted.
          </div>
        )}
      </div>
    </div>
  );
}

async function getLogsInfo(execution: Execution, runId?: string): Promise<Map<string, string>> {
  const logsInfo = new Map<string, string>();
  let podName = '';
  let podNameSpace = '';
  let cachedExecutionId = '';
  let logsDetails = '';
  let logsBannerMessage = '';
  let logsBannerAdditionalInfo = '';
  const customPropertiesMap = execution.getCustomPropertiesMap();
  const createdAt = new Date(execution.getCreateTimeSinceEpoch()).toISOString().split('T')[0];

  if (execution) {
    podName = customPropertiesMap.get(KfpExecutionProperties.POD_NAME)?.getStringValue() || '';
    podNameSpace = customPropertiesMap.get('namespace')?.getStringValue() || '';
    cachedExecutionId =
      customPropertiesMap.get(EXECUTION_KEY_CACHED_EXECUTION_ID)?.getStringValue() || '';
  }

  // TODO(jlyaoyuli): Consider to link to the cached execution.
  if (cachedExecutionId) {
    logsInfo.set(LOGS_DETAILS, 'This step output is taken from cache.');
    return logsInfo; // Early return if it is from cache.
  }

  try {
    logsDetails = await Apis.getPodLogs(runId!, podName, podNameSpace, createdAt);
    logsInfo.set(LOGS_DETAILS, logsDetails);
  } catch (err) {
    let errMsg = await errorToMessage(err);
    logsBannerMessage = 'Failed to retrieve pod logs.';
    logsInfo.set(LOGS_BANNER_MESSAGE, logsBannerMessage);
    logsBannerAdditionalInfo = 'Error response: ' + errMsg;
    logsInfo.set(LOGS_BANNER_ADDITIONAL_INFO, logsBannerAdditionalInfo);
  }
  return logsInfo;
}

interface ArtifactNodeDetailProps {
  execution?: Execution;
  linkedArtifact?: LinkedArtifact;
  namespace: string | undefined;
}
function ArtifactNodeDetail({ execution, linkedArtifact, namespace }: ArtifactNodeDetailProps) {
  const { data } = useQuery<ArtifactType[], Error>(
    ['artifact_types', { linkedArtifact }],
    () => getArtifactTypes(),
    {},
  );

  const [selectedTab, setSelectedTab] = useState(0);
  return (
    <div className={commonCss.page}>
      <MD2Tabs
        tabs={['Artifact Info', 'Visualization']}
        selectedTab={selectedTab}
        onSwitch={tab => setSelectedTab(tab)}
      />
      <div className={padding(20)}>
        {/* Artifact Info tab */}
        {selectedTab === 0 && (
          <ArtifactInfo
            execution={execution}
            artifactTypes={data}
            linkedArtifact={linkedArtifact}
            namespace={namespace}
          ></ArtifactInfo>
        )}

        {/* Visualization tab */}
        {selectedTab === 1 && execution && (
          <MetricsVisualizations
            linkedArtifacts={linkedArtifact ? [linkedArtifact] : []}
            artifactTypes={data ? data : []}
            execution={execution}
            namespace={namespace}
          />
        )}
      </div>
    </div>
  );
}

interface ArtifactNodeDetailProps {
  execution?: Execution;
  artifactTypes?: ArtifactType[];
  linkedArtifact?: LinkedArtifact;
  namespace: string | undefined;
}

function ArtifactInfo({
  execution,
  artifactTypes,
  linkedArtifact,
  namespace,
}: ArtifactNodeDetailProps) {
  if (!execution || !linkedArtifact) {
    return NODE_STATE_UNAVAILABLE;
  }

  // Static Artifact information.
  const taskName =
    execution
      .getCustomPropertiesMap()
      .get('display_name')
      ?.getStringValue() || '-';
  const artifactName =
    linkedArtifact.artifact
      .getCustomPropertiesMap()
      .get('display_name')
      ?.getStringValue() || '-';
  let artifactTypeName = artifactTypes
    ? getArtifactTypeName(artifactTypes, [linkedArtifact])
    : ['-'];

  // Runtime artifact information.
  const createdAt = new Date(linkedArtifact.artifact.getCreateTimeSinceEpoch());

  // Artifact info rows.
  const artifactInfo = [
    ['Upstream Task Name', taskName],
    ['Artifact Name', artifactName],
    ['Artifact Type', artifactTypeName],
    ['Created At', createdAt],
  ];

  let artifactParamsWithSessionInfo = getArtifactParamList([linkedArtifact], artifactTypeName);
  let artifactParams: ParamList = [];

  if (artifactParamsWithSessionInfo) {
    artifactParams = artifactParamsWithSessionInfo.params;
  }

  return (
    <div>
      <ArtifactTitle artifact={linkedArtifact.artifact}></ArtifactTitle>
      {artifactInfo && (
        <div>
          <DetailsTable title='Artifact Info' fields={artifactInfo} />
        </div>
      )}

      <div>
        <DetailsTable<string>
          key={`artifact-url`}
          title='Artifact URI'
          fields={artifactParams}
          valueComponent={ArtifactPreview}
          valueComponentProps={{
            namespace: namespace,
            sessionMap: artifactParamsWithSessionInfo.sessionMap,
          }}
        />
      </div>
    </div>
  );
}

interface SubDAGNodeDetailProps {
  element: FlowElement<FlowElementDataBase>;
  execution?: Execution;
  layers: string[];
  onLayerChange: (layers: string[]) => void;
  namespace: string | undefined;
}

function SubDAGNodeDetail({
  element,
  execution,
  layers,
  onLayerChange,
  namespace,
}: SubDAGNodeDetailProps) {
  const taskKey = getTaskKeyFromNodeKey(element.id);
  // const componentSpec = getComponentSpec(pipelineSpec, layers, taskKey);
  // if (!componentSpec) {
  //   return NODE_INFO_UNKNOWN;
  // }

  const onSubDagOpenClick = () => {
    onLayerChange([...layers, taskKey]);
  };

  const [selectedTab, setSelectedTab] = useState(0);

  return (
    <div>
      <div className={commonCss.page}>
        <div className={padding(20, 'blr')}>
          <Button variant='contained' onClick={onSubDagOpenClick}>
            Open Sub-DAG
          </Button>
        </div>
        <MD2Tabs
          tabs={['Input/Output', 'Task Details']}
          selectedTab={selectedTab}
          onSwitch={tab => setSelectedTab(tab)}
        />
        <div className={commonCss.page}>
          {/* Input/Output tab */}
          {selectedTab === 0 &&
            (() => {
              if (execution) {
                return (
                  <InputOutputTab execution={execution} namespace={namespace}></InputOutputTab>
                );
              }
              return NODE_STATE_UNAVAILABLE;
            })()}

          {/* Task Details tab */}
          {selectedTab === 1 && (
            <div className={padding(20)}>
              <DetailsTable
                title='Task Details'
                fields={getTaskDetailsFields(element, execution)}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

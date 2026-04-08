/*
 * Copyright 2024 The Kubeflow Authors
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
import { Apis } from 'src/lib/Apis';
import { getTaskKeyFromNodeKey, PipelineFlowElement } from 'src/lib/v2/StaticFlow';
import { convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import { KfpExecutionProperties } from 'src/mlmd/MlmdUtils';
import { Execution } from 'src/third_party/mlmd';

// Pod status/events related interfaces
export interface PodStatusInfo {
  phase?: string;
  message?: string;
  reason?: string;
  podIP?: string;
  nodeName?: string;
  containerStatuses?: ContainerStatusInfo[];
  cached?: boolean;
  cachedAt?: number;
}

export interface ContainerStatusInfo {
  name: string;
  ready: boolean;
  restartCount: number;
  state?: string;
  reason?: string;
  message?: string;
  exitCode?: number;
}

export interface PodEventInfo {
  type: string;
  reason: string;
  message: string;
  firstTimestamp?: string;
  lastTimestamp?: string;
  count: number;
  source?: string;
  cached?: boolean;
}

export interface PodStateSnapshot {
  timestamp: number;
  phase?: string;
  reason?: string;
  message?: string;
  containerStatuses?: ContainerStatusInfo[];
  isError: boolean;
}

export interface PodStatusResult {
  status: PodStatusInfo | null;
  events: PodEventInfo[];
  stateHistory: PodStateSnapshot[];
  cached?: boolean;
  cachedAt?: number;
}

export const ERROR_REASONS = [
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

// Derives the effective pod state based on container statuses.
// Pod phase might be "Pending" while containers are in error state.
export function deriveEffectiveState(
  phase?: string,
  containerStatuses?: ContainerStatusInfo[],
  initContainerStatuses?: ContainerStatusInfo[],
): { effectivePhase: string; errorReason?: string; errorMessage?: string } {
  // Check init containers first (they run before regular containers)
  if (initContainerStatuses && initContainerStatuses.length > 0) {
    for (const cs of initContainerStatuses) {
      if (cs.reason && ERROR_REASONS.includes(cs.reason)) {
        return { effectivePhase: 'Failed', errorReason: cs.reason, errorMessage: cs.message };
      }
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
        return { effectivePhase: 'Failed', errorReason: cs.reason, errorMessage: cs.message };
      }
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

// Resolve the component ref name of the immediate parent DAG from layers.
// The compiler (dag.go) uses component ref names as parentDagName, not task keys.
// layers = ['root'] → root DAG → returns undefined (root-level)
// layers = ['root', 'inner-pipeline'] → returns componentRef name of 'inner-pipeline'
function resolveParentDagComponentName(
  layers: string[] | undefined,
  pipelineJobString: string | undefined,
): string | undefined {
  if (!layers || layers.length <= 1 || !pipelineJobString) {
    return undefined;
  }
  try {
    const pipelineSpec = convertYamlToV2PipelineSpec(pipelineJobString);
    let currentDag = pipelineSpec.root?.dag;
    // Walk layers[1:] to find the componentRef name of the last (immediate parent) layer.
    // layers[0] is always 'root', so we skip it.
    for (let i = 1; i < layers.length; i++) {
      const taskKey = layers[i];
      const taskSpec = currentDag?.tasks[taskKey];
      const componentName = taskSpec?.componentRef?.name;
      if (!componentName) return undefined;
      if (i === layers.length - 1) {
        return componentName;
      }
      const componentSpec = pipelineSpec.components[componentName];
      currentDag = componentSpec?.dag;
      if (!currentDag) return undefined;
    }
  } catch {
    // If pipeline spec parsing fails, fall back to undefined
  }
  return undefined;
}

// Build the full task path from layers and element ID for nested DAG disambiguation.
// Must match the compiler's task_path annotation format (dag.go):
//   - Root-level tasks (parentDagName == "root"): bare task name
//   - Nested tasks: "root.<componentRefName>.<taskName>"
function buildTaskPath(
  elementId: string | undefined,
  layers: string[] | undefined,
  pipelineJobString: string | undefined,
): string | undefined {
  const leafTaskKey = elementId ? getTaskKeyFromNodeKey(elementId) : undefined;
  if (!leafTaskKey) return undefined;
  // Resolve the parent DAG's componentRef name to match the compiler's format.
  const parentComponentName = resolveParentDagComponentName(layers, pipelineJobString);
  if (parentComponentName) {
    return 'root.' + parentComponentName + '.' + leafTaskKey;
  }
  return leafTaskKey;
}

// Helper to find the best matching pod from a list.
function findBestPod(
  pods: any[],
  taskName: string | undefined,
  elementId: string | undefined,
): any {
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

  const isImplPod = (pod: any): boolean => {
    const name = pod.metadata?.name || '';
    return name.includes('-impl-') || name.includes('-launcher-') || name.includes('-user-');
  };

  const isDriverPod = (pod: any): boolean => {
    const name = pod.metadata?.name || '';
    return name.includes('-driver-') || name.includes('-dag-driver-');
  };

  const scorePod = (pod: any): number => {
    let score = 0;
    if (hasContainerError(pod)) score += 100;
    if (isImplPod(pod)) score += 50;
    if (!isDriverPod(pod)) score += 10;
    if (pod.status?.phase !== 'Succeeded') score += 25;
    return score;
  };

  const sortedPods = [...pods].sort((a: any, b: any) => scorePod(b) - scorePod(a));
  let matchingPod = sortedPods[0];

  if (taskName && sortedPods.length > 1) {
    const leafKey = elementId ? getTaskKeyFromNodeKey(elementId) : taskName;
    const found = sortedPods.find((p: any) => {
      const labels = p.metadata?.labels || {};
      const annotations = p.metadata?.annotations || {};
      const podTaskLabel = labels['pipelines.kubeflow.org/task_name'] || '';
      const podTaskPath = annotations['pipelines.kubeflow.org/task_path'] || '';
      const pName = p.metadata?.name || '';
      return (
        podTaskPath === taskName ||
        podTaskLabel === taskName ||
        podTaskLabel === leafKey ||
        podTaskPath.endsWith('.' + leafKey) ||
        podTaskLabel.endsWith('.' + leafKey) ||
        labels['component'] === leafKey ||
        pName.includes((leafKey || '').replace(/_/g, '-')) ||
        pName.includes(leafKey || '')
      );
    });
    if (found) {
      matchingPod = found;
    }
  }

  return matchingPod;
}

// Helper to parse a raw K8s container status object.
function parseContainerStatus(cs: any): ContainerStatusInfo {
  return {
    name: cs.name,
    ready: cs.ready,
    restartCount: cs.restartCount,
    state: cs.state ? Object.keys(cs.state)[0] : undefined,
    reason: cs.state?.waiting?.reason || cs.state?.terminated?.reason,
    message: cs.state?.waiting?.message || cs.state?.terminated?.message,
    exitCode: cs.state?.terminated?.exitCode,
  };
}

// Fetch pod status and events from Kubernetes API.
// Execution is optional to support early failures where MLMD execution doesn't exist yet.
export async function getPodStatusAndEvents(
  execution: Execution | undefined,
  namespace?: string,
  runId?: string,
  element?: PipelineFlowElement | null,
  layers?: string[],
  pipelineJobString?: string,
): Promise<PodStatusResult> {
  let podName = '';
  let podNameSpace = namespace || '';

  // Try to get pod info from MLMD execution custom properties
  if (execution) {
    const customPropertiesMap = execution.getCustomPropertiesMap();
    podName = customPropertiesMap.get(KfpExecutionProperties.POD_NAME)?.getStringValue() || '';
    podNameSpace = customPropertiesMap.get('namespace')?.getStringValue() || namespace || '';
  }

  // If pod name is not in MLMD, try to find it by run ID label.
  // This handles early failures like ImagePullBackOff where the driver never ran.
  if (!podName && runId && podNameSpace) {
    try {
      const taskName = buildTaskPath(element?.id, layers, pipelineJobString);
      const pods = await Apis.getPodsByRunId(runId, podNameSpace, taskName);
      if (pods && pods.length > 0) {
        const matchingPod = findBestPod(pods, taskName, element?.id);
        podName = matchingPod?.metadata?.name || '';
      }
    } catch (err) {
      // Failed to get pods - this might be expected if no pods exist yet
    }
  }

  if (!podName || !podNameSpace) {
    return { status: null, events: [], stateHistory: [], cached: false };
  }

  const resolvedTaskName = buildTaskPath(element?.id, layers, pipelineJobString);

  let status: PodStatusInfo | null = null;
  let events: PodEventInfo[] = [];
  let stateHistory: PodStateSnapshot[] = [];
  let cached = false;
  let cachedAt: number | undefined;

  // Try to get pod info
  try {
    const podInfo = await Apis.getPodInfo(podName, podNameSpace, runId, resolvedTaskName);
    if (podInfo && podInfo.status) {
      const podStatus = podInfo.status as any;

      if ((podInfo as any)._cached) {
        cached = true;
        cachedAt = (podInfo as any)._cachedAt;
      }

      if ((podInfo as any)._stateHistory) {
        stateHistory = (podInfo as any)._stateHistory;
      }

      const containerStatuses = (podStatus.containerStatuses || []).map(parseContainerStatus);
      const initContainerStatuses = (podStatus.initContainerStatuses || []).map(
        parseContainerStatus,
      );

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
    // Pod may not exist anymore - we'll show events if available
  }

  // Try to get pod events
  try {
    const eventList = await Apis.getPodEvents(podName, podNameSpace, runId, resolvedTaskName);
    if (eventList && eventList.items) {
      if ((eventList as any)._cached) {
        cached = true;
        cachedAt = cachedAt || (eventList as any)._cachedAt;
      }

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
      events.sort((a, b) => {
        const timeA = a.lastTimestamp ? new Date(a.lastTimestamp).getTime() : 0;
        const timeB = b.lastTimestamp ? new Date(b.lastTimestamp).getTime() : 0;
        return timeB - timeA;
      });
    }
  } catch (err) {
    // Events may not be available
  }

  return { status, events, stateHistory, cached, cachedAt };
}

// React hook wrapping getPodStatusAndEvents with useQuery.
export function usePodStatus(
  execution: Execution | undefined,
  namespace: string | undefined,
  runId: string | undefined,
  element: PipelineFlowElement | null | undefined,
  layers: string[],
  pipelineJobString?: string,
) {
  return useQuery<PodStatusResult, Error>({
    queryKey: ['podStatus', execution?.getId(), runId, element?.id, namespace, layers.join('/')],
    queryFn: () =>
      getPodStatusAndEvents(execution, namespace, runId, element, layers, pipelineJobString),
    enabled: !!(runId && namespace),
    refetchInterval: 30000,
  });
}

/*
 * Copyright 2019 The Kubeflow Authors
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

import { logger } from 'src/lib/Utils';
import { PipelineTaskTaskState, V2RuntimeState } from 'src/apisv2/run';

export const statusBgColors = {
  error: '#fce8e6',
  notStarted: '#f7f7f7',
  running: '#e8f0fe',
  succeeded: '#e6f4ea',
  cached: '#e6f4ea',
  terminatedOrSkipped: '#f1f3f4',
  warning: '#fef7f0',
};

export const statusProtoMap = new Map<V2RuntimeState, string>([
  [V2RuntimeState.RUNTIME_STATE_UNSPECIFIED, 'Unknown'],
  [V2RuntimeState.PENDING, 'Pending'],
  [V2RuntimeState.RUNNING, 'Running'],
  [V2RuntimeState.SUCCEEDED, 'Succeeded'],
  [V2RuntimeState.SKIPPED, 'Skipped'],
  [V2RuntimeState.FAILED, 'Failed'],
  [V2RuntimeState.CANCELING, 'Canceling'],
  [V2RuntimeState.CANCELED, 'Canceled'],
  [V2RuntimeState.PAUSED, 'Paused'],
]);

export type StatusTone = 'succeeded' | 'running' | 'failed' | 'neutral' | 'warning';
export interface StatusPresentation {
  label: string;
  tone: StatusTone;
}

const runtimeTones: Record<V2RuntimeState, StatusTone> = {
  SUCCEEDED: 'succeeded',
  RUNNING: 'running',
  FAILED: 'failed',
  PENDING: 'neutral',
  CANCELING: 'running',
  CANCELED: 'neutral',
  PAUSED: 'warning',
  SKIPPED: 'neutral',
  RUNTIME_STATE_UNSPECIFIED: 'neutral',
};

// Task states are a distinct API domain: cached results are successful, and tasks
// do not expose runtime-only states such as paused or canceling.
const taskStates: Record<PipelineTaskTaskState, StatusPresentation> = {
  SUCCEEDED: { label: 'Succeeded', tone: 'succeeded' },
  RUNNING: { label: 'Running', tone: 'running' },
  FAILED: { label: 'Failed', tone: 'failed' },
  SKIPPED: { label: 'Skipped', tone: 'neutral' },
  CACHED: { label: 'Cached', tone: 'succeeded' },
  RUNTIME_STATE_UNSPECIFIED: { label: 'Unknown', tone: 'neutral' },
};

export function getRunStatus(state?: V2RuntimeState): StatusPresentation {
  return {
    label: (state && statusProtoMap.get(state)) || 'Unknown',
    tone: (state && runtimeTones[state]) || 'neutral',
  };
}

export function getTaskStatus(state?: PipelineTaskTaskState): StatusPresentation {
  return state === undefined
    ? { label: 'Task', tone: 'neutral' }
    : taskStates[state] || taskStates.RUNTIME_STATE_UNSPECIFIED;
}

// Timeline bars describe execution time: a cache hit performed no execution, so
// it stays neutral here even though the graph marks its result successful.
export function getTaskTimelineTone(state?: PipelineTaskTaskState): StatusTone {
  return state === PipelineTaskTaskState.CACHED ? 'neutral' : getTaskStatus(state).tone;
}

export function hasFinishedV2(state?: V2RuntimeState): boolean {
  switch (state) {
    case V2RuntimeState.SUCCEEDED: // Fall through
    case V2RuntimeState.SKIPPED: // Fall through
    case V2RuntimeState.FAILED: // Fall through
    case V2RuntimeState.CANCELED:
      return true;
    case V2RuntimeState.PENDING: // Fall through
    case V2RuntimeState.RUNNING: // Fall through
    case V2RuntimeState.CANCELING: // Fall through
    case V2RuntimeState.PAUSED: // Fall through
    case V2RuntimeState.RUNTIME_STATE_UNSPECIFIED:
      return false;
    default:
      logger.warn('Unknown state:', state);
      return false;
  }
}

export function statusToBgColorV2(state?: V2RuntimeState, nodeMessage?: string): string {
  state = checkIfTerminatedV2(state, nodeMessage);
  switch (state) {
    case V2RuntimeState.FAILED:
      return statusBgColors.error;
    case V2RuntimeState.PENDING:
      return statusBgColors.notStarted;
    case V2RuntimeState.CANCELING:
    // fall through
    case V2RuntimeState.RUNNING:
      return statusBgColors.running;
    case V2RuntimeState.PAUSED:
      return statusBgColors.notStarted;
    case V2RuntimeState.SUCCEEDED:
      return statusBgColors.succeeded;
    case V2RuntimeState.SKIPPED:
    // fall through
    case V2RuntimeState.CANCELED:
      return statusBgColors.terminatedOrSkipped;
    case V2RuntimeState.RUNTIME_STATE_UNSPECIFIED:
    // fall through
    default:
      logger.verbose('Unknown state:', state);
      return statusBgColors.notStarted;
  }
}

export function checkIfTerminatedV2(
  state?: V2RuntimeState,
  nodeMessage?: string,
): V2RuntimeState | undefined {
  // Argo considers terminated runs as having "Failed", so we have to examine the failure message to
  // determine why the run failed.
  if (state === V2RuntimeState.FAILED && nodeMessage === 'terminated') {
    state = V2RuntimeState.CANCELED;
  }
  return state;
}

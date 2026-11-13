// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { V2PipelineTask } from 'src/apisv2/run';
import { Apis } from 'src/lib/Apis';

// Keys must match the launcher's backend/src/v2/component/debug_pause.go.
// Pause is an overlay on status_metadata.custom_properties - never a
// PipelineTask state.
const CUSTOM_PROP_DEBUG_PAUSE_BARRIER = 'debug_pause_barrier';
const CUSTOM_PROP_DEBUG_PAUSE_RESUME_REQUESTED = 'debug_pause_resume_requested';
const CUSTOM_PROP_DEBUG_PAUSE_INSTANCE = 'debug_pause_instance';

export type DebugPauseBarrier = 'before' | 'after' | 'on_error';

const VALID_BARRIERS: ReadonlySet<string> = new Set(['before', 'after', 'on_error']);

// The generated type says { [key: string]: object } but a string-valued
// protobuf.Value serializes as a bare JSON string - narrow at runtime.
type CustomProperties = Record<string, unknown> | undefined;

/**
 * Reads the task's current debug-pause barrier from
 * status_metadata.custom_properties, or undefined when not parked.
 */
export function getDebugPauseBarrier(
  task: V2PipelineTask | undefined,
): DebugPauseBarrier | undefined {
  const properties = task?.status_metadata?.custom_properties as CustomProperties;
  const raw = properties?.[CUSTOM_PROP_DEBUG_PAUSE_BARRIER];
  if (typeof raw !== 'string' || !VALID_BARRIERS.has(raw)) {
    return undefined;
  }
  return raw as DebugPauseBarrier;
}

export function isDebugPaused(task: V2PipelineTask | undefined): boolean {
  return getDebugPauseBarrier(task) !== undefined;
}

// Reads the per-pause instance ID. The same barrier value can appear on two
// pause epochs (e.g. pod restart); the instance distinguishes them.
export function getDebugPauseInstance(task: V2PipelineTask | undefined): string | undefined {
  const properties = task?.status_metadata?.custom_properties as CustomProperties;
  const raw = properties?.[CUSTOM_PROP_DEBUG_PAUSE_INSTANCE];
  return typeof raw === 'string' && raw !== '' ? raw : undefined;
}

/**
 * Requests resume via the existing UpdateTask RPC - no dedicated endpoint
 * needed since UpdateTask already does run-scoped authorization. The
 * launcher observes the flag on its next poll.
 *
 * UpdateTask replaces status_metadata wholesale, so refetch first and merge
 * into fresh custom_properties. The write is refused if the barrier or pause
 * instance changed since the page loaded - a stale write could resurrect an
 * older barrier and grant a resume meant for a previous pause.
 */
export async function requestDebugPauseResume(
  runId: string,
  task: V2PipelineTask,
): Promise<V2PipelineTask> {
  if (!task.task_id) {
    throw new Error('Cannot request resume for a task with no task_id.');
  }
  const expectedBarrier = getDebugPauseBarrier(task);
  if (!expectedBarrier) {
    throw new Error('This task is not paused.');
  }
  const expectedInstance = getDebugPauseInstance(task);
  const freshTask = await Apis.runServiceApiV2.task_1(runId, task.task_id);
  const currentBarrier = getDebugPauseBarrier(freshTask);
  if (currentBarrier !== expectedBarrier) {
    throw new Error(
      currentBarrier
        ? `This task moved from the "${expectedBarrier}" barrier to "${currentBarrier}" - refresh and resume again.`
        : 'This task is no longer paused - it may have already been resumed or finished.',
    );
  }
  if (expectedInstance && getDebugPauseInstance(freshTask) !== expectedInstance) {
    throw new Error('This pause was restarted since the page loaded - refresh and resume again.');
  }
  const existingProperties =
    (freshTask.status_metadata?.custom_properties as CustomProperties) || {};
  return Apis.runServiceApiV2.task_2(runId, task.task_id, {
    task_id: task.task_id,
    run_id: runId,
    status_metadata: {
      ...freshTask.status_metadata,
      custom_properties: {
        ...existingProperties,
        [CUSTOM_PROP_DEBUG_PAUSE_RESUME_REQUESTED]: 'true',
      } as unknown as { [key: string]: object },
    },
  });
}

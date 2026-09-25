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

import { V2beta1PipelineTask } from 'src/apisv2beta1/run';
import { Apis } from 'src/lib/Apis';

// These keys and values must exactly match the launcher's
// backend/src/v2/component/debug_pause.go (customPropDebugPauseBarrier /
// customPropDebugPauseResumeRequested). Debug pause is a separate overlay
// signal on a task's status_metadata.custom_properties - it never competes
// with the task's real RUNNING/SUCCEEDED/FAILED state.
const CUSTOM_PROP_DEBUG_PAUSE_BARRIER = 'debug_pause_barrier';
const CUSTOM_PROP_DEBUG_PAUSE_RESUME_REQUESTED = 'debug_pause_resume_requested';

export type DebugPauseBarrier = 'before' | 'after' | 'on_error';

const VALID_BARRIERS: ReadonlySet<string> = new Set(['before', 'after', 'on_error']);

// PipelineTaskStatusMetadata types custom_properties as
// { [key: string]: object } because it's generated from a
// map<string, google.protobuf.Value> field. In practice, a string-valued
// protobuf.Value serializes over JSON as a bare JSON string, not an object —
// the generated type is simply imprecise for this case. This helper narrows
// safely at runtime (typeof check) rather than trusting the declared type.
type CustomProperties = Record<string, unknown> | undefined;

/**
 * Reads the task's current debug-pause barrier, if any, from
 * status_metadata.custom_properties. Returns undefined when the task is not
 * currently parked (the common case for a task that never called
 * set_debug_pause(), or one that hasn't reached a configured barrier yet).
 *
 * This is a live signal reported directly by the launcher - not an
 * inference from how long a task has been running - so it accurately
 * distinguishes "genuinely parked" from "just taking a while."
 */
export function getDebugPauseBarrier(
  task: V2beta1PipelineTask | undefined,
): DebugPauseBarrier | undefined {
  const properties = task?.status_metadata?.custom_properties as CustomProperties;
  const raw = properties?.[CUSTOM_PROP_DEBUG_PAUSE_BARRIER];
  if (typeof raw !== 'string' || !VALID_BARRIERS.has(raw)) {
    return undefined;
  }
  return raw as DebugPauseBarrier;
}

export function isDebugPaused(task: V2beta1PipelineTask | undefined): boolean {
  return getDebugPauseBarrier(task) !== undefined;
}

/**
 * Requests that a paused task resume. This calls the existing UpdateTask RPC
 * directly — no purpose-built "/:resume" endpoint exists or is needed, since
 * UpdateTask already performs the same run-scoped authorization such an
 * endpoint would require. The launcher's own polling loop (in the running
 * pod) discovers this flag on its own next poll and continues by itself;
 * this call never reaches into the pod directly.
 */
export async function requestDebugPauseResume(
  runId: string,
  task: V2beta1PipelineTask,
): Promise<V2beta1PipelineTask> {
  if (!task.task_id) {
    throw new Error('Cannot request resume for a task with no task_id.');
  }
  const existingProperties = (task.status_metadata?.custom_properties as CustomProperties) || {};
  return Apis.runServiceApiV2.task_2(runId, task.task_id, {
    task_id: task.task_id,
    run_id: runId,
    status_metadata: {
      ...task.status_metadata,
      custom_properties: {
        ...existingProperties,
        [CUSTOM_PROP_DEBUG_PAUSE_RESUME_REQUESTED]: 'true',
      } as { [key: string]: object },
    },
  });
}

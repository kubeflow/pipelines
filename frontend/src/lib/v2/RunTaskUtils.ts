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

import { PipelineTaskTaskState, PipelineTaskTaskType, V2PipelineTask, V2Run } from 'src/apisv2/run';
import { Apis } from 'src/lib/Apis';
import { listAllPages } from './PaginationUtils';

const MAX_PAGE_SIZE = 200;

export function listAllRunTasks(runId: string): Promise<V2PipelineTask[]> {
  return listAllPages(async (pageToken) => {
    const response = await Apis.runServiceApiV2.tasks(
      runId,
      undefined,
      MAX_PAGE_SIZE,
      pageToken,
      undefined,
      'create_time asc',
    );
    return { items: response.tasks, nextPageToken: response.next_page_token };
  }, 'Task service');
}

export function getTaskDisplayName(task: V2PipelineTask, fallback = 'Task'): string {
  // Replace only the launcher's generic label; preserve custom loop display names.
  if (task.type === PipelineTaskTaskType.LOOP && task.display_name === 'Loop' && task.name) {
    return `${task.name} (Loop)`;
  }
  return task.display_name || task.name || fallback;
}

// Returns the pod lifecycle diagnostic for a task that has not completed successfully.
// A message left on a succeeded, cached or skipped task is stale, so it is ignored.
export function getTaskLifecycleMessage(task?: V2PipelineTask): string | undefined {
  if (!task?.lifecycle_message) {
    return undefined;
  }
  if (
    task.state === PipelineTaskTaskState.SUCCEEDED ||
    task.state === PipelineTaskTaskState.CACHED ||
    task.state === PipelineTaskTaskState.SKIPPED
  ) {
    return undefined;
  }
  return task.lifecycle_message;
}

export function getRunDisplayName(run: V2Run, fallback = 'Run'): string {
  return run.display_name || run.run_id || fallback;
}

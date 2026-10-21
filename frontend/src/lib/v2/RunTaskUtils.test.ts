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

import { Apis } from 'src/lib/Apis';
import { PipelineTaskTaskState, PipelineTaskTaskType } from 'src/apisv2/run';
import {
  getRunDisplayName,
  getTaskDisplayName,
  getTaskLifecycleMessage,
  listAllRunTasks,
} from './RunTaskUtils';

describe('listAllRunTasks', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('retrieves every page using the backend maximum page size', async () => {
    const tasksSpy = vi.spyOn(Apis.runServiceApiV2, 'tasks');
    tasksSpy
      .mockResolvedValueOnce({ tasks: [{ task_id: 'task-1' }], next_page_token: 'next-page' })
      .mockResolvedValueOnce({ tasks: [{ task_id: 'task-2' }] });

    await expect(listAllRunTasks('run-1')).resolves.toEqual([
      { task_id: 'task-1' },
      { task_id: 'task-2' },
    ]);
    expect(tasksSpy).toHaveBeenNthCalledWith(
      1,
      'run-1',
      undefined,
      200,
      undefined,
      undefined,
      'create_time asc',
    );
    expect(tasksSpy).toHaveBeenNthCalledWith(
      2,
      'run-1',
      undefined,
      200,
      'next-page',
      undefined,
      'create_time asc',
    );
  });

  it('rejects a repeated page token', async () => {
    vi.spyOn(Apis.runServiceApiV2, 'tasks').mockResolvedValue({
      tasks: [],
      next_page_token: 'repeated-page',
    });

    await expect(listAllRunTasks('run-1')).rejects.toThrow(
      'Task service returned a repeated page token: repeated-page',
    );
    expect(Apis.runServiceApiV2.tasks).toHaveBeenCalledTimes(2);
  });
});

describe('runtime display names', () => {
  it('retains a loop task identity beside its generic runtime label', () => {
    expect(
      getTaskDisplayName({
        type: PipelineTaskTaskType.LOOP,
        display_name: 'Loop',
        name: 'parallel-loop',
      }),
    ).toBe('parallel-loop (Loop)');
    expect(
      getTaskDisplayName({
        type: PipelineTaskTaskType.LOOP,
        display_name: 'Train models',
        name: 'parallel-loop',
      }),
    ).toBe('Train models');
  });
  it('prefers task display name, then name, then the supplied fallback', () => {
    expect(getTaskDisplayName({ display_name: 'Display', name: 'name' })).toBe('Display');
    expect(getTaskDisplayName({ name: 'name' })).toBe('name');
    expect(getTaskDisplayName({}, '-')).toBe('-');
  });

  it('prefers run display name, then ID, then the supplied fallback', () => {
    expect(getRunDisplayName({ display_name: 'Display', run_id: 'run-id' })).toBe('Display');
    expect(getRunDisplayName({ run_id: 'run-id' })).toBe('run-id');
    expect(getRunDisplayName({}, '-')).toBe('-');
  });
});

describe('getTaskLifecycleMessage', () => {
  it('returns the message for a task that is still pending or running', () => {
    expect(
      getTaskLifecycleMessage({
        state: PipelineTaskTaskState.RUNNING,
        lifecycle_message: 'ImagePullBackOff',
      }),
    ).toBe('ImagePullBackOff');
  });

  it('returns the message for a failed task', () => {
    expect(
      getTaskLifecycleMessage({
        state: PipelineTaskTaskState.FAILED,
        lifecycle_message: 'OOMKilled',
      }),
    ).toBe('OOMKilled');
  });

  it('returns undefined when there is no message or an empty message', () => {
    expect(getTaskLifecycleMessage(undefined)).toBeUndefined();
    expect(getTaskLifecycleMessage({ state: PipelineTaskTaskState.RUNNING })).toBeUndefined();
    expect(
      getTaskLifecycleMessage({ state: PipelineTaskTaskState.RUNNING, lifecycle_message: '' }),
    ).toBeUndefined();
  });

  it.each([
    PipelineTaskTaskState.SUCCEEDED,
    PipelineTaskTaskState.CACHED,
    PipelineTaskTaskState.SKIPPED,
  ])('ignores a stale message on a %s task', (state) => {
    expect(
      getTaskLifecycleMessage({ state, lifecycle_message: 'ImagePullBackOff' }),
    ).toBeUndefined();
  });
});

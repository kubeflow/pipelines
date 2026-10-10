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
import { getDebugPauseBarrier, isDebugPaused, requestDebugPauseResume } from './DebugPauseUtils';

function taskWithCustomProperties(
  customProperties: Record<string, unknown> | undefined,
): V2beta1PipelineTask {
  return {
    task_id: 'task-1',
    status_metadata: customProperties
      ? { custom_properties: customProperties as { [key: string]: object } }
      : undefined,
  };
}

describe('getDebugPauseBarrier', () => {
  it('returns undefined when the task is undefined', () => {
    expect(getDebugPauseBarrier(undefined)).toBeUndefined();
  });

  it('returns undefined when status_metadata is absent', () => {
    expect(getDebugPauseBarrier(taskWithCustomProperties(undefined))).toBeUndefined();
  });

  it('returns undefined when custom_properties has no barrier key', () => {
    expect(getDebugPauseBarrier(taskWithCustomProperties({}))).toBeUndefined();
  });

  it('returns undefined for an empty-string barrier (cleared after resume)', () => {
    expect(
      getDebugPauseBarrier(taskWithCustomProperties({ debug_pause_barrier: '' })),
    ).toBeUndefined();
  });

  it('returns undefined for an unrecognized barrier value', () => {
    expect(
      getDebugPauseBarrier(taskWithCustomProperties({ debug_pause_barrier: 'not-a-real-barrier' })),
    ).toBeUndefined();
  });

  it.each([['before'], ['after'], ['on_error']])('returns %s when set', (barrier) => {
    expect(
      getDebugPauseBarrier(taskWithCustomProperties({ debug_pause_barrier: barrier })),
    ).toEqual(barrier);
  });
});

describe('isDebugPaused', () => {
  it('is false when there is no barrier', () => {
    expect(isDebugPaused(taskWithCustomProperties({}))).toBe(false);
  });

  it('is true when a barrier is set', () => {
    expect(isDebugPaused(taskWithCustomProperties({ debug_pause_barrier: 'before' }))).toBe(true);
  });
});

describe('requestDebugPauseResume', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('throws when the task has no task_id', async () => {
    await expect(requestDebugPauseResume('run-1', { task_id: undefined })).rejects.toThrow(
      /task_id/,
    );
  });

  it('calls UpdateTask (task_2) with the resume flag merged into existing custom_properties', async () => {
    const updateSpy = vi.spyOn(Apis.runServiceApiV2, 'task_2').mockResolvedValueOnce({
      task_id: 'task-1',
    });

    const task = taskWithCustomProperties({
      debug_pause_barrier: 'after',
      some_other_property: 'must-be-preserved',
    });

    await requestDebugPauseResume('run-1', task);

    expect(updateSpy).toHaveBeenCalledWith('run-1', 'task-1', {
      task_id: 'task-1',
      run_id: 'run-1',
      status_metadata: {
        custom_properties: {
          debug_pause_barrier: 'after',
          some_other_property: 'must-be-preserved',
          debug_pause_resume_requested: 'true',
        },
      },
    });
  });

  it('works when the task has no existing custom_properties at all', async () => {
    const updateSpy = vi.spyOn(Apis.runServiceApiV2, 'task_2').mockResolvedValueOnce({
      task_id: 'task-1',
    });

    await requestDebugPauseResume('run-1', { task_id: 'task-1' });

    expect(updateSpy).toHaveBeenCalledWith('run-1', 'task-1', {
      task_id: 'task-1',
      run_id: 'run-1',
      status_metadata: {
        custom_properties: {
          debug_pause_resume_requested: 'true',
        },
      },
    });
  });
});

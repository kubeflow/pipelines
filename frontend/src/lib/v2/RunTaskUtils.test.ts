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
import { listAllRunTasks } from './RunTaskUtils';

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

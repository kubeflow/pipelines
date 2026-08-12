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
import { listAllPages } from './PaginationUtils';

const MAX_PAGE_SIZE = 200;

export function listAllRunTasks(runId: string): Promise<V2beta1PipelineTask[]> {
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

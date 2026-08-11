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

export async function listAllRunTasks(runId: string): Promise<V2beta1PipelineTask[]> {
  const tasks: V2beta1PipelineTask[] = [];
  const seenPageTokens = new Set<string>();
  let pageToken: string | undefined;
  do {
    const response = await Apis.runServiceApiV2.tasks(
      runId,
      undefined,
      100,
      pageToken,
      undefined,
      'create_time asc',
    );
    tasks.push(...(response.tasks || []));
    pageToken = response.next_page_token || undefined;
    if (pageToken) {
      if (seenPageTokens.has(pageToken)) {
        throw new Error(`Task service returned a repeated page token: ${pageToken}`);
      }
      seenPageTokens.add(pageToken);
    }
  } while (pageToken);
  return tasks;
}

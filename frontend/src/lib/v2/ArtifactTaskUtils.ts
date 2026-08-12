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

import { V2beta1ArtifactTask } from 'src/apisv2beta1/artifact';
import { Apis } from 'src/lib/Apis';
import { listAllPages } from './PaginationUtils';

const MAX_PAGE_SIZE = 200;

export function listAllArtifactTasks(artifactId: string): Promise<V2beta1ArtifactTask[]> {
  return listAllPages(async (pageToken) => {
    const response = await Apis.artifactServiceApiV2.artifactTasks(
      undefined,
      undefined,
      [artifactId],
      undefined,
      pageToken,
      MAX_PAGE_SIZE,
      'id asc',
    );
    return { items: response.artifact_tasks, nextPageToken: response.next_page_token };
  }, 'Artifact service');
}

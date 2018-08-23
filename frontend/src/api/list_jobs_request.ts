// Copyright 2018 Google LLC
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

import { ListPipelinesRequest } from './list_pipelines_request';

export class ListJobsRequest extends ListPipelinesRequest {
  public filterBy: string;

  constructor(pageSize: number) {
    super(pageSize);
    this.filterBy = '';
  }

  public toQueryParams(): string {
    // TODO: this isn't actually supported by the backend yet (and won't be for a while.) (5/23)
    return super.toQueryParams() +
        (this.filterBy ? '&filterBy=' + encodeURIComponent(this.filterBy) : '');
  }
}

// Valid sortKeys as specified by the backend.
export enum JobSortKeys {
  CREATED_AT = 'created_at',
  ID = 'id',
  NAME = 'name',
  PIPELINE_ID = 'pipeline_id'
}

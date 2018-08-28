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

import { BaseListRequest } from './base_list_request';

export class ListRunsRequest extends BaseListRequest {
  public jobId: string;

  constructor(jobId: string, pageSize: number) {
    super(pageSize);
    this.jobId = jobId;
  }

  public toQueryParams(): string {
    return super.toQueryParams() + '&jobId=' + this.jobId.toString();
  }
}

// Valid sortKeys as specified by the backend.
// Note that '' and 'created_at' are considered equivalent.
export enum RunSortKeys {
  CREATED_AT = 'created_at',
  NAME = 'name'
}

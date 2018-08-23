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

import { Job } from './job';

export class ListJobsResponse {
  public next_page_token?: string;
  public jobs?: Job[];

  public static buildFromObject(listJobsResponse: any): ListJobsResponse {
    const newListJobsResponse = new ListJobsResponse();
    if (listJobsResponse.next_page_token) {
      newListJobsResponse.next_page_token = listJobsResponse.next_page_token;
    }
    if (listJobsResponse.jobs && listJobsResponse.jobs.length > 0) {
      newListJobsResponse.jobs =
          listJobsResponse.jobs.map((p: any) => Job.buildFromObject(p));
    }
    return newListJobsResponse;
  }
}

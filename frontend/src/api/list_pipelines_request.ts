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

export class ListPipelinesRequest {
  public orderAscending: boolean;
  public pageSize?: number;
  public pageToken: string;
  public sortBy: string;

  constructor(pageSize?: number) {
    this.pageSize = pageSize;
    this.pageToken = '';
    this.sortBy = '';
    this.orderAscending = true;
  }

  public toQueryParams(): string {
    const params = [
      `ascending=${this.orderAscending}`,
      `pageToken=${this.pageToken}`,
      `sortBy=${encodeURIComponent(this.sortBy)}`,
    ];
    if (this.pageSize !== undefined) {
      params.push(`pageSize=${this.pageSize}`);
    }
    return params.join('&');
  }
}

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

import { listAllPages, PageTokenTracker } from './PaginationUtils';

describe('PageTokenTracker', () => {
  it('detects self-repeating and cyclic tokens without rejecting a page reload', () => {
    const tracker = new PageTokenTracker();

    expect(tracker.isRepeated('first-query', undefined, 'page-2')).toBe(false);
    expect(tracker.isRepeated('first-query', undefined, 'page-2')).toBe(false);
    expect(tracker.isRepeated('first-query', 'page-2', 'page-3')).toBe(false);
    expect(tracker.isRepeated('first-query', 'page-3', 'page-2')).toBe(true);
    expect(tracker.isRepeated('second-query', 'page-2', 'page-2')).toBe(true);
  });
});

describe('listAllPages', () => {
  it('rejects an unbounded sequence of unique page tokens', async () => {
    let page = 0;

    await expect(
      listAllPages(async () => ({ nextPageToken: `page-${++page}` }), 'Task service', 2),
    ).rejects.toThrow('Task service returned more than 2 pages. Narrow the request and retry.');
    expect(page).toBe(2);
  });
});

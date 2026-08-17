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

interface Page<T> {
  items?: T[];
  nextPageToken?: string;
}

interface PageTokenChain {
  invalidRequests: Set<string>;
  nextTokens: Set<string>;
  successors: Map<string, string>;
}

const MAX_PAGES = 10_000;

export class PageTokenTracker {
  private chains = new Map<string, PageTokenChain>();

  public isRepeated(
    contextKey: string,
    requestPageToken: string | undefined,
    nextPageToken: string,
  ): boolean {
    const requestToken = requestPageToken || '';
    let chain = this.chains.get(contextKey);
    if (!requestToken || !chain) {
      chain = { invalidRequests: new Set(), nextTokens: new Set(), successors: new Map() };
      this.chains.set(contextKey, chain);
    }
    if (!nextPageToken) {
      return false;
    }
    const previousSuccessor = chain.successors.get(requestToken);
    if (previousSuccessor === nextPageToken) {
      return chain.invalidRequests.has(requestToken);
    }
    // A changed successor means this request recovered. Keep detecting other cycles, but do not
    // permanently latch the request to its earlier bad response.
    chain.invalidRequests.delete(requestToken);
    const repeated = requestToken === nextPageToken || chain.nextTokens.has(nextPageToken);
    chain.successors.set(requestToken, nextPageToken);
    chain.nextTokens.add(nextPageToken);
    if (repeated) {
      chain.invalidRequests.add(requestToken);
    }
    return repeated;
  }
}

export async function listAllPages<T>(
  fetchPage: (pageToken?: string) => Promise<Page<T>>,
  sourceName: string,
  maxPages = MAX_PAGES,
): Promise<T[]> {
  const items: T[] = [];
  const seenPageTokens = new Set<string>();
  let pageToken: string | undefined;
  let pageCount = 0;
  do {
    if (pageCount >= maxPages) {
      throw new Error(
        `${sourceName} returned more than ${maxPages} pages. Narrow the request and retry.`,
      );
    }
    const page = await fetchPage(pageToken);
    pageCount += 1;
    items.push(...(page.items || []));
    pageToken = page.nextPageToken || undefined;
    if (pageToken) {
      if (seenPageTokens.has(pageToken)) {
        throw new Error(`${sourceName} returned a repeated page token: ${pageToken}`);
      }
      seenPageTokens.add(pageToken);
    }
  } while (pageToken);
  return items;
}

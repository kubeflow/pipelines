/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { ResponseError } from '../generated/openapi/runtime';

export class PaginationRestartRequired extends Error {
  constructor() {
    super('Pagination changed. Start this listing again from the first page.');
  }
}

// Generated fetch clients wrap HTTP errors in ResponseError. Older callers
// also throw Response directly. Clone the body so ordinary errors stay readable.
export async function throwIfPaginationRestartRequired(error: unknown): Promise<void> {
  if (error instanceof PaginationRestartRequired) throw error;
  const response = error instanceof ResponseError ? error.response : error;
  if (!(response instanceof Response)) return;
  let body: unknown;
  try {
    body = await response.clone().json();
  } catch {
    return;
  }
  if (!body || typeof body !== 'object' || !('details' in body)) return;
  if (!Array.isArray(body.details)) return;
  if (
    body.details.some(
      (detail: unknown) =>
        !!detail &&
        typeof detail === 'object' &&
        '@type' in detail &&
        detail['@type'] === 'type.googleapis.com/google.rpc.ErrorInfo' &&
        'domain' in detail &&
        detail.domain === 'kubeflow.org' &&
        'reason' in detail &&
        detail.reason === 'PAGINATION_RESTART_REQUIRED',
    )
  ) {
    throw new PaginationRestartRequired();
  }
}

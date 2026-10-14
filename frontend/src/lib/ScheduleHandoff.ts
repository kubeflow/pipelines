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

type ScheduleOperation = 'create' | 'enable' | 'disable';

async function isScheduleHandoffRejection(error: unknown): Promise<boolean> {
  const response = error instanceof ResponseError ? error.response : error;
  if (!(response instanceof Response) || response.status !== 503) return false;
  let body: unknown;
  try {
    // Leave the original error readable by the normal error UI.
    body = await response.clone().json();
  } catch {
    return false;
  }
  if (!body || typeof body !== 'object' || !('code' in body) || body.code !== 14) return false;
  if (!('details' in body) || !Array.isArray(body.details)) return false;
  return body.details.some(
    (detail: unknown) =>
      !!detail &&
      typeof detail === 'object' &&
      '@type' in detail &&
      detail['@type'] === 'type.googleapis.com/google.rpc.ErrorInfo' &&
      'domain' in detail &&
      detail.domain === 'kubeflow.org' &&
      'reason' in detail &&
      detail.reason === 'SCHEDULE_HANDOFF_IN_PROGRESS',
  );
}

// The marker guarantees rejection before mutation. Never retry generic HTTP or
// transport failures: a create might already have succeeded despite a lost response.
export async function retryScheduleHandoff<T>(
  operation: ScheduleOperation,
  submit: () => Promise<T>,
): Promise<T> {
  const delays = [1000, 2000, 4000, 8000];
  for (let attempt = 0; ; attempt++) {
    try {
      return await submit();
    } catch (error) {
      if (!(await isScheduleHandoffRejection(error))) throw error;
      if (attempt === delays.length) {
        const unchanged =
          operation === 'create'
            ? 'The recurring run was not created.'
            : operation === 'disable'
              ? 'The disable request was not applied; an enabled schedule remains enabled and may still run.'
              : 'The enable request was not applied; the schedule retains its previous state.';
        throw new Error(
          `Scheduling is temporarily unavailable during the upgrade. ${unchanged} ` +
            'Wait for the rollout to complete, then retry and confirm the schedule state. ' +
            'If this continues, contact your cluster administrator.',
          { cause: error },
        );
      }
      await new Promise<void>((resolve) => setTimeout(resolve, delays[attempt]));
    }
  }
}

/*
 * Copyright 2019 The Kubeflow Authors
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

import { logger } from 'src/lib/Utils';
import { V2beta1RuntimeState } from 'src/apisv2beta1/run';

export const statusBgColors = {
  error: '#fce8e6',
  notStarted: '#f7f7f7',
  running: '#e8f0fe',
  succeeded: '#e6f4ea',
  cached: '#e6f4ea',
  terminatedOrSkipped: '#f1f3f4',
  warning: '#fef7f0',
};

export const statusProtoMap = new Map<V2beta1RuntimeState, string>([
  [V2beta1RuntimeState.RUNTIME_STATE_UNSPECIFIED, 'Unknown'],
  [V2beta1RuntimeState.PENDING, 'Pending'],
  [V2beta1RuntimeState.RUNNING, 'Running'],
  [V2beta1RuntimeState.SUCCEEDED, 'Succeeded'],
  [V2beta1RuntimeState.SKIPPED, 'Skipped'],
  [V2beta1RuntimeState.FAILED, 'Failed'],
  [V2beta1RuntimeState.CANCELING, 'Canceling'],
  [V2beta1RuntimeState.CANCELED, 'Canceled'],
  [V2beta1RuntimeState.PAUSED, 'Paused'],
]);

export function hasFinishedV2(state?: V2beta1RuntimeState): boolean {
  switch (state) {
    case V2beta1RuntimeState.SUCCEEDED: // Fall through
    case V2beta1RuntimeState.SKIPPED: // Fall through
    case V2beta1RuntimeState.FAILED: // Fall through
    case V2beta1RuntimeState.CANCELED:
      return true;
    case V2beta1RuntimeState.PENDING: // Fall through
    case V2beta1RuntimeState.RUNNING: // Fall through
    case V2beta1RuntimeState.CANCELING: // Fall through
    case V2beta1RuntimeState.PAUSED: // Fall through
    case V2beta1RuntimeState.RUNTIME_STATE_UNSPECIFIED:
      return false;
    default:
      logger.warn('Unknown state:', state);
      return false;
  }
}

export function statusToBgColorV2(state?: V2beta1RuntimeState, nodeMessage?: string): string {
  state = checkIfTerminatedV2(state, nodeMessage);
  switch (state) {
    case V2beta1RuntimeState.FAILED:
      return statusBgColors.error;
    case V2beta1RuntimeState.PENDING:
      return statusBgColors.notStarted;
    case V2beta1RuntimeState.CANCELING:
    // fall through
    case V2beta1RuntimeState.RUNNING:
      return statusBgColors.running;
    case V2beta1RuntimeState.PAUSED:
      return statusBgColors.notStarted;
    case V2beta1RuntimeState.SUCCEEDED:
      return statusBgColors.succeeded;
    case V2beta1RuntimeState.SKIPPED:
    // fall through
    case V2beta1RuntimeState.CANCELED:
      return statusBgColors.terminatedOrSkipped;
    case V2beta1RuntimeState.RUNTIME_STATE_UNSPECIFIED:
    // fall through
    default:
      logger.verbose('Unknown state:', state);
      return statusBgColors.notStarted;
  }
}

export function checkIfTerminatedV2(
  state?: V2beta1RuntimeState,
  nodeMessage?: string,
): V2beta1RuntimeState | undefined {
  // Argo considers terminated runs as having "Failed", so we have to examine the failure message to
  // determine why the run failed.
  if (state === V2beta1RuntimeState.FAILED && nodeMessage === 'terminated') {
    state = V2beta1RuntimeState.CANCELED;
  }
  return state;
}

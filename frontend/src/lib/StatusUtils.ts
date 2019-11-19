/*
 * Copyright 2019 Google LLC
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

import { logger } from '../lib/Utils';

export const statusBgColors = {
  error: '#fce8e6',
  notStarted: '#f7f7f7',
  running: '#e8f0fe',
  succeeded: '#e6f4ea',
  terminatedOrSkipped: '#f1f3f4',
  warning: '#fef7f0',
};

export enum NodePhase {
  ERROR = 'Error',
  FAILED = 'Failed',
  PENDING = 'Pending',
  RUNNING = 'Running',
  SKIPPED = 'Skipped',
  SUCCEEDED = 'Succeeded',
  TERMINATING = 'Terminating',
  TERMINATED = 'Terminated',
  UNKNOWN = 'Unknown',
}

export function hasFinished(status?: NodePhase): boolean {
  switch (status) {
    case NodePhase.SUCCEEDED: // Fall through
    case NodePhase.FAILED: // Fall through
    case NodePhase.ERROR: // Fall through
    case NodePhase.SKIPPED: // Fall through
    case NodePhase.TERMINATED:
      return true;
    case NodePhase.PENDING: // Fall through
    case NodePhase.RUNNING: // Fall through
    case NodePhase.TERMINATING: // Fall through
    case NodePhase.UNKNOWN:
      return false;
    default:
      return false;
  }
}

export function statusToBgColor(status?: NodePhase, nodeMessage?: string): string {
  status = checkIfTerminated(status, nodeMessage);
  switch (status) {
    case NodePhase.ERROR:
    // fall through
    case NodePhase.FAILED:
      return statusBgColors.error;
    case NodePhase.PENDING:
      return statusBgColors.notStarted;
    case NodePhase.TERMINATING:
    // fall through
    case NodePhase.RUNNING:
      return statusBgColors.running;
    case NodePhase.SUCCEEDED:
      return statusBgColors.succeeded;
    case NodePhase.SKIPPED:
    // fall through
    case NodePhase.TERMINATED:
      return statusBgColors.terminatedOrSkipped;
    case NodePhase.UNKNOWN:
    // fall through
    default:
      logger.verbose('Unknown node phase:', status);
      return statusBgColors.notStarted;
  }
}

export function checkIfTerminated(status?: NodePhase, nodeMessage?: string): NodePhase | undefined {
  // Argo considers terminated runs as having "Failed", so we have to examine the failure message to
  // determine why the run failed.
  if (status === NodePhase.FAILED && nodeMessage === 'terminated') {
    status = NodePhase.TERMINATED;
  }
  return status;
}

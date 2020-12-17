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
import { NodeStatus } from '../../third_party/argo-ui/argo_template';

export const statusBgColors = {
  error: '#fce8e6',
  notStarted: '#f7f7f7',
  running: '#e8f0fe',
  succeeded: '#e6f4ea',
  cached: '#e6f4ea',
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
  CACHED = 'Cached',
  TERMINATING = 'Terminating',
  TERMINATED = 'Terminated',
  UNKNOWN = 'Unknown',
}

export function hasFinished(status?: NodePhase): boolean {
  switch (status) {
    case NodePhase.SUCCEEDED: // Fall through
    case NodePhase.CACHED: // Fall through
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
    case NodePhase.CACHED:
      return statusBgColors.cached;
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

export function parseNodePhase(node: NodeStatus): NodePhase {
  if (node.phase !== 'Succeeded') {
    return node.phase as NodePhase; // HACK: NodePhase is a string enum that has the same items as node.phase.
  }
  return wasNodeCached(node) ? NodePhase.CACHED : NodePhase.SUCCEEDED;
}

function wasNodeCached(node: NodeStatus): boolean {
  const artifacts = node.outputs?.artifacts;
  // HACK: There is a way to detect the skipped pods based on the WorkflowStatus alone.
  // All output artifacts have the pod name (same as node ID) in the URI. But for skipped
  // pods, the pod name does not match the URIs.
  // (And now there are always some output artifacts since we've enabled log archiving).
  return !artifacts || !node.id || node.type !== 'Pod'
    ? false
    : artifacts.some(artifact => artifact.s3 && !artifact.s3.key.includes(node.id));
}

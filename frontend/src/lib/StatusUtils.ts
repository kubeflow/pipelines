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
import { NodeStatus } from 'src/third_party/mlmd/argo_template';
import { V2beta1RuntimeState } from 'src/apisv2beta1/run';

// Buckets a pod lifecycle failure by where in the pod's life it happened.
// Mirrors PodFailureCategory in backend/src/common/util/pod_failure_classifier.go.
export enum PodFailureCategory {
  PROVISIONING = 'Provisioning',
  RUNTIME = 'Runtime',
  NODE = 'Node',
}

export const statusBgColors = {
  error: '#fce8e6',
  notStarted: '#f7f7f7',
  running: '#e8f0fe',
  succeeded: '#e6f4ea',
  cached: '#e6f4ea',
  terminatedOrSkipped: '#f1f3f4',
  warning: '#fef7f0',
};

// Distinct background colors per pod failure category, so a user can tell at a
// glance whether a failed step is something they can fix themselves
// (Provisioning/Runtime) or purely infrastructure-caused (Node), without
// opening the side panel. Falls back to statusBgColors.error for failures that
// aren't a recognized pod lifecycle pattern (i.e. the user's own pipeline code).
export const podFailureCategoryBgColors: Record<PodFailureCategory, string> = {
  [PodFailureCategory.PROVISIONING]: '#fdecd2',
  [PodFailureCategory.RUNTIME]: '#fce8e6',
  [PodFailureCategory.NODE]: '#ece3f7',
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
  OMITTED = 'Omitted',
}

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

export function hasFinished(status?: NodePhase): boolean {
  switch (status) {
    case NodePhase.SUCCEEDED: // Fall through
    case NodePhase.CACHED: // Fall through
    case NodePhase.FAILED: // Fall through
    case NodePhase.ERROR: // Fall through
    case NodePhase.SKIPPED: // Fall through
    case NodePhase.TERMINATED:
    case NodePhase.OMITTED:
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
    case NodePhase.FAILED: {
      const classification = classifyPodFailure(nodeMessage);
      return classification
        ? podFailureCategoryBgColors[classification.category]
        : statusBgColors.error;
    }
    case NodePhase.PENDING:
      return statusBgColors.notStarted;
    case NodePhase.OMITTED:
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

interface PodFailurePattern {
  substring: string;
  category: PodFailureCategory;
  cause: string;
  fix: string;
}

// Mirrors podFailurePatterns in backend/src/common/util/pod_failure_classifier.go.
// Kept as a parallel list rather than a shared import, since the frontend can't
// depend on backend Go code; order matters; first matching substring wins.
// Each entry's cause/fix is written for someone with no Kubernetes background,
// per the "educational hover tooltips" deliverable in #12843.
const POD_FAILURE_PATTERNS: PodFailurePattern[] = [
  {
    substring: 'ImagePullBackOff',
    category: PodFailureCategory.PROVISIONING,
    cause:
      "Kubernetes can't download your container image — wrong name/tag, a private registry, or the registry is down.",
    fix: 'Check the image name/tag on this component for typos, and confirm the image is public or the cluster has registry credentials.',
  },
  {
    substring: 'ErrImagePull',
    category: PodFailureCategory.PROVISIONING,
    cause: 'Same as above, but this is the first failed attempt rather than the repeated backoff.',
    fix: 'Same fix as ImagePullBackOff; this can also resolve itself on retry if it was a transient network blip.',
  },
  {
    substring: 'ErrImageNeverPull',
    category: PodFailureCategory.PROVISIONING,
    cause:
      'The pod is set to never download the image and expects it to already exist on the machine.',
    fix: 'Remove any imagePullPolicy: Never setting unless the image is pre-loaded on every node.',
  },
  {
    substring: 'InvalidImageName',
    category: PodFailureCategory.PROVISIONING,
    cause: 'The image reference itself is malformed, not a pull failure.',
    fix: 'Check base_image on the component for invalid characters or syntax.',
  },
  {
    // Checked before the generic Unschedulable entry below, since the scheduler's own message
    // contains both substrings and the more specific one should win.
    substring: 'Insufficient nvidia.com/gpu',
    category: PodFailureCategory.PROVISIONING,
    cause:
      'No node in the cluster currently has a free GPU matching what this component requested.',
    fix: 'Check whether the requested GPU type or count actually exists in this cluster, or whether GPU node pool autoscaling is enabled.',
  },
  {
    // A scheduling predicate failure, not a resource-capacity failure, so it needs its own entry
    // ahead of the generic Unschedulable one below: it does not reliably also contain the word
    // "Unschedulable" the way a resource-capacity message does.
    substring: 'unbound immediate PersistentVolumeClaims',
    category: PodFailureCategory.PROVISIONING,
    cause: 'This task references a PersistentVolumeClaim that has not been bound to storage yet.',
    fix: 'Check the PVC name and StorageClass this task references (for example through kubernetes.mount_pvc), and confirm the PVC exists and can be provisioned.',
  },
  {
    substring: 'Unschedulable',
    category: PodFailureCategory.PROVISIONING,
    cause:
      'No machine in the cluster currently has enough free CPU/memory/GPU for this task, or a placement rule excludes all of them.',
    fix: "Lower the component's requested cpu/memory/accelerator count, or ask a cluster admin for more capacity.",
  },
  {
    substring: 'CrashLoopBackOff',
    category: PodFailureCategory.RUNTIME,
    cause:
      'Your container starts and then exits, repeatedly — almost always a bug that fires immediately on startup.',
    fix: 'Read the container logs for the exception/stack trace at startup; this is a code bug, not infrastructure.',
  },
  {
    substring: 'OOMKilled',
    category: PodFailureCategory.RUNTIME,
    cause: 'Your code used more memory than the task was allowed.',
    fix: "Increase the component's memory request/limit, or reduce memory usage (smaller batch size, stream instead of loading everything at once).",
  },
  {
    substring: 'DeadlineExceeded',
    category: PodFailureCategory.RUNTIME,
    cause: 'The task ran longer than its configured time limit.',
    fix: 'Increase the task/pipeline timeout, or investigate why this step is slower than expected.',
  },
  {
    substring: 'ContainerCannotRun',
    category: PodFailureCategory.RUNTIME,
    cause:
      "The container runtime couldn't even start the container — a bad entrypoint/command or a permissions problem.",
    fix: "Verify the entrypoint/command in the component definition, and confirm the image's executable has run permissions.",
  },
  {
    substring: 'CreateContainerConfigError',
    category: PodFailureCategory.RUNTIME,
    cause:
      "Usually means a referenced Kubernetes Secret or ConfigMap doesn't exist, or is missing an expected key.",
    fix: 'Check that any secret/configmap this task references actually exists in the target namespace.',
  },
  {
    substring: 'CreateContainerError',
    category: PodFailureCategory.RUNTIME,
    cause: 'Low-level container setup failure — often a bad volume mount or security setting.',
    fix: 'Check any custom pod-spec patches applied via kfp-kubernetes (volumes, security context).',
  },
  {
    substring: 'RunContainerError',
    category: PodFailureCategory.RUNTIME,
    cause: 'The container was created but the runtime failed to execute it.',
    fix: 'Usually an image build problem (missing shared library, wrong CPU architecture) — check the image build, not the component code.',
  },
  {
    substring: 'NodeLost',
    category: PodFailureCategory.NODE,
    cause:
      'The machine running your task disappeared from the cluster (crashed, or was reclaimed).',
    fix: 'Nothing to fix in your code — just retry. If this happens often, ask a cluster admin whether nodes are being reclaimed too aggressively (e.g. spot/preemptible policy).',
  },
  {
    substring: 'Preempted',
    category: PodFailureCategory.NODE,
    cause: 'A higher-priority workload needed the resources your task was using.',
    fix: 'Nothing wrong with your code; retry, or request a higher priority class for critical runs.',
  },
  {
    // Raw Kubernetes condition/status Reason value for the same event
    // "Preempted" describes from Argo's rendered node message.
    substring: 'PreemptionByScheduler',
    category: PodFailureCategory.NODE,
    cause: 'A higher-priority workload needed the resources your task was using.',
    fix: 'Nothing wrong with your code; retry, or request a higher priority class for critical runs.',
  },
  {
    substring: 'Evicted',
    category: PodFailureCategory.NODE,
    cause: 'The machine was low on memory/disk and Kubernetes removed your task to protect it.',
    fix: "Reduce the task's memory/disk footprint, or ask a cluster admin about node capacity.",
  },
  {
    // Raw Kubernetes condition/status Reason value for the same event
    // "Evicted" describes from Argo's rendered node message.
    substring: 'TerminationByKubelet',
    category: PodFailureCategory.NODE,
    cause: 'The machine was low on memory/disk and Kubernetes removed your task to protect it.',
    fix: "Reduce the task's memory/disk footprint, or ask a cluster admin about node capacity.",
  },
];

export interface PodFailureClassification {
  category: PodFailureCategory;
  reason: string;
  cause: string;
  fix: string;
}

// classifyPodFailure inspects a raw Argo node message and classifies it into a
// PodFailureCategory with a plain-English cause and fix, or undefined if the
// message doesn't match a known pod lifecycle failure pattern.
export function classifyPodFailure(nodeMessage?: string): PodFailureClassification | undefined {
  if (!nodeMessage) {
    return undefined;
  }
  const pattern = POD_FAILURE_PATTERNS.find((p) => nodeMessage.includes(p.substring));
  if (!pattern) {
    return undefined;
  }
  return {
    category: pattern.category,
    reason: pattern.substring,
    cause: pattern.cause,
    fix: pattern.fix,
  };
}

// isPodLifecycleFailure returns true if nodeMessage looks like it was caused by
// a Kubernetes pod lifecycle issue rather than an error in the user's own
// pipeline code.
export function isPodLifecycleFailure(nodeMessage?: string): boolean {
  return classifyPodFailure(nodeMessage) !== undefined;
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
    : artifacts.some((artifact) => artifact.s3 && !artifact.s3.key.includes(node.id));
}

// separate these helper function for paritial v2 api integration
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
    case V2beta1RuntimeState.RUNTIME_STATE_UNSPECIFIED:
      return false;
    default:
      logger.warn('Unknown state:', state);
      throw new Error('Unexpected runtime state!');
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

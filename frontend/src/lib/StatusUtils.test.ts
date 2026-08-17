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

import {
  classifyPodFailure,
  isPodLifecycleFailure,
  NodePhase,
  podFailureCategoryBgColors,
  PodFailureCategory,
  statusBgColors,
  statusToBgColor,
} from './StatusUtils';

describe('classifyPodFailure', () => {
  it('returns undefined for an undefined message', () => {
    expect(classifyPodFailure(undefined)).toBeUndefined();
  });

  it('returns undefined for an empty message', () => {
    expect(classifyPodFailure('')).toBeUndefined();
  });

  it('returns undefined for an ordinary user pipeline error', () => {
    expect(classifyPodFailure('exit status 1: division by zero in user script')).toBeUndefined();
  });

  const cases: Array<[string, string, PodFailureCategory]> = [
    [
      'Failed to pull image "bad-tag:latest": ImagePullBackOff',
      'ImagePullBackOff',
      PodFailureCategory.PROVISIONING,
    ],
    ['rpc error: ErrImagePull: failed to pull', 'ErrImagePull', PodFailureCategory.PROVISIONING],
    [
      'ErrImageNeverPull: image not present locally',
      'ErrImageNeverPull',
      PodFailureCategory.PROVISIONING,
    ],
    [
      'InvalidImageName: could not parse reference',
      'InvalidImageName',
      PodFailureCategory.PROVISIONING,
    ],
    [
      '0/1 nodes are available: 1 Insufficient cpu. Unschedulable',
      'Unschedulable',
      PodFailureCategory.PROVISIONING,
    ],
    [
      'back-off restarting failed container: CrashLoopBackOff',
      'CrashLoopBackOff',
      PodFailureCategory.RUNTIME,
    ],
    [
      'container terminated with reason OOMKilled, exit code 137',
      'OOMKilled',
      PodFailureCategory.RUNTIME,
    ],
    [
      'pod ci-step exceeded active deadline: DeadlineExceeded',
      'DeadlineExceeded',
      PodFailureCategory.RUNTIME,
    ],
    ['ContainerCannotRun: exec format error', 'ContainerCannotRun', PodFailureCategory.RUNTIME],
    [
      'CreateContainerConfigError: secret "foo" not found',
      'CreateContainerConfigError',
      PodFailureCategory.RUNTIME,
    ],
    [
      'CreateContainerError: failed to create containerd task',
      'CreateContainerError',
      PodFailureCategory.RUNTIME,
    ],
    [
      'RunContainerError: failed to start container',
      'RunContainerError',
      PodFailureCategory.RUNTIME,
    ],
    ['node has been marked NodeLost by the controller', 'NodeLost', PodFailureCategory.NODE],
    [
      'pod was Preempted to make room for a higher priority pod',
      'Preempted',
      PodFailureCategory.NODE,
    ],
    ['pod Evicted due to node memory pressure', 'Evicted', PodFailureCategory.NODE],
  ];

  it.each(cases)('classifies %j as reason %j in category %j', (message, reason, category) => {
    const result = classifyPodFailure(message);
    expect(result).toBeDefined();
    expect(result!.reason).toBe(reason);
    expect(result!.category).toBe(category);
    expect(result!.cause.length).toBeGreaterThan(0);
    expect(result!.fix.length).toBeGreaterThan(0);
  });

  it('matches the first pattern when a message contains multiple substrings', () => {
    const result = classifyPodFailure('ImagePullBackOff eventually led to OOMKilled during retry');
    expect(result?.reason).toBe('ImagePullBackOff');
    expect(result?.category).toBe(PodFailureCategory.PROVISIONING);
  });
});

describe('isPodLifecycleFailure', () => {
  it('returns true for a recognized pod lifecycle failure message', () => {
    expect(isPodLifecycleFailure('back-off restarting failed container: CrashLoopBackOff')).toBe(
      true,
    );
  });

  it('returns false for an unrecognized message', () => {
    expect(isPodLifecycleFailure('exit status 1: division by zero in user script')).toBe(false);
  });

  it('returns false for an undefined message', () => {
    expect(isPodLifecycleFailure(undefined)).toBe(false);
  });
});

describe('statusToBgColor with pod failure categories', () => {
  it('returns the Provisioning color for an Unschedulable failure', () => {
    const color = statusToBgColor(NodePhase.FAILED, '0/1 nodes are available: Unschedulable');
    expect(color).toBe(podFailureCategoryBgColors[PodFailureCategory.PROVISIONING]);
  });

  it('returns the Runtime color for an OOMKilled failure', () => {
    const color = statusToBgColor(NodePhase.FAILED, 'container terminated with reason OOMKilled');
    expect(color).toBe(podFailureCategoryBgColors[PodFailureCategory.RUNTIME]);
  });

  it('returns the Node color for a Preempted failure', () => {
    const color = statusToBgColor(NodePhase.FAILED, 'pod was Preempted to make room');
    expect(color).toBe(podFailureCategoryBgColors[PodFailureCategory.NODE]);
  });

  it('falls back to the generic error color for an unrecognized failure message', () => {
    const color = statusToBgColor(
      NodePhase.FAILED,
      'exit status 1: division by zero in user script',
    );
    expect(color).toBe(statusBgColors.error);
  });

  it('falls back to the generic error color when there is no message', () => {
    const color = statusToBgColor(NodePhase.FAILED, undefined);
    expect(color).toBe(statusBgColors.error);
  });

  it('the three category colors are distinct from each other', () => {
    // Runtime intentionally reuses statusBgColors.error, since OOMKilled/CrashLoopBackOff
    // etc. are genuinely code-side failures, same as an uncategorized pipeline error.
    const colors = Object.values(podFailureCategoryBgColors);
    expect(new Set(colors).size).toBe(colors.length);
  });
});

/*
 * Copyright 2018 The Kubeflow Authors
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
  NodePhase,
  hasFinished,
  statusBgColors,
  statusToBgColor,
  checkIfTerminated,
  parseNodePhase,
  parsePodLifecycleFailure,
  getPodDiagnosticSummary,
} from './StatusUtils';
import { NodeStatus, S3Artifact, Artifact } from 'third_party/argo-ui/argo_template';

describe('StatusUtils', () => {
  describe('hasFinished', () => {
    [
      NodePhase.ERROR,
      NodePhase.FAILED,
      NodePhase.SUCCEEDED,
      NodePhase.CACHED,
      NodePhase.SKIPPED,
      NodePhase.TERMINATED,
      NodePhase.OMITTED,
    ].forEach((status) => {
      it(`returns \'true\' if status is: ${status}`, () => {
        expect(hasFinished(status)).toBe(true);
      });
    });

    [NodePhase.PENDING, NodePhase.RUNNING, NodePhase.UNKNOWN, NodePhase.TERMINATING].forEach(
      (status) => {
        it(`returns \'false\' if status is: ${status}`, () => {
          expect(hasFinished(status)).toBe(false);
        });
      },
    );

    it("returns 'false' if status is undefined", () => {
      expect(hasFinished(undefined)).toBe(false);
    });

    it("returns 'false' if status is invalid", () => {
      expect(hasFinished('bad phase' as any)).toBe(false);
    });
  });

  describe('statusToBgColor', () => {
    it('handles an invalid phase', () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementationOnce(() => null);
      expect(statusToBgColor('bad phase' as any)).toEqual(statusBgColors.notStarted);
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'bad phase');
    });

    it("handles an 'Unknown' phase", () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementationOnce(() => null);
      expect(statusToBgColor(NodePhase.UNKNOWN)).toEqual(statusBgColors.notStarted);
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'Unknown');
    });

    it("returns color 'not started' if status is undefined", () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementationOnce(() => null);
      expect(statusToBgColor(undefined)).toEqual(statusBgColors.notStarted);
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', undefined);
    });

    it("returns color 'not started' if status is 'Omitted'", () => {
      expect(statusToBgColor(NodePhase.OMITTED)).toEqual(statusBgColors.notStarted);
    });

    it("returns color 'not started' if status is 'Pending'", () => {
      expect(statusToBgColor(NodePhase.PENDING)).toEqual(statusBgColors.notStarted);
    });

    [NodePhase.ERROR, NodePhase.FAILED].forEach((status) => {
      it(`returns color \'error\' if status is: ${status}`, () => {
        expect(statusToBgColor(status)).toEqual(statusBgColors.error);
      });
    });

    [NodePhase.RUNNING, NodePhase.TERMINATING].forEach((status) => {
      it(`returns color \'running\' if status is: ${status}`, () => {
        expect(statusToBgColor(status)).toEqual(statusBgColors.running);
      });
    });

    [NodePhase.SKIPPED, NodePhase.TERMINATED].forEach((status) => {
      it(`returns color \'terminated or skipped\' if status is: ${status}`, () => {
        expect(statusToBgColor(status)).toEqual(statusBgColors.terminatedOrSkipped);
      });
    });

    [NodePhase.SUCCEEDED, NodePhase.CACHED].forEach((status) => {
      it(`returns color 'succeeded' if status is '${status}'`, () => {
        expect(statusToBgColor(status)).toEqual(statusBgColors.succeeded);
      });
    });
  });

  describe('checkIfTerminated', () => {
    it("returns status 'terminated' if status is 'failed' and error message is 'terminated'", () => {
      expect(checkIfTerminated(NodePhase.FAILED, 'terminated')).toEqual(NodePhase.TERMINATED);
    });

    [
      NodePhase.SUCCEEDED,
      NodePhase.ERROR,
      NodePhase.SKIPPED,
      NodePhase.PENDING,
      NodePhase.RUNNING,
      NodePhase.TERMINATING,
      NodePhase.OMITTED,
      NodePhase.UNKNOWN,
    ].forEach((status) => {
      it(`returns the original status, even if message is 'terminated', if status is: ${status}`, () => {
        expect(checkIfTerminated(status, 'terminated')).toEqual(status);
      });
    });

    it("returns 'failed' if status is 'failed' and no error message is provided", () => {
      expect(checkIfTerminated(NodePhase.FAILED)).toEqual(NodePhase.FAILED);
    });

    it("returns 'failed' if status is 'failed' and empty error message is provided", () => {
      expect(checkIfTerminated(NodePhase.FAILED, '')).toEqual(NodePhase.FAILED);
    });

    it("returns 'failed' if status is 'failed' and arbitrary error message is provided", () => {
      expect(checkIfTerminated(NodePhase.FAILED, 'some random error')).toEqual(NodePhase.FAILED);
    });
  });

  describe('parseNodePhase', () => {
    const DEFAULT_NODE_STATUS = {
      phase: 'Succeeded',
      id: 'file-passing-pipelines-55slt-2894085459',
      outputs: {
        artifacts: [
          {
            s3: {
              key: 'artifacts/file-passing-pipelines-55slt/file-passing-pipelines-55slt-2894085459/sum-numbers-output.tgz',
            },
          } as unknown as Artifact,
        ],
      },
    } as unknown as NodeStatus;

    it('returns node original phase if not successful', () => {
      expect(
        parseNodePhase({
          ...DEFAULT_NODE_STATUS,
          phase: 'Failed',
        }),
      ).toEqual('Failed');
    });

    it('returns succeeded phase for a normal node', () => {
      expect(
        parseNodePhase({
          ...DEFAULT_NODE_STATUS,
          phase: 'Succeeded',
        }),
      ).toEqual('Succeeded');
    });

    it('returns cached phase for a cached node', () => {
      expect(
        parseNodePhase({
          ...DEFAULT_NODE_STATUS,
          id: 'file-passing-pipelines-55slt-2894085459',
          type: 'Pod',
          phase: 'Succeeded', // Cached nodes have phase == 'Succeeded'
          outputs: {
            artifacts: [
              {
                s3: {
                  // HACK: A cached node's artifacts will refer to a path that doesn't match its own id.
                  key: 'artifacts/file-passing-pipelines-mjpph/file-passing-pipelines-mjpph-1802581193/sum-numbers-output.tgz',
                },
              } as Artifact,
            ],
          },
        }),
      ).toEqual('Cached');
    });

    it('returns succeeded phase for a retry node', () => {
      expect(
        parseNodePhase({
          ...DEFAULT_NODE_STATUS,
          id: 'file-passing-pipelines-55slt-2894085459',
          type: 'Retry',
          phase: 'Succeeded', // Cached nodes have phase == 'Succeeded'
          outputs: {
            artifacts: [
              {
                s3: {
                  // HACK: A cached node's artifacts will refer to a path that doesn't match its own id.
                  key: 'artifacts/file-passing-pipelines-mjpph/file-passing-pipelines-mjpph-1802581193/sum-numbers-output.tgz',
                },
              } as Artifact,
            ],
          },
        }),
      ).toEqual('Succeeded');
    });
  });

  describe('parsePodLifecycleFailure', () => {
    it('returns null for empty or undefined message', () => {
      expect(parsePodLifecycleFailure(undefined)).toBeNull();
      expect(parsePodLifecycleFailure('')).toBeNull();
    });

    it('identifies OOMKilled failures', () => {
      expect(parsePodLifecycleFailure('Command failed: OOMKilled')).toEqual('OOMKilled');
      expect(parsePodLifecycleFailure('Container process terminated: Out of Memory')).toEqual(
        'OOMKilled',
      );
    });

    it('identifies ImagePullBackOff failures', () => {
      expect(
        parsePodLifecycleFailure('Back-off pulling image "gcr.io/my-proj/image:v1": ImagePullBackOff'),
      ).toEqual('ImagePullBackOff');
    });

    it('identifies ErrImagePull failures', () => {
      expect(parsePodLifecycleFailure('Error: Image pull failed')).toEqual('ErrImagePull');
      expect(parsePodLifecycleFailure('ErrImagePull')).toEqual('ErrImagePull');
    });

    it('identifies CrashLoopBackOff failures', () => {
      expect(
        parsePodLifecycleFailure('back-off restarting failed container main in pod'),
      ).toEqual('CrashLoopBackOff');
      expect(parsePodLifecycleFailure('CrashLoopBackOff')).toEqual('CrashLoopBackOff');
    });

    it('identifies NodeLost failures', () => {
      expect(parsePodLifecycleFailure('Node lost connection to API server')).toEqual('NodeLost');
      expect(parsePodLifecycleFailure('NodeLost')).toEqual('NodeLost');
    });

    it('identifies Evicted failures', () => {
      expect(parsePodLifecycleFailure('Pod evicted due to disk pressure')).toEqual('Evicted');
    });

    it('returns null for unrecognized messages', () => {
      expect(parsePodLifecycleFailure('Generic task failure message')).toBeNull();
    });
  });

  describe('getPodDiagnosticSummary', () => {
    it('returns OOMKilled diagnostic guidance', () => {
      const summary = getPodDiagnosticSummary('OOMKilled');
      expect(summary.title).toContain('Out of Memory');
      expect(summary.suggestion).toContain('memory requests/limits');
    });

    it('returns ImagePullBackOff diagnostic guidance', () => {
      const summary = getPodDiagnosticSummary('ImagePullBackOff');
      expect(summary.title).toContain('Image Pull Failure');
      expect(summary.suggestion).toContain('imagePullSecrets');
    });

    it('returns CrashLoopBackOff diagnostic guidance', () => {
      const summary = getPodDiagnosticSummary('CrashLoopBackOff');
      expect(summary.title).toContain('Crash Loop');
      expect(summary.suggestion).toContain('execution logs');
    });

    it('returns fallback summary for unknown failures', () => {
      const summary = getPodDiagnosticSummary('UnknownFailure');
      expect(summary.title).toEqual('Pipeline Step Failure');
    });
  });
});


// Copyright 2024 The Kubeflow Authors
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
import { vi, describe, it, expect, afterEach, beforeEach } from 'vitest';

// Use vi.hoisted so the mock fns are available when vi.mock factory runs (it's hoisted above imports)
const { mockListNamespacedPod, mockListNamespacedEvent } = vi.hoisted(() => ({
  mockListNamespacedPod: vi.fn(),
  mockListNamespacedEvent: vi.fn(),
}));

vi.mock('./k8s-helper.js', () => ({
  TEST_ONLY: {
    k8sV1Client: {
      listNamespacedPod: mockListNamespacedPod,
      listNamespacedEvent: mockListNamespacedEvent,
    },
  },
}));

vi.mock('./pod-events-cache.js', () => ({
  getCachedPodInfo: vi.fn().mockResolvedValue(null),
  savePodInfo: vi.fn().mockResolvedValue(undefined),
}));

import { TEST_ONLY as WATCHER_TEST_EXPORT } from './pod-watcher.js';

describe('pod-watcher', () => {
  describe('getTaskNameFromPod', () => {
    it('prefers task_path annotation over task_name label', () => {
      const pod = {
        metadata: {
          labels: {
            'pipelines.kubeflow.org/task_name': 'print-op1',
            component: 'print-op1',
          },
          annotations: {
            'pipelines.kubeflow.org/task_path': 'root.comp-inner-pipeline.print-op1',
          },
        },
      } as any;
      expect(WATCHER_TEST_EXPORT.getTaskNameFromPod(pod)).toBe(
        'root.comp-inner-pipeline.print-op1',
      );
    });

    it('falls back to task_name label when task_path is absent', () => {
      const pod = {
        metadata: {
          labels: {
            'pipelines.kubeflow.org/task_name': 'print-op1',
            component: 'print-op1',
          },
          annotations: {},
        },
      } as any;
      expect(WATCHER_TEST_EXPORT.getTaskNameFromPod(pod)).toBe('print-op1');
    });

    it('falls back to component label when both annotation and task_name are absent', () => {
      const pod = {
        metadata: {
          labels: {
            component: 'train-model',
          },
          annotations: {},
        },
      } as any;
      expect(WATCHER_TEST_EXPORT.getTaskNameFromPod(pod)).toBe('train-model');
    });

    it('returns undefined when no labels or annotations match', () => {
      const pod = {
        metadata: {
          labels: {},
          annotations: {},
        },
      } as any;
      expect(WATCHER_TEST_EXPORT.getTaskNameFromPod(pod)).toBeUndefined();
    });
  });

  describe('getRunIdFromPod', () => {
    it('returns pipeline/runid label', () => {
      const pod = {
        metadata: {
          labels: { 'pipeline/runid': 'run-123' },
        },
      } as any;
      expect(WATCHER_TEST_EXPORT.getRunIdFromPod(pod)).toBe('run-123');
    });

    it('falls back to pipelines.kubeflow.org/run_id label', () => {
      const pod = {
        metadata: {
          labels: { 'pipelines.kubeflow.org/run_id': 'run-456' },
        },
      } as any;
      expect(WATCHER_TEST_EXPORT.getRunIdFromPod(pod)).toBe('run-456');
    });
  });

  describe('listRunningPipelinePods pagination', () => {
    beforeEach(() => {
      mockListNamespacedPod.mockReset();
      vi.spyOn(console, 'error').mockImplementation(() => null);
      vi.spyOn(console, 'log').mockImplementation(() => null);
    });

    afterEach(() => {
      vi.restoreAllMocks();
    });

    it('follows continuation tokens across multiple pages', async () => {
      const makePod = (name: string) => ({
        metadata: { name, namespace: 'test-ns' },
        status: { phase: 'Running' },
      });

      // Page 1: 100 pods with a continue token
      mockListNamespacedPod.mockResolvedValueOnce({
        items: Array.from({ length: 100 }, (_, i) => makePod(`pod-${i}`)),
        metadata: { _continue: 'token-page-2' },
      });
      // Page 2: 50 pods, no more pages
      mockListNamespacedPod.mockResolvedValueOnce({
        items: Array.from({ length: 50 }, (_, i) => makePod(`pod-${100 + i}`)),
        metadata: {},
      });

      const pods = await WATCHER_TEST_EXPORT.listRunningPipelinePods('test-ns');

      expect(pods).toHaveLength(150);
      expect(mockListNamespacedPod).toHaveBeenCalledTimes(2);

      // Verify second call used the continue token
      expect(mockListNamespacedPod).toHaveBeenNthCalledWith(
        2,
        expect.objectContaining({ _continue: 'token-page-2' }),
      );
    });

    it('stops at MAX_PAGES safety cap', async () => {
      const makePod = (name: string) => ({
        metadata: { name, namespace: 'test-ns' },
        status: { phase: 'Running' },
      });

      // Return a continue token on every page
      mockListNamespacedPod.mockImplementation(() =>
        Promise.resolve({
          items: [makePod(`pod-${Math.random()}`)],
          metadata: { _continue: 'next-token' },
        }),
      );

      const pods = await WATCHER_TEST_EXPORT.listRunningPipelinePods('test-ns');

      // MAX_PAGES is 20, so we should get exactly 20 pods (one per page)
      expect(pods).toHaveLength(20);
      expect(mockListNamespacedPod).toHaveBeenCalledTimes(20);
    });

    it('falls back to unfiltered listing when field selector fails', async () => {
      const makePod = (name: string, phase: string) => ({
        metadata: { name, namespace: 'test-ns' },
        status: { phase },
      });

      // First call (with fieldSelector) fails
      mockListNamespacedPod.mockRejectedValueOnce(new Error('field selector not supported'));
      // Fallback call (no fieldSelector) succeeds with mixed phases
      mockListNamespacedPod.mockResolvedValueOnce({
        items: [
          makePod('running-pod', 'Running'),
          makePod('pending-pod', 'Pending'),
          makePod('succeeded-pod', 'Succeeded'),
        ],
        metadata: {},
      });

      const pods = await WATCHER_TEST_EXPORT.listRunningPipelinePods('test-ns');

      // Only Running and Pending pods should be returned in fallback mode
      expect(pods).toHaveLength(2);
      expect(pods.map((p: any) => p.metadata.name)).toContain('running-pod');
      expect(pods.map((p: any) => p.metadata.name)).toContain('pending-pod');
    });

    it('returns empty array when both listing strategies fail', async () => {
      mockListNamespacedPod.mockRejectedValue(new Error('forbidden'));

      const pods = await WATCHER_TEST_EXPORT.listRunningPipelinePods('test-ns');

      expect(pods).toHaveLength(0);
    });
  });
});

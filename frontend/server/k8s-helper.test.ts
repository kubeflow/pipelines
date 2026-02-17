// Copyright 2020 The Kubeflow Authors
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
import { vi, describe, it, expect, afterEach, beforeEach, Mock, SpyInstance } from 'vitest';
import {
  TEST_ONLY as K8S_TEST_EXPORT,
  getPodLogs,
  getPod,
  getConfigMap,
  listPodEvents,
  getArgoWorkflow,
  getK8sSecret,
} from './k8s-helper.js';

describe('k8s-helper', () => {
  describe('parseTensorboardLogDir', () => {
    const podTemplateSpec = {
      spec: {
        containers: [
          {
            volumeMounts: [
              {
                name: 'output',
                mountPath: '/data',
              },
              {
                name: 'artifact',
                subPath: 'pipeline1',
                mountPath: '/data1',
              },
              {
                name: 'artifact',
                subPath: 'pipeline2',
                mountPath: '/data2',
              },
            ],
          },
        ],
        volumes: [
          {
            name: 'output',
            hostPath: {
              path: '/data/output',
              type: 'Directory',
            },
          },
          {
            name: 'artifact',
            persistentVolumeClaim: {
              claimName: 'artifact_pvc',
            },
          },
        ],
      },
    };

    it('handles not volume storage', () => {
      const logdir = 'gs://testbucket/test/key/path';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual(logdir);
    });

    it('handles not volume storage with Series', () => {
      const logdir =
        'Series1:gs://testbucket/test/key/path1,Series2:gs://testbucket/test/key/path2';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual(logdir);
    });

    it('handles volume storage without subPath', () => {
      const logdir = 'volume://output';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual('/data');
    });

    it('handles volume storage without subPath with Series', () => {
      const logdir = 'Series1:volume://output/volume/path1,Series2:volume://output/volume/path2';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual('Series1:/data/volume/path1,Series2:/data/volume/path2');
    });

    it('handles volume storage with subPath', () => {
      const logdir = 'volume://artifact/pipeline1';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual('/data1');
    });

    it('handles volume storage with subPath with Series', () => {
      const logdir =
        'Series1:volume://artifact/pipeline1/path1,Series2:volume://artifact/pipeline2/path2';
      const url = K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec);
      expect(url).toEqual('Series1:/data1/path1,Series2:/data2/path2');
    });

    it('handles volume storage without subPath throw volume not configured error', () => {
      const logdir = 'volume://other/path';
      expect(() => K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec)).toThrowError(
        'Cannot find file "volume://other/path" in pod "unknown": volume "other" not configured',
      );
    });

    it('handles volume storage without subPath throw volume not configured error with Series', () => {
      const logdir = 'Series1:volume://output/volume/path1,Series2:volume://other/volume/path2';
      expect(() => K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec)).toThrowError(
        'Cannot find file "volume://other/volume/path2" in pod "unknown": volume "other" not configured',
      );
    });

    it('handles volume storage without subPath throw volume not mounted', () => {
      const noMountPodTemplateSpec = {
        spec: {
          volumes: [
            {
              name: 'artifact',
              persistentVolumeClaim: {
                claimName: 'artifact_pvc',
              },
            },
          ],
        },
      };
      const logdir = 'volume://artifact/path1';
      expect(() =>
        K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, noMountPodTemplateSpec),
      ).toThrowError(
        'Cannot find file "volume://artifact/path1" in pod "unknown": container "" not found',
      );
    });

    it('handles volume storage without volumeMounts throw volume not mounted', () => {
      const noMountPodTemplateSpec = {
        spec: {
          containers: [
            {
              volumeMounts: [
                {
                  name: 'other',
                  mountPath: '/data',
                },
              ],
            },
          ],
          volumes: [
            {
              name: 'artifact',
              persistentVolumeClaim: {
                claimName: 'artifact_pvc',
              },
            },
          ],
        },
      };
      const logdir = 'volume://artifact/path';
      expect(() => K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec)).toThrowError(
        'Cannot find file "volume://artifact/path" in pod "unknown": volume "artifact" not mounted',
      );
    });

    it('handles volume storage with subPath throw volume mount not found', () => {
      const logdir = 'volume://artifact/other';
      expect(() => K8S_TEST_EXPORT.parseTensorboardLogDir(logdir, podTemplateSpec)).toThrowError(
        'Cannot find file "volume://artifact/other" in pod "unknown": volume "artifact" not mounted or volume "artifact" with subPath (which is prefix of other) not mounted',
      );
    });
  });

  describe('getPodLogs', () => {
    let readNamespacedPodLogSpy: SpyInstance;

    beforeEach(() => {
      readNamespacedPodLogSpy = vi.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'readNamespacedPodLog');
    });

    afterEach(() => {
      readNamespacedPodLogSpy.mockRestore();
    });

    it('returns pod logs when successful', async () => {
      const mockLogs = 'container log output';
      readNamespacedPodLogSpy.mockResolvedValue(mockLogs);

      const logs = await getPodLogs('test-pod', 'test-namespace', 'main');

      expect(readNamespacedPodLogSpy).toHaveBeenCalledWith({
        name: 'test-pod',
        namespace: 'test-namespace',
        container: 'main',
      });
      expect(logs).toBe(mockLogs);
    });

    it('uses default container name "main" when not specified', async () => {
      readNamespacedPodLogSpy.mockResolvedValue('logs');

      await getPodLogs('test-pod', 'test-namespace');

      expect(readNamespacedPodLogSpy).toHaveBeenCalledWith({
        name: 'test-pod',
        namespace: 'test-namespace',
        container: 'main',
      });
    });

    it('returns empty string when response is empty', async () => {
      readNamespacedPodLogSpy.mockResolvedValue('');

      const logs = await getPodLogs('test-pod', 'test-namespace');

      expect(logs).toBe('');
    });

    it('throws error when API call fails', async () => {
      const errorBody = { message: 'Pod not found', code: 404 };
      readNamespacedPodLogSpy.mockRejectedValue({ body: errorBody });

      await expect(getPodLogs('nonexistent-pod', 'test-namespace')).rejects.toThrow(
        JSON.stringify(errorBody),
      );
    });
  });

  describe('getPod', () => {
    let readNamespacedPodSpy: SpyInstance;

    beforeEach(() => {
      readNamespacedPodSpy = vi.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'readNamespacedPod');
    });

    afterEach(() => {
      readNamespacedPodSpy.mockRestore();
    });

    it('returns pod when found', async () => {
      const mockPod = {
        metadata: { name: 'test-pod', namespace: 'test-namespace' },
        spec: { containers: [] },
        status: { phase: 'Running' },
      };
      readNamespacedPodSpy.mockResolvedValue(mockPod);

      const [pod, error] = await getPod('test-pod', 'test-namespace');

      expect(readNamespacedPodSpy).toHaveBeenCalledWith({
        name: 'test-pod',
        namespace: 'test-namespace',
      });
      expect(pod).toEqual(mockPod);
      expect(error).toBeUndefined();
    });

    it('returns error when pod not found', async () => {
      readNamespacedPodSpy.mockRejectedValue({ body: { message: 'not found' } });

      const [pod, error] = await getPod('nonexistent-pod', 'test-namespace');

      expect(pod).toBeUndefined();
      expect(error).toEqual({
        message: 'Could not get pod nonexistent-pod in namespace test-namespace',
      });
    });

    it('returns invalid resource name error for invalid pod name', async () => {
      readNamespacedPodSpy.mockRejectedValue({ body: { message: 'invalid' } });

      const [pod, error] = await getPod('../invalid-name', 'test-namespace');

      expect(pod).toBeUndefined();
      expect(error).toEqual({ message: 'Invalid resource name' });
    });

    it('returns invalid resource name error for invalid namespace', async () => {
      readNamespacedPodSpy.mockRejectedValue({ body: { message: 'invalid' } });

      const [pod, error] = await getPod('test-pod', '../invalid-namespace');

      expect(pod).toBeUndefined();
      expect(error).toEqual({ message: 'Invalid resource name' });
    });
  });

  describe('getConfigMap', () => {
    let readNamespacedConfigMapSpy: SpyInstance;

    beforeEach(() => {
      readNamespacedConfigMapSpy = vi.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'readNamespacedConfigMap');
    });

    afterEach(() => {
      readNamespacedConfigMapSpy.mockRestore();
    });

    it('returns configmap when found', async () => {
      const mockConfigMap = {
        metadata: { name: 'test-config', namespace: 'test-namespace' },
        data: { key1: 'value1', key2: 'value2' },
      };
      readNamespacedConfigMapSpy.mockResolvedValue(mockConfigMap);

      const [configMap, error] = await getConfigMap('test-config', 'test-namespace');

      expect(readNamespacedConfigMapSpy).toHaveBeenCalledWith({
        name: 'test-config',
        namespace: 'test-namespace',
      });
      expect(configMap).toEqual(mockConfigMap);
      expect(error).toBeUndefined();
    });

    it('returns error when configmap not found', async () => {
      readNamespacedConfigMapSpy.mockRejectedValue({ body: { message: 'not found' } });

      const [configMap, error] = await getConfigMap('nonexistent', 'test-namespace');

      expect(configMap).toBeUndefined();
      expect(error).toEqual({
        message: 'Could not get configMap nonexistent in namespace test-namespace',
      });
    });

    it('returns invalid resource name error for invalid configmap name', async () => {
      readNamespacedConfigMapSpy.mockRejectedValue({ body: { message: 'invalid' } });

      const [configMap, error] = await getConfigMap('../invalid', 'test-namespace');

      expect(configMap).toBeUndefined();
      expect(error).toEqual({ message: 'Invalid resource name' });
    });

    it('returns invalid resource name error for invalid namespace', async () => {
      readNamespacedConfigMapSpy.mockRejectedValue({ body: { message: 'invalid' } });

      const [configMap, error] = await getConfigMap('test-config', '../invalid');

      expect(configMap).toBeUndefined();
      expect(error).toEqual({ message: 'Invalid resource name' });
    });
  });

  describe('listPodEvents', () => {
    let listNamespacedEventSpy: SpyInstance;

    beforeEach(() => {
      listNamespacedEventSpy = vi.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'listNamespacedEvent');
    });

    afterEach(() => {
      listNamespacedEventSpy.mockRestore();
    });

    it('returns events when successful', async () => {
      const mockEvents = {
        items: [
          { message: 'Created container', reason: 'Created' },
          { message: 'Started container', reason: 'Started' },
        ],
      };
      listNamespacedEventSpy.mockResolvedValue(mockEvents);

      const [events, error] = await listPodEvents('test-pod', 'test-namespace');

      expect(listNamespacedEventSpy).toHaveBeenCalledWith({
        namespace: 'test-namespace',
        fieldSelector:
          'involvedObject.namespace=test-namespace,involvedObject.name=test-pod,involvedObject.kind=Pod',
      });
      expect(events).toEqual(mockEvents);
      expect(error).toBeUndefined();
    });

    it('returns error when API call fails', async () => {
      listNamespacedEventSpy.mockRejectedValue({ body: { message: 'forbidden' } });

      const [events, error] = await listPodEvents('test-pod', 'test-namespace');

      expect(events).toBeUndefined();
      expect(error).toEqual({
        message: 'Error when listing pod events for pod test-pod in namespace test-namespace',
      });
    });

    it('returns invalid resource name error for invalid pod name', async () => {
      listNamespacedEventSpy.mockRejectedValue({ body: { message: 'invalid' } });

      const [events, error] = await listPodEvents('../invalid', 'test-namespace');

      expect(events).toBeUndefined();
      expect(error).toEqual({ message: 'Invalid resource name' });
    });

    it('returns invalid resource name error for invalid namespace', async () => {
      listNamespacedEventSpy.mockRejectedValue({ body: { message: 'invalid' } });

      const [events, error] = await listPodEvents('test-pod', '../invalid');

      expect(events).toBeUndefined();
      expect(error).toEqual({ message: 'Invalid resource name' });
    });
  });

  describe('getArgoWorkflow', () => {
    // Note: getArgoWorkflow requires serverNamespace which is read from
    // /var/run/secrets/kubernetes.io/serviceaccount/namespace at module load time.
    // In a non-cluster environment, this file doesn't exist so serverNamespace is undefined.
    // These tests verify the namespace validation behavior and error handling.

    it('throws error when serverNamespace is not available', async () => {
      // This test verifies behavior when running outside a kubernetes cluster
      await expect(getArgoWorkflow('test-workflow')).rejects.toThrow(
        'Cannot get namespace from /var/run/secrets/kubernetes.io/serviceaccount/namespace',
      );
    });
  });

  describe('getK8sSecret', () => {
    let readNamespacedSecretSpy: SpyInstance;

    beforeEach(() => {
      readNamespacedSecretSpy = vi.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'readNamespacedSecret');
    });

    afterEach(() => {
      readNamespacedSecretSpy.mockRestore();
    });

    it('returns decoded secret value when found', async () => {
      const secretValue = 'my-secret-value';
      const base64Value = Buffer.from(secretValue).toString('base64');
      readNamespacedSecretSpy.mockResolvedValue({
        data: { 'secret-key': base64Value },
      });

      const result = await getK8sSecret('test-secret', 'secret-key', 'test-namespace');

      expect(readNamespacedSecretSpy).toHaveBeenCalledWith({
        name: 'test-secret',
        namespace: 'test-namespace',
      });
      expect(result).toBe(secretValue);
    });

    it('returns empty string when secret key does not exist', async () => {
      readNamespacedSecretSpy.mockResolvedValue({
        data: { 'other-key': 'c29tZXRoaW5n' },
      });

      const result = await getK8sSecret('test-secret', 'nonexistent-key', 'test-namespace');

      expect(result).toBe('');
    });

    it('returns empty string when data is undefined', async () => {
      readNamespacedSecretSpy.mockResolvedValue({});

      const result = await getK8sSecret('test-secret', 'secret-key', 'test-namespace');

      expect(result).toBe('');
    });

    it('throws error when API call fails', async () => {
      readNamespacedSecretSpy.mockRejectedValue(new Error('API error'));

      await expect(getK8sSecret('test-secret', 'secret-key', 'test-namespace')).rejects.toThrow(
        'API error',
      );
    });

    it('throws error when no namespace is provided and serverNamespace is unavailable', async () => {
      // This test verifies behavior when running outside a kubernetes cluster
      // and no providedNamespace is given
      await expect(getK8sSecret('test-secret', 'secret-key')).rejects.toThrow(
        'Cannot get namespace from /var/run/secrets/kubernetes.io/serviceaccount/namespace',
      );
    });
  });
});

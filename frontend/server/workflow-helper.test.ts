// Copyright 2019 The Kubeflow Authors
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
import { vi, describe, it, expect, beforeEach, Mock } from 'vitest';
import { PassThrough } from 'stream';
import { Client as MinioClient } from 'minio';
import {
  createPodLogsMinioRequestConfig,
  composePodLogsStreamHandler,
  getPodLogsStreamFromK8s,
  getPodLogsStreamFromWorkflow,
  getPodLogsMinioRequestConfigfromWorkflow,
  toGetPodLogsStream,
  getKeyFormatFromArtifactRepositories,
} from './workflow-helper.js';
import {
  getK8sSecret,
  getArgoWorkflow,
  getPodLogs,
  getConfigMap,
  getServerNamespace,
} from './k8s-helper.js';
import { V1ConfigMap, V1ObjectMeta } from '@kubernetes/client-node';

vi.mock('minio');
vi.mock('./k8s-helper');

describe('workflow-helper', () => {
  const minioConfig = {
    accessKey: 'minio',
    endPoint: 'seaweedfs.kubeflow',
    secretKey: 'minio123',
  };

  beforeEach(() => {
    vi.resetAllMocks();
  });

  describe('composePodLogsStreamHandler', () => {
    it('returns the stream from the default handler if there is no errors.', async () => {
      const defaultStream = new PassThrough();
      const defaultHandler = vi.fn((_podName: string, _createdAt: string, _namespace?: string) =>
        Promise.resolve(defaultStream),
      );
      const stream = await composePodLogsStreamHandler(defaultHandler)(
        'podName',
        '2024-08-13',
        'namespace',
      );
      expect(defaultHandler).toBeCalledWith('podName', '2024-08-13', 'namespace');
      expect(stream).toBe(defaultStream);
    });

    it('returns the stream from the fallback handler if there is any error.', async () => {
      const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
      const fallbackStream = new PassThrough();
      const defaultHandler = vi.fn((_podName: string, _createdAt: string, _namespace?: string) =>
        Promise.reject('unknown error'),
      );
      const fallbackHandler = vi.fn((_podName: string, _createdAt: string, _namespace?: string) =>
        Promise.resolve(fallbackStream),
      );
      const stream = await composePodLogsStreamHandler(defaultHandler, fallbackHandler)(
        'podName',
        '2024-08-13',
        'namespace',
      );
      expect(defaultHandler).toBeCalledWith('podName', '2024-08-13', 'namespace');
      expect(fallbackHandler).toBeCalledWith('podName', '2024-08-13', 'namespace');
      expect(warn).toHaveBeenCalledWith(
        expect.stringContaining('Primary pod-log source failed; falling back to archive'),
      );
      expect(stream).toBe(fallbackStream);
    });

    it('throws error if both handler and fallback fails.', async () => {
      const defaultHandler = vi.fn((_podName: string, _createdAt: string, _namespace?: string) =>
        Promise.reject('unknown error for default'),
      );
      const fallbackHandler = vi.fn((_podName: string, _createdAt: string, _namespace?: string) =>
        Promise.reject('unknown error for fallback'),
      );
      await expect(
        composePodLogsStreamHandler(defaultHandler, fallbackHandler)(
          'podName',
          '2024-08-13',
          'namespace',
        ),
      ).rejects.toEqual('unknown error for fallback');
    });
  });

  describe('getPodLogsStreamFromK8s', () => {
    it('returns the pod log stream using k8s api.', async () => {
      const mockedGetPodLogs: Mock = getPodLogs as any;
      mockedGetPodLogs.mockResolvedValueOnce('pod logs');

      const stream = await getPodLogsStreamFromK8s('podName', '', 'namespace');
      expect(mockedGetPodLogs).toBeCalledWith('podName', 'namespace', 'main');
      expect(stream.read().toString()).toBe('pod logs');
    });
  });

  describe('toGetPodLogsStream', () => {
    it('wraps a getMinioRequestConfig function to return the corresponding object stream.', async () => {
      const objStream = new PassThrough();
      objStream.end('some fake logs.');

      const client = new MinioClient(minioConfig);
      const mockedClientGetObject: Mock = client.getObject as any;
      mockedClientGetObject.mockResolvedValueOnce(objStream);
      const configs = {
        bucket: 'bucket',
        client,
        key: 'folder/key',
      };
      const createRequest = vi.fn((_podName: string, _createdAt: string, _namespace?: string) =>
        Promise.resolve(configs),
      );
      const stream = await toGetPodLogsStream(createRequest)('podName', '2024-08-13', 'namespace');
      expect(mockedClientGetObject).toBeCalledWith('bucket', 'folder/key');
    });
  });

  describe('getKeyFormatFromArtifactRepositories', () => {
    it('returns a keyFormat string from the artifact-repositories configmap.', async () => {
      const artifactRepositories = {
        'artifact-repositories':
          'archiveLogs: true\n' +
          's3:\n' +
          '  accessKeySecret:\n' +
          '    key: accesskey\n' +
          '    name: mlpipeline-minio-artifact\n' +
          '  bucket: mlpipeline\n' +
          '  endpoint: seaweedfs.kubeflow:9000\n' +
          '  insecure: true\n' +
          '  keyFormat: foo\n' +
          '  secretKeySecret:\n' +
          '    key: secretkey\n' +
          '    name: mlpipeline-minio-artifact',
      };

      const mockedConfigMap: V1ConfigMap = {
        apiVersion: 'v1',
        kind: 'ConfigMap',
        metadata: new V1ObjectMeta(),
        data: artifactRepositories,
        binaryData: {},
      };

      const mockedGetConfigMap: Mock = getConfigMap as any;
      mockedGetConfigMap.mockResolvedValueOnce([mockedConfigMap, undefined]);
      const res = await getKeyFormatFromArtifactRepositories('');
      expect(mockedGetConfigMap).toBeCalledTimes(1);
      expect(res).toEqual('foo');
    });
  });

  describe('createPodLogsMinioRequestConfig', () => {
    it('returns a MinioRequestConfig factory with the provided minioClientOptions, bucket, and prefix.', async () => {
      const mockedClient: Mock = MinioClient as any;
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        'artifacts/{{workflow.name}}/{{workflow.creationTimestamp.Y}}/{{workflow.creationTimestamp.m}}/{{workflow.creationTimestamp.d}}/{{pod.name}}',
        true,
      );
      const request = await requestFunc(
        'workflow-name-system-container-impl-foo',
        '2024-08-13',
        'namespace',
      );

      expect(mockedClient).toBeCalledWith(minioConfig);
      expect(request.client).toBeInstanceOf(MinioClient);
      expect(request.bucket).toBe('bucket');
      expect(request.key).toBe(
        'artifacts/workflow-name/2024/08/13/workflow-name-system-container-impl-foo/main.log',
      );
    });

    it('scopes the key to the namespace when auth is enabled and the keyFormat embeds it.', async () => {
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        'artifacts/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}',
        false,
        true, // authEnabled
      );
      const request = await requestFunc(
        'workflow-name-system-container-impl-foo',
        '2024-08-13',
        'user-ns',
      );
      expect(request.key).toBe(
        'artifacts/user-ns/workflow-name/workflow-name-system-container-impl-foo/main.log',
      );
    });

    it('uses the server namespace for a namespace-scoped archive key in standalone mode.', async () => {
      const mockedGetServerNamespace: Mock = getServerNamespace as any;
      mockedGetServerNamespace.mockReturnValue('kubeflow');
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        'private-artifacts/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}',
        false,
        false,
      );

      const request = await requestFunc(
        'workflow-name-system-container-impl-foo',
        '2024-08-13',
        '',
      );

      expect(request.key).toBe(
        'private-artifacts/kubeflow/workflow-name/workflow-name-system-container-impl-foo/main.log',
      );
    });

    it('ignores a tenant-controlled keyFormat override when auth is enabled.', async () => {
      const mockedGetConfigMap: Mock = getConfigMap as any;
      mockedGetConfigMap.mockResolvedValueOnce([
        {
          data: {
            'artifact-repositories':
              's3:\n  keyFormat: artifacts/victim-ns/{{workflow.namespace}}/{{pod.name}}\n',
          },
        },
        undefined,
      ]);
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        'artifacts/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}',
        true, // artifactRepositoriesLookup
        true, // authEnabled
      );

      const request = await requestFunc(
        'workflow-name-system-container-impl-foo',
        '2024-08-13',
        'user-ns',
      );

      expect(mockedGetConfigMap).not.toBeCalled();
      expect(request.key).toBe(
        'artifacts/user-ns/workflow-name/workflow-name-system-container-impl-foo/main.log',
      );
    });

    it('fails closed when auth is enabled but the keyFormat is not namespace-scoped.', async () => {
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        'artifacts/{{workflow.name}}/{{pod.name}}',
        false,
        true, // authEnabled
      );
      await expect(
        requestFunc('workflow-name-system-container-impl-foo', '2024-08-13', 'user-ns'),
      ).rejects.toThrow(/{{workflow.namespace}}/);
    });

    it('fails closed when auth is enabled but no namespace is provided.', async () => {
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        'artifacts/{{workflow.namespace}}/{{pod.name}}',
        false,
        true, // authEnabled
      );
      await expect(
        requestFunc('workflow-name-system-container-impl-foo', '2024-08-13', ''),
      ).rejects.toThrow(/namespace/);
    });

    it('rejects a key containing a ".." path segment.', async () => {
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        'artifacts/{{workflow.namespace}}/{{pod.name}}',
        false,
        true, // authEnabled
      );
      await expect(requestFunc('..', '2024-08-13', 'user-ns')).rejects.toThrow(/\.\./);
    });

    it('fails closed when the namespace tag is not its own path segment (concatenated).', async () => {
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        'artifacts/{{workflow.namespace}}{{pod.name}}',
        false,
        true, // authEnabled
      );
      await expect(
        requestFunc('workflow-name-system-container-impl-foo', '2024-08-13', 'user-ns'),
      ).rejects.toThrow(/path segment/);
    });

    it('fails closed when the namespace tag is adjacent to another field via a delimiter.', async () => {
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        'artifacts/{{workflow.namespace}}-{{pod.name}}/main',
        false,
        true, // authEnabled
      );
      await expect(
        requestFunc('workflow-name-system-container-impl-foo', '2024-08-13', 'user-ns'),
      ).rejects.toThrow(/path segment/);
    });

    it('fails closed when a caller-controlled field precedes the namespace tag.', async () => {
      // Namespace is a bounded segment, but {{pod.name}} appears before it, so
      // the namespace is not a deterministic prefix: a tenant whose namespace name
      // coincides with a pod/workflow segment of another namespace's key could
      // collide with it. This must be rejected.
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        'archive/{{pod.name}}/{{workflow.namespace}}',
        false,
        true, // authEnabled
      );
      await expect(
        requestFunc('workflow-name-system-container-impl-foo', '2024-08-13', 'user-ns'),
      ).rejects.toThrow(/caller-controlled field/);
    });

    it.each([
      'archive/{{custom.tenantValue}}/{{workflow.namespace}}/{{pod.name}}',
      'archive/{{workflow.namespace}}/{{workflow.namespace}}/{{pod.name}}',
    ])('fails closed for an ambiguous namespace prefix template %s', async (keyFormat) => {
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        keyFormat,
        false,
        true,
      );
      await expect(
        requestFunc('workflow-name-system-container-impl-foo', '2024-08-13', 'user-ns'),
      ).rejects.toThrow(/namespace/);
    });

    it('accepts the namespace tag as a bounded prefix segment ahead of caller fields.', async () => {
      const requestFunc = await createPodLogsMinioRequestConfig(
        minioConfig,
        'bucket',
        'logs/archive/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}',
        false,
        true, // authEnabled
      );
      const request = await requestFunc(
        'workflow-name-system-container-impl-foo',
        '2024-08-13',
        'user-ns',
      );
      expect(request.key).toBe(
        'logs/archive/user-ns/workflow-name/workflow-name-system-container-impl-foo/main.log',
      );
    });
  });

  describe('getPodLogsStreamFromWorkflow', () => {
    it('returns a getPodLogsStream function that retrieves an object stream using the workflow status corresponding to the pod name.', async () => {
      const sampleWorkflow = {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
        status: {
          artifactRepositoryRef: {
            artifactRepository: {
              archiveLogs: true,
              s3: {
                accessKeySecret: { key: 'accessKey', name: 'accessKeyName' },
                bucket: 'bucket',
                endpoint: 'seaweedfs.kubeflow',
                insecure: true,
                key: 'prefix/workflow-name/workflow-name-system-container-impl-abc/some-artifact.csv',
                secretKeySecret: { key: 'secretKey', name: 'secretKeyName' },
              },
            },
          },
          nodes: {
            'workflow-name-abc': {
              outputs: {
                artifacts: [
                  {
                    name: 'main-logs',
                    s3: {
                      key: 'prefix/workflow-name/workflow-name-system-container-impl-abc/main.log',
                    },
                  },
                ],
              },
            },
          },
        },
      };

      const mockedGetArgoWorkflow: Mock = getArgoWorkflow as any;
      mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);

      // The run namespace matches the server's own namespace, so reading the
      // object-store credential Secret is permitted.
      const mockedGetServerNamespace: Mock = getServerNamespace as any;
      mockedGetServerNamespace.mockReturnValue('kubeflow');

      const mockedGetK8sSecret: Mock = getK8sSecret as any;
      mockedGetK8sSecret.mockResolvedValue('someSecret');

      const objStream = new PassThrough();
      const mockedClient: Mock = MinioClient as any;
      // In Vitest, auto-mocked class instances get their own mock methods.
      // Set up prototype mock so new instances inherit it.
      MinioClient.prototype.getObject = vi.fn().mockResolvedValueOnce(objStream) as any;
      objStream.end('some fake logs.');

      const stream = await getPodLogsStreamFromWorkflow(
        'workflow-name-system-container-impl-abc',
        '2024-07-09',
        'kubeflow',
        { authEnabled: false },
      );

      expect(mockedGetArgoWorkflow).toBeCalledWith('workflow-name', 'kubeflow');

      expect(mockedGetK8sSecret).toBeCalledTimes(2);
      expect(mockedGetK8sSecret).toBeCalledWith('accessKeyName', 'accessKey', 'kubeflow');
      expect(mockedGetK8sSecret).toBeCalledWith('secretKeyName', 'secretKey', 'kubeflow');

      expect(mockedClient).toBeCalledTimes(1);
      expect(mockedClient).toBeCalledWith({
        accessKey: 'someSecret',
        endPoint: 'seaweedfs.kubeflow',
        port: 80,
        secretKey: 'someSecret',
        useSSL: false,
      });
      // Access the instance created by the constructor to check getObject
      const clientInstance = mockedClient.mock.results[0].value;
      expect(clientInstance.getObject).toBeCalledTimes(1);
      expect(clientInstance.getObject).toBeCalledWith(
        'bucket',
        'prefix/workflow-name/workflow-name-system-container-impl-abc/main.log',
      );
    });

    it('does not read the object-store Secret when the run namespace is not the server namespace (security)', async () => {
      const sampleWorkflow = {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
        status: {
          artifactRepositoryRef: {
            artifactRepository: {
              archiveLogs: true,
              s3: {
                accessKeySecret: { key: 'accessKey', name: 'accessKeyName' },
                bucket: 'bucket',
                endpoint: 'seaweedfs.kubeflow',
                insecure: true,
                key: 'prefix/workflow-name/workflow-name-system-container-impl-abc/some-artifact.csv',
                secretKeySecret: { key: 'secretKey', name: 'secretKeyName' },
              },
            },
          },
          nodes: {
            'workflow-name-abc': {
              outputs: {
                artifacts: [
                  {
                    name: 'main-logs',
                    s3: {
                      key: 'prefix/workflow-name/workflow-name-system-container-impl-abc/main.log',
                    },
                  },
                ],
              },
            },
          },
        },
      };

      const mockedGetArgoWorkflow: Mock = getArgoWorkflow as any;
      mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);

      // The run namespace is a customer namespace, different from the server's
      // own namespace, so the credential Secret must NOT be read.
      const mockedGetServerNamespace: Mock = getServerNamespace as any;
      mockedGetServerNamespace.mockReturnValue('kubeflow');

      const mockedGetK8sSecret: Mock = getK8sSecret as any;

      // The server's own object-store credentials are provided via the
      // environment, matching the deployment's MINIO_ACCESS_KEY/MINIO_SECRET_KEY.
      const previousAccessKey = process.env.MINIO_ACCESS_KEY;
      const previousSecretKey = process.env.MINIO_SECRET_KEY;
      process.env.MINIO_ACCESS_KEY = 'server-access-key';
      process.env.MINIO_SECRET_KEY = 'server-secret-key';

      const objStream = new PassThrough();
      const mockedClient: Mock = MinioClient as any;
      MinioClient.prototype.getObject = vi.fn().mockResolvedValueOnce(objStream) as any;
      objStream.end('some fake logs.');

      try {
        await getPodLogsStreamFromWorkflow(
          'workflow-name-system-container-impl-abc',
          '2024-07-09',
          'my-user-namespace',
          { authEnabled: false },
        );
      } finally {
        process.env.MINIO_ACCESS_KEY = previousAccessKey;
        process.env.MINIO_SECRET_KEY = previousSecretKey;
      }

      expect(mockedGetK8sSecret).not.toBeCalled();
      // The client is built using the server's own environment credentials
      // rather than the customer-namespace Secret, so the workflow-status log
      // path works against the shared store instead of failing anonymously.
      expect(mockedClient).toBeCalledWith({
        accessKey: 'server-access-key',
        endPoint: 'seaweedfs.kubeflow',
        port: 80,
        secretKey: 'server-secret-key',
        useSSL: false,
      });
    });

    it('reads the object-store Secret from the server namespace when the run namespace is omitted (standalone)', async () => {
      const sampleWorkflow = {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
        status: {
          artifactRepositoryRef: {
            artifactRepository: {
              archiveLogs: true,
              s3: {
                accessKeySecret: { key: 'accessKey', name: 'accessKeyName' },
                bucket: 'bucket',
                endpoint: 'seaweedfs.kubeflow',
                insecure: true,
                key: 'prefix/workflow-name/workflow-name-system-container-impl-abc/some-artifact.csv',
                secretKeySecret: { key: 'secretKey', name: 'secretKeyName' },
              },
            },
          },
          nodes: {
            'workflow-name-abc': {
              outputs: {
                artifacts: [
                  {
                    name: 'main-logs',
                    s3: {
                      key: 'prefix/workflow-name/workflow-name-system-container-impl-abc/main.log',
                    },
                  },
                ],
              },
            },
          },
        },
      };

      const mockedGetArgoWorkflow: Mock = getArgoWorkflow as any;
      mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);

      // Standalone mode omits the namespace; the run is effectively in the server
      // namespace, so the credential Secret is read from the server namespace.
      const mockedGetServerNamespace: Mock = getServerNamespace as any;
      mockedGetServerNamespace.mockReturnValue('kubeflow');

      const mockedGetK8sSecret: Mock = getK8sSecret as any;
      mockedGetK8sSecret.mockResolvedValue('custom-store-secret');

      const objStream = new PassThrough();
      const mockedClient: Mock = MinioClient as any;
      MinioClient.prototype.getObject = vi.fn().mockResolvedValueOnce(objStream) as any;
      objStream.end('some fake logs.');

      await getPodLogsStreamFromWorkflow(
        'workflow-name-system-container-impl-abc',
        '2024-07-09',
        undefined,
        { authEnabled: false },
      );

      // The Secret is read from the server namespace, never a user namespace.
      expect(mockedGetK8sSecret).toBeCalledTimes(2);
      expect(mockedGetK8sSecret).toBeCalledWith('accessKeyName', 'accessKey', 'kubeflow');
      expect(mockedGetK8sSecret).toBeCalledWith('secretKeyName', 'secretKey', 'kubeflow');

      // The custom object-store credentials from the Secret are honored rather
      // than falling back to default env credentials.
      expect(mockedClient).toBeCalledWith({
        accessKey: 'custom-store-secret',
        endPoint: 'seaweedfs.kubeflow',
        port: 80,
        secretKey: 'custom-store-secret',
        useSSL: false,
      });
    });

    it('falls back to env credentials for an omitted namespace when the artifact references no Secret', async () => {
      const sampleWorkflow = {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
        status: {
          artifactRepositoryRef: {
            artifactRepository: {
              archiveLogs: true,
              s3: {
                // No accessKeySecret / secretKeySecret: the artifact repository
                // does not reference a credential Secret.
                bucket: 'bucket',
                endpoint: 'seaweedfs.kubeflow',
                insecure: true,
                key: 'prefix/workflow-name/workflow-name-system-container-impl-abc/some-artifact.csv',
              },
            },
          },
          nodes: {
            'workflow-name-abc': {
              outputs: {
                artifacts: [
                  {
                    name: 'main-logs',
                    s3: {
                      key: 'prefix/workflow-name/workflow-name-system-container-impl-abc/main.log',
                    },
                  },
                ],
              },
            },
          },
        },
      };

      const mockedGetArgoWorkflow: Mock = getArgoWorkflow as any;
      mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);

      const mockedGetServerNamespace: Mock = getServerNamespace as any;
      mockedGetServerNamespace.mockReturnValue('kubeflow');

      const mockedGetK8sSecret: Mock = getK8sSecret as any;

      const previousAccessKey = process.env.MINIO_ACCESS_KEY;
      const previousSecretKey = process.env.MINIO_SECRET_KEY;
      process.env.MINIO_ACCESS_KEY = 'server-access-key';
      process.env.MINIO_SECRET_KEY = 'server-secret-key';

      const objStream = new PassThrough();
      const mockedClient: Mock = MinioClient as any;
      MinioClient.prototype.getObject = vi.fn().mockResolvedValueOnce(objStream) as any;
      objStream.end('some fake logs.');

      try {
        await getPodLogsStreamFromWorkflow(
          'workflow-name-system-container-impl-abc',
          '2024-07-09',
          undefined,
          { authEnabled: false },
        );
      } finally {
        process.env.MINIO_ACCESS_KEY = previousAccessKey;
        process.env.MINIO_SECRET_KEY = previousSecretKey;
      }

      // With no Secret referenced, no Secret read is attempted and the frontend's
      // own configured env credentials are used.
      expect(mockedGetK8sSecret).not.toBeCalled();
      expect(mockedClient).toBeCalledWith({
        accessKey: 'server-access-key',
        endPoint: 'seaweedfs.kubeflow',
        port: 80,
        secretKey: 'server-secret-key',
        useSSL: false,
      });
    });

    it('fails closed in multi-user mode when the workflow-recorded log key is not scoped to the authorized namespace.', async () => {
      const sampleWorkflow = {
        status: {
          artifactRepositoryRef: {
            artifactRepository: {
              archiveLogs: true,
              s3: { bucket: 'bucket', endpoint: 'seaweedfs.kubeflow', insecure: true, key: 'x' },
            },
          },
          nodes: {
            'workflow-name-abc': {
              outputs: {
                artifacts: [
                  {
                    name: 'main-logs',
                    // Key belongs to a different namespace / is not namespace-scoped.
                    s3: { key: 'artifacts/other-ns/workflow-name/pod/main.log' },
                  },
                ],
              },
            },
          },
        },
      };
      const mockedGetArgoWorkflow: Mock = getArgoWorkflow as any;
      mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);

      await expect(
        getPodLogsMinioRequestConfigfromWorkflow(
          'workflow-name-system-container-impl-abc',
          '2024-07-09',
          'user-ns',
          {
            authEnabled: true,
            trustedKeyFormat: 'artifacts/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}',
          },
        ),
      ).rejects.toThrow(/authorized namespace/);
    });

    it('allows a workflow-recorded log key that embeds the authorized namespace as a segment.', async () => {
      const sampleWorkflow = {
        status: {
          artifactRepositoryRef: {
            artifactRepository: {
              archiveLogs: true,
              s3: { bucket: 'bucket', endpoint: 'seaweedfs.kubeflow', insecure: true, key: 'x' },
            },
          },
          nodes: {
            'workflow-name-abc': {
              outputs: {
                artifacts: [
                  {
                    name: 'main-logs',
                    s3: { key: 'artifacts/user-ns/workflow-name/pod/main.log' },
                  },
                ],
              },
            },
          },
        },
      };
      const mockedGetArgoWorkflow: Mock = getArgoWorkflow as any;
      mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);
      const mockedGetServerNamespace: Mock = getServerNamespace as any;
      mockedGetServerNamespace.mockReturnValue('kubeflow');

      const previousAccessKey = process.env.MINIO_ACCESS_KEY;
      const previousSecretKey = process.env.MINIO_SECRET_KEY;
      process.env.MINIO_ACCESS_KEY = 'server-access-key';
      process.env.MINIO_SECRET_KEY = 'server-secret-key';
      try {
        const request = await getPodLogsMinioRequestConfigfromWorkflow(
          'workflow-name-system-container-impl-abc',
          '2024-07-09',
          'user-ns',
          {
            authEnabled: true,
            trustedKeyFormat: 'artifacts/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}',
            trustedBucket: 'bucket',
            trustedStore: {
              accessKey: 'server-access',
              endPoint: 'seaweedfs.kubeflow',
              port: 80,
              secretKey: 'server-secret',
              useSSL: false,
            },
          },
        );
        expect(request.bucket).toBe('bucket');
        expect(request.key).toBe('artifacts/user-ns/workflow-name/pod/main.log');
      } finally {
        process.env.MINIO_ACCESS_KEY = previousAccessKey;
        process.env.MINIO_SECRET_KEY = previousSecretKey;
      }
    });

    it.each([
      {
        bucket: 'bucket',
        endpoint: '169.254.169.254:80',
        name: 'workflow-controlled endpoint',
        namespace: 'user-ns',
      },
      {
        bucket: 'other-bucket',
        endpoint: 'seaweedfs.kubeflow',
        name: 'workflow-controlled bucket',
        namespace: 'user-ns',
      },
      {
        bucket: 'bucket',
        endpoint: 'https://seaweedfs.kubeflow',
        name: 'TLS-conflicting workflow-controlled endpoint',
        namespace: 'user-ns',
      },
      {
        bucket: 'bucket',
        endpoint: '169.254.169.254:80',
        name: 'server-namespace workflow-controlled endpoint',
        namespace: 'kubeflow',
      },
    ])('rejects a $name when auth is enabled', async ({ bucket, endpoint, namespace }) => {
      const sampleWorkflow = {
        status: {
          artifactRepositoryRef: {
            artifactRepository: {
              archiveLogs: true,
              s3: { bucket, endpoint, insecure: true, key: 'x' },
            },
          },
          nodes: {
            'workflow-name-abc': {
              outputs: {
                artifacts: [
                  {
                    name: 'main-logs',
                    s3: { key: `artifacts/${namespace}/workflow-name/pod/main.log` },
                  },
                ],
              },
            },
          },
        },
      };
      const mockedGetArgoWorkflow: Mock = getArgoWorkflow as any;
      mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);
      const mockedClient: Mock = MinioClient as any;

      await expect(
        getPodLogsMinioRequestConfigfromWorkflow(
          'workflow-name-system-container-impl-abc',
          '2024-07-09',
          namespace,
          {
            authEnabled: true,
            trustedKeyFormat: 'artifacts/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}',
            trustedBucket: 'bucket',
            trustedStore: { endPoint: 'seaweedfs.kubeflow', port: 80, useSSL: false },
          },
        ),
      ).rejects.toThrow(/workflow-controlled artifact (endpoint|bucket)|invalid or conflicts/);
      expect(mockedClient).not.toBeCalled();
    });

    it('pairs a trusted cross-namespace AWS store with its AWS credentials', async () => {
      const sampleWorkflow = {
        status: {
          artifactRepositoryRef: {
            artifactRepository: {
              archiveLogs: true,
              s3: { bucket: 'bucket', endpoint: 's3.example.test', insecure: false, key: 'x' },
            },
          },
          nodes: {
            'workflow-name-abc': {
              outputs: {
                artifacts: [
                  {
                    name: 'main-logs',
                    s3: { key: 'artifacts/user-ns/workflow-name/pod/main.log' },
                  },
                ],
              },
            },
          },
        },
      };
      const mockedGetArgoWorkflow: Mock = getArgoWorkflow as any;
      mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);
      const mockedGetServerNamespace: Mock = getServerNamespace as any;
      mockedGetServerNamespace.mockReturnValue('kubeflow');
      const mockedClient: Mock = MinioClient as any;

      await getPodLogsMinioRequestConfigfromWorkflow(
        'workflow-name-system-container-impl-abc',
        '2024-07-09',
        'user-ns',
        {
          authEnabled: true,
          trustedKeyFormat: 'artifacts/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}',
          trustedBucket: 'bucket',
          trustedStore: {
            accessKey: 'aws-access-key',
            endPoint: 's3.example.test',
            region: 'eu-west-1',
            secretKey: 'aws-secret-key',
            useSSL: true,
          },
        },
      );

      expect(mockedClient).toBeCalledWith({
        accessKey: 'aws-access-key',
        endPoint: 's3.example.test',
        port: 443,
        region: 'eu-west-1',
        secretKey: 'aws-secret-key',
        useSSL: true,
      });
    });

    it('fails closed when the trusted keyFormat is not namespace-scoped, even if the key contains the namespace.', async () => {
      const sampleWorkflow = {
        status: {
          artifactRepositoryRef: {
            artifactRepository: {
              archiveLogs: true,
              s3: { bucket: 'bucket', endpoint: 'seaweedfs.kubeflow', insecure: true, key: 'x' },
            },
          },
          nodes: {
            'workflow-name-abc': {
              outputs: {
                artifacts: [
                  {
                    name: 'main-logs',
                    // 'user-ns' appears only as a coincidental (non-prefix) segment.
                    s3: { key: 'artifacts/other-ns/user-ns/pod/main.log' },
                  },
                ],
              },
            },
          },
        },
      };
      const mockedGetArgoWorkflow: Mock = getArgoWorkflow as any;
      mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);

      await expect(
        getPodLogsMinioRequestConfigfromWorkflow(
          'workflow-name-system-container-impl-abc',
          '2024-07-09',
          'user-ns',
          {
            authEnabled: true,
            // Trusted keyFormat has no {{workflow.namespace}} → no safe prefix.
            trustedKeyFormat: 'artifacts/{{workflow.name}}/{{pod.name}}',
          },
        ),
      ).rejects.toThrow(/namespace/);
    });

    // Regression: the workflow-recorded endpoint and the trusted store endpoint
    // are the same in-cluster Service written with different DNS forms. Manifests
    // write `seaweedfs.<ns>.svc[.cluster.local]:9000` while configs.ts builds the
    // trusted `seaweedfs.<ns>`. These must compare equal, or the workflow-status
    // path is dead on arrival in multi-user mode.
    for (const [workflowEndpoint, clusterDomain] of [
      ['seaweedfs.kubeflow.svc:9000', '.svc.cluster.local'],
      ['seaweedfs.kubeflow.svc.cluster.local:9000', '.svc.cluster.local'],
      ['seaweedfs.kubeflow.svc.cluster.corp:9000', 'cluster.corp'],
    ]) {
      it(`accepts the manifest endpoint form "${workflowEndpoint}" as the trusted store`, async () => {
        const sampleWorkflow = {
          status: {
            artifactRepositoryRef: {
              artifactRepository: {
                archiveLogs: true,
                s3: {
                  bucket: 'mlpipeline',
                  endpoint: workflowEndpoint,
                  insecure: true,
                  key: 'x',
                },
              },
            },
            nodes: {
              'workflow-name-abc': {
                outputs: {
                  artifacts: [
                    {
                      name: 'main-logs',
                      s3: { key: 'private-artifacts/user-ns/workflow-name/pod/main.log' },
                    },
                  ],
                },
              },
            },
          },
        };
        const mockedGetArgoWorkflow: Mock = getArgoWorkflow as any;
        mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);
        const mockedGetServerNamespace: Mock = getServerNamespace as any;
        mockedGetServerNamespace.mockReturnValue('kubeflow');
        const mockedClient: Mock = MinioClient as any;

        const request = await getPodLogsMinioRequestConfigfromWorkflow(
          'workflow-name-system-container-impl-abc',
          '2024-07-09',
          'user-ns',
          {
            authEnabled: true,
            trustedKeyFormat:
              'private-artifacts/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}',
            trustedBucket: 'mlpipeline',
            clusterDomain,
            trustedStore: {
              endPoint: 'seaweedfs.kubeflow',
              port: 9000,
              useSSL: false,
              accessKey: 'server-access-key',
              secretKey: 'server-secret-key',
            },
          },
        );

        // The read is accepted (no throw); the client connects to the real
        // workflow-recorded host, and the shared trusted-store credentials are used.
        expect(request.bucket).toBe('mlpipeline');
        expect(request.key).toBe('private-artifacts/user-ns/workflow-name/pod/main.log');
        expect(mockedClient).toBeCalledWith(
          expect.objectContaining({
            accessKey: 'server-access-key',
            secretKey: 'server-secret-key',
            endPoint: workflowEndpoint.split(':')[0],
            port: 9000,
            useSSL: false,
          }),
        );
      });
    }

    it('still rejects a genuinely foreign endpoint host', async () => {
      const sampleWorkflow = {
        status: {
          artifactRepositoryRef: {
            artifactRepository: {
              archiveLogs: true,
              s3: {
                bucket: 'mlpipeline',
                endpoint: 'attacker.evil.example:9000',
                insecure: true,
                key: 'x',
              },
            },
          },
          nodes: {
            'workflow-name-abc': {
              outputs: {
                artifacts: [
                  {
                    name: 'main-logs',
                    s3: { key: 'private-artifacts/user-ns/workflow-name/pod/main.log' },
                  },
                ],
              },
            },
          },
        },
      };
      const mockedGetArgoWorkflow: Mock = getArgoWorkflow as any;
      mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);
      const mockedGetServerNamespace: Mock = getServerNamespace as any;
      mockedGetServerNamespace.mockReturnValue('kubeflow');

      await expect(
        getPodLogsMinioRequestConfigfromWorkflow(
          'workflow-name-system-container-impl-abc',
          '2024-07-09',
          'user-ns',
          {
            authEnabled: true,
            trustedKeyFormat:
              'private-artifacts/{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}',
            trustedBucket: 'mlpipeline',
            clusterDomain: '.svc.cluster.local',
            trustedStore: { endPoint: 'seaweedfs.kubeflow', port: 9000, useSSL: false },
          },
        ),
      ).rejects.toThrow(/workflow-controlled artifact endpoint/);
    });
  });
});

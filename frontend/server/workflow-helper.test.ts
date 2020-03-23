// Copyright 2019 Google LLC
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
import { PassThrough } from 'stream';
import { Client as MinioClient } from 'minio';
import {
  createPodLogsMinioRequestConfig,
  composePodLogsStreamHandler,
  getPodLogsStreamFromK8s,
  getPodLogsStreamFromWorkflow,
  toGetPodLogsStream,
} from './workflow-helper';
import { getK8sSecret, getArgoWorkflow, getPodLogs } from './k8s-helper';

jest.mock('minio');
jest.mock('./k8s-helper');

describe('workflow-helper', () => {
  const minioConfig = {
    accessKey: 'minio',
    endPoint: 'minio-service.kubeflow',
    secretKey: 'minio123',
  };

  beforeEach(() => {
    jest.resetAllMocks();
  });

  describe('composePodLogsStreamHandler', () => {
    it('returns the stream from the default handler if there is no errors.', async () => {
      const defaultStream = new PassThrough();
      const defaultHandler = jest.fn((_podName: string, _namespace?: string) =>
        Promise.resolve(defaultStream),
      );
      const stream = await composePodLogsStreamHandler(defaultHandler)('podName', 'namespace');
      expect(defaultHandler).toBeCalledWith('podName', 'namespace');
      expect(stream).toBe(defaultStream);
    });

    it('returns the stream from the fallback handler if there is any error.', async () => {
      const fallbackStream = new PassThrough();
      const defaultHandler = jest.fn((_podName: string, _namespace?: string) =>
        Promise.reject('unknown error'),
      );
      const fallbackHandler = jest.fn((_podName: string, _namespace?: string) =>
        Promise.resolve(fallbackStream),
      );
      const stream = await composePodLogsStreamHandler(defaultHandler, fallbackHandler)(
        'podName',
        'namespace',
      );
      expect(defaultHandler).toBeCalledWith('podName', 'namespace');
      expect(fallbackHandler).toBeCalledWith('podName', 'namespace');
      expect(stream).toBe(fallbackStream);
    });

    it('throws error if both handler and fallback fails.', async () => {
      const defaultHandler = jest.fn((_podName: string, _namespace?: string) =>
        Promise.reject('unknown error for default'),
      );
      const fallbackHandler = jest.fn((_podName: string, _namespace?: string) =>
        Promise.reject('unknown error for fallback'),
      );
      await expect(
        composePodLogsStreamHandler(defaultHandler, fallbackHandler)('podName', 'namespace'),
      ).rejects.toEqual('unknown error for fallback');
    });
  });

  describe('getPodLogsStreamFromK8s', () => {
    it('returns the pod log stream using k8s api.', async () => {
      const mockedGetPodLogs: jest.Mock = getPodLogs as any;
      mockedGetPodLogs.mockResolvedValueOnce('pod logs');

      const stream = await getPodLogsStreamFromK8s('podName', 'namespace');
      expect(mockedGetPodLogs).toBeCalledWith('podName', 'namespace');
      expect(stream.read().toString()).toBe('pod logs');
    });
  });

  describe('toGetPodLogsStream', () => {
    it('wraps a getMinioRequestConfig function to return the corresponding object stream.', async () => {
      const objStream = new PassThrough();
      objStream.end('some fake logs.');

      const client = new MinioClient(minioConfig);
      const mockedClientGetObject: jest.Mock = client.getObject as any;
      mockedClientGetObject.mockResolvedValueOnce(objStream);
      const configs = {
        bucket: 'bucket',
        client,
        key: 'folder/key',
      };
      const createRequest = jest.fn((_podName: string, _namespace?: string) =>
        Promise.resolve(configs),
      );
      const stream = await toGetPodLogsStream(createRequest)('podName', 'namespace');
      expect(mockedClientGetObject).toBeCalledWith('bucket', 'folder/key');
    });
  });

  describe('createPodLogsMinioRequestConfig', () => {
    it('returns a MinioRequestConfig factory with the provided minioClientOptions, bucket, and prefix.', async () => {
      const mockedClient: jest.Mock = MinioClient as any;
      const requestFunc = await createPodLogsMinioRequestConfig(minioConfig, 'bucket', 'prefix');
      const request = await requestFunc('workflow-name-abc', 'namespace');

      expect(mockedClient).toBeCalledWith(minioConfig);
      expect(request.client).toBeInstanceOf(MinioClient);
      expect(request.bucket).toBe('bucket');
      expect(request.key).toBe('prefix/workflow-name/workflow-name-abc/main.log');
    });
  });

  describe('getPodLogsStreamFromWorkflow', () => {
    it('returns a getPodLogsStream function that retrieves an object stream using the workflow status corresponding to the pod name.', async () => {
      const sampleWorkflow = {
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
        status: {
          nodes: {
            'workflow-name-abc': {
              outputs: {
                artifacts: [
                  {
                    name: 'some-artifact.csv',
                    s3: {
                      accessKeySecret: { key: 'accessKey', name: 'accessKeyName' },
                      bucket: 'bucket',
                      endpoint: 'minio-service.kubeflow',
                      insecure: true,
                      key: 'prefix/workflow-name/workflow-name-abc/some-artifact.csv',
                      secretKeySecret: { key: 'secretKey', name: 'secretKeyName' },
                    },
                  },
                  {
                    archiveLogs: true,
                    name: 'main.log',
                    s3: {
                      accessKeySecret: { key: 'accessKey', name: 'accessKeyName' },
                      bucket: 'bucket',
                      endpoint: 'minio-service.kubeflow',
                      insecure: true,
                      key: 'prefix/workflow-name/workflow-name-abc/main.log',
                      secretKeySecret: { key: 'secretKey', name: 'secretKeyName' },
                    },
                  },
                ],
              },
            },
          },
        },
      };

      const mockedGetArgoWorkflow: jest.Mock = getArgoWorkflow as any;
      mockedGetArgoWorkflow.mockResolvedValueOnce(sampleWorkflow);

      const mockedGetK8sSecret: jest.Mock = getK8sSecret as any;
      mockedGetK8sSecret.mockResolvedValue('someSecret');

      const objStream = new PassThrough();
      const mockedClient: jest.Mock = MinioClient as any;
      const mockedClientGetObject: jest.Mock = MinioClient.prototype.getObject as any;
      mockedClientGetObject.mockResolvedValueOnce(objStream);
      objStream.end('some fake logs.');

      const stream = await getPodLogsStreamFromWorkflow('workflow-name-abc');

      expect(mockedGetArgoWorkflow).toBeCalledWith('workflow-name');

      expect(mockedGetK8sSecret).toBeCalledTimes(2);
      expect(mockedGetK8sSecret).toBeCalledWith('accessKeyName', 'accessKey');
      expect(mockedGetK8sSecret).toBeCalledWith('secretKeyName', 'secretKey');

      expect(mockedClient).toBeCalledTimes(1);
      expect(mockedClient).toBeCalledWith({
        accessKey: 'someSecret',
        endPoint: 'minio-service.kubeflow',
        port: 80,
        secretKey: 'someSecret',
        useSSL: false,
      });
      expect(mockedClientGetObject).toBeCalledTimes(1);
      expect(mockedClientGetObject).toBeCalledWith(
        'bucket',
        'prefix/workflow-name/workflow-name-abc/main.log',
      );
    });
  });
});

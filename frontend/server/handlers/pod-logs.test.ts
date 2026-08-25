// Copyright 2026 The Kubeflow Authors
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
import express from 'express';
import requests from 'supertest';
import { beforeEach, describe, expect, it, Mock, vi } from 'vitest';
import * as workflowHelper from '../workflow-helper.js';
import { getPodLogsHandler } from './pod-logs.js';

vi.mock('../workflow-helper.js', () => ({
  composePodLogsStreamHandler: vi.fn(
    (
      primary: (...args: any[]) => Promise<PassThrough>,
      fallback?: (...args: any[]) => Promise<PassThrough>,
    ) =>
      async (...args: any[]) => {
        try {
          return await primary(...args);
        } catch (error) {
          if (!fallback) throw error;
          return fallback(...args);
        }
      },
  ),
  createPodLogsMinioRequestConfig: vi.fn(() => vi.fn()),
  getPodLogsMinioRequestConfigfromWorkflow: vi.fn(async () => ({
    bucket: 'bucket',
    client: {},
    key: 'key',
  })),
  getPodLogsStreamFromK8s: vi.fn(async () => {
    throw new Error('pod logs unavailable');
  }),
  toGetPodLogsStream: vi.fn(
    (createRequest: (...args: any[]) => Promise<unknown>) =>
      async (...args: any[]) => {
        await createRequest(...args);
        const stream = new PassThrough();
        stream.end('archived logs');
        return stream;
      },
  ),
}));

describe('getPodLogsHandler', () => {
  beforeEach(() => vi.clearAllMocks());

  it('wires operator archive trust options into the workflow-status fallback', async () => {
    const trustedStore = {
      accessKey: 'server-access',
      endPoint: 'seaweedfs.kubeflow',
      port: 9000,
      secretKey: 'server-secret',
      useSSL: false,
    };
    const app = express();
    app.get(
      '/logs',
      getPodLogsHandler(
        {
          archiveArtifactory: 'minio',
          archiveBucketName: 'mlpipeline',
          archiveLogs: true,
          artifactRepositoriesLookup: false,
          keyFormat: 'private-artifacts/{{workflow.namespace}}/{{pod.name}}',
        },
        {
          minio: trustedStore,
          aws: {
            accessKey: '',
            endPoint: 's3.amazonaws.com',
            region: 'us-east-1',
            secretKey: '',
            useSSL: true,
          },
        },
        'main',
        vi.fn(async () => undefined),
        true,
        'cluster.corp',
      ),
    );

    await requests(app)
      .get('/logs?podname=workflow-pod&createdat=2026-08-23&podnamespace=user-ns')
      .expect(200, 'archived logs');

    expect(workflowHelper.getPodLogsMinioRequestConfigfromWorkflow as Mock).toHaveBeenCalledWith(
      'workflow-pod',
      '2026-08-23',
      'user-ns',
      {
        authEnabled: true,
        trustedBucket: 'mlpipeline',
        trustedKeyFormat: 'private-artifacts/{{workflow.namespace}}/{{pod.name}}',
        clusterDomain: 'cluster.corp',
        trustedStore,
      },
    );
  });

  it('reaches the configured archive fallback without podnamespace in standalone mode', async () => {
    const archiveRequest = vi.fn(async () => ({ bucket: 'bucket', client: {}, key: 'key' }));
    (workflowHelper.createPodLogsMinioRequestConfig as Mock).mockReturnValue(archiveRequest);
    (workflowHelper.getPodLogsMinioRequestConfigfromWorkflow as Mock).mockRejectedValue(
      new Error('workflow unavailable'),
    );
    const app = express();
    app.get(
      '/logs',
      getPodLogsHandler(
        {
          archiveArtifactory: 'minio',
          archiveBucketName: 'mlpipeline',
          archiveLogs: true,
          artifactRepositoriesLookup: false,
          keyFormat: 'private-artifacts/{{workflow.namespace}}/{{pod.name}}',
        },
        {
          minio: {
            accessKey: 'server-access',
            endPoint: 'seaweedfs.kubeflow',
            port: 9000,
            secretKey: 'server-secret',
            useSSL: false,
          },
          aws: {
            accessKey: '',
            endPoint: 's3.amazonaws.com',
            region: 'us-east-1',
            secretKey: '',
            useSSL: true,
          },
        },
        'main',
        vi.fn(async () => undefined),
        false,
      ),
    );

    await requests(app)
      .get('/logs?podname=workflow-pod&createdat=2026-08-23')
      .expect(200, 'archived logs');
    expect(archiveRequest).toHaveBeenCalledWith('workflow-pod', '2026-08-23', undefined);
  });
});

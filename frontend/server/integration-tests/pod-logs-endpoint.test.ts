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

import express from 'express';
import { load as loadYaml } from 'js-yaml';
import { Client as MinioClient } from 'minio';
import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { PassThrough } from 'node:stream';
import { fileURLToPath } from 'node:url';
import requests from 'supertest';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { loadConfigs, type ProcessEnv } from '../configs.js';
import { getPodLogsHandler } from '../handlers/pod-logs.js';
import { getArgoWorkflow, getK8sSecret, getPodLogs, getServerNamespace } from '../k8s-helper.js';
import type { ArtifactRepository, PartialArgoWorkflow } from '../workflow-helper.js';

vi.mock('minio');
vi.mock('../k8s-helper.js');

const podName = 'workflow-1-system-container-impl-12345';
const logContent = 'archived pod logs\n';
const getObject = vi.fn();

type ArchivedArtifactRepository = ArtifactRepository & { s3: { keyFormat: string } };

function workflowWithEndpoint(
  endpoint: string,
  insecure: boolean,
  logKey = 'tenant/task/main.log',
): PartialArgoWorkflow {
  return {
    status: {
      artifactRepositoryRef: {
        artifactRepository: {
          archiveLogs: true,
          s3: {
            endpoint,
            insecure,
            bucket: 'workflow-logs',
            key: 'unused-repository-key',
            accessKeySecret: { name: 'workflow-store', key: 'access-key' },
            secretKeySecret: { name: 'workflow-store', key: 'secret-key' },
          },
        },
      },
      nodes: {
        'workflow-1-12345': {
          outputs: {
            artifacts: [{ name: 'main-logs', s3: { key: logKey } }],
          },
        },
      },
    },
  };
}

function workflowWithRepository(repository: ArchivedArtifactRepository, namespace: string) {
  const logKey = `${repository.s3.keyFormat
    .replaceAll('{{workflow.namespace}}', namespace)
    .replaceAll('{{workflow.name}}', 'workflow-1')
    .replaceAll('{{workflow.creationTimestamp.Y}}', '2026')
    .replaceAll('{{workflow.creationTimestamp.m}}', '09')
    .replaceAll('{{workflow.creationTimestamp.d}}', '05')
    .replaceAll('{{pod.name}}', podName)}/main.log`;
  const workflow = workflowWithEndpoint(repository.s3.endpoint, repository.s3.insecure, logKey);
  workflow.status.artifactRepositoryRef = { artifactRepository: repository };
  return { workflow, logKey };
}

function createRequest(env: ProcessEnv = {}) {
  const configs = loadConfigs(['node', 'server.js', '/tmp', '3000'], {
    MINIO_HOST: 'operator-store',
    MINIO_NAMESPACE: 'kubeflow',
    MINIO_PORT: '9000',
    MINIO_SSL: 'false',
    MINIO_ACCESS_KEY: 'operator-access-key',
    MINIO_SECRET_KEY: 'operator-secret-key',
    AWS_S3_ENDPOINT: 'https://operator-s3.example.com:9443',
    ARGO_ARCHIVE_LOGS: 'false',
    ARGO_ARTIFACT_REPOSITORIES_LOOKUP: 'false',
    ...env,
  });
  const app = express();
  app.get(
    '/k8s/pod/logs',
    getPodLogsHandler(
      configs.argo,
      configs.artifacts,
      configs.pod.logContainerName,
      vi.fn().mockResolvedValue(undefined),
      false,
    ),
  );
  return requests(app);
}

describe('/k8s/pod/logs workflow artifact endpoints', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.spyOn(console, 'log').mockImplementation(() => undefined);
    vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    vi.stubEnv('MINIO_ACCESS_KEY', 'frontend-access-key');
    vi.stubEnv('MINIO_SECRET_KEY', 'frontend-secret-key');
    vi.mocked(getPodLogs).mockRejectedValue(new Error('pod no longer exists'));
    vi.mocked(getServerNamespace).mockReturnValue('kubeflow');
    vi.mocked(getK8sSecret).mockImplementation(async (_name, key) => `workflow-${key}`);
    getObject.mockImplementation(async () => {
      const stream = new PassThrough();
      stream.end(logContent);
      return stream;
    });
    vi.mocked(MinioClient).mockImplementation(function () {
      return { getObject } as unknown as MinioClient;
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
  });

  it.each([
    ['server namespace', 'kubeflow'],
    ['user namespace', 'tenant'],
    ['omitted namespace', undefined],
  ])(
    'rejects an untrusted workflow endpoint in the %s before credentials or storage IO',
    async (_description, namespace) => {
      vi.mocked(getArgoWorkflow).mockResolvedValue(
        workflowWithEndpoint('https://untrusted.example.com', false),
      );

      const response = await createRequest()
        .get('/k8s/pod/logs')
        .query({ podname: podName, ...(namespace ? { podnamespace: namespace } : {}) })
        .expect(500);

      expect(response.text).toContain(
        'Artifact store endpoint https://untrusted.example.com is not allowed.',
      );
      expect(response.text).toContain(
        'Ask a cluster operator to add this exact origin to the cluster-level ALLOWED_ARTIFACT_ENDPOINTS setting.',
      );
      expect(getPodLogs).toHaveBeenCalledWith(podName, namespace, 'main');
      expect(getArgoWorkflow).toHaveBeenCalledWith('workflow-1', namespace);
      expect(getK8sSecret).not.toHaveBeenCalled();
      expect(MinioClient).not.toHaveBeenCalled();
      expect(getObject).not.toHaveBeenCalled();
    },
  );

  it.each([
    ['operator-store.kubeflow:9000', false],
    ['operator-store.kubeflow:9001', true],
    ['https://operator-s3.example.com:9444', false],
    ['https://extra-store.example.com:9444', false],
  ])(
    'requires the exact trusted scheme and port for %s (insecure=%s)',
    async (endpoint, insecure) => {
      vi.mocked(getArgoWorkflow).mockResolvedValue(workflowWithEndpoint(endpoint, insecure));

      const response = await createRequest({
        ALLOWED_ARTIFACT_ENDPOINTS: 'https://extra-store.example.com:9443',
      })
        .get('/k8s/pod/logs')
        .query({ podname: podName, podnamespace: 'kubeflow' })
        .expect(500);

      expect(response.text).toContain(
        'Ask a cluster operator to add this exact origin to the cluster-level ALLOWED_ARTIFACT_ENDPOINTS setting.',
      );
      expect(getK8sSecret).not.toHaveBeenCalled();
      expect(MinioClient).not.toHaveBeenCalled();
      expect(getObject).not.toHaveBeenCalled();
    },
  );

  it.each([
    ['configured MinIO', 'operator-store.kubeflow:9000', true, 'operator-store.kubeflow', 9000],
    [
      'configured S3',
      'https://operator-s3.example.com:9443',
      false,
      'operator-s3.example.com',
      9443,
    ],
    [
      'additional origin',
      'https://extra-store.example.com:9443',
      false,
      'extra-store.example.com',
      9443,
    ],
  ])(
    'retrieves workflow logs from the %s origin',
    async (_description, endpoint, insecure, endPoint, port) => {
      vi.mocked(getArgoWorkflow).mockResolvedValue(workflowWithEndpoint(endpoint, insecure));

      const response = await createRequest({
        ALLOWED_ARTIFACT_ENDPOINTS: 'https://extra-store.example.com:9443',
      })
        .get('/k8s/pod/logs')
        .query({ podname: podName, podnamespace: 'kubeflow' })
        .expect(200, logContent);

      expect(response.headers['content-type']).toMatch(/^text\/plain/);
      expect(response.headers['content-disposition']).toBe('attachment');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
      expect(getK8sSecret).toHaveBeenCalledTimes(2);
      expect(getK8sSecret).toHaveBeenCalledWith('workflow-store', 'access-key', 'kubeflow');
      expect(getK8sSecret).toHaveBeenCalledWith('workflow-store', 'secret-key', 'kubeflow');
      expect(MinioClient).toHaveBeenCalledExactlyOnceWith({
        accessKey: 'workflow-access-key',
        secretKey: 'workflow-secret-key',
        endPoint,
        port,
        useSSL: !insecure,
      });
      expect(getObject).toHaveBeenCalledExactlyOnceWith('workflow-logs', 'tenant/task/main.log');
    },
  );

  it.each(['kubeflow', 'pipelines'])(
    'retrieves the stock manifest archive in installation namespace %s without fallback',
    async (installationNamespace) => {
      const manifest = loadYaml(
        readFileSync(
          new URL(
            '../../../manifests/kustomize/third-party/argo/base/workflow-controller-configmap-patch.yaml',
            import.meta.url,
          ),
          'utf8',
        ),
      ) as { data: { artifactRepository: string } };
      const repository = loadYaml(
        manifest.data.artifactRepository
          .replaceAll('$(kfp-namespace)', installationNamespace)
          .replaceAll('$(kfp-artifact-bucket-name)', 'mlpipeline'),
      ) as ArchivedArtifactRepository;
      const { workflow, logKey } = workflowWithRepository(repository, 'tenant');
      vi.mocked(getArgoWorkflow).mockResolvedValue(workflow);

      await createRequest({
        MINIO_HOST: 'seaweedfs',
        MINIO_NAMESPACE: installationNamespace,
        AWS_S3_ENDPOINT: '',
        ALLOWED_ARTIFACT_ENDPOINTS: '',
      })
        .get('/k8s/pod/logs')
        .query({ podname: podName, podnamespace: 'tenant' })
        .expect(200, logContent);

      expect(repository.s3.endpoint).toBe(`seaweedfs.${installationNamespace}.svc:9000`);
      expect(logKey).toBe(`private-artifacts/tenant/workflow-1/2026/09/05/${podName}/main.log`);
      expect(getK8sSecret).not.toHaveBeenCalled();
      expect(MinioClient).toHaveBeenCalledExactlyOnceWith({
        accessKey: 'frontend-access-key',
        secretKey: 'frontend-secret-key',
        endPoint: `seaweedfs.${installationNamespace}.svc`,
        port: 9000,
        useSSL: false,
      });
      expect(getObject).toHaveBeenCalledExactlyOnceWith('mlpipeline', logKey);
    },
  );

  it.each([
    ['default domain', 'seaweedfs', undefined, 'seaweedfs.kubeflow.svc.cluster.local'],
    ['custom domain', 'seaweedfs', '.svc.cluster.corp', 'seaweedfs.kubeflow.svc.cluster.corp'],
    [
      'domain without leading dot',
      'seaweedfs',
      'svc.cluster.corp',
      'seaweedfs.kubeflow.svc.cluster.corp',
    ],
    [
      'custom object store',
      'operator-store',
      '.svc.cluster.corp',
      'operator-store.kubeflow.svc.cluster.corp',
    ],
  ])(
    'retrieves the actual profile-controller archive with %s without an allowlist or fallback',
    async (_description, objectStoreHost, clusterDomain, endPoint) => {
      const repository = JSON.parse(
        execFileSync(
          'python3',
          [
            fileURLToPath(new URL('./testdata/profile-artifact-repository.py', import.meta.url)),
            objectStoreHost,
            clusterDomain ?? '',
            'tenant',
          ],
          { encoding: 'utf8', timeout: 5000 },
        ),
      ) as ArchivedArtifactRepository;
      const { workflow, logKey } = workflowWithRepository(repository, 'tenant');
      vi.mocked(getArgoWorkflow).mockResolvedValue(workflow);

      await createRequest({
        MINIO_HOST: objectStoreHost,
        ...(clusterDomain ? { CLUSTER_DOMAIN: clusterDomain } : {}),
        AWS_S3_ENDPOINT: '',
        ALLOWED_ARTIFACT_ENDPOINTS: '',
      })
        .get('/k8s/pod/logs')
        .query({ podname: podName, podnamespace: 'tenant' })
        .expect(200, logContent);

      expect(getK8sSecret).not.toHaveBeenCalled();
      expect(MinioClient).toHaveBeenCalledExactlyOnceWith({
        accessKey: 'frontend-access-key',
        secretKey: 'frontend-secret-key',
        endPoint,
        port: 9000,
        useSSL: false,
      });
      expect(logKey).toBe(`private-artifacts/tenant/workflow-1/2026/09/05/${podName}/main.log`);
      expect(getObject).toHaveBeenCalledExactlyOnceWith('mlpipeline', logKey);
    },
  );

  it.each([
    ['seaweedfs.tenant.svc:9000', true],
    ['other-store.kubeflow.svc:9000', true],
    ['seaweedfs.kubeflow.svc.evil.example:9000', true],
    ['seaweedfs.kubeflow.svc.cluster.corp.evil.example:9000', true],
    ['seaweedfs.kubeflow.svc.cluster.local:9000', true],
    ['seaweedfs.kubeflow.svc:9000', false],
    ['seaweedfs.kubeflow.svc:9001', true],
    ['seaweedfs.kubeflow.svc.cluster.corp:9000', false],
    ['seaweedfs.kubeflow.svc.cluster.corp:9001', true],
  ])(
    'rejects a workflow archive outside the generated exact origins: %s (insecure=%s)',
    async (endpoint, insecure) => {
      vi.mocked(getArgoWorkflow).mockResolvedValue(workflowWithEndpoint(endpoint, insecure));

      const response = await createRequest({
        MINIO_HOST: 'seaweedfs',
        CLUSTER_DOMAIN: '.svc.cluster.corp',
        AWS_S3_ENDPOINT: '',
        ALLOWED_ARTIFACT_ENDPOINTS: '',
      })
        .get('/k8s/pod/logs')
        .query({ podname: podName, podnamespace: 'tenant' })
        .expect(500);

      expect(response.text).toContain(
        'Ask a cluster operator to add this exact origin to the cluster-level ALLOWED_ARTIFACT_ENDPOINTS setting.',
      );
      expect(getK8sSecret).not.toHaveBeenCalled();
      expect(MinioClient).not.toHaveBeenCalled();
      expect(getObject).not.toHaveBeenCalled();
    },
  );

  it('uses frontend credentials for a trusted user-namespace workflow without reading its Secrets', async () => {
    vi.mocked(getArgoWorkflow).mockResolvedValue(
      workflowWithEndpoint('operator-store.kubeflow:9000', true),
    );

    await createRequest()
      .get('/k8s/pod/logs')
      .query({ podname: podName, podnamespace: 'tenant' })
      .expect(200, logContent);

    expect(getK8sSecret).not.toHaveBeenCalled();
    expect(MinioClient).toHaveBeenCalledExactlyOnceWith({
      accessKey: 'frontend-access-key',
      secretKey: 'frontend-secret-key',
      endPoint: 'operator-store.kubeflow',
      port: 9000,
      useSSL: false,
    });
    expect(getObject).toHaveBeenCalledExactlyOnceWith('workflow-logs', 'tenant/task/main.log');
  });

  it('falls back to the operator archive after rejecting an untrusted workflow when archiveLogs is enabled', async () => {
    vi.mocked(getArgoWorkflow).mockResolvedValue(
      workflowWithEndpoint('https://untrusted.example.com', false),
    );

    await createRequest({
      ARGO_ARCHIVE_LOGS: 'true',
      ARGO_ARCHIVE_ARTIFACTORY: 'minio',
      ARGO_ARCHIVE_BUCKETNAME: 'operator-logs',
      ARGO_KEYFORMAT: 'archive/{{workflow.namespace}}/{{pod.name}}',
    })
      .get('/k8s/pod/logs')
      .query({ podname: podName, podnamespace: 'kubeflow' })
      .expect(200, logContent);

    expect(getArgoWorkflow).toHaveBeenCalledWith('workflow-1', 'kubeflow');
    expect(getK8sSecret).not.toHaveBeenCalled();
    expect(MinioClient).toHaveBeenCalledExactlyOnceWith({
      accessKey: 'operator-access-key',
      secretKey: 'operator-secret-key',
      endPoint: 'operator-store.kubeflow',
      port: 9000,
      useSSL: false,
    });
    expect(getObject).toHaveBeenCalledExactlyOnceWith(
      'operator-logs',
      `archive/kubeflow/${podName}/main.log`,
    );
  });
});

// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import type { Request } from 'express';
import { PassThrough } from 'node:stream';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { getAuthorizedPodLogs, getPodForLogAuthorization } from '../k8s-helper.js';
import { createS3ArtifactStore, getObjectStream } from '../artifact-store.js';
import { createAuthorizedPodLogsReader } from './authorized-pod-logs.js';

vi.mock('../k8s-helper.js', () => ({
  getAuthorizedPodLogs: vi.fn(),
  getPodForLogAuthorization: vi.fn(),
}));
vi.mock('../artifact-store.js', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../artifact-store.js')>()),
  createS3ArtifactStore: vi.fn(),
  getObjectStream: vi.fn(),
}));

const context = {
  runId: 'run-a',
  namespace: 'team-a',
  workflowName: 'workflow-a',
  archiveKey: 'private-artifacts/team-a/workflow-a/2026/11/12/pod-a/main.log',
};
const options = {
  endPoint: 'store.example',
  port: 9000,
  useSSL: false,
  accessKey: 'synthetic',
  secretKey: 'synthetic',
};
const reader = (
  containerName = 'main',
  nativeStandaloneReader?: (req: Request) => Promise<PassThrough>,
) =>
  createAuthorizedPodLogsReader(
    'http://api.example',
    'apis/v2',
    {
      archiveLogs: true,
      archiveBucketName: 'logs',
      archiveArtifactory: 'minio',
      keyFormat: '',
      artifactRepositoriesLookup: false,
      artifactRepositoryEndpoints: ['http://store-alias.example:9000'],
    },
    {
      minio: options,
      aws: {
        endPoint: 's3.amazonaws.com',
        region: 'test',
        accessKey: 'synthetic',
        secretKey: 'synthetic',
        useSSL: true,
      },
    },
    containerName,
    nativeStandaloneReader,
  );
const request = () =>
  ({
    query: { runid: 'run-a', podname: 'pod-a', podnamespace: 'untrusted' },
    headers: { authorization: 'Bearer synthetic', 'kubeflow-userid': 'alice', host: 'ui.example' },
  }) as unknown as Request;
const api = vi.fn();
let archive: PassThrough;

beforeEach(() => {
  vi.clearAllMocks();
  archive = new PassThrough();
  archive.end('SYNTHETIC_HISTORICAL_LOG\n');
  vi.stubGlobal('fetch', api);
  api.mockResolvedValue({ ok: true, json: async () => ({ ...context }) });
  vi.mocked(getPodForLogAuthorization).mockRejectedValue({ code: 404 });
  vi.mocked(getObjectStream).mockResolvedValue(archive);
});
afterEach(() => vi.unstubAllGlobals());

describe('authorized pod logs', () => {
  it('retains native standalone readers only after an explicit matching native context', async () => {
    api.mockResolvedValue({
      ok: true,
      json: async () => ({
        runId: 'run-a',
        namespace: '',
        workflowName: '',
        archiveOnly: false,
      }),
    });
    const native = new PassThrough();
    native.end('NATIVE_STANDALONE_LOG');
    const fallback = vi.fn().mockResolvedValue(native);
    const req = request();
    const stream = await reader('main', fallback)(req);
    let content = '';
    for await (const chunk of stream) content += chunk.toString();
    expect(content).toBe('NATIVE_STANDALONE_LOG');
    expect(fallback).toHaveBeenCalledWith(req);
    expect(api).toHaveBeenCalled();
    expect(getPodForLogAuthorization).not.toHaveBeenCalled();
    expect(getObjectStream).not.toHaveBeenCalled();
  });

  it('reads standalone imported archives without invoking the native callback', async () => {
    api.mockResolvedValue({ ok: true, json: async () => ({ ...context, archiveOnly: true }) });
    const fallback = vi.fn();
    const stream = await reader('main', fallback)(request());
    let content = '';
    for await (const chunk of stream) content += chunk.toString();
    expect(content).toBe('SYNTHETIC_HISTORICAL_LOG\n');
    expect(fallback).not.toHaveBeenCalled();
    expect(getPodForLogAuthorization).not.toHaveBeenCalled();
    expect(getAuthorizedPodLogs).not.toHaveBeenCalled();
  });

  it.each([
    { ...context },
    { ...context, runId: 'different-run', archiveOnly: false },
    { ...context, archiveOnly: 'false' },
  ])('does not restore standalone fallback for an invalid context %j', async (body) => {
    api.mockResolvedValue({ ok: true, json: async () => body });
    const fallback = vi.fn();
    await expect(reader('main', fallback)(request())).rejects.toThrow();
    expect(fallback).not.toHaveBeenCalled();
    expect(getPodForLogAuthorization).not.toHaveBeenCalled();
    expect(getObjectStream).not.toHaveBeenCalled();
  });

  it('does not restore standalone fallback after API authorization failure', async () => {
    api.mockResolvedValue({ ok: false, status: 403 });
    const fallback = vi.fn();
    await expect(reader('main', fallback)(request())).rejects.toMatchObject({ statusCode: 403 });
    expect(fallback).not.toHaveBeenCalled();
    expect(getPodForLogAuthorization).not.toHaveBeenCalled();
    expect(getObjectStream).not.toHaveBeenCalled();
  });

  it('reads imported archives without touching a colliding live Pod', async () => {
    api.mockResolvedValue({ ok: true, json: async () => ({ ...context, archiveOnly: true }) });
    vi.mocked(getPodForLogAuthorization).mockResolvedValue({
      metadata: {
        namespace: 'team-a',
        labels: { 'pipeline/runid': 'run-a' },
      },
    });
    vi.mocked(getAuthorizedPodLogs).mockResolvedValue('UNRELATED_LIVE_POD_LOG');
    const stream = await reader()(request());
    let content = '';
    for await (const chunk of stream) content += chunk.toString();
    expect(content).toBe('SYNTHETIC_HISTORICAL_LOG\n');
    expect(getPodForLogAuthorization).not.toHaveBeenCalled();
    expect(getAuthorizedPodLogs).not.toHaveBeenCalled();
    expect(getObjectStream).toHaveBeenCalledWith(
      expect.objectContaining({ key: context.archiveKey }),
    );
  });

  it.each([
    { archiveKey: context.archiveKey.replace('/team-a/', '/team-b/') },
    { archiveKey: context.archiveKey.replace('/workflow-a/', '/workflow-b/') },
    { archiveBucket: 'foreign-bucket' },
    { archiveEndpoint: 'foreign.example:9000', archiveInsecure: true },
    { namespace: '' },
    { workflowName: '' },
  ])('denies imported storage or identity mismatch %j before any I/O', async (override) => {
    api.mockResolvedValue({
      ok: true,
      json: async () => ({ ...context, archiveOnly: true, ...override }),
    });
    await expect(reader()(request())).rejects.toMatchObject({ statusCode: 403 });
    expect(getPodForLogAuthorization).not.toHaveBeenCalled();
    expect(getAuthorizedPodLogs).not.toHaveBeenCalled();
    expect(createS3ArtifactStore).not.toHaveBeenCalled();
    expect(getObjectStream).not.toHaveBeenCalled();
  });

  it.each(['true', 1, null, {}])('rejects invalid archiveOnly flag %j', async (archiveOnly) => {
    api.mockResolvedValue({ ok: true, json: async () => ({ ...context, archiveOnly }) });
    await expect(reader()(request())).rejects.toMatchObject({ statusCode: 502 });
    expect(getPodForLogAuthorization).not.toHaveBeenCalled();
    expect(getObjectStream).not.toHaveBeenCalled();
  });

  it('retains live behavior for an explicit false archiveOnly flag', async () => {
    api.mockResolvedValue({ ok: true, json: async () => ({ ...context, archiveOnly: false }) });
    vi.mocked(getPodForLogAuthorization).mockResolvedValue({
      metadata: {
        namespace: 'team-a',
        labels: { 'pipeline/runid': 'run-a' },
      },
    });
    vi.mocked(getAuthorizedPodLogs).mockResolvedValue('NATIVE_LIVE_LOG');
    const stream = await reader()(request());
    let content = '';
    for await (const chunk of stream) content += chunk.toString();
    expect(content).toBe('NATIVE_LIVE_LOG');
    expect(getObjectStream).not.toHaveBeenCalled();
  });

  it('authorizes historical logs with original credentials and operator store configuration', async () => {
    const stream = await reader()(request());
    let content = '';
    for await (const chunk of stream) content += chunk.toString();
    expect(content).toBe('SYNTHETIC_HISTORICAL_LOG\n');
    expect(api.mock.calls[0][0]).toBe(
      'http://api.example/apis/v2/runs/run-a/nodes/pod-a/log?archive_context=true',
    );
    const headers = api.mock.calls[0][1].headers as Headers;
    expect(headers.get('authorization')).toBe('Bearer synthetic');
    expect(headers.get('kubeflow-userid')).toBe('alice');
    expect(headers.has('host')).toBe(false);
    expect(getPodForLogAuthorization).toHaveBeenCalledWith('pod-a', 'team-a');
    expect(createS3ArtifactStore).toHaveBeenCalledWith(options, 's3', 'logs');
    expect(getObjectStream).toHaveBeenCalledWith(
      expect.objectContaining({ key: context.archiveKey }),
    );
  });

  it('reads a bound live main container without touching storage', async () => {
    vi.mocked(getPodForLogAuthorization).mockResolvedValue({
      metadata: { namespace: 'team-a', labels: { 'pipeline/runid': 'run-a' } },
    });
    vi.mocked(getAuthorizedPodLogs).mockResolvedValue('live log');
    const stream = await reader()(request());
    expect(getAuthorizedPodLogs).toHaveBeenCalledWith('pod-a', 'team-a', 'main');
    expect(createS3ArtifactStore).not.toHaveBeenCalled();
    stream.destroy();
  });

  it('preserves custom live containers but does not substitute main archives', async () => {
    vi.mocked(getPodForLogAuthorization).mockResolvedValue({
      metadata: { namespace: 'team-a', labels: { 'pipeline/runid': 'run-a' } },
    });
    vi.mocked(getAuthorizedPodLogs).mockResolvedValue('custom live log');
    const stream = await reader('sidecar')(request());
    expect(getAuthorizedPodLogs).toHaveBeenCalledWith('pod-a', 'team-a', 'sidecar');
    stream.destroy();
    vi.mocked(getPodForLogAuthorization).mockRejectedValue({ code: 404 });
    await expect(reader('sidecar')(request())).rejects.toMatchObject({ statusCode: 404 });
    expect(getObjectStream).not.toHaveBeenCalled();
  });

  it.each([401, 403, 404, 500])('does not fall back after API status %s', async (status) => {
    api.mockResolvedValue({ ok: false, status });
    await expect(reader()(request())).rejects.toMatchObject({ statusCode: status });
    expect(getPodForLogAuthorization).not.toHaveBeenCalled();
    expect(createS3ArtifactStore).not.toHaveBeenCalled();
  });

  it.each([403, 429, 500])('does not fall back after Kubernetes status %s', async (code) => {
    vi.mocked(getPodForLogAuthorization).mockRejectedValue({ code });
    await expect(reader()(request())).rejects.toMatchObject({ code });
    expect(createS3ArtifactStore).not.toHaveBeenCalled();
  });

  it.each([404, 403])('allows only a missing live-log response to fall back: %s', async (code) => {
    vi.mocked(getPodForLogAuthorization).mockResolvedValue({
      metadata: { namespace: 'team-a', labels: { 'pipeline/runid': 'run-a' } },
    });
    vi.mocked(getAuthorizedPodLogs).mockRejectedValue({ code });
    if (code === 404) {
      expect(await reader()(request())).toBe(archive);
      expect(getObjectStream).toHaveBeenCalled();
    } else {
      await expect(reader()(request())).rejects.toMatchObject({ code });
      expect(getObjectStream).not.toHaveBeenCalled();
    }
  });

  it('rejects a pod belonging to another run without reading any logs', async () => {
    vi.mocked(getPodForLogAuthorization).mockResolvedValue({
      metadata: { namespace: 'team-a', labels: { 'pipeline/runid': 'run-b' } },
    });
    await expect(reader()(request())).rejects.toMatchObject({ statusCode: 403 });
    expect(getAuthorizedPodLogs).not.toHaveBeenCalled();
    expect(createS3ArtifactStore).not.toHaveBeenCalled();
  });

  it.each([
    { archiveKey: context.archiveKey.replace('/team-a/', '/team-b/') },
    { archiveKey: context.archiveKey.replace('/workflow-a/', '/workflow-b/') },
    { archiveBucket: 'victim-bucket' },
    { archiveEndpoint: 'attacker.example:9000', archiveInsecure: true },
    { archiveKey: context.archiveKey.replace('/main.log', '/secret.bin') },
    { runId: 'different-run' },
    { namespace: '' },
  ])('rejects untrusted archive hints or invalid identity %j', async (override) => {
    api.mockResolvedValue({ ok: true, json: async () => ({ ...context, ...override }) });
    await expect(reader()(request())).rejects.toMatchObject({ statusCode: 403 });
    expect(createS3ArtifactStore).not.toHaveBeenCalled();
    expect(getObjectStream).not.toHaveBeenCalled();
  });

  it('accepts the configured store alias without routing to it', async () => {
    api.mockResolvedValue({
      ok: true,
      json: async () => ({
        ...context,
        archiveBucket: 'logs',
        archiveEndpoint: 'store-alias.example:9000',
        archiveInsecure: true,
      }),
    });
    await reader()(request());
    expect(createS3ArtifactStore).toHaveBeenCalledWith(options, 's3', 'logs');
  });

  it('uses stored workflow date hints only inside the authorized layout', async () => {
    api.mockResolvedValue({
      ok: true,
      json: async () => ({
        ...context,
        archiveKey: undefined,
        workflowCreationTimestamp: '2026-11-13T00:00:01Z',
      }),
    });
    await reader()(request());
    expect(getObjectStream).toHaveBeenCalledWith(
      expect.objectContaining({ key: context.archiveKey.replace('/12/', '/13/') }),
    );
  });

  it.each([null, [], 'not a context', { runId: 'run-a' }])(
    'rejects malformed context %j',
    async (body) => {
      api.mockResolvedValue({ ok: true, json: async () => body });
      await expect(reader()(request())).rejects.toMatchObject({ statusCode: 502 });
      expect(getPodForLogAuthorization).not.toHaveBeenCalled();
    },
  );

  it('uses createdat only as a locator inside the authorized workflow root', async () => {
    api.mockResolvedValue({ ok: true, json: async () => ({ ...context, archiveKey: undefined }) });
    const req = request();
    req.query.createdat = '2026-11-13';
    await reader()(req);
    expect(getObjectStream).toHaveBeenCalledWith(
      expect.objectContaining({ key: context.archiveKey.replace('/12/', '/13/') }),
    );
  });

  it('requires a run id before any upstream request', async () => {
    const req = request();
    delete req.query.runid;
    await expect(reader()(req)).rejects.toMatchObject({ statusCode: 400 });
    expect(api).not.toHaveBeenCalled();
  });
});

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

import type { Request, Response } from 'express';
import { PassThrough } from 'node:stream';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { AuthorizeFn } from '../helpers/auth.js';

const getPodLogsStream = vi.fn();

vi.mock('../workflow-helper.js', () => ({
  composePodLogsStreamHandler: vi.fn(() => getPodLogsStream),
  createPodLogsMinioRequestConfig: vi.fn(),
  getPodLogsStreamFromK8s: vi.fn(),
  getPodLogsStreamFromWorkflow: vi.fn(),
  toGetPodLogsStream: vi.fn(),
}));

import { getPodLogsHandler } from './pod-logs.js';

function makeHandler() {
  return getPodLogsHandler(
    {
      archiveArtifactory: 'minio',
      archiveBucketName: '',
      archiveLogs: false,
      artifactRepositoriesLookup: false,
      keyFormat: '',
    },
    {
      aws: { endPoint: 's3.amazonaws.com' },
      minio: { endPoint: 'minio-service.kubeflow' },
    },
    'main',
    vi.fn() as unknown as AuthorizeFn,
    false,
  );
}

function makeRequest(): Request {
  return {
    query: { podname: 'pod-1' },
  } as unknown as Request;
}

function makeResponse() {
  const response = new PassThrough() as PassThrough & Response;
  response.setHeader = vi.fn();
  response.type = vi.fn(() => response);
  response.status = vi.fn(() => response);
  response.send = vi.fn(() => response);
  response.destroy = vi.fn(() => response) as typeof response.destroy;
  return response;
}

describe('getPodLogsHandler stream failures', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('serves pod logs as a plain-text attachment with MIME sniffing disabled', async () => {
    const logs = new PassThrough();
    getPodLogsStream.mockResolvedValue(logs);
    const response = makeResponse();

    await makeHandler()(makeRequest(), response, vi.fn());

    expect(response.setHeader).toHaveBeenCalledWith('X-Content-Type-Options', 'nosniff');
    expect(response.setHeader).toHaveBeenCalledWith('Content-Disposition', 'attachment');
    expect(response.type).toHaveBeenCalledWith('text/plain');
    logs.end();
  });

  it('destroys a committed response when archived logs fail mid-stream', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const logs = new PassThrough();
    const destroyLogs = vi.spyOn(logs, 'destroy');
    getPodLogsStream.mockResolvedValue(logs);
    const response = makeResponse();
    Object.defineProperty(response, 'headersSent', { configurable: true, value: true });

    await makeHandler()(makeRequest(), response, vi.fn());
    logs.emit('error', new Error('storage connection reset'));

    await vi.waitFor(() => expect(response.destroy).toHaveBeenCalledOnce());
    expect(destroyLogs).toHaveBeenCalledOnce();
    expect(response.status).not.toHaveBeenCalled();
    expect(response.send).not.toHaveBeenCalled();
    expect(consoleError).toHaveBeenCalledWith(
      '[pod-logs] aborting committed response:',
      expect.any(Error),
    );
  });

  it('destroys the archived log stream when the client disconnects', async () => {
    const logs = new PassThrough();
    const destroyLogs = vi.spyOn(logs, 'destroy');
    getPodLogsStream.mockResolvedValue(logs);
    const response = makeResponse();

    await makeHandler()(makeRequest(), response, vi.fn());
    response.emit('close');

    expect(destroyLogs).toHaveBeenCalledOnce();
    expect(response.status).not.toHaveBeenCalled();
    expect(response.send).not.toHaveBeenCalled();
  });

  it('destroys an archived log stream acquired after the client has disconnected', async () => {
    const logs = new PassThrough();
    const destroyLogs = vi.spyOn(logs, 'destroy');
    const pipeLogs = vi.spyOn(logs, 'pipe');
    let resolveLogs: ((stream: PassThrough) => void) | undefined;
    getPodLogsStream.mockImplementation(
      () =>
        new Promise<PassThrough>((resolve) => {
          resolveLogs = resolve;
        }),
    );
    const response = makeResponse();

    const handling = makeHandler()(makeRequest(), response, vi.fn());
    await vi.waitFor(() => expect(getPodLogsStream).toHaveBeenCalledOnce());
    Object.defineProperty(response, 'destroyed', { configurable: true, value: true });
    response.emit('close');
    resolveLogs?.(logs);
    await handling;

    expect(destroyLogs).toHaveBeenCalledOnce();
    expect(pipeLogs).not.toHaveBeenCalled();
    expect(response.status).not.toHaveBeenCalled();
    expect(response.send).not.toHaveBeenCalled();
  });

  it('returns a 404 when an archive lookup fails before headers are sent', async () => {
    const logs = new PassThrough();
    getPodLogsStream.mockResolvedValue(logs);
    const response = makeResponse();
    Object.defineProperty(response, 'headersSent', { configurable: true, value: false });

    await makeHandler()(makeRequest(), response, vi.fn());
    logs.destroy(new Error('Unable to find pod log archive information'));

    await vi.waitFor(() => expect(response.status).toHaveBeenCalledWith(404));
    expect(response.send).toHaveBeenCalledWith('pod not found');
    expect(response.destroy).not.toHaveBeenCalled();
  });

  it('reports a storage failure emitted before the request listener is attached', async () => {
    const streamError = new Error('immediate storage failure');
    const logs = new PassThrough();
    logs.on('error', () => undefined);
    logs.destroy(streamError);
    await new Promise<void>((resolve) => setImmediate(resolve));
    getPodLogsStream.mockResolvedValue(logs);
    const response = makeResponse();
    Object.defineProperty(response, 'headersSent', { configurable: true, value: false });
    const pipeLogs = vi.spyOn(logs, 'pipe');

    await makeHandler()(makeRequest(), response, vi.fn());

    expect(response.status).toHaveBeenCalledWith(500);
    expect(response.send).toHaveBeenCalledWith(
      'Could not get main container logs: Error: immediate storage failure',
    );
    expect(pipeLogs).not.toHaveBeenCalled();
  });
});

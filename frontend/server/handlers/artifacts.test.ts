// Copyright 2025 The Kubeflow Authors
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

import { getEventListeners } from 'events';
import { PassThrough, Writable } from 'stream';
import { describe, it, expect, vi } from 'vitest';
import type { Request, Response } from 'express';
import type { Client as MinioClient } from 'minio';
import { resolveArtifactCoordinates } from '../helpers/artifact-coordinates.js';
import {
  pipePreviewResponse,
  sendArtifactError,
  streamDirectoryAsTarGz,
  TEST_ONLY,
  waitForArtifactOperation,
} from './artifacts.js';

vi.mock('../k8s-helper.js');

function makeRequest(path: string, query: Record<string, unknown> = {}): Request {
  return { path, query } as unknown as Request;
}

describe('sendArtifactError', () => {
  it('destroys a committed response so a partial download is not reported as complete', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const response = {
      headersSent: true,
      destroy: vi.fn(),
      end: vi.fn(),
    } as unknown as Response;

    sendArtifactError(response, 500, 'storage stream failed');

    expect(response.destroy).toHaveBeenCalledOnce();
    expect(response.end).not.toHaveBeenCalled();
    expect(consoleError).toHaveBeenCalledWith(
      '[artifacts] aborting committed response: storage stream failed',
    );
  });

  it('does not attempt to send an error body after a stream destroyed the response', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const response = {
      headersSent: false,
      destroyed: true,
      writableEnded: false,
      destroy: vi.fn(),
      status: vi.fn(),
      type: vi.fn(),
      send: vi.fn(),
    } as unknown as Response;

    sendArtifactError(response, 500, 'storage stream failed');

    expect(response.destroy).toHaveBeenCalledOnce();
    expect(response.status).not.toHaveBeenCalled();
    expect(response.type).not.toHaveBeenCalled();
    expect(response.send).not.toHaveBeenCalled();
    expect(consoleError).toHaveBeenCalledWith(
      '[artifacts] aborting committed response: storage stream failed',
    );
  });
});

describe('artifact stream lifecycle', () => {
  it('removes archive abort listeners after each completed object operation', async () => {
    const controller = new AbortController();

    for (let index = 0; index < 1000; index++) {
      await expect(
        waitForArtifactOperation(Promise.resolve(index), controller.signal),
      ).resolves.toBe(index);
      expect(getEventListeners(controller.signal, 'abort')).toHaveLength(0);
    }
  });

  it('destroys the upstream preview stream without reporting a client disconnect as an error', async () => {
    const source = new PassThrough();
    const response = new PassThrough() as unknown as Response;
    const onError = vi.fn();

    pipePreviewResponse(source, response, 0, onError);
    response.destroy();

    await vi.waitFor(() => expect(source.destroyed).toBe(true));
    expect(onError).not.toHaveBeenCalled();
    expect(source.destroyed).toBe(true);
  });

  it('destroys a preview source acquired after the client has already disconnected', () => {
    const source = new PassThrough();
    const pipeSource = vi.spyOn(source, 'pipe');
    const response = new PassThrough() as unknown as Response;
    const onError = vi.fn();
    response.destroy();

    pipePreviewResponse(source, response, 0, onError);

    expect(source.destroyed).toBe(true);
    expect(pipeSource).not.toHaveBeenCalled();
    expect(onError).not.toHaveBeenCalled();
  });

  it('keeps an uncommitted response writable when the first directory object cannot be fetched', async () => {
    const client = {
      getObject: vi.fn(async () => {
        throw new Error('first object unavailable');
      }),
      listObjectsV2Query: vi.fn(async () => ({
        objects: [{ name: 'directory/file.txt', size: 4 }],
        isTruncated: false,
        nextContinuationToken: '',
      })),
    } as unknown as MinioClient;
    const response = new PassThrough() as unknown as Response;
    response.setHeader = vi.fn();

    await expect(
      streamDirectoryAsTarGz({ bucket: 'ml-pipeline', key: 'directory', client }, response),
    ).rejects.toThrow('first object unavailable');

    expect(response.destroyed).toBe(false);
    expect(response.setHeader).not.toHaveBeenCalled();
  });

  it('settles a directory archive when the client disconnects during an entry', async () => {
    const objectStream = new PassThrough();
    const client = {
      getObject: vi.fn(async () => objectStream),
      listObjectsV2Query: vi.fn(async () => ({
        objects: [{ name: 'directory/file.txt', size: 4 }],
        isTruncated: false,
        nextContinuationToken: '',
      })),
    } as unknown as MinioClient;
    const response = new PassThrough() as unknown as Response;
    response.setHeader = vi.fn();

    const archive = streamDirectoryAsTarGz(
      { bucket: 'ml-pipeline', key: 'directory', client },
      response,
    );
    await vi.waitFor(() => expect(client.getObject).toHaveBeenCalledOnce());
    response.destroy();

    await expect(archive).rejects.toThrow();
    expect(objectStream.destroyed).toBe(true);
  });

  it('does not report a directory archive client disconnect as a server error', async () => {
    const objectStream = new PassThrough();
    const client = {
      getObject: vi
        .fn()
        .mockRejectedValueOnce(Object.assign(new Error('not found'), { code: 'NoSuchKey' }))
        .mockResolvedValueOnce(objectStream),
      listObjectsV2Query: vi.fn(async () => ({
        objects: [{ name: 'directory/file.txt', size: 4 }],
        isTruncated: false,
        nextContinuationToken: '',
      })),
    } as unknown as MinioClient;
    const response = new PassThrough() as unknown as Response;
    response.setHeader = vi.fn();
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    consoleError.mockClear();
    const handler = TEST_ONLY.getMinioArtifactHandler({
      bucket: 'ml-pipeline',
      key: 'directory',
      client,
      tryExtract: false,
    });

    const handling = handler({} as Request, response);
    await vi.waitFor(() => expect(client.getObject).toHaveBeenCalledTimes(2));
    response.destroy();

    await expect(handling).resolves.toBeUndefined();
    expect(consoleError).not.toHaveBeenCalled();
    expect(objectStream.destroyed).toBe(true);
  });

  it('does not report a disconnect during the final archive flush as a server error', async () => {
    const objectStream = new PassThrough();
    let finalResponseWrite: (() => void) | undefined;
    let responseWriteCount = 0;
    const client = {
      getObject: vi
        .fn()
        .mockRejectedValueOnce(Object.assign(new Error('not found'), { code: 'NoSuchKey' }))
        .mockResolvedValueOnce(objectStream),
      listObjectsV2Query: vi.fn(async () => ({
        objects: [{ name: 'directory/file.txt', size: 4 }],
        isTruncated: false,
        nextContinuationToken: '',
      })),
    } as unknown as MinioClient;
    const response = new Writable({
      write(_chunk, _encoding, callback) {
        responseWriteCount += 1;
        if (responseWriteCount === 1) {
          callback();
        } else {
          finalResponseWrite = callback;
        }
      },
    }) as unknown as Response;
    response.setHeader = vi.fn();
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    consoleError.mockClear();
    const handler = TEST_ONLY.getMinioArtifactHandler({
      bucket: 'ml-pipeline',
      key: 'directory',
      client,
      tryExtract: false,
    });

    const handling = handler({} as Request, response);
    await vi.waitFor(() => expect(client.getObject).toHaveBeenCalledTimes(2));
    objectStream.end('data');
    await vi.waitFor(() => expect(finalResponseWrite).toBeDefined());
    response.destroy();

    await expect(handling).resolves.toBeUndefined();
    expect(consoleError).not.toHaveBeenCalled();
  });

  it('closes a directory object stream returned after the client disconnects', async () => {
    const objectStream = new PassThrough();
    let resolveObject: ((stream: PassThrough) => void) | undefined;
    const client = {
      getObject: vi.fn(
        () =>
          new Promise<PassThrough>((resolve) => {
            resolveObject = resolve;
          }),
      ),
      listObjectsV2Query: vi.fn(async () => ({
        objects: [{ name: 'directory/file.txt', size: 4 }],
        isTruncated: false,
        nextContinuationToken: '',
      })),
    } as unknown as MinioClient;
    const response = new PassThrough() as unknown as Response;
    response.setHeader = vi.fn();

    const archive = streamDirectoryAsTarGz(
      { bucket: 'ml-pipeline', key: 'directory', client },
      response,
    );
    await vi.waitFor(() => expect(client.getObject).toHaveBeenCalledOnce());
    response.destroy();

    await expect(archive).rejects.toThrow();
    resolveObject?.(objectStream);
    await vi.waitFor(() => expect(objectStream.destroyed).toBe(true));
  });

  it('settles when the client disconnects while the initial directory listing is pending', async () => {
    const client = {
      getObject: vi.fn(),
      listObjectsV2Query: vi.fn(() => new Promise(() => undefined)),
    } as unknown as MinioClient;
    const response = new PassThrough() as unknown as Response;
    response.setHeader = vi.fn();

    const archive = streamDirectoryAsTarGz(
      { bucket: 'ml-pipeline', key: 'directory', client },
      response,
    );
    response.destroy();

    await expect(archive).rejects.toThrow('closed before archive streaming started');
    expect(client.getObject).not.toHaveBeenCalled();
  });
});

describe('resolveArtifactCoordinates', () => {
  describe('path-based routes', () => {
    it('extracts coordinates from /artifacts/:source/:bucket/* style URLs', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/hello/world.txt');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello/world.txt',
      });
    });

    it('extracts coordinates from /pipeline-prefixed routes', () => {
      const req = makeRequest('/pipeline/artifacts/s3/my-bucket/path/to/file.csv');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'my-bucket',
        key: 'path/to/file.csv',
      });
    });

    it('extracts coordinates from a non-/pipeline base path', () => {
      const req = makeRequest('/foo/bar/artifacts/minio/my-bucket/some/key');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'my-bucket',
        key: 'some/key',
      });
    });

    it('decodes percent-encoded path segments once (matching Express req.params semantics)', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/hello%2Fworld.txt');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello/world.txt',
      });
    });

    it('preserves a literal %2F when the URL is double-encoded (%252F)', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/hello%252Fworld.txt');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello%2Fworld.txt',
      });
    });

    it('returns null on malformed percent-encoding (fail-closed)', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/bad%ZZkey');
      expect(resolveArtifactCoordinates(req)).toBeNull();
    });

    it('handles keys that contain multiple slashes', () => {
      const req = makeRequest('/artifacts/gcs/my-bucket/a/b/c/d.json');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'gcs',
        bucket: 'my-bucket',
        key: 'a/b/c/d.json',
      });
    });
  });

  describe('query-based fallback (/artifacts/get and unrecognized paths)', () => {
    it('falls back to query when path is /artifacts/get', () => {
      const req = makeRequest('/artifacts/get', {
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello/world.txt',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello/world.txt',
      });
    });

    it('falls back to query when path is /pipeline/artifacts/get', () => {
      const req = makeRequest('/pipeline/artifacts/get', {
        source: 's3',
        bucket: 'my-bucket',
        key: 'data.csv',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'my-bucket',
        key: 'data.csv',
      });
    });

    it('falls back to query when path does not match the artifact patterns', () => {
      const req = makeRequest('/foo/bar', {
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'k',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'k',
      });
    });
  });

  describe('missing or non-string query values', () => {
    it('returns empty strings when query params are absent', () => {
      const req = makeRequest('/artifacts/get');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: '',
        bucket: '',
        key: '',
      });
    });

    it('rejects array-valued query params (treats them as missing)', () => {
      const req = makeRequest('/artifacts/get', {
        source: ['minio', 'sneaky'],
        bucket: 'ml-pipeline',
        key: 'k',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: '',
        bucket: 'ml-pipeline',
        key: 'k',
      });
    });

    it('rejects object-valued query params (treats them as missing)', () => {
      const req = makeRequest('/artifacts/get', {
        source: { nested: 'minio' },
        bucket: 'ml-pipeline',
        key: 'k',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: '',
        bucket: 'ml-pipeline',
        key: 'k',
      });
    });
  });

  describe('coordinate-source spoofing defense', () => {
    it('uses path coordinates when both path and query are present', () => {
      // Attacker plants benign values in query, real values in path.
      const req = makeRequest('/artifacts/minio/victim-bucket/secret.txt', {
        source: 'minio',
        bucket: 'safe-bucket',
        key: 'safe-key',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'victim-bucket',
        key: 'secret.txt',
      });
    });

    it('treats /artifacts/get/x/y as a path-based route with source=get (not the get endpoint)', () => {
      // Only an exact /artifacts/get path uses the query string; any other
      // path that matches /:source/:bucket/* uses path values, even if the
      // first segment happens to literally be "get". The downstream handler
      // then rejects source="get" as an unknown storage source (500).
      const req = makeRequest('/artifacts/get/some-bucket/some-key', {
        source: 'minio',
        bucket: 'safe-bucket',
        key: 'safe-key',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'get',
        bucket: 'some-bucket',
        key: 'some-key',
      });
    });
  });
});

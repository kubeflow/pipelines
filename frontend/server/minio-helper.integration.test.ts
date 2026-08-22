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

import { createServer } from 'http';
import type { AddressInfo } from 'net';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createMinioClient } from './minio-helper.js';

vi.mock('./k8s-helper.js', () => ({ getK8sSecret: vi.fn() }));

describe('MinIO retry integration', () => {
  beforeEach(() => vi.spyOn(Math, 'random').mockReturnValue(0));
  afterEach(() => vi.restoreAllMocks());

  it.each([
    [0, 3],
    [5, 5],
  ])('applies maxRetries=%i to HTTP 500 responses', async (maxRetries, attempts) => {
    let requestCount = 0;
    const server = createServer((_request, response) => {
      requestCount += 1;
      response.writeHead(500).end();
    });
    await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));

    try {
      const { port } = server.address() as AddressInfo;
      const client = await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: '127.0.0.1',
          pathStyle: true,
          port,
          region: 'us-east-1',
          retryOptions: { baseDelayMs: 0, maximumDelayMs: 0, maximumRetryCount: 0 },
          secretKey: 'secretkey',
          useSSL: false,
        },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: { fromEnv: 'true', maxRetries: String(maxRetries) },
        }),
      );

      await expect(client.getObject('bucket', 'key')).rejects.toThrow();
      expect(requestCount).toBe(attempts);
    } finally {
      await new Promise<void>((resolve, reject) =>
        server.close((error) => (error ? reject(error) : resolve())),
      );
    }
  });

  it('shares one total-attempt budget across HTTP and transport failures', async () => {
    let requestCount = 0;
    const server = createServer((_request, response) => {
      requestCount += 1;
      if (requestCount === 2) {
        response.socket?.destroy();
        return;
      }
      response.writeHead(500).end();
    });
    await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));

    try {
      const { port } = server.address() as AddressInfo;
      const client = await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: '127.0.0.1',
          pathStyle: true,
          port,
          region: 'us-east-1',
          retryOptions: { baseDelayMs: 0, maximumDelayMs: 0, maximumRetryCount: 0 },
          secretKey: 'secretkey',
          useSSL: false,
        },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: { fromEnv: 'true', maxRetries: '3' },
        }),
      );

      await expect(client.getObject('bucket', 'key')).rejects.toThrow();
      expect(requestCount).toBe(3);
    } finally {
      await new Promise<void>((resolve, reject) =>
        server.close((error) => (error ? reject(error) : resolve())),
      );
    }
  });

  it.each([
    [400, 'SlowDown', 3],
    [400, 'RequestTimeout', 3],
    [400, 'Throttling', 3],
    [408, 'SlowDown', 3],
    [429, 'RequestTimeout', 3],
    [499, 'Throttling', 3],
    [520, 'SlowDown', 3],
    [408, 'UnknownError', 1],
    [429, 'UnknownError', 1],
    [499, 'UnknownError', 1],
    [520, 'UnknownError', 1],
  ])('matches Go retries for HTTP %i with S3 code %s', async (status, code, attempts) => {
    let requestCount = 0;
    const server = createServer((_request, response) => {
      requestCount += 1;
      response
        .writeHead(status, { 'content-type': 'application/xml' })
        .end(`<Error><Code>${code}</Code><Message>failed</Message></Error>`);
    });
    await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));

    try {
      const { port } = server.address() as AddressInfo;
      const client = await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: '127.0.0.1',
          pathStyle: true,
          port,
          region: 'us-east-1',
          retryOptions: { baseDelayMs: 0, maximumDelayMs: 0, maximumRetryCount: 0 },
          secretKey: 'secretkey',
          useSSL: false,
        },
        'minio',
        JSON.stringify({ Provider: 'minio', Params: { fromEnv: 'true' } }),
      );

      await expect(client.getObject('bucket', 'key')).rejects.toThrow();
      expect(requestCount).toBe(attempts);
    } finally {
      await new Promise<void>((resolve, reject) =>
        server.close((error) => (error ? reject(error) : resolve())),
      );
    }
  });

  it('does not retry service-controlled text that resembles MinIO retry output', async () => {
    let requestCount = 0;
    const server = createServer((_request, response) => {
      requestCount += 1;
      response
        .writeHead(400, { 'content-type': 'application/xml' })
        .end(
          '<Error><Code>AccessDenied</Code>' +
            '<Message>Retryable HTTP status: 500</Message></Error>',
        );
    });
    await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));

    try {
      const { port } = server.address() as AddressInfo;
      const client = await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: '127.0.0.1',
          pathStyle: true,
          port,
          region: 'us-east-1',
          secretKey: 'secretkey',
          useSSL: false,
        },
        'minio',
        JSON.stringify({ Provider: 'minio', Params: { fromEnv: 'true' } }),
      );

      await expect(client.getObject('bucket', 'key')).rejects.toThrow();
      expect(requestCount).toBe(1);
    } finally {
      await new Promise<void>((resolve, reject) =>
        server.close((error) => (error ? reject(error) : resolve())),
      );
    }
  });

  it('sends endpoint base paths with Go-compatible escape normalization', async () => {
    let requestPath: string | undefined;
    const server = createServer((request, response) => {
      requestPath = request.url;
      response.writeHead(200).end('artifact');
    });
    await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));

    try {
      const { port } = server.address() as AddressInfo;
      const endpoint = `http://127.0.0.1:${port}/%2e%2e/%41/%7e/%2f/%3A/raw[bracket]`;
      const client = await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: '127.0.0.1',
          pathStyle: true,
          port,
          region: 'us-east-1',
          secretKey: 'secretkey',
          useSSL: false,
        },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint,
            fromEnv: 'true',
            nativeQuery: 'true',
            use_path_style: 'true',
          },
        }),
      );

      await client.getObject('bucket', 'key');
      expect(requestPath).toBe('/../A/~///:/raw[bracket]/bucket/key');
    } finally {
      await new Promise<void>((resolve, reject) =>
        server.close((error) => (error ? reject(error) : resolve())),
      );
    }
  });

  it.each([
    ['/%252e/%252f', '/%2e/%2f/bucket/key'],
    ['/%FF/%C0%AF/%ED%A0%80', '/%FF/%C0%AF/%ED%A0%80/bucket/key'],
  ])('preserves Go byte-path semantics for endpoint path %s', async (basePath, expectedPath) => {
    let requestPath: string | undefined;
    const server = createServer((request, response) => {
      requestPath = request.url;
      response.writeHead(200).end('artifact');
    });
    await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));

    try {
      const { port } = server.address() as AddressInfo;
      const client = await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: '127.0.0.1',
          pathStyle: true,
          port,
          region: 'us-east-1',
          secretKey: 'secretkey',
          useSSL: false,
        },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: `http://127.0.0.1:${port}${basePath}`,
            fromEnv: 'true',
            nativeQuery: 'true',
            use_path_style: 'true',
          },
        }),
      );

      await client.getObject('bucket', 'key');
      expect(requestPath).toBe(expectedPath);
    } finally {
      await new Promise<void>((resolve, reject) =>
        server.close((error) => (error ? reject(error) : resolve())),
      );
    }
  });
});

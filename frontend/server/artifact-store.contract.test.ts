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
import { Agent, createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { once } from 'node:events';
import type { AddressInfo } from 'node:net';
import { randomUUID } from 'node:crypto';
import {
  CreateBucketCommand,
  DeleteBucketCommand,
  DeleteObjectsCommand,
  PutBucketPolicyCommand,
  PutObjectCommand,
  S3Client,
} from '@aws-sdk/client-s3';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { fromNodeProviderChain } from '@aws-sdk/credential-providers';
import { createS3ArtifactStore } from './artifact-store.js';

vi.mock('@aws-sdk/credential-providers', () => ({ fromNodeProviderChain: vi.fn() }));
vi.mock('./k8s-helper.js', () => ({ getK8sSecret: vi.fn() }));

const credentials = { accessKeyId: 'kfp-contract', secretAccessKey: 'kfp-contract-secret' };
const cleanup: Array<() => Promise<void> | void> = [];
afterEach(async () => {
  for (const close of cleanup.splice(0).reverse()) await close();
  vi.clearAllMocks();
});

async function wireServer(handle: (req: IncomingMessage, res: ServerResponse) => void) {
  const server = createServer(handle);
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  cleanup.push(() => {
    server.closeAllConnections();
    return new Promise<void>((resolve) => server.close(() => resolve()));
  });
  return (server.address() as AddressInfo).port;
}

function errorResponse(res: ServerResponse, status: number, code: string) {
  res.writeHead(status, { 'content-type': 'application/xml' });
  res.end(`<Error><Code>${code}</Code><Message>contract fixture</Message></Error>`);
}

// These tests verify requests and response handling against local protocol fixtures. They do not
// claim live provider compatibility or exercise a provider's authentication service.
describe('S3 artifact store wire contract', () => {
  it.each([
    {
      provider: 'AWS S3',
      endpoint: 's3.us-west-2.amazonaws.com',
      region: 'us-west-2',
      pathStyle: false,
    },
    {
      provider: 'Google Cloud Storage XML API',
      endpoint: 'storage.googleapis.com',
      region: 'auto',
      pathStyle: true,
    },
    {
      provider: 'Cloudflare R2',
      endpoint: 'contract-account.r2.cloudflarestorage.com',
      region: 'auto',
      pathStyle: true,
    },
    {
      provider: 'IBM Cloud Object Storage',
      endpoint: 's3.us-south.cloud-object-storage.appdomain.cloud',
      region: 'us-south',
      pathStyle: true,
    },
  ])('signs and addresses a mocked $provider endpoint', async ({ endpoint, region, pathStyle }) => {
    const requests: IncomingMessage[] = [];
    const port = await wireServer((req, res) => {
      requests.push(req);
      res.end('cloud fixture');
    });
    const agent = new Agent({
      lookup: (_hostname, options, callback) =>
        options.all
          ? callback(null, [{ address: '127.0.0.1', family: 4 }])
          : callback(null, '127.0.0.1', 4),
    });
    cleanup.push(() => agent.destroy());
    const store = await createS3ArtifactStore(
      {
        endPoint: endpoint,
        port,
        useSSL: false,
        region,
        pathStyle,
        transportAgent: agent,
        accessKey: credentials.accessKeyId,
        secretKey: credentials.secretAccessKey,
        sessionToken: 'temporary-session-token',
        maxAttempts: 1,
      },
      's3',
      'contract-bucket',
    );
    const stream = await store.openRead('folder/a +%雪.txt');
    expect(await streamToString(stream)).toBe('cloud fixture');
    expect(requests).toHaveLength(1);
    expect(requests[0].headers.host).toBe(
      `${pathStyle ? '' : 'contract-bucket.'}${endpoint}:${port}`,
    );
    expect(new URL(requests[0].url!, 'http://fixture').pathname).toBe(
      `${pathStyle ? '/contract-bucket' : ''}/folder/a%20%2B%25%E9%9B%AA.txt`,
    );
    expect(requests[0].headers.authorization).toContain(`/` + region + '/s3/aws4_request');
    expect(requests[0].headers['x-amz-security-token']).toBe('temporary-session-token');
  });

  it('refreshes expiring default-chain credentials between actual signed requests', async () => {
    let generation = 0;
    const resolveCredentials = vi.fn(async () => ({
      ...credentials,
      accessKeyId: `access-${++generation}`,
      sessionToken: `token-${generation}`,
      expiration: new Date(Date.now() + 60000),
    }));
    vi.mocked(fromNodeProviderChain).mockReturnValue(resolveCredentials);
    const requests: IncomingMessage[] = [];
    const port = await wireServer((req, res) => {
      requests.push(req);
      res.end('artifact');
    });
    const store = await createS3ArtifactStore(
      { endPoint: '127.0.0.1', port, useSSL: false },
      's3',
      'bucket',
    );
    for (let i = 0; i < 2; i++)
      expect(await streamToString(await store.openRead('artifact'))).toBe('artifact');
    expect(requests).toHaveLength(2);
    expect(requests[0].headers['x-amz-security-token']).not.toBe(
      requests[1].headers['x-amz-security-token'],
    );
    for (const request of requests) {
      const generation = String(request.headers['x-amz-security-token']).replace('token-', '');
      expect(request.headers.authorization).toContain(`Credential=access-${generation}/`);
    }
    expect(resolveCredentials.mock.calls.length).toBeGreaterThanOrEqual(3);
  });

  it('propagates a truncated response body to the reader', async () => {
    const port = await wireServer((_req, res) => {
      res.writeHead(200, { 'content-length': 100000 });
      res.write('incomplete');
      setImmediate(() => res.destroy());
    });
    const store = await createS3ArtifactStore(
      { endPoint: '127.0.0.1', port, useSSL: false },
      'minio',
      'bucket',
    );
    await expect(store.openRead('broken').then(streamToString)).rejects.toThrow();
  });

  it('discovers the region of an AWS bucket without a configured region', async () => {
    const requests: IncomingMessage[] = [];
    const port = await wireServer((req, res) => {
      requests.push(req);
      if (requests.length === 1) {
        res.setHeader('x-amz-bucket-region', 'us-west-2');
        errorResponse(res, 301, 'PermanentRedirect');
      } else res.end('regional artifact');
    });
    const agent = new Agent({
      lookup: (_hostname, options, callback) =>
        options.all
          ? callback(null, [{ address: '127.0.0.1', family: 4 }])
          : callback(null, '127.0.0.1', 4),
    });
    cleanup.push(() => agent.destroy());
    const store = await createS3ArtifactStore(
      {
        endPoint: 's3.amazonaws.com',
        port,
        useSSL: false,
        transportAgent: agent,
        accessKey: credentials.accessKeyId,
        secretKey: credentials.secretAccessKey,
      },
      's3',
      'bucket',
    );
    expect(await streamToString(await store.openRead('artifact'))).toBe('regional artifact');
    expect(requests).toHaveLength(2);
    expect(requests[1].headers.authorization).toContain('/us-west-2/s3/aws4_request');
  });

  it('corrects a custom endpoint signing region without following its Location header', async () => {
    const requests: IncomingMessage[] = [];
    const resolvedHosts: string[] = [];
    const port = await wireServer((req, res) => {
      requests.push(req);
      if (requests.length === 1) {
        res.setHeader('location', 'http://untrusted.invalid/artifact');
        res.setHeader('x-amz-bucket-region', 'contract-region');
        errorResponse(res, 301, 'PermanentRedirect');
      } else res.end('custom artifact');
    });
    const agent = new Agent({
      lookup: (hostname, options, callback) => {
        resolvedHosts.push(hostname);
        // Every possible connection is forced onto loopback; a changed host still fails assertions.
        return options.all
          ? callback(null, [{ address: '127.0.0.1', family: 4 }])
          : callback(null, '127.0.0.1', 4);
      },
    });
    cleanup.push(() => agent.destroy());
    const store = await createS3ArtifactStore(
      {
        endPoint: 'storage.contract.invalid',
        port,
        useSSL: false,
        transportAgent: agent,
        accessKey: credentials.accessKeyId,
        secretKey: credentials.secretAccessKey,
      },
      's3',
      'bucket',
    );
    expect(await streamToString(await store.openRead('artifact'))).toBe('custom artifact');
    expect(requests).toHaveLength(2);
    expect(requests.map((request) => `http://${request.headers.host}${request.url}`)).toEqual([
      `http://storage.contract.invalid:${port}/bucket/artifact?x-id=GetObject`,
      `http://storage.contract.invalid:${port}/bucket/artifact?x-id=GetObject`,
    ]);
    expect(resolvedHosts.length).toBeGreaterThan(0);
    expect(resolvedHosts.every((host) => host === 'storage.contract.invalid')).toBe(true);
    expect(requests[1].headers.authorization).toContain('/contract-region/s3/aws4_request');
  });

  it('does not sign anonymous requests', async () => {
    let authorization: string | undefined;
    let token: string | string[] | undefined;
    const port = await wireServer((req, res) => {
      authorization = req.headers.authorization;
      token = req.headers['x-amz-security-token'];
      res.end('public artifact');
    });
    const store = await createS3ArtifactStore(
      {
        endPoint: '127.0.0.1',
        port,
        useSSL: false,
        accessKey: credentials.accessKeyId,
        secretKey: credentials.secretAccessKey,
      },
      's3',
      'public',
      JSON.stringify({ Provider: 's3', Params: { fromEnv: 'true', anonymous: 'true' } }),
    );
    expect(await streamToString(await store.openRead('artifact'))).toBe('public artifact');
    expect(authorization).toBeUndefined();
    expect(token).toBeUndefined();
  });

  it.each([
    [404, 'NoSuchKey'],
    [403, 'AccessDenied'],
  ] as const)('preserves %s %s errors', async (status, code) => {
    const port = await wireServer((_req, res) => errorResponse(res, status, code));
    const store = await createS3ArtifactStore(
      { endPoint: '127.0.0.1', port, useSSL: false, maxAttempts: 1 },
      'minio',
      'bucket',
    );
    await expect(store.openRead('missing')).rejects.toMatchObject({ code });
  });

  it('returns a live stream before the response ends and cancels it', async () => {
    let response!: ServerResponse;
    const port = await wireServer((_req, res) => {
      response = res;
      res.writeHead(200, { 'content-type': 'application/octet-stream' });
      res.write('first chunk');
    });
    const store = await createS3ArtifactStore(
      { endPoint: '127.0.0.1', port, useSSL: false },
      'minio',
      'bucket',
    );
    const controller = new AbortController();
    const stream = await store.openRead('large', { signal: controller.signal });
    expect(response.writableEnded).toBe(false);
    const closed = once(response, 'close');
    stream.on('error', () => {});
    controller.abort();
    await closed;
    expect(stream.destroyed).toBe(true);
  });

  it('stops pagination when cancelled between pages', async () => {
    let requests = 0;
    const port = await wireServer((_req, res) => {
      requests++;
      res.setHeader('content-type', 'application/xml');
      res.end(
        '<ListBucketResult><IsTruncated>true</IsTruncated><NextContinuationToken>next</NextContinuationToken><Contents><Key>first</Key><Size>1</Size></Contents></ListBucketResult>',
      );
    });
    const store = await createS3ArtifactStore(
      { endPoint: '127.0.0.1', port, useSSL: false },
      'minio',
      'bucket',
    );
    const controller = new AbortController();
    const iterator = store.list('', { signal: controller.signal })[Symbol.asyncIterator]();
    expect((await iterator.next()).value).toEqual({ key: 'first', size: 1 });
    controller.abort();
    await expect(iterator.next()).rejects.toMatchObject({ name: 'AbortError' });
    expect(requests).toBe(1);
  });

  it('retries a transient service failure within the configured limit', async () => {
    let requests = 0;
    const port = await wireServer((_req, res) => {
      requests++;
      if (requests === 1) errorResponse(res, 503, 'SlowDown');
      else res.end('recovered');
    });
    const store = await createS3ArtifactStore(
      { endPoint: '127.0.0.1', port, useSSL: false, maxAttempts: 2 },
      'minio',
      'bucket',
    );
    expect(await streamToString(await store.openRead('artifact'))).toBe('recovered');
    expect(requests).toBe(2);
  });

  it('follows opaque pagination tokens without duplicating or decoding keys', async () => {
    const seen: URL[] = [];
    const port = await wireServer((req, res) => {
      const url = new URL(req.url!, 'http://fixture');
      seen.push(url);
      const second = url.searchParams.has('continuation-token');
      res.setHeader('content-type', 'application/xml');
      res.end(
        `<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><IsTruncated>${!second}</IsTruncated>${second ? '' : '<NextContinuationToken>opaque+/%=</NextContinuationToken>'}<Contents><Key>${second ? 'prefix/雪 %2F' : 'prefix/first'}</Key><Size>${second ? 0 : 12}</Size></Contents></ListBucketResult>`,
      );
    });
    const store = await createS3ArtifactStore(
      { endPoint: '127.0.0.1', port, useSSL: false },
      'minio',
      'bucket',
    );
    const objects = [];
    for await (const object of store.list('prefix/', { pageSize: 1 })) objects.push(object);
    expect(objects).toEqual([
      { key: 'prefix/first', size: 12 },
      { key: 'prefix/雪 %2F', size: 0 },
    ]);
    expect(seen).toHaveLength(2);
    expect(seen[1].searchParams.get('continuation-token')).toBe('opaque+/%=');
    expect(seen.every((url) => url.searchParams.get('prefix') === 'prefix/')).toBe(true);
  });
});

async function streamToString(stream: NodeJS.ReadableStream): Promise<string> {
  const chunks: Buffer[] = [];
  for await (const chunk of stream) chunks.push(Buffer.from(chunk));
  return Buffer.concat(chunks).toString('utf8');
}

for (const [service, variable] of [
  ['SeaweedFS', 'KFP_SEAWEEDFS_ENDPOINT'],
  ['MinIO', 'KFP_MINIO_ENDPOINT'],
] as const) {
  describe.skipIf(!process.env[variable])(`${service} real-service contract`, () => {
    it('reads encoded keys, lists over 1000 objects, handles missing objects and public reads', async () => {
      const endpoint = new URL(process.env[variable]!);
      const bucket = `kfp-contract-${randomUUID()}`;
      const fixture = new S3Client({
        endpoint: endpoint.origin,
        region: service === 'MinIO' ? 'contract-region' : 'us-east-1',
        forcePathStyle: true,
        credentials,
      });
      const keys = Array.from({ length: 1003 }, (_, i) => `prefix/${String(i).padStart(4, '0')}`);
      const special = 'prefix/space + percent% unicode雪/%2F.txt';
      keys.push(special);
      await fixture.send(new CreateBucketCommand({ Bucket: bucket }));
      try {
        // Bound fixture setup concurrency so the test also runs on modest developer machines.
        for (let i = 0; i < keys.length; i += 20) {
          await Promise.all(
            keys.slice(i, i + 20).map((Key) =>
              fixture.send(
                new PutObjectCommand({
                  Bucket: bucket,
                  Key,
                  Body: Buffer.from(Key === special ? 'encoded artifact 雪' : ''),
                  ContentLength: Key === special ? Buffer.byteLength('encoded artifact 雪') : 0,
                }),
              ),
            ),
          );
        }
        const config = {
          endPoint: endpoint.hostname,
          port: Number(endpoint.port),
          useSSL: endpoint.protocol === 'https:',
          // Leave the signing region unspecified to exercise server-directed correction.
          pathStyle: true,
          accessKey: credentials.accessKeyId,
          secretKey: credentials.secretAccessKey,
          maxAttempts: 1,
        };
        const store = await createS3ArtifactStore(config, 's3', bucket);
        expect(await streamToString(await store.openRead(special))).toBe('encoded artifact 雪');
        const objects = [];
        for await (const object of store.list('prefix/')) objects.push(object);
        expect(objects.map((object) => object.key).sort()).toEqual([...keys].sort());
        expect(objects.find((object) => object.key === special)?.size).toBe(
          Buffer.byteLength('encoded artifact 雪'),
        );
        await expect(store.openRead('missing')).rejects.toMatchObject({ code: 'NoSuchKey' });
        const cancelled = new AbortController();
        cancelled.abort();
        await expect(store.openRead(special, { signal: cancelled.signal })).rejects.toMatchObject({
          name: 'AbortError',
        });
        await fixture.send(
          new PutBucketPolicyCommand({
            Bucket: bucket,
            Policy: JSON.stringify({
              Version: '2012-10-17',
              Statement: [
                {
                  Effect: 'Allow',
                  Principal: '*',
                  Action: ['s3:GetObject'],
                  Resource: [`arn:aws:s3:::${bucket}/*`],
                },
              ],
            }),
          }),
        );
        const anonymous = await createS3ArtifactStore(
          config,
          's3',
          bucket,
          JSON.stringify({ Provider: 's3', Params: { fromEnv: 'true', anonymous: 'true' } }),
        );
        expect(await streamToString(await anonymous.openRead(special))).toBe('encoded artifact 雪');
      } finally {
        for (let i = 0; i < keys.length; i += 1000)
          await fixture.send(
            new DeleteObjectsCommand({
              Bucket: bucket,
              Delete: { Objects: keys.slice(i, i + 1000).map((Key) => ({ Key })) },
            }),
          );
        await fixture.send(new DeleteBucketCommand({ Bucket: bucket }));
        fixture.destroy();
      }
    }, 120000);
  });
}

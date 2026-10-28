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
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { Readable } from 'node:stream';
import { NodeHttpHandler } from '@smithy/node-http-handler';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { createS3ArtifactStore } from './artifact-store.js';
vi.mock('./k8s-helper.js', () => ({ getK8sSecret: vi.fn() }));
const directories: string[] = [];
afterEach(async () => {
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
  for (const directory of directories.splice(0)) await rm(directory, { recursive: true });
});
async function readWithAuthorizedAwsEndpoint() {
  const handle = vi.spyOn(NodeHttpHandler.prototype, 'handle').mockImplementation(async () => ({
    response: { statusCode: 200, headers: {}, body: Readable.from(['artifact']) },
  }));
  const authorize = vi.fn();
  const store = await createS3ArtifactStore(
    {
      endPoint: 's3.amazonaws.com',
      region: 'us-east-1',
      accessKey: 'fixture-key',
      secretKey: 'fixture-secret',
      maxAttempts: 1,
    },
    's3',
    'fixture-bucket',
    undefined,
    undefined,
    undefined,
    undefined,
    authorize,
  );
  const stream = await store.openRead('artifact.txt');
  expect(await stream.toArray()).toHaveLength(1);
  expect(authorize).toHaveBeenCalledWith(
    expect.objectContaining({ origin: 'https://s3.amazonaws.com' }),
  );
  expect(handle).toHaveBeenCalledTimes(1);
  expect(handle.mock.calls[0][0]).toMatchObject({
    protocol: 'https:',
    hostname: 'fixture-bucket.s3.us-east-1.amazonaws.com',
  });
}
describe('artifact endpoint policy and ambient AWS SDK configuration', () => {
  it.each(['AWS_ENDPOINT_URL', 'AWS_ENDPOINT_URL_S3'])(
    'does not let %s replace the authorized endpoint',
    async (variable) => {
      vi.stubEnv(variable, 'http://unapproved.invalid:9000');
      await readWithAuthorizedAwsEndpoint();
    },
  );
  it('does not let the shared AWS profile replace the authorized endpoint', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'kfp-endpoint-policy-'));
    directories.push(directory);
    const configFile = join(directory, 'config');
    await writeFile(
      configFile,
      '[profile endpoint-policy-fixture]\nendpoint_url = http://unapproved.invalid:9000\n',
    );
    vi.stubEnv('AWS_CONFIG_FILE', configFile);
    vi.stubEnv('AWS_PROFILE', 'endpoint-policy-fixture');
    await readWithAuthorizedAwsEndpoint();
  });
});

describe('artifact bucket-name compatibility', () => {
  it.each([
    'arn:aws:s3:us-east-1:123456789012:accesspoint/test-access',
    'bucket/other',
    'Uppercase',
    'bucket..name',
    '192.168.1.1',
    'ab',
    'a'.repeat(64),
  ])('rejects unsupported bucket %s before reads or listings reach transport', async (bucket) => {
    const handle = vi.spyOn(NodeHttpHandler.prototype, 'handle');
    const store = await createS3ArtifactStore(
      {
        endPoint: 's3.amazonaws.com',
        region: 'us-east-1',
        accessKey: 'fixture-key',
        secretKey: 'fixture-secret',
      },
      's3',
      bucket,
    );
    await expect(store.openRead('artifact.txt')).rejects.toMatchObject({
      name: 'InvalidBucketNameError',
    });
    await expect(store.list('prefix/')[Symbol.asyncIterator]().next()).rejects.toMatchObject({
      name: 'InvalidBucketNameError',
    });
    expect(handle).not.toHaveBeenCalled();
  });
});

describe('artifact endpoint compatibility with ambient AWS feature flags', () => {
  it.each([
    ['AWS_USE_DUALSTACK_ENDPOINT', 'storage.example.com', 'storage.example.com'],
    ['AWS_USE_FIPS_ENDPOINT', 'storage.example.com', 'storage.example.com'],
    [
      'AWS_USE_DUALSTACK_ENDPOINT',
      's3.us-east-1.amazonaws.com',
      'fixture-bucket.s3.us-east-1.amazonaws.com',
    ],
    [
      'AWS_USE_FIPS_ENDPOINT',
      's3.us-east-1.amazonaws.com',
      'fixture-bucket.s3.us-east-1.amazonaws.com',
    ],
  ])('ignores %s for configured %s', async (variable, endpoint, hostname) => {
    vi.stubEnv(variable, 'true');
    const handle = vi.spyOn(NodeHttpHandler.prototype, 'handle').mockImplementation(async () => ({
      response: { statusCode: 200, headers: {}, body: Readable.from(['artifact']) },
    }));
    const store = await createS3ArtifactStore(
      {
        endPoint: endpoint,
        region: 'us-east-1',
        accessKey: 'fixture-key',
        secretKey: 'fixture-secret',
      },
      's3',
      'fixture-bucket',
    );
    const stream = await store.openRead('artifact.txt');
    expect(await stream.toArray()).toHaveLength(1);
    expect(handle).toHaveBeenCalledTimes(1);
    expect(handle.mock.calls[0][0]).toMatchObject({ protocol: 'https:', hostname });
  });
});

describe('custom S3 bucket names that resemble AWS specialized buckets', () => {
  it.each([
    ['ordinary--x-s3', true, false],
    ['ordinary--op-s3', true, false],
    ['ordinary--x-s3', false, false],
    ['ordinary--op-s3', false, false],
    ['dotted.bucket--x-s3', false, false],
    ['dotted.bucket--op-s3', false, false],
    ['ordinary--x-s3', true, true],
    ['ordinary--op-s3', false, true],
  ] as const)(
    'uses generic S3 for %s (path style %s, anonymous %s)',
    async (bucket, pathStyle, anonymous) => {
      const handle = vi
        .spyOn(NodeHttpHandler.prototype, 'handle')
        .mockImplementation(async (request) => ({
          response: {
            statusCode: 200,
            headers: {
              'content-type': request.query['list-type'] ? 'application/xml' : 'text/plain',
            },
            body: Readable.from([
              request.query['list-type']
                ? '<ListBucketResult><IsTruncated>false</IsTruncated><Contents><Key>folder/key.txt</Key><Size>8</Size></Contents></ListBucketResult>'
                : 'artifact',
            ]),
          },
        }));
      const store = await createS3ArtifactStore(
        {
          endPoint: 'storage.example.com',
          region: 'Region_One',
          pathStyle,
          endpointBasePath: '/gateway%2Froot',
          ...(anonymous ? {} : { accessKey: 'fixture-key', secretKey: 'fixture-secret' }),
          maxAttempts: 1,
        },
        'minio',
        bucket,
      );
      const stream = await store.openRead('folder/space +%雪.txt');
      expect(await stream.toArray()).toHaveLength(1);
      const objects = [];
      for await (const entry of store.list('folder/')) objects.push(entry);
      expect(objects).toEqual([{ key: 'folder/key.txt', size: 8 }]);
      expect(handle).toHaveBeenCalledTimes(2);
      const usesPath = pathStyle || bucket.includes('.');
      for (const [request] of handle.mock.calls) {
        expect(request.hostname).toBe(
          usesPath ? 'storage.example.com' : `${bucket}.storage.example.com`,
        );
        expect(request.query).not.toHaveProperty('session');
        if (anonymous) expect(request.headers.authorization).toBeUndefined();
        else expect(request.headers.authorization).toContain('/Region_One/s3/aws4_request');
      }
      expect(handle.mock.calls[0][0].path).toBe(
        `/gateway%2Froot${usesPath ? '/' + bucket : ''}/folder/space%20%2B%25%E9%9B%AA.txt`,
      );
      expect(handle.mock.calls[1][0].query['list-type']).toBe('2');
    },
  );
});

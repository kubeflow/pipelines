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
import { expect, vi } from 'vitest';
import type { S3ClientConfig, GetObjectCommand, ListObjectsV2Command } from '@aws-sdk/client-s3';
import type { Readable } from 'node:stream';
type AwsCredentialIdentity = {
  accessKeyId: string;
  secretAccessKey: string;
  sessionToken?: string;
};

interface TransportFixture {
  getObject(bucket: string, key: string): Promise<Readable>;
  listPage?(
    bucket: string,
    prefix: string,
    token: string,
    delimiter: string,
    maxKeys: number,
  ): Promise<{
    objects: { name: string; size: number }[];
    isTruncated: boolean;
    nextContinuationToken: string;
  }>;
}

// Fixtures describe responses at the SDK boundary; real request serialization is
// covered separately by the HTTP compatibility tests.
export const MockS3Client = vi.fn(function (_config: S3ClientConfig): TransportFixture {
  return { getObject: (...args) => MockS3Client.prototype.getObject(...args) };
});
const resolvedCredentials = new WeakMap<object, AwsCredentialIdentity>();

export function createS3Mock(actual: typeof import('@aws-sdk/client-s3')) {
  return vi.fn(function (config: S3ClientConfig) {
    const fixture = MockS3Client(config);
    return {
      config: { ...config },
      middlewareStack: { add: vi.fn() },
      destroy: vi.fn(),
      async send(
        command: GetObjectCommand | ListObjectsV2Command,
        options?: { abortSignal?: AbortSignal },
      ) {
        const credentials =
          typeof config.credentials === 'function'
            ? await config.credentials()
            : config.credentials;
        if (credentials) resolvedCredentials.set(config, credentials);
        if (command instanceof actual.GetObjectCommand) {
          return { Body: await fixture.getObject(command.input.Bucket!, command.input.Key!) };
        }
        const input = command.input;
        const operation = fixture.listPage?.(
          input.Bucket!,
          input.Prefix!,
          input.ContinuationToken || '',
          '',
          input.MaxKeys!,
        );
        const page = await new Promise<Awaited<typeof operation>>((resolve, reject) => {
          const abort = () => reject(Object.assign(new Error('Aborted'), { name: 'AbortError' }));
          if (options?.abortSignal?.aborted) return abort();
          options?.abortSignal?.addEventListener('abort', abort, { once: true });
          Promise.resolve(operation)
            .then(resolve, reject)
            .finally(() => options?.abortSignal?.removeEventListener('abort', abort));
        });
        return {
          Contents: page?.objects.map(({ name, size }) => ({ Key: name, Size: size })),
          IsTruncated: page?.isTruncated,
          NextContinuationToken: page?.nextContinuationToken,
        };
      },
    };
  });
}

/** Assert the endpoint and credentials selected by the application. */
export function expectedS3Options(options: Record<string, unknown>) {
  return {
    asymmetricMatch(config: S3ClientConfig) {
      const endpoint =
        typeof config.endpoint === 'string'
          ? new URL(config.endpoint)
          : new URL('https://s3.amazonaws.com');
      const credentials =
        resolvedCredentials.get(config) ??
        (typeof config.credentials === 'object' ? config.credentials : undefined);
      const observed = {
        endPoint: endpoint.hostname.replace(/^\[|\]$/g, ''),
        port: Number(endpoint.port || (endpoint.protocol === 'https:' ? 443 : 80)),
        useSSL: endpoint.protocol === 'https:',
        region: config.region,
        accessKey: credentials?.accessKeyId,
        secretKey: credentials?.secretAccessKey,
        sessionToken: credentials?.sessionToken,
        pathStyle: config.forcePathStyle,
      };
      const expected = { ...options };
      if (typeof expected.endPoint === 'string' && /^[^:]+:\d+$/.test(expected.endPoint)) {
        const parsed = new URL(
          `${expected.useSSL === false ? 'http' : 'https'}://${expected.endPoint}`,
        );
        expected.endPoint = parsed.hostname;
        expected.port = Number(parsed.port || (parsed.protocol === 'http:' ? 80 : 443));
      }
      if ('port' in expected && expected.port === undefined)
        expected.port = endpoint.protocol === 'https:' ? 443 : 80;
      if ('useSSL' in expected && expected.useSSL === undefined) expected.useSSL = true;
      if ('region' in expected && expected.region === undefined) expected.region = 'us-east-1';
      for (const field of ['accessKey', 'secretKey']) {
        if (field in expected && expected[field] === undefined) expected[field] = '';
      }
      return expect.objectContaining(expected).asymmetricMatch(observed);
    },
    toString: () => 'S3 endpoint and credentials',
    getExpectedType: () => 'object',
  };
}

export async function resolveS3TestCredentials() {
  for (const [config] of MockS3Client.mock.calls) {
    if (typeof config.credentials === 'function')
      resolvedCredentials.set(config, await config.credentials());
  }
}

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

import { createServer, ServerResponse } from 'node:http';
import { createRequire } from 'node:module';
import { Readable, Transform } from 'node:stream';
import { pipeline } from 'node:stream/promises';
import { describe, expect, it } from 'vitest';
import type * as Minio from 'minio';

// Resolve from MinIO so these exercise its actual overridden dependencies,
// including installations where npm nests them instead of hoisting them.
const require = createRequire(import.meta.url);
const minioRequire = createRequire(require.resolve('minio/package.json'));
const queryString: {
  parse(input: string): Record<string, unknown>;
  stringify(input: Record<string, unknown>): string;
} = minioRequire('query-string');
const jsonLineParser: { default: { asStream(): Transform } } = minioRequire(
  'stream-json/jsonl/parser.js',
);

async function withStore(
  respond: (response: ServerResponse) => void,
  run: (port: number) => Promise<void>,
): Promise<void> {
  const server = createServer((_request, response) => respond(response));
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
  const address = server.address();
  if (!address || typeof address === 'string') throw new Error('Expected a TCP test server');
  try {
    await run(address.port);
  } finally {
    server.closeAllConnections();
    await new Promise<void>((resolve, reject) => {
      server.close((error) => (error ? reject(error) : resolve()));
    });
  }
}

describe('MinIO dependency compatibility patch', () => {
  it('decodes Unicode, malformed escapes, repeated values, and query round trips', () => {
    expect(
      queryString.parse('key=%E2%9C%93+ok&bad=%E0%A4%A&repeat=1&repeat=2&literal=%252F'),
    ).toEqual({
      key: '✓ ok',
      bad: '%E0%A4%A',
      repeat: ['1', '2'],
      literal: '%2F',
    });
    const value = {
      prefix: '模型/a b',
      events: ['s3:ObjectCreated:*', 's3:ObjectRemoved:*'],
    };
    expect(queryString.parse(queryString.stringify(value))).toEqual(value);
  });

  it('preserves JSONL key/value output, split UTF-8, heartbeats, and a final unterminated line', async () => {
    const input = Buffer.from(
      '{"Records":[{"key":"模型"}]}\n\n{"Records":null}\n{"Records":[{"key":"last"}]}',
    );
    const output: unknown[] = [];
    await pipeline(
      Readable.from([...input].map((byte) => Buffer.from([byte]))),
      jsonLineParser.default.asStream(),
      async (source) => {
        for await (const item of source) output.push(item);
      },
    );
    expect(output).toEqual([
      { key: 0, value: { Records: [{ key: '模型' }] } },
      { key: 1, value: { Records: null } },
      { key: 2, value: { Records: [{ key: 'last' }] } },
    ]);
  });

  for (const format of ['CommonJS', 'ESM']) {
    const load = async (): Promise<typeof Minio> =>
      format === 'CommonJS' ? require('minio') : import('minio');

    it(`${format} client emits notification records from the real HTTP response`, async () => {
      const { Client } = await load();
      await withStore(
        (response) => {
          response.end('{"Records":null}\n\n{"Records":[{"id":1}]}\n{"Records":[{"id":2}]}');
        },
        async (port) => {
          const client = new Client({
            endPoint: '127.0.0.1',
            port,
            useSSL: false,
          });
          const poller = client.listenBucketNotification('bucket', '', '', []);
          const records: unknown[] = [];
          try {
            await new Promise<void>((resolve, reject) => {
              poller.on('error', reject);
              poller.on('notification', (record) => {
                records.push(record);
                if (records.length === 2) {
                  poller.stop();
                  resolve();
                }
              });
            });
            expect(records).toEqual([{ id: 1 }, { id: 2 }]);
          } finally {
            poller.stop();
          }
        },
      );
    });

    it(`${format} client propagates malformed JSON errors`, async () => {
      const { Client } = await load();
      await withStore(
        (response) => response.end('{invalid}\n'),
        async (port) => {
          const client = new Client({
            endPoint: '127.0.0.1',
            port,
            useSSL: false,
          });
          const poller = client.listenBucketNotification('bucket', '', '', []);
          try {
            const error = await new Promise<unknown>((resolve) => {
              poller.on('error', (error) => {
                poller.stop();
                resolve(error);
              });
            });
            expect(error).toBeInstanceOf(SyntaxError);
          } finally {
            poller.stop();
          }
        },
      );
    });

    it(`${format} client propagates interrupted response stream errors`, async () => {
      const { Client } = await load();
      await withStore(
        (response) => {
          response.writeHead(200, { 'Content-Type': 'application/json' });
          response.write('{"Records":');
          setImmediate(() => response.destroy());
        },
        async (port) => {
          const client = new Client({
            endPoint: '127.0.0.1',
            port,
            useSSL: false,
          });
          const poller = client.listenBucketNotification('bucket', '', '', []);
          try {
            const error = await new Promise<unknown>((resolve) => {
              poller.on('error', (error) => {
                poller.stop();
                resolve(error);
              });
            });
            expect(error).toBeInstanceOf(Error);
          } finally {
            poller.stop();
          }
        },
      );
    });
  }
});

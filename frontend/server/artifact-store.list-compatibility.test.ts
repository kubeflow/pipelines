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
import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { once } from 'node:events';
import type { AddressInfo } from 'node:net';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { createS3ArtifactStore } from './artifact-store.js';

vi.mock('./k8s-helper.js', () => ({ getK8sSecret: vi.fn() }));
const cleanup: Array<() => Promise<void>> = [];
afterEach(async () => {
  for (const close of cleanup.splice(0)) await close();
});

async function fixture(handle: (req: IncomingMessage, res: ServerResponse) => void) {
  const server = createServer(handle);
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  cleanup.push(() => {
    server.closeAllConnections();
    return new Promise<void>((resolve) => server.close(() => resolve()));
  });
  return createS3ArtifactStore(
    {
      endPoint: '127.0.0.1',
      port: (server.address() as AddressInfo).port,
      useSSL: false,
      region: 'us-east-1',
      accessKey: 'fixture',
      secretKey: 'fixture-secret',
    },
    's3',
    'artifact-bucket',
  );
}

const keys = [
  '\n',
  'dir/control\u0001.txt',
  'dir/space name.txt',
  'dir/literal+plus.txt',
  'dir/%252F.txt',
  'dir/雪.txt',
  'dir/file ',
  'dir/ newline\n',
];

describe('artifact listing key compatibility', () => {
  it('retains a whitespace-only object key that XML text normalization would erase', async () => {
    const store = await fixture((req, res) => {
      const encoded =
        new URL(req.url!, 'http://fixture').searchParams.get('encoding-type') === 'url';
      res.setHeader('content-type', 'application/xml');
      res.end(
        `<ListBucketResult>${encoded ? '<EncodingType>url</EncodingType>' : ''}<IsTruncated>false</IsTruncated><Contents><Key>${encoded ? '%0A' : '&#xA;'}</Key><Size>4</Size></Contents></ListBucketResult>`,
      );
    });
    expect(await Array.fromAsync(store.list(''))).toEqual([{ key: '\n', size: 4 }]);
  });

  it.each(['percent', 'form'])(
    'decodes %s URL-encoded keys once and preserves opaque continuation tokens',
    async (encoding) => {
      const requests: URL[] = [];
      const token = 'opaque+%2F/雪==';
      const store = await fixture((req, res) => {
        const url = new URL(req.url!, 'http://fixture');
        requests.push(url);
        const first = requests.length === 1;
        const contents = (first ? keys.slice(0, 3) : keys.slice(3))
          .map((key) => {
            const encoded = encodeURIComponent(key);
            return `<Contents><Key>${encoding === 'form' ? encoded.replace(/%20/g, '+') : encoded}</Key><Size>4</Size></Contents>`;
          })
          .join('');
        res.setHeader('content-type', 'application/xml');
        res.end(
          `<ListBucketResult><EncodingType>url</EncodingType><IsTruncated>${first}</IsTruncated>${first ? `<NextContinuationToken>${token}</NextContinuationToken>` : ''}${contents}</ListBucketResult>`,
        );
      });
      expect(await Array.fromAsync(store.list('dir/ +%雪', { pageSize: 3 }))).toEqual(
        keys.map((key) => ({ key, size: 4 })),
      );
      expect(requests).toHaveLength(2);
      for (const url of requests) {
        expect(url.searchParams.get('encoding-type')).toBe('url');
        expect(url.searchParams.get('prefix')).toBe('dir/ +%雪');
        expect(url.searchParams.get('max-keys')).toBe('3');
      }
      expect(requests[1].searchParams.get('continuation-token')).toBe(token);
    },
  );

  it('retains literal percent and plus characters when a provider returns unencoded keys', async () => {
    const store = await fixture((_req, res) => {
      res.setHeader('content-type', 'application/xml');
      res.end(
        '<ListBucketResult><IsTruncated>false</IsTruncated><Contents><Key>dir/a+%2F%name.txt</Key><Size>9</Size></Contents></ListBucketResult>',
      );
    });
    expect(await Array.fromAsync(store.list('dir/'))).toEqual([
      { key: 'dir/a+%2F%name.txt', size: 9 },
    ]);
  });
});

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

import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);
// Resolve from each consumer so nested vulnerable copies cannot hide behind a
// patched top-level dependency.
const kubernetesRequire = createRequire(require.resolve('@kubernetes/client-node'));
const authRequire = createRequire(require.resolve('google-auth-library'));
const gaxiosRequire = createRequire(authRequire.resolve('gaxios'));

it('counts empty YAML merge sources against the Kubernetes parser budget', () => {
  const yaml = kubernetesRequire('js-yaml');
  const source = 'sources: &sources [{}, {}, {}, {}]\nresult:\n  <<: *sources\n';
  expect(() => yaml.load(source, { maxTotalMergeKeys: 3 })).toThrow(/maxTotalMergeKeys/);
  expect(yaml.load(source, { maxTotalMergeKeys: 10 }).result).toEqual({});
});

it('rejects undersized output buffers in the UUID dependency used by gaxios', () => {
  const { v5 } = gaxiosRequire('uuid');
  expect(() => v5('probe', v5.DNS, new Uint8Array(8))).toThrow(RangeError);
});

it('preserves gaxios multipart generation with the patched UUID dependency', async () => {
  const { Gaxios } = authRequire('gaxios') as typeof import('gaxios');
  const client = new Gaxios();
  const result = await client.request({
    url: 'https://example.invalid/',
    method: 'POST',
    multipart: [{ headers: { 'Content-Type': 'text/plain' }, content: 'probe' }],
    adapter: async (options) => {
      expect(options.headers?.['Content-Type']).toMatch(/boundary=[0-9a-f-]{36}$/);
      let body = '';
      for await (const chunk of options.body as AsyncIterable<Buffer>) body += chunk.toString();
      expect(body).toContain('probe');
      return { status: 200, statusText: 'OK', headers: {}, config: options, data: undefined };
    },
  });
  expect(result.status).toBe(200);
});

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

import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import semver from 'semver';
import qs from 'qs';

// Check nested copies as well as hoisted ones. These fixed release lines do not
// replace fresh advisory scans when qualifying a release.
const tooling = {
  vite: '>=8.2.4 <9',
  vitest: '>=4.1.12 <5',
  '@vitest/mocker': '>=4.1.12 <5',
  '@vitest/coverage-v8': '>=4.1.12 <5',
};
const scopes = {
  '../mock-backend/package-lock.json': {
    'body-parser': '>=2.4.0 <3',
    qs: '>=6.16.1 <7',
  },
  '../../test/frontend-integration-test/package-lock.json': {
    undici: '>=6.29.1 <7 || >=7.30.1 <8',
    'serialize-javascript': '>=7.1.3 <8',
  },
  '../package-lock.json': {
    ...tooling,
    qs: '>=6.16.1 <7',
    storybook: '>=10.6.2 <11',
    zod: '>=4.6.6 <5',
  },
  '../server/package-lock.json': {
    ...tooling,
    'body-parser': '>=2.4.0 <3',
    'follow-redirects': '>=1.16.1 <2',
    'stream-json': '>=3.7.0 <4',
  },
};

for (const [file, requirements] of Object.entries(scopes)) {
  describe(file, () => {
    const lock = JSON.parse(readFileSync(new URL(file, import.meta.url), 'utf8'));
    for (const [name, range] of Object.entries(requirements)) {
      it(`locks every ${name} copy to a qualified patched release`, () => {
        const copies = Object.entries(lock.packages).filter(([path]) =>
          path.endsWith(`node_modules/${name}`),
        );
        expect(copies.length).toBeGreaterThan(0);
        for (const [path, entry] of copies) {
          expect(semver.satisfies(entry.version, range), `${path}@${entry.version}`).toBe(true);
        }
      });
    }
  });
}

// GHSA-6pq8-jhm2-2vcw: comma serialization must handle JSON-derived objects
// whose own toString value cannot be called. Test both coercion paths.
describe('qs comma serialization', () => {
  it.each([false, true])('handles hostile objects with encodeValuesOnly=%s', (encodeValuesOnly) => {
    const input = JSON.parse('{"a":[{"toString":"x"}]}');
    expect(qs.stringify(input, { arrayFormat: 'comma', encodeValuesOnly })).toBe(
      'a=%5Bobject%20Object%5D',
    );
  });

  it('preserves ordinary comma-separated values', () => {
    expect(qs.stringify({ a: ['x', 'y'] }, { arrayFormat: 'comma' })).toBe('a=x%2Cy');
  });
});

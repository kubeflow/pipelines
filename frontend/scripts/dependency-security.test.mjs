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

// Check nested copies as well as hoisted ones. These fixed release lines do not
// replace fresh advisory scans when qualifying a release.
const tooling = {
  vite: '>=8.2.4 <9',
  vitest: '>=4.1.12 <5',
  '@vitest/mocker': '>=4.1.12 <5',
  '@vitest/coverage-v8': '>=4.1.12 <5',
};
const scopes = {
  '../package-lock.json': {
    ...tooling,
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

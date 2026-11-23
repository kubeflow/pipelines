// @vitest-environment node
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
import { satisfies } from 'semver';
import { describe, expect, it } from 'vitest';

it('rejects vulnerable selector parsers if a dependency reintroduces them', () => {
  const lock = JSON.parse(readFileSync(new URL('../package-lock.json', import.meta.url), 'utf8'));
  const parsers = Object.entries(lock.packages).filter(([path]) =>
    path.endsWith('/postcss-selector-parser'),
  );
  // Tailwind 4 removes this dependency; any future copies must stay patched.
  for (const [path, dependency] of parsers) {
    expect(satisfies(dependency.version, '>=7.1.6'), `${path}@${dependency.version}`).toBe(true);
  }
});

// GHSA-3g3m-2fjr-6pw4 and GHSA-g3hv-75wg-m249 require patched major releases.
it('keeps glob traversal and minimatch optimization on patched releases', () => {
  const lock = JSON.parse(readFileSync(new URL('../package-lock.json', import.meta.url), 'utf8'));
  for (const [name, minimum] of Object.entries({ glob: '>=14.0.2', minimatch: '>=11.0.1' })) {
    const packages = Object.entries(lock.packages).filter(([path]) => path.endsWith(`/${name}`));
    expect(packages.length, name).toBeGreaterThan(0);
    for (const [path, dependency] of packages) {
      expect(satisfies(dependency.version, minimum), `${path}@${dependency.version}`).toBe(true);
    }
  }
});

// Keep the patched release ranges while allowing master's newer tool majors.
// Nested dependency copies must satisfy the same bounds as hoisted copies.
const tooling = {
  vite: '>=8.2.4 <9',
  vitest: '>=4.1.12 <5 || >=5.0.5 <6',
  '@vitest/mocker': '>=4.1.12 <5 || >=5.0.5 <6',
  '@vitest/coverage-v8': '>=4.1.12 <5 || >=5.0.5 <6',
};
const scopes = {
  '../mock-backend/package-lock.json': {
    'body-parser': '>=2.4.0 <3',
    qs: '>=6.16.1 <7',
  },
  '../../test/frontend-integration-test/package-lock.json': {
    undici: '>=6.29.1 <7 || >=7.30.1 <8 || >=8.12.3 <9',
    'serialize-javascript': '>=7.1.3 <8',
  },
  '../package-lock.json': {
    ...tooling,
    storybook: '>=10.6.2 <11',
    zod: '>=4.6.6 <5',
  },
  '../server/package-lock.json': {
    ...tooling,
    'body-parser': '>=2.4.0 <3',
    'follow-redirects': '>=1.16.1 <2',
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
          expect(satisfies(entry.version, range), `${path}@${entry.version}`).toBe(true);
        }
      });
    }
  });
}

it('does not install the unused fs stub or pin the obsolete body-parser override', () => {
  const manifest = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'));
  const lock = JSON.parse(readFileSync(new URL('../package-lock.json', import.meta.url), 'utf8'));
  expect(manifest.devDependencies).not.toHaveProperty('fs');
  expect(manifest.overrides).not.toHaveProperty('body-parser');
  expect(lock.packages).not.toHaveProperty('node_modules/fs');
});

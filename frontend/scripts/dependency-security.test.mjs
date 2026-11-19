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
import { expect, it } from 'vitest';

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

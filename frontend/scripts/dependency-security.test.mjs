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
import { minVersion, satisfies } from 'semver';
import { expect, it } from 'vitest';

it('patches every selector parser, including nested Tailwind/PostCSS copies', () => {
  const lock = JSON.parse(readFileSync(new URL('../package-lock.json', import.meta.url), 'utf8'));
  const parsers = Object.entries(lock.packages).filter(([path]) =>
    path.endsWith('/postcss-selector-parser'),
  );
  // Tailwind 4 no longer installs this parser; retain the override for future transitive copies.
  const manifest = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'));
  expect(satisfies(minVersion(manifest.overrides['postcss-selector-parser']), '>=7.1.6')).toBe(true);
  for (const [path, dependency] of parsers) {
    expect(satisfies(dependency.version, '>=7.1.6'), `${path}@${dependency.version}`).toBe(true);
  }
});

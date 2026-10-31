// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const lock = JSON.parse(readFileSync(new URL('./package-lock.json', import.meta.url), 'utf8')) as {
  packages: Record<string, { version?: string }>;
};

// Guard the vulnerable transitive dependencies that motivated the S3 client migration.
// Checking every lockfile instance also covers a future dependency reintroducing them.
describe('artifact storage dependency security', () => {
  it.each([
    ['stream-json', 3, 7],
    ['decode-uri-component', 0, 5],
  ] as const)('does not lock a vulnerable %s release', (name, fixedMajor, fixedMinor) => {
    for (const [path, entry] of Object.entries(lock.packages)) {
      if (!path.endsWith(`/node_modules/${name}`) && path !== `node_modules/${name}`) continue;
      const [major, minor] = (entry.version ?? '').split('.').map(Number);
      expect(
        major > fixedMajor || (major === fixedMajor && minor >= fixedMinor),
        `${path}@${entry.version} predates the fixed ${fixedMajor}.${fixedMinor} release`,
      ).toBe(true);
    }
  });
});

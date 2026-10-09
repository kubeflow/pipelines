// @vitest-environment node

/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { mkdtemp, mkdir, readFile, rm, writeFile, readdir } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { gunzipSync } from 'node:zlib';
import { afterEach, expect, it } from 'vitest';
import { precompressAssets } from './precompress-assets.mjs';

let directory;
afterEach(async () => {
  if (directory) await rm(directory, { recursive: true, force: true });
});

it('emits only beneficial public JS/CSS sidecars with exact final bytes', async () => {
  directory = await mkdtemp(join(tmpdir(), 'kfp-precompressed-'));
  await mkdir(join(directory, 'static'));
  const assets = {
    'static/Editor-abcd.js':
      'const model = "pipeline";\n'.repeat(100) + '//# sourceMappingURL=Editor.js.map\n',
    'static/worker-yaml-abcd.js': 'function validate() {}\n'.repeat(100),
    'static/theme-abcd.css': '.editor { color: black; }\n'.repeat(100),
    'static/tiny.js': 'x',
    'static/Editor.js.map': '{}'.repeat(100),
    'index.html': '<html></html>'.repeat(100),
  };
  for (const [name, bytes] of Object.entries(assets)) await writeFile(join(directory, name), bytes);
  await precompressAssets(directory, Object.keys(assets));
  expect(
    (await readdir(join(directory, 'static'))).filter((name) => name.endsWith('.gz')).sort(),
  ).toEqual(['Editor-abcd.js.gz', 'theme-abcd.css.gz', 'worker-yaml-abcd.js.gz']);
  for (const name of [
    'static/Editor-abcd.js',
    'static/worker-yaml-abcd.js',
    'static/theme-abcd.css',
  ]) {
    expect(gunzipSync(await readFile(join(directory, `${name}.gz`))).toString()).toBe(assets[name]);
    expect((await readFile(join(directory, name))).toString()).toBe(assets[name]);
  }
  expect(await readdir(directory)).not.toContain('index.html.gz');
});

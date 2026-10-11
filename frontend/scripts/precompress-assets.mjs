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

import { readFile, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { gzipSync } from 'node:zlib';

export async function precompressAssets(directory, filenames) {
  for (const filename of filenames) {
    // Only public, generated assets: never HTML, source maps, or dynamic responses.
    if (!/^static\/[\w.-]+\.(?:js|css)$/.test(filename)) continue;
    const path = resolve(directory, filename);
    const original = await readFile(path);
    const compressed = gzipSync(original, { level: 9 });
    if (compressed.length < original.length) await writeFile(`${path}.gz`, compressed);
  }
}

/** @returns {import('vite').Plugin} */
export function precompressAssetsPlugin() {
  return {
    name: 'kfp-precompress-static-assets',
    apply: 'build',
    // Read final output bytes, including Rollup's sourceMappingURL comments.
    async writeBundle(options, bundle) {
      if (!options.dir) throw new Error('Static asset compression requires an output directory');
      await precompressAssets(options.dir, Object.keys(bundle));
    },
  };
}

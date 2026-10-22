/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */
import { readFile, readdir, writeFile } from 'node:fs/promises';
import path from 'node:path';

const inventoryFile = '.vite/third-party-licenses.json';

// Vite owns the bundled-package inventory. Supplement its first license file with
// additional LICENSE/NOTICE files and fonts copied verbatim from public/.
export async function writeThirdPartyNotices(outputDirectory, packageDirectories, frontendRoot) {
  const entries = JSON.parse(await readFile(path.join(outputDirectory, inventoryFile), 'utf8'));
  const sections = [
    'THIRD-PARTY NOTICES',
    'Licenses for dependencies distributed in this browser build.',
  ];
  for (const entry of entries) {
    if (!entry.text?.trim() && entry.name === 'victory-vendor' && entry.version === '37.3.6') {
      entry.text = await readFile(
        path.join(frontendRoot, 'third_party/victory-vendor/37.3.6/LICENSE'),
        'utf8',
      );
    }
    if (!entry.text?.trim()) {
      throw new Error(
        `No license text for bundled dependency ${entry.name}@${entry.version}. ` +
          'Restore its license file or supply a reviewed notice before distributing this build.',
      );
    }
    sections.push(
      `${entry.name}@${entry.version} (${entry.identifier || 'see license below'})`,
      entry.text,
    );
    const directory = packageDirectories.get(`${entry.name}@${entry.version}`);
    if (!directory)
      throw new Error(
        `Cannot locate ${entry.name}@${entry.version}; check the Vite license inventory.`,
      );
    for (const item of (await readdir(directory, { withFileTypes: true })).sort((a, b) =>
      a.name.localeCompare(b.name),
    )) {
      if (!item.isFile()) continue;
      const filename = item.name;
      if (!/^(licen[sc]e|copying|notice)(?:[.-]|$)/i.test(filename)) continue;
      const text = (await readFile(path.join(directory, filename), 'utf8')).trim();
      if (text && text !== entry.text.trim()) sections.push(`${entry.name}: ${filename}`, text);
    }
  }
  const victory = packageDirectories.get('victory-vendor@37.3.6');
  if (victory) {
    for (const vendor of (await readdir(path.join(victory, 'lib-vendor'))).sort()) {
      sections.push(
        `victory-vendor: ${vendor}`,
        (await readFile(path.join(victory, 'lib-vendor', vendor, 'LICENSE'), 'utf8')).trim(),
      );
    }
  }
  // Tailwind's generated CSS enters Vite as a local file, not a package module.
  const tailwindDirectory = path.join(frontendRoot, 'node_modules/tailwindcss');
  const tailwind = JSON.parse(await readFile(path.join(tailwindDirectory, 'package.json'), 'utf8'));
  sections.push(
    `${tailwind.name}@${tailwind.version} (generated CSS)`,
    (await readFile(path.join(tailwindDirectory, 'LICENSE'), 'utf8')).trim(),
  );
  const fontsDirectory = path.join(outputDirectory, 'fonts');
  for (const font of (await readdir(fontsDirectory)).sort()) {
    sections.push(
      `Font: ${font}`,
      (await readFile(path.join(fontsDirectory, font, 'OFL.txt'), 'utf8')).trim(),
    );
  }
  await writeFile(
    path.join(outputDirectory, 'THIRD_PARTY_NOTICES.txt'),
    sections.join('\n\n') + '\n',
  );
}

/** @returns {import('vite').Plugin} */
export function thirdPartyNoticesPlugin() {
  let frontendRoot;
  return {
    name: 'kfp-third-party-notices',
    apply: 'build',
    configResolved(config) {
      frontendRoot = config.root;
    },
    async writeBundle(options, bundle) {
      if (!options.dir) throw new Error('Third-party notices require a build output directory.');
      const directories = new Set();
      for (const chunk of Object.values(bundle)) {
        if (chunk.type !== 'chunk') continue;
        for (const moduleId of chunk.moduleIds) {
          const marker = '/node_modules/';
          const offset = moduleId.lastIndexOf(marker);
          if (offset < 0 || moduleId.startsWith('\0')) continue;
          const parts = moduleId.slice(offset + marker.length).split('/');
          const packageName = parts.slice(0, parts[0].startsWith('@') ? 2 : 1).join('/');
          directories.add(moduleId.slice(0, offset + marker.length) + packageName);
        }
      }
      const packages = new Map();
      for (const directory of directories) {
        const metadata = JSON.parse(await readFile(path.join(directory, 'package.json'), 'utf8'));
        packages.set(`${metadata.name}@${metadata.version}`, directory);
      }
      await writeThirdPartyNotices(options.dir, packages, frontendRoot);
    },
  };
}

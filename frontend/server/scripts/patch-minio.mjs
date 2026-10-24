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
import { readFileSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, join } from 'node:path';

// MinIO 8.0.7 has not adopted the fixed decoder/JSONL parser APIs. Keep this
// narrowly scoped compatibility patch until upstream supports these versions.
// Node 24 supports synchronous require() of these ESM dependencies.
const require = createRequire(new URL('../package.json', import.meta.url));
const minioRoot = dirname(require.resolve('minio/package.json'));
const minioRequire = createRequire(join(minioRoot, 'package.json'));
const queryStringPath = minioRequire.resolve('query-string');
const versions = [
  [join(minioRoot, 'package.json'), '8.0.7'],
  [join(dirname(queryStringPath), 'package.json'), '7.1.3'],
  [join(dirname(minioRequire.resolve('stream-json')), '../package.json'), '3.7.0'],
  [
    join(dirname(createRequire(queryStringPath).resolve('decode-uri-component')), 'package.json'),
    '0.5.0',
  ],
];
for (const [file, expected] of versions) {
  const { version } = JSON.parse(readFileSync(file, 'utf8'));
  if (version !== expected) {
    throw new Error(
      `Review the MinIO compatibility patch: ${file} is ${version}, expected ${expected}`,
    );
  }
}

const patches = [
  [
    join(minioRoot, 'dist/main/notification.js'),
    [
      ['require("stream-json/jsonl/Parser.js")', 'require("stream-json/jsonl/parser.js").default'],
      ['_Parser.make()', '_Parser.asStream()'],
    ],
  ],
  [
    join(minioRoot, 'dist/esm/notification.mjs'),
    [
      ["from 'stream-json/jsonl/Parser.js'", "from 'stream-json/jsonl/parser.js'"],
      ['jsonLineParser.make()', 'jsonLineParser.asStream()'],
    ],
  ],
  [
    queryStringPath,
    [["require('decode-uri-component');", "require('decode-uri-component').default;"]],
  ],
];

// Validate every replacement before writing anything; rerunning is harmless.
const updated = patches.map(([file, replacements]) => {
  let source = readFileSync(file, 'utf8');
  for (const [before, after] of replacements) {
    const alreadyPatched = source.split(after);
    if (alreadyPatched.length === 2 && !alreadyPatched.join('').includes(before)) continue;
    const count = source.split(before).length - 1;
    if (count !== 1) {
      throw new Error(`Review the MinIO compatibility patch: expected one ${before} in ${file}`);
    }
    source = source.replace(before, after);
  }
  return [file, source];
});
for (const [file, source] of updated) writeFileSync(file, source);

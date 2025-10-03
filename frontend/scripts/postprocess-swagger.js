/*
 * Copyright 2025 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const fs = require('fs');
const path = require('path');

const projectRoot = path.resolve(__dirname, '..');
const searchRoots = [
  path.join(projectRoot, 'src'),
  path.join(projectRoot, 'server', 'src'),
];

const replacements = [
  {
    regex: /delete\s+\(localVarUrlObj\s+as\s+any\)\.search\s*;/g,
    replacement: 'localVarUrlObj.search = undefined;',
  },
  {
    regex: /delete\s+localVarUrlObj\.search\s*;/g,
    replacement: 'localVarUrlObj.search = undefined;',
  },
];

const updatedFiles = [];

function processFile(filePath) {
  let contents = fs.readFileSync(filePath, 'utf8');
  let updated = contents;

  for (const {regex, replacement} of replacements) {
    updated = updated.replace(regex, replacement);
  }

  if (updated !== contents) {
    fs.writeFileSync(filePath, updated);
    updatedFiles.push(path.relative(projectRoot, filePath));
  }
}

function walk(dirPath) {
  for (const entry of fs.readdirSync(dirPath)) {
    if (entry === 'node_modules' || entry === '.swagger-codegen') {
      continue;
    }

    const fullPath = path.join(dirPath, entry);
    const stats = fs.statSync(fullPath);

    if (stats.isDirectory()) {
      walk(fullPath);
    } else if (stats.isFile() && fullPath.endsWith('.ts')) {
      processFile(fullPath);
    }
  }
}

for (const root of searchRoots) {
  if (fs.existsSync(root)) {
    walk(root);
  }
}

if (updatedFiles.length) {
  console.log(`Updated ${updatedFiles.length} file(s) to replace unsafe query-string deletion:`);
  for (const file of updatedFiles) {
    console.log(` - ${file}`);
  }
}



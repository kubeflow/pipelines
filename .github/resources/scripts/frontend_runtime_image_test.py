# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Check a built frontend image's production dependencies and HTTP startup."""

import os
import subprocess
import unittest


@unittest.skipUnless(
    os.environ.get("KFP_FRONTEND_IMAGE"),
    "Set KFP_FRONTEND_IMAGE to test a built image",
)
class FrontendRuntimeImageTest(unittest.TestCase):

    def test_development_dependencies_are_absent(self):
        script = r"""
const assert = require('node:assert/strict');
const fs = require('node:fs');
for (const name of ['@vitest/mocker', 'vitest', 'vite', 'nanoid',
                    'postcss', 'source-map-js', 'typescript']) {
  assert(!fs.existsSync(`/server/node_modules/${name}`), `${name} must be absent`);
}
"""
        self.run_node(script)

    def test_minio_patch_and_http_startup(self):
        script = r"""
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const {spawn, execFileSync} = require('node:child_process');
const {createRequire} = require('node:module');
assert.equal(typeof require('minio').Client, 'function');
const minioRoot = path.dirname(require.resolve('minio/package.json'));
const minioRequire = createRequire(path.join(minioRoot, 'package.json'));
const targets = [path.join(minioRoot, 'dist/main/notification.js'),
                 path.join(minioRoot, 'dist/esm/notification.mjs'),
                 minioRequire.resolve('query-string')];
const before = targets.map(file => fs.readFileSync(file, 'utf8'));
// Validation must not repair the image: the qualified patches already exist.
execFileSync(process.execPath, ['scripts/patch-minio.mjs']);
assert.deepEqual(targets.map(file => fs.readFileSync(file, 'utf8')), before);
const child = spawn(process.execPath, ['dist/server.js', '/client/', '3000'],
                    {stdio: ['ignore', 'pipe', 'pipe']});
let output = '';
child.stdout.on('data', data => output += data);
child.stderr.on('data', data => output += data);
(async () => {
  try {
    for (let attempt = 0; attempt < 40; attempt++) {
      if (child.exitCode !== null) throw new Error(`Server exited: ${output}`);
      try {
        const response = await fetch('http://127.0.0.1:3000/');
        assert.equal(response.status, 200);
        assert.match(await response.text(), /<html/i);
        console.log('Production dependencies, MinIO compatibility, and HTTP startup passed');
        return;
      } catch (error) {
        if (attempt === 39) throw new Error(`${error}\n${output}`);
        await new Promise(resolve => setTimeout(resolve, 250));
      }
    }
  } finally {
    child.kill();
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
"""
        self.run_node(script)

    def run_node(self, script):
        result = subprocess.run(
            [
                "docker",
                "run",
                "--rm",
                "--entrypoint",
                "node",
                os.environ["KFP_FRONTEND_IMAGE"],
                "--input-type=commonjs",
                "-e",
                script,
            ],
            check=False,
            capture_output=True,
            text=True,
            timeout=90,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()

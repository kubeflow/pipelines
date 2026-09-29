#!/usr/bin/env python3
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
"""Exercise Dockerfile download selection without downloading or building."""

import os
from pathlib import Path
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[3]
API_DOCKERFILE = ROOT / 'backend/api/Dockerfile'
RELEASE_DOCKERFILE = ROOT / 'release/Dockerfile.release'
STUBS = '''
curl() { printf '%s\\n' "$@"; return "${CURL_EXIT_CODE:-0}"; }
chmod() { :; }
tar() { echo extracted; }
rm() { :; }
cd() { :; }
go() { :; }
jq() { echo v0.36.4; }
'''


def run_instruction(path, marker):
    instructions = path.read_text().replace('\\\n', '').splitlines()
    return next(line[4:]
                for line in instructions
                if line.startswith('RUN ') and marker in line)


class ToolImageArchitectureTest(unittest.TestCase):

    def run_download(self, path, marker, architecture, curl_exit_code=0):
        return subprocess.run(
            ['sh', '-c', STUBS + run_instruction(path, marker)],
            env={
                **os.environ,
                'TARGETARCH': architecture,
                'PROTOC_VERSION': '31.1',
                'GIT_CLIFF_VERSION': '2.10.0',
                'CURL_EXIT_CODE': str(curl_exit_code),
            },
            capture_output=True,
            text=True,
            timeout=10,
        )

    def test_downloads_correct_assets_for_each_architecture(self):
        for arch, protoc_arch, cliff_arch in (
            ('amd64', 'x86_64', 'x86_64'),
            ('arm64', 'aarch_64', 'aarch64'),
        ):
            downloads = (
                (API_DOCKERFILE, 'protoc_arch=',
                 'protocolbuffers/protobuf/releases/download/v31.1/'
                 f'protoc-31.1-linux-{protoc_arch}.zip'),
                (API_DOCKERFILE, 'swagger_linux_',
                 'go-swagger/go-swagger/releases/download/v0.36.4/'
                 f'swagger_linux_{arch}'),
                (RELEASE_DOCKERFILE, 'yq_linux_',
                 f'mikefarah/yq/releases/download/3.4.1/yq_linux_{arch}'),
                (RELEASE_DOCKERFILE, 'git_cliff_arch=',
                 'orhun/git-cliff/releases/download/v2.10.0/'
                 f'git-cliff-2.10.0-{cliff_arch}-unknown-linux-gnu.tar.gz'),
            )
            for path, marker, asset in downloads:
                with self.subTest(architecture=arch, marker=marker):
                    result = self.run_download(path, marker, arch)
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertIn('https://github.com/' + asset,
                                  result.stdout.splitlines())

    def test_rejects_unsupported_architectures_before_download(self):
        for path, marker in (
            (API_DOCKERFILE, 'protoc_arch='),
            (RELEASE_DOCKERFILE, 'amd64|arm64'),
            (RELEASE_DOCKERFILE, 'git_cliff_arch='),
        ):
            for arch in ('', 's390x', 'arm'):
                with self.subTest(path=path, marker=marker, architecture=arch):
                    result = self.run_download(path, marker, arch)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn('Unsupported TARGETARCH:', result.stderr)
                    self.assertEqual(result.stdout, '')

    def test_download_failure_stops_archive_extraction(self):
        result = self.run_download(
            RELEASE_DOCKERFILE, 'git_cliff_arch=', 'arm64', curl_exit_code=22)
        self.assertEqual(result.returncode, 22, result.stderr)
        self.assertNotIn('extracted', result.stdout)

    def test_target_arch_is_declared_in_each_build_stage(self):
        for path in (API_DOCKERFILE, RELEASE_DOCKERFILE):
            with self.subTest(path=path):
                lines = path.read_text().splitlines()
                start = next(i for i, line in enumerate(lines)
                             if line.startswith('FROM '))
                self.assertEqual(lines[start + 1], 'ARG TARGETARCH')


if __name__ == '__main__':
    unittest.main()

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

import shlex
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
TARGETS = (
    '.github/workflows/e2e-test.yml',
    'manifests/kustomize/third-party/seaweedfs/test.sh',
)


class StorageMigrationWaitTest(unittest.TestCase):
    def test_consumers_wait_for_durable_completion(self):
        for relative in TARGETS:
            with self.subTest(path=relative):
                text = (ROOT / relative).read_text()
                self.assertFalse(
                    'deployment/seaweedfs-credential-rotation' in text,
                    'Waiting on the removed controller cannot observe '
                    'migration completion',
                )
                commands = [
                    shlex.split(line.strip())
                    for line in text.splitlines()
                    if line.strip().startswith('kubectl ')
                    and 'secret/seaweedfs-admin-credentials' in line
                ]
                self.assertEqual(len(commands), 1)
                args = commands[0]
                self.assertIn('wait', args)
                self.assertIn('--timeout=600s', args)
                self.assertIn(
                    r'--for=jsonpath={.metadata.annotations.pipelines\.kubeflow\.org/'
                    'storage-credential-phase}=DefaultRevoked',
                    args,
                )
                # Completion must precede reading credentials or exposing storage.
                wait_index = text.index('secret/seaweedfs-admin-credentials')
                consumer = (
                    'access_key=$(kubectl'
                    if relative.endswith('.yml')
                    else 'port-forward svc/seaweedfs'
                )
                self.assertLess(wait_index, text.index(consumer))


if __name__ == '__main__':
    unittest.main()

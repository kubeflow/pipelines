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
"""Exercise the backend Dockerfile's sample revision substitutions offline."""

import json
import re
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
COMMIT = '0123456789abcdef0123456789abcdef01234567'


class SampleConfigRevisionTest(unittest.TestCase):
    def rewrite(self, value):
        dockerfile = (ROOT / 'backend/Dockerfile').read_text()
        expressions = re.findall(
            r'sed -E "([^"]+)" -i /config/sample_config.json', dockerfile
        )
        self.assertEqual(len(expressions), 2)
        for expression in expressions:
            result = subprocess.run(
                ['sed', '-E', expression.replace('${COMMIT_SHA}', COMMIT)],
                input=value,
                text=True,
                capture_output=True,
                check=False,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            value = result.stdout
        return value

    def test_current_and_legacy_branch_links_are_pinned(self):
        for branch in ('main', 'master'):
            for kind in ('tree', 'blob'):
                with self.subTest(branch=branch, kind=kind):
                    prefix = f'https://github.com/kubeflow/pipelines/{kind}/'
                    self.assertEqual(
                        self.rewrite(f'{prefix}{branch}/samples\n'),
                        f'{prefix}{COMMIT}/samples\n',
                    )

    def test_encoded_branch_links_are_pinned(self):
        for branch in ('main', 'master'):
            with self.subTest(branch=branch):
                self.assertEqual(
                    self.rewrite(
                        f'github.com%252Fkubeflow%252Fpipelines%252Fblob%252F{branch}%252Fsamples\n'
                    ),
                    f'github.com%252Fkubeflow%252Fpipelines%252Fblob%252F{COMMIT}%252Fsamples\n',
                )

    def test_release_and_commit_links_are_preserved(self):
        for revision in ('2.16.0', 'main-experiment', 'master-backup', COMMIT):
            with self.subTest(revision=revision):
                link = f'https://github.com/kubeflow/pipelines/tree/{revision}/samples\n'
                self.assertEqual(self.rewrite(link), link)
                encoded = link.replace('/', '%252F')
                self.assertEqual(self.rewrite(encoded), encoded)

    def test_bundled_sample_links_are_pinned(self):
        original = json.loads(
            (
                ROOT / 'backend/src/apiserver/config/sample_config.json'
            ).read_text()
        )
        rewritten = json.loads(self.rewrite(json.dumps(original)))
        for before, after in zip(original['pipelines'], rewritten['pipelines']):
            self.assertIn(f'/tree/{COMMIT}/', after['description'])
            self.assertEqual(before['file'], after['file'])


if __name__ == '__main__':
    unittest.main()

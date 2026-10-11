# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import configure_frontend_apt as apt


class FrontendAptTest(unittest.TestCase):

    def test_mirror_replacement_preserves_suites_signatures_and_other_sources(
            self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'apt.conf.d').mkdir()
            (root / 'sources.list.d').mkdir()
            body = (
                'URIs: http://azure.archive.ubuntu.com/ubuntu\n'
                'Suites: noble noble-updates\n'
                'Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg\n'
                'https://packages.microsoft.com/ubuntu/24.04/prod\n'
                'https://azure.archive.ubuntu.com/ubuntu-other\n')
            sources = [
                root / 'apt-mirrors.txt', root / 'sources.list',
                root / 'sources.list.d/ubuntu.sources',
                root / 'sources.list.d/legacy.list'
            ]
            for source in sources:
                source.write_text(body)
            apt.configure(root)
            for source in sources:
                self.assertEqual(
                    source.read_text(),
                    body.replace('http://azure.archive.ubuntu.com/ubuntu\n',
                                 'https://archive.ubuntu.com/ubuntu\n'))
            limits = root / 'apt.conf.d/99-kfp-browser-acquisition'
            self.assertIn('Acquire::http::Timeout "30";', limits.read_text())
            self.assertIn('Acquire::https::Timeout "30";', limits.read_text())
            self.assertIn('Acquire::Retries "2";', limits.read_text())
            apt.configure(root)
            self.assertEqual(limits.read_text(), apt.APT_LIMITS)

    def test_local_and_self_hosted_execution_cannot_modify_apt(self):
        for environment in ({}, {
                'CI': 'true',
                'GITHUB_ACTIONS': 'true',
                'RUNNER_ENVIRONMENT': 'self-hosted'
        }):
            with mock.patch.dict(os.environ, environment, clear=True):
                with mock.patch.object(apt, 'configure') as configure:
                    with self.assertRaisesRegex(RuntimeError, 'disposable'):
                        apt.main()
                    configure.assert_not_called()


if __name__ == '__main__':
    unittest.main()

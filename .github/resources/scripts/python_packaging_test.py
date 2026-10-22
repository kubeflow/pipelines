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

from pathlib import Path
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]


class PythonPackagingTest(unittest.TestCase):

    def test_readthedocs_bounds_font_download_concurrency(self) -> None:
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/readthedocs-builds.yml').read_text())
        job = workflow['jobs']['test-readthedocs-builds']
        variable = 'SPHINX_IMMATERIAL_FONT_FETCH_MAX_WORKERS'
        environment = job.get('env', {})
        self.assertEqual(int(environment.get(variable, 128)), 4)
        self.assertFalse(job.get('continue-on-error', False))
        builds = [
            step for step in job['steps'] if step['name'] in
            ('Build KFP SDK Docs', 'Build KFP Kubernetes SDK Docs')
        ]
        self.assertEqual(len(builds), 2)
        for step in builds:
            with self.subTest(step=step['name']):
                effective = {**environment, **step.get('env', {})}
                self.assertEqual(int(effective[variable]), 4)
                self.assertEqual(step['run'].strip(),
                                 'sphinx-build -b html . _build/html')
                self.assertFalse(step.get('continue-on-error', False))


if __name__ == '__main__':
    unittest.main()

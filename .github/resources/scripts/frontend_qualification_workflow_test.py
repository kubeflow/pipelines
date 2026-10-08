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
"""Keep frontend presubmit validation fast and complete."""

from itertools import product
import json
from pathlib import Path
import unittest

from generate_ci_workflow_inventory import UniqueKeyLoader
import yaml

ROOT = Path(__file__).resolve().parents[3]


def workflow(name):
    return yaml.load(
        (ROOT / '.github/workflows' / name).read_text(), Loader=UniqueKeyLoader)


def lanes(job):
    matrix = job['strategy']['matrix']
    excluded = {
        (row['os'], row['browser']) for row in matrix.get('exclude', [])
    }
    return set(product(matrix['os'], matrix['browser'])) - excluded


class FrontendQualificationWorkflowTest(unittest.TestCase):

    def test_presubmit_bundles_after_validation_without_repeating_it(self):
        scripts = json.loads(
            (ROOT / 'frontend/package.json').read_text())['scripts']
        steps = workflow('frontend.yml')['jobs']['frontend-tests']['steps']
        checks = next(i for i, step in enumerate(steps)
                      if 'npm run test:ci' in step.get('run', ''))
        bundle = next(i for i, step in enumerate(steps)
                      if 'npm run build:bundle' in step.get('run', ''))
        self.assertLess(checks, bundle)
        for check in ('lint', 'typecheck'):
            self.assertIn(f'npm run {check}', scripts['test:ci'])
            self.assertIn(f'npm run {check}', scripts['build'])
            self.assertNotIn(check, scripts['build:bundle'])
        self.assertIn('npm run build:tailwind', scripts['build:bundle'])
        self.assertIn('vite build', scripts['build:bundle'])
        self.assertNotIn('npm run build', scripts['test:ci'])

    def test_presubmit_runs_linux_chromium_against_shared_bundle(self):
        presubmit = workflow('frontend.yml')
        self.assertEqual(
            lanes(presubmit['jobs']['browser-tests']),
            {('ubuntu-latest', 'chromium')})
        self.assertIn('pull_request', presubmit['on'])
        self.assertEqual(presubmit['jobs']['browser-tests']['needs'],
                         'frontend-tests')


if __name__ == '__main__':
    unittest.main()

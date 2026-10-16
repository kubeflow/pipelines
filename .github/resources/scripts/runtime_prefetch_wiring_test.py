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
"""Keep runtime prefetch opt-ins consistent with deployment and cleanup."""

from pathlib import Path
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]


class RuntimePrefetchWiringTest(unittest.TestCase):

    def test_every_api_e2e_cluster_prefetch_matches_deployment(self):
        for name in ('api-server-tests.yml', 'e2e-test.yml'):
            workflow = yaml.safe_load(
                (ROOT / '.github/workflows' / name).read_text())
            for job in workflow['jobs'].values():
                steps = job.get('steps', [])
                creates = [
                    s for s in steps
                    if s.get('uses') == './.github/actions/create-cluster'
                ]
                if not creates:
                    continue
                deploy = next(
                    s for s in steps
                    if s.get('uses') == './.github/actions/deploy')['with']
                create = creates[0]['with']
                self.assertEqual(create['runtime_image_path'],
                                 deploy['image_path'])
                for before, after, default in (('runtime_argo_version',
                                                'argo_version', ''),
                                               ('runtime_db_type', 'db_type',
                                                ''), ('runtime_modelcar',
                                                      'load_modelcar_fixture',
                                                      'false')):
                    self.assertEqual(
                        create.get(before, default), deploy.get(after, default))

    def test_cleanup_precedes_download_and_always_stops_before_return(self):
        action = yaml.safe_load(
            (ROOT / '.github/actions/create-cluster/action.yml').read_text())
        steps = action['runs']['steps']
        names = [step['name'] for step in steps]
        self.assertLess(
            names.index('Free up disk space on runner'),
            names.index('Prefetch runtime images during cluster setup'))
        self.assertLess(
            names.index('Prefetch runtime images during cluster setup'),
            names.index('Restore Kind node image cache'))
        self.assertEqual(steps[-1]['name'],
                         'Stop runtime prefetch on every exit')
        self.assertIn('always()', steps[-1]['if'])
        self.assertIn('success()', steps[-2]['if'])

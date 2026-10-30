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

"""Keep workflow coverage and publication safe during the branch rename."""

import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOWS = ROOT / '.github/workflows'
DEFAULT_REF = "github.ref == format('refs/heads/{0}', github.event.repository.default_branch)"


class MainBranchWorkflowsTest(unittest.TestCase):
    def test_main_has_every_master_event_filter(self):
        for path in sorted(WORKFLOWS.glob('*.y*ml')):
            workflow = yaml.safe_load(path.read_text())
            events = workflow.get('on', workflow.get(True, {}))
            if not isinstance(events, dict):
                continue
            for event, config in events.items():
                if not isinstance(config, dict):
                    continue
                branches = config.get('branches', [])
                if 'master' in branches:
                    with self.subTest(workflow=path.name, event=event):
                        self.assertIn('main', branches)

    def test_shared_runtime_tags_only_publish_from_default_branch(self):
        workflow = yaml.safe_load(
            (WORKFLOWS / 'image-builds-master.yml').read_text()
        )
        job = workflow['jobs']['build-images-for-master']
        self.assertIn(DEFAULT_REF, job.get('if', ''))
        self.assertEqual(job['with']['target_tag'], 'master')
        self.assertEqual(
            workflow['jobs']['create-manifests']['with']['target_tag'], 'master'
        )

    def test_tool_tags_remain_compatible_after_rename(self):
        workflow = yaml.safe_load(
            (WORKFLOWS / 'build-tools-images.yml').read_text()
        )
        self.assertIn(
            "github.ref_name == 'main' && 'master'",
            workflow['env']['IMAGE_TAG'],
        )
        self.assertIn(DEFAULT_REF, workflow['jobs']['build-tools']['if'])
        self.assertIn(
            "github.ref_name != 'master' && github.ref_name != 'main'",
            workflow['jobs']['build-tools']['if'],
        )

    def test_promotion_requires_current_default_branch(self):
        workflow = yaml.safe_load(
            (WORKFLOWS / 'release-promote-latest.yml').read_text()
        )
        for name, job in workflow['jobs'].items():
            with self.subTest(job=name):
                self.assertIn(DEFAULT_REF, job['if'])
                self.assertIn(
                    "github.repository == 'kubeflow/pipelines'", job['if']
                )


if __name__ == '__main__':
    unittest.main()

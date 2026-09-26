#!/usr/bin/env python3
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

import json
from pathlib import Path
import tempfile
import unittest

from arm64_presubmit_image import image_record
from arm64_smoke import CI_IMAGES
from arm64_smoke import image_refs
import yaml

ROOT = Path(__file__).resolve().parents[3]
SHA = 'a' * 40


class Arm64PresubmitTest(unittest.TestCase):

    def test_build_records_are_consumable_without_retagging(self):
        with tempfile.TemporaryDirectory() as directory:
            for image in CI_IMAGES.values():
                record = image_record(image, SHA, 'kind-registry:5000')
                Path(directory,
                     f'{image}-arm64.json').write_text(json.dumps(record))
            refs = image_refs(directory, SHA, 'local')
        self.assertEqual(
            refs, {
                image: f'kind-registry:5000/{artifact}:ci'
                for image, artifact in CI_IMAGES.items()
            })

    def test_wrong_source_or_unknown_image_is_rejected(self):
        for image, sha in [('apiserver', 'master'), ('retired-image', SHA)]:
            with self.subTest(
                    image=image, sha=sha), self.assertRaises(ValueError):
                image_record(image, sha, 'kind-registry:5000')

    def test_one_arm_producer_and_architecture_scoped_consumer(self):
        workflows = ROOT / '.github/workflows'
        consumers = []
        for path in workflows.glob('*.yml'):
            jobs = yaml.safe_load(path.read_text()).get('jobs', {})
            for job in jobs.values():
                if (job.get('uses') == './.github/workflows/image-builds.yml'
                        and job.get('with', {}).get('architecture') == 'arm64'):
                    consumers.append(path.name)
        self.assertEqual(consumers, ['arm64-presubmit.yml'])
        workflow = yaml.safe_load(
            (workflows / 'arm64-presubmit.yml').read_text())
        smoke = workflow['jobs']['smoke']
        self.assertEqual(smoke['needs'], 'build')
        self.assertEqual(smoke['runs-on'], 'ubuntu-24.04-arm')
        download = next(step for step in smoke['steps'] if step.get('uses') ==
                        './.github/actions/download-artifact-with-retry')
        self.assertEqual(download['with']['pattern'], '*-arm64')
        action = next(step for step in smoke['steps']
                      if step.get('uses') == './.github/actions/arm64-smoke')
        self.assertEqual(action['with']['image_mode'], 'local')
        self.assertEqual(action['with']['source_sha'], '${{ github.sha }}')

    def test_runtime_cache_and_default_consumers_remain_amd64(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/image-builds.yml').read_text())
        event = workflow.get('on', workflow.get(True))['workflow_call']
        self.assertEqual(event['inputs']['architecture']['default'], 'amd64')
        self.assertEqual(workflow['jobs']['runtime-base-images']['if'],
                         "${{ inputs.architecture == 'amd64' }}")
        matrix = workflow['jobs']['image-build']['strategy']['matrix'][
            'include']
        self.assertEqual({entry['image'] for entry in matrix},
                         set(CI_IMAGES.values()))

    def test_filters_cover_compiler_and_local_module_inputs(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/arm64-presubmit.yml').read_text())
        event = workflow.get('on', workflow.get(True))
        paths = event['pull_request']['paths']
        for path in ('kubernetes_platform/**', 'sdk/python/**', 'samples/**',
                     'pyproject.toml', 'uv.lock', 'api/**', 'go.mod', 'go.sum'):
            self.assertIn(path, paths)


if __name__ == '__main__':
    unittest.main()

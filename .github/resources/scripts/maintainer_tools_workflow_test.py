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
"""Keep native tooling checks ahead of shared-tag publication."""

from pathlib import Path
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/build-tools-images.yml'
NOT_PULL_REQUEST = "github.event_name != 'pull_request'"


class MaintainerToolsWorkflowTest(unittest.TestCase):

    def setUp(self):
        self.workflow = yaml.safe_load(WORKFLOW.read_text())
        self.jobs = self.workflow['jobs']
        self.build = self.jobs['build-tools']
        self.compare = self.jobs['compare-generated']
        self.publish = self.jobs['publish-tools']

    def step_running(self, job, marker):
        return next(
            step for step in job['steps'] if marker in step.get('run', ''))

    def step_using(self, job, action):
        return next(step for step in job['steps']
                    if step.get('uses', '').split('@')[0] == action)

    def test_builds_on_both_native_runners(self):
        self.assertEqual(self.build['strategy']['matrix']['include'], [
            {
                'arch': 'amd64',
                'runner': 'ubuntu-latest'
            },
            {
                'arch': 'arm64',
                'runner': 'ubuntu-24.04-arm'
            },
        ])
        self.assertEqual(self.build['runs-on'], '${{ matrix.runner }}')
        self.assertFalse(self.build['strategy']['fail-fast'])
        self.assertEqual(self.build['env']['PLATFORM'],
                         'linux/${{ matrix.arch }}')
        self.assertNotIn('setup-qemu', WORKFLOW.read_text())

    def test_release_image_uses_generator_from_same_local_build(self):
        buildx = self.step_using(self.build, 'docker/setup-buildx-action')
        self.assertEqual(buildx['with']['driver'], 'docker')
        generator = self.step_running(self.build,
                                      '--file backend/api/Dockerfile')
        release = self.step_running(self.build,
                                    '--file release/Dockerfile.release')
        self.assertIn('--tag kfp-api-generator:ci', generator['run'])
        self.assertIn('--build-arg BASE_IMAGE=kfp-api-generator:ci',
                      release['run'])
        for step in (generator, release):
            self.assertIn('docker build --platform "$PLATFORM"', step['run'])
            self.assertNotIn('--pull', step['run'])
        steps = self.build['steps']
        self.assertLess(steps.index(generator), steps.index(release))
        self.assertLess(
            steps.index(release),
            steps.index(
                self.step_running(self.build, 'maintainer_tools_smoke.sh')))

    def test_pull_requests_test_but_cannot_stage_or_promote(self):
        login = self.step_using(self.build, 'docker/login-action')
        staging = self.step_running(self.build, 'docker push')
        references = next(
            step for step in self.build['steps']
            if step.get('with', {}).get('name', '').startswith('tool-digests-'))
        for step in (login, staging, references):
            self.assertEqual(step['if'], NOT_PULL_REQUEST)
        self.assertEqual(self.publish['if'], NOT_PULL_REQUEST)
        self.assertNotIn('if', self.compare)
        self.assertNotIn(
            'if', self.step_running(self.build, 'maintainer_tools_smoke.sh'))

    def test_stages_tested_images_with_unique_run_tags_and_source_records(self):
        staging = self.step_running(self.build, 'docker push')
        steps = self.build['steps']
        smoke = self.step_running(self.build, 'maintainer_tools_smoke.sh')
        self.assertLess(steps.index(smoke), steps.index(staging))
        self.assertEqual(self.build['env']['SOURCE_SHA'], '${{ github.sha }}')
        self.assertEqual(self.build['env']['RUN_TAG'],
                         'run-${{ github.run_id }}-${{ github.run_attempt }}')
        checkout = self.step_using(self.build, 'actions/checkout')
        self.assertEqual(checkout['with']['ref'], '${{ github.sha }}')
        command = staging['run']
        self.assertIn('docker tag "${image}:ci" "$uploaded"', command)
        self.assertIn('${RUN_TAG}-${ARCH}', command)
        self.assertIn('imagetools create --prefer-index=true', command)
        self.assertIn('--arg source_sha "$SOURCE_SHA"', command)
        self.assertIn('--arg platform "$PLATFORM"', command)
        self.assertIn(
            'digest: $digest, source_sha: $source_sha, platform: $platform',
            command)
        self.assertNotIn('$IMAGE_TAG', command)

    def test_comparison_and_promotion_require_successful_native_smokes(self):
        self.assertEqual(self.compare['needs'], 'build-tools')
        self.assertEqual(
            set(self.publish['needs']), {'build-tools', 'compare-generated'})
        comparison = self.step_running(self.compare, 'diff -ru')
        self.assertIn('tool-output-amd64-$RUN_ATTEMPT', comparison['run'])
        self.assertIn('tool-output-arm64-$RUN_ATTEMPT', comparison['run'])
        for job in (self.build, self.compare, self.publish):
            self.assertNotIn('continue-on-error', job)
            for step in job['steps']:
                self.assertNotIn('continue-on-error', step)

    def test_artifacts_are_scoped_to_same_run_attempt(self):
        action = './.github/actions/download-artifact-with-retry'
        generated = self.step_using(self.compare, action)['with']
        self.assertEqual(generated['pattern'],
                         'tool-output-*-${{ github.run_attempt }}')
        self.assertNotEqual(generated.get('merge-multiple'), 'true')
        digests = self.step_using(self.publish, action)['with']
        self.assertEqual(digests['pattern'],
                         'tool-digests-*-${{ github.run_attempt }}')
        self.assertEqual(digests['merge-multiple'], 'true')
        for job in (self.build, self.publish):
            for step in job['steps']:
                if step.get('uses', '').startswith('actions/upload-artifact@'):
                    self.assertIn('${{ github.run_attempt }}',
                                  step['with']['name'])
                    self.assertEqual(step['with']['if-no-files-found'], 'error')

    def test_shared_tags_use_validated_indexes_without_latest(self):
        promotion = self.step_running(self.publish, 'publish_image_index.py')
        self.assertEqual(promotion['env']['SOURCE_SHA'], '${{ github.sha }}')
        command = promotion['run']
        self.assertIn('for image in kfp-api-generator kfp-release;', command)
        self.assertIn('--source-sha "$SOURCE_SHA"', command)
        self.assertIn('--platforms linux/amd64,linux/arm64', command)
        self.assertIn('--digests "tool-digests/$image"', command)
        self.assertIn('--target-tag "$IMAGE_TAG" --set-latest false', command)
        self.assertNotIn('docker push', command)


if __name__ == '__main__':
    unittest.main()

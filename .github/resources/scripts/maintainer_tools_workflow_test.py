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

import unittest
from pathlib import Path

import yaml
from artifact_workflow_test_support import evaluate

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/build-tools-images.yml'
BUILD_ACTION = './.github/actions/build-tools-images'


class MaintainerToolsWorkflowTest(unittest.TestCase):
    def setUp(self):
        self.workflow = yaml.safe_load(WORKFLOW.read_text())
        self.jobs = self.workflow['jobs']
        self.check = self.jobs['build-tools-check']
        self.build = self.jobs['build-tools']
        self.native_build = yaml.safe_load(
            (ROOT / BUILD_ACTION / 'action.yml').read_text()
        )['runs']
        self.compare = self.jobs['compare-generated']
        self.publish = self.jobs['publish-tools']

    def step_running(self, job, marker):
        return next(
            step for step in job['steps'] if marker in step.get('run', '')
        )

    def step_using(self, job, action):
        return next(
            step
            for step in job['steps']
            if step.get('uses', '').split('@')[0] == action
        )

    def test_builds_on_both_native_runners(self):
        self.assertEqual(
            self.build['strategy']['matrix']['include'],
            [
                {'arch': 'amd64', 'runner': 'ubuntu-latest'},
                {'arch': 'arm64', 'runner': 'ubuntu-24.04-arm'},
            ],
        )
        self.assertEqual(self.build['runs-on'], '${{ matrix.runner }}')
        self.assertEqual(self.check['strategy'], self.build['strategy'])
        self.assertEqual(self.check['runs-on'], self.build['runs-on'])
        for job in (self.check, self.build):
            self.assertEqual(
                self.step_using(job, BUILD_ACTION)['with']['arch'],
                '${{ matrix.arch }}',
            )
        self.assertFalse(self.build['strategy']['fail-fast'])
        self.assertEqual(
            self.build['env']['PLATFORM'], 'linux/${{ matrix.arch }}'
        )
        self.assertNotIn('setup-qemu', WORKFLOW.read_text())

    def test_release_image_uses_generator_from_same_local_build(self):
        buildx = self.step_using(
            self.native_build, 'docker/setup-buildx-action'
        )
        self.assertEqual(buildx['with']['driver'], 'docker')
        generator = self.step_running(
            self.native_build, '--file backend/api/Dockerfile'
        )
        release = self.step_running(
            self.native_build, '--file release/Dockerfile.release'
        )
        self.assertIn('--tag "$TOOL_GENERATOR_IMAGE:ci"', generator['run'])
        self.assertIn(
            '--build-arg "BASE_IMAGE=$TOOL_GENERATOR_IMAGE:ci"', release['run']
        )
        for step in (generator, release):
            self.assertIn('docker build --platform "$PLATFORM"', step['run'])
            self.assertNotIn('--pull', step['run'])
            self.assertEqual(
                step['env']['PLATFORM'], 'linux/${{ inputs.arch }}'
            )
        steps = self.native_build['steps']
        self.assertLess(steps.index(generator), steps.index(release))
        self.assertLess(
            steps.index(release),
            steps.index(
                self.step_running(
                    self.native_build, 'maintainer_tools_smoke.sh'
                )
            ),
        )

    def test_pull_requests_test_but_cannot_stage_or_promote(self):
        self.assertEqual(self.workflow['permissions'], {'contents': 'read'})
        self.assertEqual(self.check['permissions'], {'contents': 'read'})
        self.assertEqual(
            self.build['permissions'], {'contents': 'read', 'packages': 'write'}
        )
        self.assertEqual(self.publish['permissions'], self.build['permissions'])
        for event in ('pull_request', 'push', 'workflow_dispatch'):
            with self.subTest(event=event):
                context = {
                    'github': {
                        'event_name': event,
                        'ref': 'refs/heads/main',
                        'ref_name': 'main',
                        'event': {'repository': {'default_branch': 'main'}},
                    },
                    'needs': {
                        'build-tools': {'result': 'success'},
                        'compare-generated': {'result': 'success'},
                    },
                }
                is_pr = event == 'pull_request'
                self.assertEqual(evaluate(self.check['if'], context), is_pr)
                self.assertEqual(evaluate(self.build['if'], context), not is_pr)
                self.assertEqual(
                    evaluate(
                        self.publish['if'],
                        context,
                        {'cancelled': lambda: False},
                    ),
                    not is_pr,
                )
        for job in (self.check, self.native_build):
            for step in job['steps']:
                self.assertNotIn('docker/login-action', step.get('uses', ''))
                self.assertNotIn('docker push', step.get('run', ''))
        self.assertNotIn(
            'if',
            self.step_running(self.native_build, 'maintainer_tools_smoke.sh'),
        )

    def test_only_current_default_can_publish_shared_development_tag(self):
        for default in ('master', 'main'):
            for branch in ('master', 'main', 'release-2.18', 'feature-tools'):
                with self.subTest(default=default, branch=branch):
                    context = {
                        'github': {
                            'event_name': 'workflow_dispatch',
                            'ref': f'refs/heads/{branch}',
                            'ref_name': branch,
                            'event': {
                                'repository': {'default_branch': default}
                            },
                        }
                    }
                    self.assertEqual(
                        evaluate(self.build['if'], context),
                        branch == default or branch not in ('master', 'main'),
                    )
                    if branch == default:
                        self.assertEqual(
                            evaluate(
                                self.workflow['env']['IMAGE_TAG'], context
                            ),
                            'master',
                        )

    def test_checkouts_do_not_leave_git_credentials(self):
        for job in self.jobs.values():
            checkout = self.step_using(job, 'actions/checkout')
            self.assertIs(checkout['with']['persist-credentials'], False)

    def test_stages_tested_images_with_unique_run_tags_and_source_records(self):
        staging = self.step_running(self.build, 'docker push')
        steps = self.build['steps']
        native_build = self.step_using(self.build, BUILD_ACTION)
        self.assertLess(steps.index(native_build), steps.index(staging))
        self.assertEqual(self.build['env']['SOURCE_SHA'], '${{ github.sha }}')
        self.assertEqual(
            self.build['env']['RUN_TAG'],
            'run-${{ github.run_id }}-${{ github.run_attempt }}',
        )
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
            command,
        )
        self.assertNotIn('$IMAGE_TAG', command)

    def test_comparison_and_promotion_require_successful_native_smokes(self):
        self.assertEqual(
            set(self.compare['needs']), {'build-tools-check', 'build-tools'}
        )
        self.assertEqual(
            set(self.publish['needs']), {'build-tools', 'compare-generated'}
        )
        comparison = self.step_running(self.compare, 'diff -ru')
        self.assertIn('tool-output-amd64-$RUN_ATTEMPT', comparison['run'])
        self.assertIn('tool-output-arm64-$RUN_ATTEMPT', comparison['run'])
        for job in (
            self.check,
            self.build,
            self.native_build,
            self.compare,
            self.publish,
        ):
            self.assertNotIn('continue-on-error', job)
            for step in job['steps']:
                self.assertNotIn('continue-on-error', step)

    def test_comparison_runs_after_selected_producer_and_blocks_failure(self):
        for check, build, cancelled, expected in (
            ('success', 'skipped', False, True),
            ('skipped', 'success', False, True),
            ('failure', 'skipped', False, False),
            ('skipped', 'failure', False, False),
            ('cancelled', 'skipped', False, False),
            ('skipped', 'cancelled', False, False),
            ('skipped', 'skipped', False, False),
            ('success', 'skipped', True, False),
            ('skipped', 'success', True, False),
        ):
            with self.subTest(check=check, build=build, cancelled=cancelled):
                actual = evaluate(
                    self.compare['if'],
                    {
                        'needs': {
                            'build-tools-check': {'result': check},
                            'build-tools': {'result': build},
                        },
                    },
                    {'cancelled': lambda: cancelled},
                )
                self.assertEqual(actual, expected)

    def test_publication_requires_successful_build_and_comparison(self):
        for build in ('success', 'failure', 'cancelled', 'skipped'):
            for comparison in ('success', 'failure', 'cancelled', 'skipped'):
                for cancelled in (False, True):
                    with self.subTest(
                        build=build, comparison=comparison, cancelled=cancelled
                    ):
                        actual = evaluate(
                            self.publish['if'],
                            {
                                'github': {'event_name': 'push'},
                                'needs': {
                                    'build-tools': {'result': build},
                                    'compare-generated': {'result': comparison},
                                },
                            },
                            {'cancelled': lambda: cancelled},
                        )
                        self.assertEqual(
                            actual,
                            build == comparison == 'success' and not cancelled,
                        )

    def test_artifacts_are_scoped_to_same_run_attempt(self):
        action = './.github/actions/download-artifact-with-retry'
        generated = self.step_using(self.compare, action)['with']
        self.assertEqual(
            generated['pattern'], 'tool-output-*-${{ github.run_attempt }}'
        )
        self.assertNotEqual(generated.get('merge-multiple'), 'true')
        digests = self.step_using(self.publish, action)['with']
        self.assertEqual(
            digests['pattern'], 'tool-digests-*-${{ github.run_attempt }}'
        )
        self.assertEqual(digests['merge-multiple'], 'true')
        for job in (self.build, self.native_build, self.publish):
            for step in job['steps']:
                if step.get('uses', '').startswith('actions/upload-artifact@'):
                    self.assertIn(
                        '${{ github.run_attempt }}', step['with']['name']
                    )
                    self.assertEqual(step['with']['if-no-files-found'], 'error')

    def test_shared_tags_use_validated_indexes_without_latest(self):
        promotion = self.step_running(self.publish, 'publish_image_index.py')
        self.assertEqual(promotion['env']['SOURCE_SHA'], '${{ github.sha }}')
        command = promotion['run']
        self.assertIn('for image in "${TOOL_IMAGE_ARTIFACTS[@]}";', command)
        self.assertIn('--source-sha "$SOURCE_SHA"', command)
        self.assertIn('--platforms linux/amd64,linux/arm64', command)
        self.assertIn('--digests "tool-digests/$image"', command)
        self.assertIn('--target-tag "$IMAGE_TAG" --set-latest false', command)
        self.assertNotIn('docker push', command)


if __name__ == '__main__':
    unittest.main()

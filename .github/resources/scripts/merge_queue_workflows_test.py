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
"""Check queue event wiring without running cluster or image builds."""

import unittest
from pathlib import Path

import yaml
from artifact_workflow_test_support import evaluate as evaluate_expression
from generate_ci_workflow_inventory import UniqueKeyLoader, build_inventory

ROOT = Path(__file__).resolve().parents[3]


def workflow(name):
    return yaml.load(
        (ROOT / '.github/workflows' / name).read_text(), Loader=UniqueKeyLoader
    )


def evaluate(expression, event, matrix_check='static', **overrides):
    """Evaluate the simple event/ref expressions used by these workflows."""
    context = {
        'event_name': event,
        'base_ref': '',
        'head_ref': '',
        'ref_name': 'gh-readonly-queue/master/pr-7-candidate',
        'repository': 'kubeflow/pipelines',
        'ref': 'refs/heads/master',
        'event': {
            'repository': {'default_branch': 'master'},
            'merge_group': {
                'head_sha': 'a' * 40,
                'base_sha': 'b' * 40,
            }
            if event == 'merge_group'
            else {},
            'pull_request': {
                'base': {'sha': 'c' * 40},
                'head': {'repo': {'full_name': 'contributor/pipelines'}},
            }
            if event == 'pull_request'
            else {'base': {}, 'head': {'repo': {}}},
            'before': 'd' * 40 if event == 'push' else '',
        },
    }
    context.update(overrides)
    return evaluate_expression(
        expression,
        {
            'github': context,
            'matrix': {'check': matrix_check},
            'needs': {
                'build-tools': {'result': 'success'},
                'compare-generated': {'result': 'success'},
            },
        },
        {'cancelled': lambda: False},
    )


class MergeQueueWorkflowsTest(unittest.TestCase):
    def test_every_pr_test_lane_runs_for_master_queue_without_path_filters(
        self,
    ):
        records = [
            {
                'path': path.relative_to(ROOT).as_posix(),
                'content': path.read_text(),
            }
            for path in (ROOT / '.github/workflows').iterdir()
            if path.suffix in ('.yml', '.yaml')
        ]
        inventory = build_inventory(records)
        lanes = [
            item
            for item in inventory['workflows']
            if item['pull_request'] is not None
        ]
        self.assertGreater(len(lanes), 30)
        for lane in lanes:
            with self.subTest(path=lane['path']):
                self.assertEqual(
                    lane['merge_group'],
                    {'types': ['checks_requested'], 'branches': ['master']},
                )
        paused = next(
            item for item in lanes if item['path'].endswith('/upgrade-test.yml')
        )
        self.assertTrue(paused['disabled_for_migration'])

    def test_inventory_distinguishes_missing_and_unfiltered_queue_trigger(self):
        for events, expected in [
            ('pull_request', None),
            ('[pull_request, merge_group]', {}),
            (
                '\n  merge_group:\n    types: [checks_requested]\n'
                '    branches: [master]',
                {'types': ['checks_requested'], 'branches': ['master']},
            ),
        ]:
            with self.subTest(events=events):
                inventory = build_inventory(
                    [
                        {
                            'path': '.github/workflows/example.yml',
                            'content': f'name: Example\non: {events}\njobs: {{}}\n',
                        }
                    ]
                )
                self.assertEqual(
                    inventory['workflows'][0]['merge_group'], expected
                )

    def test_precommit_uses_queue_base_and_preserves_pr_and_push_ranges(self):
        steps = workflow('pre-commit.yml')['jobs']['pre-commit']['steps']
        step = next(
            step for step in steps if step.get('id') == 'pre-commit-range'
        )
        expression = step['env']['EVENT_BASE_SHA']
        for event, expected in [
            ('merge_group', 'b' * 40),
            ('pull_request', 'c' * 40),
            ('push', 'd' * 40),
        ]:
            with self.subTest(event=event):
                self.assertEqual(evaluate(expression, event), expected)

    def test_frontend_drift_check_runs_against_immutable_queue_base(self):
        steps = workflow('frontend.yml')['jobs']['frontend-checks']['steps']
        step = next(
            step
            for step in steps
            if step.get('name') == 'Check for toolchain version drift'
        )
        for event in ['pull_request', 'merge_group']:
            self.assertTrue(evaluate(step['if'], event))
        self.assertFalse(evaluate(step['if'], 'push'))
        self.assertEqual(
            evaluate(step['env']['BASE_SHA'], 'merge_group'), 'b' * 40
        )
        self.assertIn('--base-ref "$BASE_SHA"', step['run'])

    def test_queue_builds_use_read_only_job_and_never_publish(self):
        definition = workflow('build-tools-images.yml')
        jobs = definition['jobs']
        self.assertEqual(
            jobs['build-tools-check']['permissions'], {'contents': 'read'}
        )
        for event in [
            'merge_group',
            'pull_request',
            'push',
            'workflow_dispatch',
        ]:
            with self.subTest(event=event):
                self.assertEqual(
                    evaluate(jobs['build-tools-check']['if'], event),
                    event in ['merge_group', 'pull_request'],
                )
                for name in ['build-tools', 'publish-tools']:
                    self.assertEqual(
                        evaluate(jobs[name]['if'], event),
                        event in ['push', 'workflow_dispatch'],
                    )
        self.assertEqual(
            evaluate(definition['env']['IMAGE_TAG'], 'push', ref_name='main'),
            'master',
        )
        self.assertEqual(
            evaluate(
                definition['env']['IMAGE_TAG'],
                'pull_request',
                base_ref='release-2.18',
            ),
            'release-2.18',
        )

    def test_third_party_upgrades_compare_immutable_queue_base(self):
        for job in workflow('third-party-image-upgrades.yml')['jobs'].values():
            step = next(
                step
                for step in job['steps']
                if step.get('with', {}).get('path') == 'baseline'
            )
            self.assertEqual(
                evaluate(step['with']['ref'], 'merge_group'), 'b' * 40
            )

    def test_remote_test_sources_use_queue_commit(self):
        for name in [
            'kfp-sdk-tests.yml',
            'kfp-sdk-unit-tests.yml',
            'kfp-kubernetes-native-migration-tests.yaml',
        ]:
            for job in workflow(name)['jobs'].values():
                for step in job.get('steps', []):
                    if 'PULL_NUMBER' not in step.get('env', {}):
                        continue
                    with self.subTest(workflow=name, step=step.get('name')):
                        self.assertEqual(
                            evaluate(
                                step['env']['TEST_SOURCE_REF'], 'merge_group'
                            ),
                            'a' * 40,
                        )
        steps = workflow('presubmit-backend.yml')['jobs']['backend-tests'][
            'steps'
        ]
        env = next(
            step['env']
            for step in steps
            if step.get('name') == 'Run Backend Tests'
        )
        self.assertEqual(evaluate(env['GIT_BRANCH'], 'merge_group'), 'a' * 40)
        self.assertEqual(
            evaluate(env['GIT_REPO'], 'merge_group'), 'kubeflow/pipelines'
        )


if __name__ == '__main__':
    unittest.main()

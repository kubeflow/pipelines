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
"""Dispatched queue candidates retain automatic CI coverage and read-only lanes."""

import json
import os
import re
import subprocess
import unittest
from pathlib import Path

from artifact_workflow_test_support import evaluate
from merge_queue_workflows_test import workflow

ROOT = Path(__file__).resolve().parents[3]
SHA = 'a' * 40
BASE = 'b' * 40
QUEUE = 'refs/heads/gh-readonly-queue/master/pr-7-candidate'


def context(event='workflow_dispatch', ref=QUEUE, **overrides):
    value = {
        'github': {
            'event_name': event,
            'ref': ref,
            'ref_name': ref.removeprefix('refs/heads/'),
            'sha': SHA,
            'event': {
                'repository': {'default_branch': 'master'},
                'merge_group': {'head_sha': SHA, 'base_sha': BASE}
                if event == 'merge_group'
                else {},
            },
        },
        'inputs': {
            'queue_head_sha': SHA,
            'queue_base_ref': 'master',
            'queue_base_sha': BASE,
            'test_label': 'NoTests',
        },
        'matrix': {
            'cache_enabled': 'true',
            'test_shard': 'B',
            'check': 'static',
        },
        'steps': {'deploy': {'outcome': 'success'}},
        'needs': {'build-tools': {'result': 'success'}},
    }
    value.update(overrides)
    return value


def guard_case(branch='master', overrides=None, moved=None, malformed=None):
    script = workflow('queue-dispatch-guard.yml')['jobs']['validate']['steps'][
        0
    ]['with']['script']
    ref = QUEUE.replace('/master/', f'/{branch}/')
    env = {
        **os.environ,
        'QUEUE_HEAD_SHA': SHA,
        'QUEUE_BASE_REF': branch,
        'QUEUE_BASE_SHA': BASE,
        **(overrides or {}),
    }
    actual = {
        'ref': ref,
        'sha': SHA,
        'eventName': 'workflow_dispatch',
        'repo': {'owner': 'kubeflow', 'repo': 'pipelines'},
        **(malformed or {}),
    }
    wrapper = f"""
const context = {json.dumps(actual)};
const moved = {json.dumps(moved)};
const github = {{rest: {{git: {{getRef: async ({{ref}}) => ({{data: {{
  ref: 'refs/' + ref, object: {{type: 'commit', sha: ref === moved ? {'d' * 40!r} :
    ref.startsWith('heads/gh-readonly-queue/') ? {SHA!r} : {BASE!r}}}
}}}})}}}}}};
(async () => {{
  try {{ {script}; console.log(JSON.stringify({{ok:true}})); }}
  catch (error) {{ console.log(JSON.stringify({{error:error.message}})); }}
}})();
"""
    result = subprocess.run(
        ['node'],
        input=wrapper,
        env=env,
        text=True,
        capture_output=True,
        check=True,
    )
    return json.loads(result.stdout)


class QueueDispatchWorkflowsTest(unittest.TestCase):
    def test_every_queue_lane_accepts_dispatch_and_validates_its_identity(self):
        lanes = []
        for path in (ROOT / '.github/workflows').iterdir():
            if path.suffix not in ('.yaml', '.yml'):
                continue
            definition = workflow(path.name)
            if (
                'merge_group' not in definition['on']
                or path.name == 'upgrade-test.yml'
            ):
                continue
            lanes.append(path.name)
            with self.subTest(workflow=path.name):
                inputs = definition['on']['workflow_dispatch']['inputs']
                for key in (
                    'queue_head_sha',
                    'queue_base_ref',
                    'queue_base_sha',
                ):
                    self.assertEqual(inputs[key]['type'], 'string')
                    self.assertFalse(inputs[key]['required'])
                guard = definition['jobs']['queue-dispatch-guard']
                self.assertEqual(guard['permissions'], {'contents': 'read'})
                self.assertEqual(
                    guard['uses'],
                    './.github/workflows/queue-dispatch-guard.yml',
                )
                self.assertTrue(evaluate(guard['if'], context(inputs={})))
                self.assertFalse(evaluate(guard['if'], context('merge_group')))
                self.assertFalse(
                    evaluate(guard['if'], context(ref='refs/heads/master'))
                )
        self.assertGreaterEqual(len(lanes), 32)

    def test_guard_requires_exact_head_base_and_live_refs_for_both_branches(
        self,
    ):
        for branch in ['master', 'main']:
            with self.subTest(branch=branch):
                self.assertEqual(guard_case(branch), {'ok': True})
                for key in [
                    'QUEUE_HEAD_SHA',
                    'QUEUE_BASE_REF',
                    'QUEUE_BASE_SHA',
                ]:
                    self.assertIn('error', guard_case(branch, {key: ''}))
                self.assertIn(
                    'error', guard_case(branch, {'QUEUE_HEAD_SHA': 'd' * 40})
                )
                self.assertIn(
                    'error',
                    guard_case(branch, moved=f'heads/{branch}'),
                )
                self.assertIn(
                    'error',
                    guard_case(
                        branch,
                        moved=f'heads/gh-readonly-queue/{branch}/pr-7-candidate',
                    ),
                )
        self.assertIn('error', guard_case('release-2.18'))
        self.assertIn('error', guard_case(malformed={'eventName': 'push'}))
        self.assertIn('error', guard_case(malformed={'sha': 'd' * 40}))

    def test_queue_dispatch_does_not_select_manual_shards_or_test_filters(self):
        for name in [
            'api-server-tests.yml',
            'e2e-test.yml',
            'compiler-tests.yml',
        ]:
            for job in workflow(name)['jobs'].values():
                for key in ('test_shard', 'test_label'):
                    expression = (
                        job.get('strategy', {}).get('matrix', {}).get(key)
                    )
                    if (
                        not isinstance(expression, str)
                        or 'fromJSON' not in expression
                    ):
                        continue
                    expression = expression.removeprefix(
                        '${{ fromJSON('
                    ).removesuffix(') }}')
                    native = evaluate(expression, context('merge_group'))
                    self.assertEqual(evaluate(expression, context()), native)
                    self.assertEqual(
                        evaluate(expression, context(inputs={})), native
                    )
                    self.assertNotEqual(
                        evaluate(expression, context(ref='refs/heads/master')),
                        native,
                    )
                for step in job.get('steps', []):
                    # Evaluate the actual shell branch condition, including an
                    # adversarial narrow test_label on a queue dispatch.
                    for expression in re.findall(
                        r'if \[ "\$\{\{ (.*?) \}\}" == "true" \]; then',
                        step.get('run', ''),
                    ):
                        if 'workflow_dispatch' not in expression:
                            continue
                        self.assertFalse(evaluate(expression, context()))
                        self.assertFalse(
                            evaluate(expression, context(inputs={}))
                        )
                        self.assertTrue(
                            evaluate(
                                expression, context(ref='refs/heads/master')
                            )
                        )

    def test_queue_dispatch_keeps_api_migration_assertions_and_fixture_coverage(
        self,
    ):
        definition = workflow('api-server-tests.yml')
        assertions = []
        for job in definition['jobs'].values():
            for step in job.get('steps', []):
                options = step.get('with', {})
                if 'required_migration_specs' in options:
                    assertions.append(options['required_migration_specs'])
                if 'prepare_cache_fixtures' in options:
                    assertions.append(options['prepare_cache_fixtures'])
        self.assertGreaterEqual(len(assertions), 2)
        for expression in assertions:
            self.assertTrue(evaluate(expression, context()))
            self.assertTrue(evaluate(expression, context(inputs={})))

    def test_queue_dispatch_cannot_publish_tools_images(self):
        jobs = workflow('build-tools-images.yml')['jobs']
        for branch in ['master', 'main', 'release-2.18']:
            queue = QUEUE.replace('/master/', f'/{branch}/')
            for inputs in [{}, context()['inputs']]:
                with self.subTest(branch=branch, inputs=inputs):
                    ctx = context(ref=queue, inputs=inputs)
                    self.assertTrue(
                        evaluate(jobs['build-tools-check']['if'], ctx)
                    )
                    for name in ['build-tools', 'publish-tools']:
                        self.assertFalse(
                            evaluate(
                                jobs[name]['if'],
                                ctx,
                                {'cancelled': lambda: False},
                            )
                        )

    def test_dispatched_queue_diff_checks_use_verified_base(self):
        steps = workflow('frontend.yml')['jobs']['frontend-checks']['steps']
        drift = next(
            s
            for s in steps
            if s.get('name') == 'Check for toolchain version drift'
        )
        self.assertTrue(evaluate(drift['if'], context()))
        self.assertEqual(evaluate(drift['env']['BASE_SHA'], context()), BASE)
        for job in workflow('third-party-image-upgrades.yml')['jobs'].values():
            for step in job.get('steps', []):
                if step.get('with', {}).get('path') == 'baseline':
                    self.assertEqual(
                        evaluate(step['with']['ref'], context()), BASE
                    )


if __name__ == '__main__':
    unittest.main()

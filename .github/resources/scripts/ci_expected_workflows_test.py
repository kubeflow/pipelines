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
"""Execute the production workflow coverage verifier against API fixtures."""

import ast
import json
from pathlib import Path
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/ci_expected_workflows.js'


def node(body):
    script = f'const gate = require({json.dumps(str(MODULE))});\n' + body
    result = subprocess.run(['node'],
                            input=script,
                            check=True,
                            capture_output=True,
                            text=True,
                            cwd=ROOT)
    return json.loads(result.stdout)


def verify(runs=None,
           files=None,
           changed_files=1,
           trigger=None,
           fresh_after=None,
           workflow_paths=None,
           registration_started_at=None,
           now=0):
    if runs is None:
        runs = [good_run()]
    if files is None:
        files = [{'filename': 'frontend/package.json'}]
    if trigger is None:
        trigger = {'branches': ['master'], 'paths': ['frontend/**']}
    if workflow_paths is None:
        workflow_paths = ['.github/workflows/frontend.yml']
    fixture = {
        'runs': runs,
        'files': files,
        'pullRequest': {
            'number': 7,
            'changed_files': changed_files,
            'head': {
                'sha': 'head',
                'ref': 'dependency',
                'repo': {
                    'full_name': 'owner/repo'
                }
            },
            'base': {
                'ref': 'master',
                'sha': 'base'
            },
        },
        'inventory': {
            'version':
                1,
            'workflows': [{
                'path': path,
                'header_sha256': 'header',
                'pull_request': trigger,
            } for path in workflow_paths]
        },
        'workflowFiles': [{
            'path': path,
            'header_sha256': 'header',
        } for path in workflow_paths],
        'freshAfter': fresh_after,
        'registrationStartedAt': registration_started_at,
        'now': now,
    }
    return node(f'''
const fixture = {json.dumps(fixture)};
const requests = [];
const github = {{
  rest: {{pulls: {{listFiles: 'files'}}, actions: {{listWorkflowRunsForRepo: 'runs'}}}},
  paginate: async (route, options) => {{
    requests.push({{route, options}});
    return fixture[route];
  }},
}};
gate.verifyExpectedWorkflows({{...fixture, github, owner: 'owner', repo: 'repo'}})
  .then(result => console.log(JSON.stringify({{...result, requests}})))
  .catch(error => console.log(JSON.stringify({{error: error.message}})));
''')


def good_run(**overrides):
    return {
        'id': 100,
        'path': '.github/workflows/frontend.yml',
        'run_attempt': 1,
        'event': 'pull_request',
        'head_sha': 'head',
        'head_branch': 'dependency',
        'head_repository': {
            'full_name': 'owner/repo'
        },
        'status': 'completed',
        'conclusion': 'success',
        'created_at': '2026-09-07T12:00:00Z',
        'run_started_at': '2026-09-07T12:00:00Z',
        'pull_requests': [],
        **overrides,
    }


class ExpectedWorkflowsTest(unittest.TestCase):

    def test_completion_selector_covers_every_pr_workflow(self):
        inventory = json.loads(
            (ROOT / '.github/resources/ci-workflow-inventory.json').read_text())
        expected = [
            workflow['name']
            for workflow in inventory['workflows']
            if workflow['pull_request'] is not None
        ]
        workflow = (ROOT / '.github/workflows/ci-checks.yml').read_text()
        selector = workflow.split('  workflow_run:\n',
                                  1)[1].split('    workflows:\n', 1)[1]
        names = []
        for line in selector.splitlines():
            if line.startswith('      - '):
                names.append(ast.literal_eval(line[len('      - '):]))
            elif line.strip() and not line.lstrip().startswith('#'):
                break
        self.assertCountEqual(names, expected)

    def test_inventory_matches_every_workflow_trigger_header(self):
        result = node('''
const loaded = gate.loadLocalInventory(process.cwd());
console.log(JSON.stringify(loaded.inventory.workflows.length));
''')
        self.assertGreater(result, 30)

    def test_ordered_exclusions_and_zero_directory_glob(self):
        result = node('''
const patterns = ['**/*.py', '!sdk/**', 'sdk/keep.py'];
console.log(JSON.stringify(['root.py', 'a/b.py', 'sdk/no.py', 'sdk/keep.py']
  .map(path => gate.matchesPatterns(path, patterns))));
''')
        self.assertEqual(result, [True, True, False, True])

    def test_unknown_syntax_and_trigger_keys_fail_closed(self):
        for trigger in ({
                'paths': ['frontend/[ab].js']
        }, {
                'types': ['labeled']
        }, {
                'paths': ['frontend/**'],
                'paths-ignore': ['docs/**']
        }):
            with self.subTest(trigger=trigger):
                self.assertIn('error', verify(trigger=trigger))

    def test_missing_workflow_cannot_be_green(self):
        result = verify(runs=[])
        self.assertFalse(result['passed'])
        self.assertEqual(result['state'], 'pending')
        self.assertEqual(result['missing'], ['.github/workflows/frontend.yml'])
        self.assertIn('has not registered', result['reasons'][0])

    def test_missing_workflow_registration_grace_expires(self):
        for now, expected in [(15 * 60 * 1000 - 1, 'pending'),
                              (15 * 60 * 1000, 'failure')]:
            with self.subTest(now=now):
                result = verify(
                    runs=[],
                    registration_started_at='1970-01-01T00:00:00Z',
                    now=now)
                self.assertEqual(result['state'], expected)
                self.assertFalse(result['passed'])
                self.assertEqual(result['missing'],
                                 ['.github/workflows/frontend.yml'])
                if expected == 'failure':
                    self.assertIn('after 15 minutes', result['reasons'][0])
        result = verify(
            registration_started_at='1970-01-01T00:00:00Z', now=15 * 60 * 1000)
        self.assertEqual(result['state'], 'success')
        self.assertTrue(result['passed'])
        self.assertEqual(result['missing'], [])

    def test_invalid_registration_timestamps_fail_closed(self):
        for timestamp in ['', 'invalid', 0, [], {}]:
            with self.subTest(timestamp=timestamp):
                self.assertIn('error',
                              verify(registration_started_at=timestamp))
        for now in [None, 'invalid']:
            with self.subTest(now=now):
                self.assertIn('error', verify(now=now))

    def test_active_workflows_are_pending(self):
        for status in [
                'queued', 'requested', 'waiting', 'in_progress', 'pending'
        ]:
            with self.subTest(status=status):
                result = verify(runs=[good_run(status=status, conclusion=None)])
                self.assertEqual(result['state'], 'pending')
                self.assertFalse(result['passed'])
                self.assertEqual(result['missing'], [])

    def test_malformed_or_unsuccessful_runs_fail_closed(self):
        for status, conclusion in [
            ('completed', 'cancelled'), ('completed', 'failure'),
            ('completed', 'timed_out'), ('completed', 'action_required'),
            ('completed', 'stale'), ('completed', 'skipped'),
            ('completed', 'neutral'), ('completed', None),
            ('completed', 'unknown'), ('completed', ['success']),
            ('in_progress', 'success'), ('unknown', None), (None, None)
        ]:
            with self.subTest(status=status, conclusion=conclusion):
                result = verify(
                    runs=[good_run(status=status, conclusion=conclusion)])
                self.assertEqual(result['state'], 'failure')
                self.assertFalse(result['passed'])

    def test_matching_run_requires_identity_and_attempt_timestamps(self):
        for field in ['id', 'run_attempt', 'created_at', 'run_started_at']:
            with self.subTest(field=field, missing=True):
                run = good_run()
                del run[field]
                result = verify(runs=[run])
                self.assertEqual(result['state'], 'failure')
                self.assertFalse(result['passed'])
        invalid_values = {
            'id': [None, 0, -1, 1.5, '100', True, 9007199254740992],
            'run_attempt': [None, 0, -1, 1.5, '1', True, 9007199254740992],
            'created_at': [None, 0, True, '', 'invalid', [], {}],
            'run_started_at': [None, 0, True, '', 'invalid', [], {}],
        }
        for field, values in invalid_values.items():
            for value in values:
                with self.subTest(field=field, value=value):
                    result = verify(runs=[good_run(**{field: value})])
                    self.assertEqual(result['state'], 'failure')
                    self.assertFalse(result['passed'])

    def test_malformed_competing_run_cannot_fall_back_to_success(self):
        for field in ['id', 'run_attempt', 'created_at', 'run_started_at']:
            malformed = good_run(id=99, conclusion='failure')
            del malformed[field]
            for runs in [[malformed, good_run()], [good_run(), malformed]]:
                with self.subTest(field=field, runs=runs):
                    result = verify(runs=runs)
                    self.assertEqual(result['state'], 'failure')
                    self.assertFalse(result['passed'])

    def test_only_unstarted_first_attempt_can_have_null_start_time(self):
        result = verify(runs=[
            good_run(status='queued', conclusion=None, run_started_at=None)
        ])
        self.assertEqual(result['state'], 'pending')
        self.assertFalse(result['passed'])
        for overrides in [{'run_attempt': 2}, {'status': 'in_progress'}]:
            with self.subTest(overrides=overrides):
                result = verify(runs=[
                    good_run(
                        **{
                            'status': 'queued',
                            'conclusion': None,
                            'run_started_at': None,
                            **overrides,
                        })
                ])
                self.assertEqual(result['state'], 'failure')
                self.assertFalse(result['passed'])

    def test_failure_takes_precedence_over_pending_workflow(self):
        paths = [
            '.github/workflows/frontend.yml', '.github/workflows/backend.yml'
        ]
        result = verify(
            workflow_paths=paths,
            runs=[
                good_run(status='in_progress', conclusion=None),
                good_run(path=paths[1], conclusion='failure')
            ])
        self.assertEqual(result['state'], 'failure')
        self.assertFalse(result['passed'])
        self.assertIn('completed/failure', result['reasons'][0])
        self.assertIn('in_progress', result['reasons'][1])

    def test_only_latest_execution_counts(self):
        for status, conclusion in [('queued', None), ('in_progress', None),
                                   ('completed', 'cancelled'),
                                   ('completed', 'action_required'),
                                   ('completed', 'failure'),
                                   ('completed', 'skipped')]:
            with self.subTest(status=status, conclusion=conclusion):
                result = verify(runs=[
                    good_run(),
                    good_run(id=101, status=status, conclusion=conclusion)
                ])
                self.assertFalse(result['passed'])
        result = verify(runs=[good_run(conclusion='failure'), good_run(id=101)])
        self.assertTrue(result['passed'])
        self.assertEqual(result['state'], 'success')

    def test_current_attempt_cannot_reuse_earlier_success(self):
        result = verify(runs=[
            good_run(run_attempt=2, status='in_progress', conclusion=None)
        ])
        self.assertFalse(result['passed'])
        self.assertEqual(result['state'], 'pending')
        result = verify(runs=[
            good_run(id=101),
            good_run(
                id=100,
                run_attempt=2,
                status='completed',
                conclusion='cancelled',
                run_started_at='2026-09-07T14:00:00Z')
        ])
        self.assertFalse(result['passed'])
        self.assertEqual(result['state'], 'failure')

    def test_retargeted_base_needs_new_execution_not_old_run_rerun(self):
        cutoff = '2026-09-07T13:00:00Z'
        result = verify(
            runs=[
                good_run(run_attempt=2, run_started_at='2026-09-07T14:00:00Z')
            ],
            fresh_after=cutoff)
        self.assertFalse(result['passed'])
        result = verify(
            runs=[good_run(created_at='2026-09-07T14:00:00Z')],
            fresh_after=cutoff)
        self.assertTrue(result['passed'])

    def test_wrong_head_repo_event_or_base_cannot_supply_evidence(self):
        for overrides in ({
                'head_sha': 'old'
        }, {
                'path': '.github/workflows/unrelated.yml'
        }, {
                'event': 'push'
        }, {
                'head_branch': 'another'
        }, {
                'head_repository': {
                    'full_name': 'attacker/repo'
                }
        }, {
                'pull_requests': [{
                    'number': 7,
                    'base': {
                        'ref': 'release-2.17'
                    }
                }]
        }):
            with self.subTest(overrides=overrides):
                self.assertFalse(verify(runs=[good_run(**overrides)])['passed'])

    def test_active_workflow_cannot_hide_stale_base_evidence(self):
        for options in [
            {
                'fresh_after': '2026-09-07T13:00:00Z'
            },
            {
                'runs': [
                    good_run(
                        status='in_progress',
                        conclusion=None,
                        pull_requests=[{
                            'number': 7,
                            'base': {
                                'ref': 'release'
                            }
                        }])
                ]
            },
        ]:
            with self.subTest(options=options):
                result = verify(
                    **{
                        'runs':
                            [good_run(status='in_progress', conclusion=None)],
                        **options,
                    })
                self.assertEqual(result['state'], 'failure')
                self.assertFalse(result['passed'])

    def test_renamed_source_still_requires_its_workflow(self):
        result = verify(files=[{
            'filename': 'docs/package.json',
            'previous_filename': 'frontend/package.json'
        }])
        self.assertTrue(result['passed'])

    def test_file_and_run_history_truncation_fail_closed(self):
        self.assertIn('error', verify(changed_files=2))
        self.assertIn('error', verify(runs=[good_run()] * 1000))

    def test_api_reads_are_paginated_and_head_scoped(self):
        result = verify()
        self.assertTrue(result['passed'])
        self.assertEqual(result['requests'][1]['options']['head_sha'], 'head')
        self.assertEqual(result['requests'][1]['options']['event'],
                         'pull_request')
        self.assertEqual(result['requests'][0]['options']['per_page'], 100)

    def test_all_expected_lanes_share_one_paginated_run_snapshot(self):
        result = node(f'''
const inventory = gate.loadLocalInventory(process.cwd());
const example = {json.dumps(good_run())};
let reads = 0;
const github = {{
  rest: {{pulls: {{listFiles: 'files'}}, actions: {{listWorkflowRunsForRepo: 'runs'}}}},
  paginate: async route => {{
    if (route === 'files') return [{{filename: 'frontend/package.json'}}];
    reads++;
    return inventory.inventory.workflows.map(workflow => ({{...example, path: workflow.path}}));
  }},
}};
gate.verifyExpectedWorkflows({{github, owner: 'owner', repo: 'repo', ...inventory,
  pullRequest: {{number: 7, changed_files: 1, base: {{ref: 'master'}},
    head: {{sha: 'head', ref: 'dependency', repo: {{full_name: 'owner/repo'}}}}}},
}}).then(result => console.log(JSON.stringify({{...result, reads}})));
''')
        self.assertTrue(result['passed'])
        self.assertGreater(len(result['expected']), 1)
        self.assertEqual(result['reads'], 1)

    def test_no_expected_workflow_is_not_vacuous_success(self):
        result = verify(files=[{'filename': 'README.md'}])
        self.assertFalse(result['passed'])
        self.assertEqual(result['state'], 'failure')

    def test_real_inventory_uses_current_frontend_coverage(self):
        result = node('''
const {inventory} = gate.loadLocalInventory(process.cwd());
console.log(JSON.stringify(inventory.workflows.filter(workflow =>
  workflow.pull_request !== null && gate.applicable(workflow.pull_request,
    'master', ['frontend/package.json']))
  .map(workflow => workflow.path)));
''')
        self.assertIn('.github/workflows/e2e-test-frontend.yml', result)
        self.assertIn('.github/workflows/pre-commit.yml', result)

    def test_added_or_modified_workflow_invalidates_inventory(self):
        result = node('''
const inventory = {version: 1, workflows: []};
try {
  gate.validateInventory(inventory, [{path: '.github/workflows/new.yml', header_sha256: 'new'}]);
  console.log(JSON.stringify(false));
} catch (error) { console.log(JSON.stringify(true)); }
''')
        self.assertTrue(result)

    def test_dependency_bumps_do_not_change_trigger_header(self):
        result = node('''
const before = 'name: CI\\non:\\n  pull_request:\\npermissions:\\n  contents: read\\njobs:\\n  test:\\n    uses: actions/example@v1\\n';
const after = before.replace('@v1', '@v2');
console.log(JSON.stringify(gate.triggerHeader(before) === gate.triggerHeader(after)));
''')
        self.assertTrue(result)

    def test_trigger_changes_change_header_and_key_order_is_supported(self):
        result = node('''
const before = 'on:\\n  pull_request:\\n    paths: [frontend/**]\\npermissions: {}\\nname: Frontend\\njobs: {}\\n';
const after = before.replace('frontend/**', 'backend/**');
const header = gate.triggerHeader(before);
console.log(JSON.stringify([header.includes('name: Frontend'),
  header !== gate.triggerHeader(after), !header.includes('permissions')]));
''')
        self.assertEqual(result, [True, True, True])


if __name__ == '__main__':
    unittest.main()

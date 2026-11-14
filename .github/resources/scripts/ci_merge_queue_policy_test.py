#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Exercise the queue activation policy and PR publisher enforcement
boundary."""

import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import ci_pass_eligibility_test as publisher

ROOT = Path(__file__).resolve().parents[3]
HELPER = ROOT / '.github/resources/scripts/merge_queue_policy.js'


class MergeQueuePolicyTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        (self.root / '.github').mkdir()
        (self.root / '.github/resources').symlink_to(ROOT / '.github/resources')
        self.policy(['master', 'main'])

    def policy(self, branches):
        (self.root / '.github/merge-queue-policy.json').write_text(
            json.dumps({'version': 1, 'enabled_branches': branches})
        )

    def exercise(self, **options):
        with mock.patch.object(publisher, 'ROOT', self.root):
            return publisher.exercise(
                {'queueRules': [{'type': 'merge_queue'}], **options},
                policy_root=self.root,
            )

    def node(self, operation, **options):
        script = """
const {queueEnabled, loadPolicy} = require(process.argv[1]);
const gate = require(process.argv[2]);
const root = process.argv[3], options = JSON.parse(process.argv[4]);
const context = {repo: {owner: 'kubeflow', repo: 'pipelines'}, eventName: 'push',
  ref: `refs/heads/${options.branch || 'master'}`};
const reads = [];
const github = {paginate: async (route, parameters) => {
  reads.push({route, parameters});
  if (options.apiError) throw new Error('API unavailable');
  return options.rules || [];
}};
(async () => {
  let result, error;
  try {
    if (options.operation === 'load') result = loadPolicy(root);
    if (options.operation === 'enabled') result = await queueEnabled({github, context,
      branch: options.branch || 'master', root});
    if (options.operation === 'revoke') result = await gate.revokeStaleGreens({github, context,
      tip: 'b'.repeat(40), root});
  } catch (e) {error = e.message;}
  console.log(JSON.stringify({result, error, reads}));
})();
"""
        result = subprocess.run(
            [
                'node',
                '-e',
                script,
                str(HELPER),
                str(publisher.MODULE),
                str(self.root),
                json.dumps({'operation': operation, **options}),
            ],
            text=True,
            check=True,
            capture_output=True,
        )
        return json.loads(result.stdout)

    def test_checked_in_policy_enables_only_master(self):
        policy = json.loads(
            (ROOT / '.github/merge-queue-policy.json').read_text()
        )
        self.assertEqual(policy, {'version': 1, 'enabled_branches': ['master']})

    def test_activation_payload_preserves_required_checks_and_queue_scope(self):
        ruleset = json.loads(
            (ROOT / 'docs/agents/merge-queue-ruleset.json').read_text()
        )
        self.assertEqual(ruleset['enforcement'], 'active')
        self.assertEqual(ruleset['bypass_actors'], [])
        self.assertEqual(
            ruleset['conditions']['ref_name'],
            {'include': ['refs/heads/master'], 'exclude': []},
        )
        rules = {rule['type']: rule['parameters'] for rule in ruleset['rules']}
        queue = rules['merge_queue']
        self.assertEqual(queue['merge_method'], 'SQUASH')
        self.assertEqual(queue['grouping_strategy'], 'ALLGREEN')
        self.assertEqual(queue['max_entries_to_build'], 2)
        self.assertEqual(queue['max_entries_to_merge'], 2)
        checks = rules['required_status_checks']
        self.assertFalse(checks['strict_required_status_checks_policy'])
        self.assertFalse(checks['do_not_enforce_on_create'])
        self.assertEqual(
            checks['required_status_checks'],
            [
                {'context': 'ci-passed', 'integration_id': 15368},
                {'context': 'DCO', 'integration_id': 1861},
            ],
        )
        classic = json.loads(
            (ROOT / 'docs/agents/merge-queue-required-checks.json').read_text()
        )
        self.assertEqual(
            classic,
            {
                'strict': False,
                'checks': [
                    {'context': 'DCO', 'app_id': 1861},
                    {'context': 'ci-passed', 'app_id': 15368},
                ],
            },
        )

    def test_master_only_policy_requires_enforcement_and_leaves_main_disabled(
        self,
    ):
        self.policy(['master'])
        self.assertIn('enforced merge_queue', self.node('enabled')['error'])
        self.assertTrue(
            self.node('enabled', rules=[{'type': 'merge_queue'}])['result']
        )
        for branch in ['main', 'release-2.18']:
            result = self.node('enabled', branch=branch)
            self.assertFalse(result['result'])
            self.assertEqual(result['reads'], [])

    def test_default_policy_preserves_old_base_freshness(self):
        self.policy([])
        result = self.exercise(baseArrivedAt='2026-09-07T12:00:00Z')
        self.assertEqual(result['status'], 'pending', result)
        self.assertFalse(
            [call for call in result['calls'] if call[0] == 'queue-rules']
        )
        self.assertEqual(self.node('enabled')['result'], False)

    def test_enforced_queue_admits_tested_head_without_fresh_base_run(self):
        for branch in ['master', 'main']:
            with self.subTest(branch=branch):
                result = self.exercise(
                    baseArrivedAt='2026-09-07T12:00:00Z',
                    pr={
                        'base': {
                            'ref': branch,
                            'sha': 'b' * 40,
                            'repo': {'full_name': 'kubeflow/pipelines'},
                        }
                    },
                )
                self.assertEqual(result['status'], 'success', result)
                self.assertTrue(
                    result['descriptions'][-1].startswith(
                        f'Queue admission {branch} CI passed;'
                    )
                )
                self.assertEqual(result['requestCounts']['associated'], 0)
                self.assertEqual(
                    [
                        call
                        for call in result['calls']
                        if call[0] == 'queue-rules'
                    ],
                    [['queue-rules', branch]] * 3,
                )
                self.assertEqual(result['requestCounts']['inventory'], 3)

    def test_success_binds_exact_workflow_attempts_and_rejects_later_successful_rerun(
        self,
    ):
        result = self.exercise()
        self.assertEqual(result['status'], 'success', result)
        self.assertRegex(
            result['targetUrls'][-1], r'#queue-attempts=[0-9a-f]{64}$'
        )
        updated = self.exercise(
            runPatch={'run_attempt': 2},
            beforeFinalStatus='success',
            beforeFinalDescription=result['descriptions'][-1],
            beforeFinalTarget=result['targetUrls'][-1],
        )
        self.assertIn(['status', 'success', 'head'], updated['calls'])
        self.assertNotEqual(updated['targetUrls'][-1], result['targetUrls'][-1])
        unchanged = self.exercise(
            beforeFinalStatus='success',
            beforeFinalDescription=result['descriptions'][-1],
            beforeFinalTarget=result['targetUrls'][-1].replace('/99#', '/98#'),
        )
        self.assertNotIn(['status', 'success', 'head'], unchanged['calls'])
        changed = self.exercise(drift='successful-rerun')
        self.assertIn(['status', 'success', 'head'], changed['calls'])
        self.assertEqual(changed['status'], 'pending', changed)

    def test_missing_or_unreadable_enforcement_never_passes(self):
        for options in [{'queueRules': []}, {'queueRulesFailure': True}]:
            with self.subTest(options=options):
                result = self.exercise(**options)
                self.assertEqual(result['status'], 'failure', result)
                self.assertNotIn(['status', 'success', 'head'], result['calls'])

    def test_rule_removed_after_success_revokes_green(self):
        result = self.exercise(queueRemovedAfterSuccess=True)
        self.assertIn(['status', 'success', 'head'], result['calls'])
        self.assertEqual(result['status'], 'failure', result)

    def test_queue_admission_still_requires_current_ci_and_retarget_evidence(
        self,
    ):
        for options in [
            {'retarget': True},
            {'missing': True},
            {'conclusion': 'failure'},
            {'runStatus': 'in_progress'},
            {'checkConclusion': 'failure'},
            {'pr': {'labels': [{'name': 'needs-ok-to-test'}]}},
            {'drift': 'rerun'},
            {'drift': 'external-failure'},
        ]:
            with self.subTest(options=options):
                self.assertNotEqual(
                    self.exercise(**options)['status'], 'success'
                )

    def test_release_branches_keep_existing_behavior(self):
        result = self.exercise(
            pr={
                'base': {
                    'ref': 'release-2.18',
                    'sha': 'b' * 40,
                    'repo': {'full_name': 'kubeflow/pipelines'},
                }
            }
        )
        self.assertEqual(result['status'], 'success', result)
        self.assertFalse(
            [call for call in result['calls'] if call[0] == 'queue-rules']
        )
        self.assertTrue(
            result['descriptions'][-1].startswith(
                'Expected CI and all checks passed;'
            )
        )

    def test_recovery_revisits_queue_green_when_policy_or_enforcement_changes(
        self,
    ):
        current = self.exercise(verifyRecovery=True)
        self.assertEqual(current['recoveryAfter'], [])
        options = {
            'verifyRecovery': True,
            'initialStatus': 'success',
            'initialDescription': current['descriptions'][-1],
        }
        missing_rule = self.exercise(**options, queueRules=[])
        self.assertEqual(len(missing_rule['recoveryBefore']), 1)
        self.assertEqual(missing_rule['status'], 'failure')
        self.policy([])
        disabled = self.exercise(
            **options, baseArrivedAt='2026-09-07T12:00:00Z'
        )
        self.assertEqual(len(disabled['recoveryBefore']), 1)
        self.assertEqual(disabled['status'], 'pending')

    def test_invalid_policy_fails_closed(self):
        for branches in [['release-2.18'], ['master', 'master'], '*', [None]]:
            with self.subTest(branches=branches):
                self.policy(branches)
                self.assertIn(
                    'Invalid merge queue policy', self.node('load')['error']
                )

    def test_revoker_skips_only_with_enforced_queue(self):
        for branch in ['master', 'main']:
            with self.subTest(branch=branch):
                result = self.node(
                    'revoke', branch=branch, rules=[{'type': 'merge_queue'}]
                )
                self.assertEqual(result['result'], 0)
                self.assertEqual(len(result['reads']), 1)
                self.assertIn(
                    'enforced', self.node('revoke', branch=branch)['error']
                )
                self.assertIn(
                    'API unavailable',
                    self.node('revoke', branch=branch, apiError=True)['error'],
                )

    def test_effective_rule_lookup_is_bound_to_target_branch(self):
        result = self.node(
            'enabled', branch='main', rules=[{'type': 'merge_queue'}]
        )
        self.assertTrue(result['result'])
        self.assertEqual(
            result['reads'],
            [
                {
                    'route': 'GET /repos/{owner}/{repo}/rules/branches/{branch}',
                    'parameters': {
                        'owner': 'kubeflow',
                        'repo': 'pipelines',
                        'branch': 'main',
                        'per_page': 100,
                    },
                }
            ],
        )


if __name__ == '__main__':
    unittest.main()

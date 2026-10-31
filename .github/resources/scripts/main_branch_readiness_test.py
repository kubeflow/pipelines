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

import copy
import io
import json
import subprocess
import unittest
from unittest import mock

import main_branch_readiness as readiness

PREFIX = 'repos/kubeflow/pipelines'
SHA = 'a' * 40


def fixture():
    rules = {
        14287837: [
            {
                'type': 'required_status_checks',
                'parameters': {
                    'required_status_checks': [
                        {'context': 'DCO', 'integration_id': 1861}
                    ]
                },
            }
        ],
        14287367: [{'type': 'deletion'}, {'type': 'non_fast_forward'}],
        14287485: [{'type': 'pull_request'}],
    }
    responses = {
        PREFIX: {'default_branch': 'master'},
        f'{PREFIX}/git/ref/heads/master': {'object': {'sha': SHA}},
        f'{PREFIX}/branches?per_page=100': [
            [{'name': 'master'}],
            [{'name': 'release-2.18'}],
        ],
        f'{PREFIX}/branches/master/protection': {
            'required_status_checks': {
                'strict': True,
                'checks': [
                    {'context': 'DCO', 'app_id': 1861},
                    {'context': 'ci-passed', 'app_id': 15368},
                ],
            },
            'allow_force_pushes': {'enabled': False},
            'allow_deletions': {'enabled': False},
            'required_linear_history': {'enabled': True},
            'required_conversation_resolution': {'enabled': True},
        },
        f'{PREFIX}/environments/pypi': {
            'deployment_branch_policy': {
                'protected_branches': False,
                'custom_branch_policies': True,
            },
            'protection_rules': [{'type': 'branch_policy'}],
        },
        f'{PREFIX}/environments/pypi/deployment-branch-policies?per_page=100': [
            {
                'branch_policies': [
                    {'type': 'branch', 'name': 'master'},
                    {'type': 'branch', 'name': 'main'},
                ]
            },
            {
                'branch_policies': [
                    {'type': 'branch', 'name': name}
                    for name in (
                        'release-*',
                        'kfp-kubernetes-*',
                        'kfp-pipeline-spec-*',
                        'kfp-server-api-*',
                    )
                ]
                + [{'type': 'tag', 'name': 'sdk-*'}]
            },
        ],
        f'{PREFIX}/actions/oidc/customization/sub': {'use_default': True},
    }
    for repo, number in readiness.PREPARATION:
        responses[f'repos/{repo}/pulls/{number}'] = {
            'merged': True,
            'base': {'ref': 'master'},
            'merge_commit_sha': 'b' * 40,
        }
    for rule_id, content in rules.items():
        responses[f'{PREFIX}/rulesets/{rule_id}'] = {
            'enforcement': 'active',
            'target': 'branch',
            'conditions': {
                'ref_name': {
                    'include': [
                        'refs/heads/main',
                        'refs/heads/master',
                        '~DEFAULT_BRANCH',
                    ],
                    'exclude': [],
                }
            },
            'rules': content,
        }
    config = {
        'config_version_sha': 'c' * 40,
        'tide': {
            'context_options': {
                'orgs': {
                    'kubeflow': {
                        'repos': {
                            'pipelines': {
                                'branches': {
                                    branch: {'required-contexts': ['ci-passed']}
                                    for branch in ('master', 'main')
                                },
                            }
                        }
                    }
                }
            },
            'queries': [
                {
                    'orgs': ['kubeflow'],
                    'repos': ['kubeflow/pipelines'],
                    'labels': ['approved', 'lgtm'],
                    'missingLabels': ['do-not-merge/hold'],
                },
                {
                    'repos': ['kubeflow/pipelines'],
                    'author': 'dependabot[bot]',
                    'includedBranches': ['master', 'main'],
                    'missingLabels': ['do-not-merge/hold', 'needs-ok-to-test'],
                },
            ],
        },
    }
    return responses, config


class ReadinessTest(unittest.TestCase):
    def setUp(self):
        self.responses, self.config = fixture()
        self.calls = []

    def get(self, path, paginated=False):
        self.calls.append((path, paginated))
        value = self.responses[path]
        if isinstance(value, Exception):
            raise value
        return copy.deepcopy(value)

    def collect(self, **kwargs):
        return readiness.collect(
            get=self.get, fetch_config=lambda: (self.config, 'd' * 64), **kwargs
        )

    def assert_blocked(self, report, name, status='BLOCKED'):
        self.assertFalse(report['automated_checks_passed'])
        self.assertEqual(report['decision'], 'NO-GO')
        self.assertEqual(
            next(c for c in report['checks'] if c['name'] == name)['status'],
            status,
        )

    def test_passing_automated_checks_still_require_owner_review(self):
        report = self.collect(expected_sha=SHA)
        self.assertTrue(report['automated_checks_passed'])
        self.assertEqual(report['decision'], 'OWNER_REVIEW_REQUIRED')
        self.assertGreaterEqual(len(report['owner_gates']), 6)
        self.assertEqual(report['evidence']['master_sha'], SHA)
        self.assertIn(
            'not rename or bootstrap authorization', report['limitations']
        )
        for endpoint in (
            'branches?per_page=100',
            'environments/pypi/deployment-branch-policies?per_page=100',
        ):
            self.assertIn((f'{PREFIX}/{endpoint}', True), self.calls)

    def test_each_open_preparation_pr_blocks(self):
        for repo, number in readiness.PREPARATION:
            with self.subTest(repo=repo):
                self.setUp()
                self.responses[f'repos/{repo}/pulls/{number}']['merged'] = False
                self.assert_blocked(self.collect(), f'{repo}#{number} merged')

    def test_permission_failure_is_unknown_and_does_not_stop_other_checks(self):
        self.responses[f'{PREFIX}/branches/master/protection'] = (
            readiness.EvidenceError('secret-token hook-url')
        )
        report = self.collect()
        self.assert_blocked(report, 'classic protection baseline', 'UNKNOWN')
        self.assertEqual(report['checks'][-1]['status'], 'PASS')
        self.assertNotIn('secret-token', json.dumps(report))
        self.assertNotIn('hook-url', json.dumps(report))

    def test_main_on_later_page_blocks(self):
        self.responses[f'{PREFIX}/branches?per_page=100'][-1].append(
            {'name': 'main'}
        )
        self.assert_blocked(self.collect(), 'main does not already exist')

    def test_empty_branch_evidence_cannot_prove_main_absent(self):
        for value in ([], [[]]):
            with self.subTest(value=value):
                self.responses[f'{PREFIX}/branches?per_page=100'] = value
                report = self.collect()
                self.assertFalse(report['automated_checks_passed'])
                self.assertNotEqual(
                    next(
                        c
                        for c in report['checks']
                        if c['name'] == 'main does not already exist'
                    )['status'],
                    'PASS',
                )

    def test_expected_sha_mismatch_blocks(self):
        self.assert_blocked(
            self.collect(expected_sha='e' * 40), 'master matches operator SHA'
        )

    def test_sha_drift_during_collection_blocks(self):
        original = self.get
        count = 0

        def changing_get(path, paginated=False):
            nonlocal count
            result = original(path, paginated)
            if path == f'{PREFIX}/git/ref/heads/master':
                count += 1
                if count == 2:
                    result['object']['sha'] = 'e' * 40
            return result

        self.get = changing_get
        self.assert_blocked(
            self.collect(), 'master SHA unchanged during collection'
        )

    def test_default_branch_drift_blocks(self):
        original = self.get
        count = 0

        def changing_get(path, paginated=False):
            nonlocal count
            result = original(path, paginated)
            if path == PREFIX:
                count += 1
                if count == 2:
                    result['default_branch'] = 'main'
            return result

        self.get = changing_get
        self.assert_blocked(
            self.collect(), 'default branch unchanged during collection'
        )

    def test_malformed_tide_is_unknown(self):
        self.config['tide'] = None
        self.assert_blocked(
            self.collect(),
            'deployed Tide accepts and gates both branches',
            'UNKNOWN',
        )

    def test_master_only_tide_query_blocks(self):
        self.config['tide']['queries'][1]['includedBranches'] = ['master']
        self.assert_blocked(
            self.collect(), 'deployed Tide accepts and gates both branches'
        )

    def test_missing_main_required_context_blocks(self):
        self.config['tide']['context_options']['orgs']['kubeflow']['repos'][
            'pipelines'
        ]['branches']['main']['required-contexts'] = []
        self.assert_blocked(
            self.collect(), 'deployed Tide accepts and gates both branches'
        )

    def test_optional_ci_passed_override_blocks(self):
        self.config['tide']['context_options']['optional-contexts'] = ['ci-.*']
        self.assert_blocked(
            self.collect(), 'deployed Tide accepts and gates both branches'
        )

    def test_additional_unreviewed_query_blocks(self):
        self.config['tide']['queries'].append(
            {
                'repos': ['kubeflow/pipelines'],
                'missingLabels': ['do-not-merge/hold'],
            }
        )
        self.assert_blocked(
            self.collect(), 'deployed Tide accepts and gates both branches'
        )

    def test_unexpected_bot_query_blocks(self):
        self.config['tide']['queries'].append(
            {
                'repos': ['kubeflow/pipelines'],
                'author': 'another-bot',
                'labels': ['approved', 'lgtm'],
                'missingLabels': ['do-not-merge/hold'],
            }
        )
        self.assert_blocked(
            self.collect(), 'deployed Tide accepts and gates both branches'
        )

    def test_scalar_required_contexts_are_unknown(self):
        self.config['tide']['context_options']['orgs']['kubeflow']['repos'][
            'pipelines'
        ]['branches']['main']['required-contexts'] = 'ci-passed'
        self.assert_blocked(
            self.collect(),
            'deployed Tide accepts and gates both branches',
            'UNKNOWN',
        )

    def test_scalar_optional_contexts_are_unknown(self):
        self.config['tide']['context_options']['optional-contexts'] = (
            'unrelated'
        )
        self.assert_blocked(
            self.collect(),
            'deployed Tide accepts and gates both branches',
            'UNKNOWN',
        )

    def test_scalar_query_orgs_are_unknown(self):
        self.config['tide']['queries'][0].pop('repos')
        self.config['tide']['queries'][0]['orgs'] = 'kubeflow'
        self.assert_blocked(
            self.collect(),
            'deployed Tide accepts and gates both branches',
            'UNKNOWN',
        )

    def test_partial_optional_context_pattern_blocks(self):
        self.config['tide']['context_options']['optional-contexts'] = ['passed']
        self.assert_blocked(
            self.collect(), 'deployed Tide accepts and gates both branches'
        )

    def test_hold_gate_required_for_each_applicable_query(self):
        for index in (0, 1):
            with self.subTest(index=index):
                self.setUp()
                self.config['tide']['queries'][index]['missingLabels'].remove(
                    'do-not-merge/hold'
                )
                self.assert_blocked(
                    self.collect(),
                    'deployed Tide accepts and gates both branches',
                )

    def test_dependabot_test_authorization_gate_is_preserved(self):
        self.config['tide']['queries'][1]['missingLabels'].remove(
            'needs-ok-to-test'
        )
        self.assert_blocked(
            self.collect(), 'deployed Tide accepts and gates both branches'
        )

    def test_skip_unknown_contexts_is_not_accepted(self):
        self.config['tide']['context_options']['skip-unknown-contexts'] = True
        self.assert_blocked(
            self.collect(), 'deployed Tide accepts and gates both branches'
        )

    def test_invalid_optional_context_regex_is_unknown(self):
        self.config['tide']['context_options']['optional-contexts'] = ['[']
        self.assert_blocked(
            self.collect(),
            'deployed Tide accepts and gates both branches',
            'UNKNOWN',
        )

    def test_malformed_master_ref_never_crashes_collection(self):
        for value in (
            None,
            [],
            {'object': None},
            {'object': []},
            {'object': {'sha': 123}},
            {'object': {'sha': 'short'}},
        ):
            with self.subTest(value=value):
                self.responses[f'{PREFIX}/git/ref/heads/master'] = value
                report = self.collect()
                self.assertEqual(report['decision'], 'NO-GO')
                self.assertNotIn('master_sha', report['evidence'])
                self.assertEqual(report['checks'][-1]['status'], 'PASS')

    def test_invalid_deployed_revision_is_unknown(self):
        self.config['config_version_sha'] = 'not-a-commit'
        self.assert_blocked(
            self.collect(),
            'deployed Tide accepts and gates both branches',
            'UNKNOWN',
        )

    def test_protection_check_app_identity_matters(self):
        checks = self.responses[f'{PREFIX}/branches/master/protection'][
            'required_status_checks'
        ]['checks']
        checks[1]['app_id'] = 999
        self.assert_blocked(self.collect(), 'classic protection baseline')

    def test_ruleset_exclusion_blocks(self):
        self.responses[f'{PREFIX}/rulesets/14287367']['conditions']['ref_name'][
            'exclude'
        ] = ['refs/heads/main']
        self.assert_blocked(
            self.collect(), 'ruleset 14287367 covers both branches'
        )

    def test_missing_existing_publishing_policy_blocks(self):
        policies = self.responses[
            f'{PREFIX}/environments/pypi/deployment-branch-policies?per_page=100'
        ]
        policies[1]['branch_policies'].pop()
        self.assert_blocked(
            self.collect(), 'pypi branch/tag allowlist preserved with main'
        )

    def test_unknown_api_shape_is_not_a_pass(self):
        self.responses[f'{PREFIX}/actions/oidc/customization/sub'] = {}
        self.assert_blocked(
            self.collect(), 'OIDC default subject template', 'UNKNOWN'
        )


class CommandLineTest(unittest.TestCase):
    def test_exit_code_never_implies_owner_authorization(self):
        for automated_ready, expected_code in ((False, 1), (True, 2)):
            with self.subTest(automated_ready=automated_ready):
                report = {
                    'automated_checks_passed': automated_ready,
                    'decision': 'OWNER_REVIEW_REQUIRED'
                    if automated_ready
                    else 'NO-GO',
                }
                with (
                    mock.patch.object(
                        readiness, 'collect', return_value=report
                    ),
                    mock.patch(
                        'sys.argv', ['main_branch_readiness.py', '--json']
                    ),
                    mock.patch(
                        'sys.stdout', new_callable=io.StringIO
                    ) as output,
                ):
                    self.assertEqual(readiness.main(), expected_code)
                    self.assertEqual(json.loads(output.getvalue()), report)


class ReadOnlyTransportTest(unittest.TestCase):
    @mock.patch.object(readiness.subprocess, 'run')
    def test_github_requests_are_get_and_paginated_output_is_preserved(
        self, run
    ):
        run.return_value = subprocess.CompletedProcess(
            [], 0, stdout='[[{"name":"master"}],[{"name":"main"}]]'
        )
        result = readiness.github(
            'repos/kubeflow/pipelines/branches?per_page=100', paginated=True
        )
        self.assertEqual(result[1][0]['name'], 'main')
        command = run.call_args.args[0]
        self.assertEqual(
            command[:6],
            ['gh', 'api', '--hostname', 'github.com', '--method', 'GET'],
        )
        self.assertEqual(command[-2:], ['--paginate', '--slurp'])
        self.assertEqual(run.call_args.kwargs['timeout'], 60)
        self.assertNotIn('shell', run.call_args.kwargs)

    @mock.patch.object(readiness.subprocess, 'run')
    def test_github_errors_do_not_expose_raw_output(self, run):
        run.return_value = subprocess.CompletedProcess(
            [], 1, stdout='token secret', stderr='private-hook-url'
        )
        with self.assertRaises(readiness.EvidenceError) as caught:
            readiness.github('repos/kubeflow/pipelines')
        self.assertNotIn('secret', str(caught.exception))
        self.assertNotIn('private-hook', str(caught.exception))

    @mock.patch.object(readiness.subprocess, 'run')
    def test_timeout_is_an_evidence_error(self, run):
        run.side_effect = subprocess.TimeoutExpired(['gh'], 60)
        with self.assertRaises(readiness.EvidenceError):
            readiness.github('repos/kubeflow/pipelines')

    @mock.patch.object(readiness.subprocess, 'run')
    def test_invalid_json_is_an_evidence_error(self, run):
        run.return_value = subprocess.CompletedProcess(
            [], 0, stdout='<html>not JSON</html>'
        )
        with self.assertRaises(readiness.EvidenceError):
            readiness.github('repos/kubeflow/pipelines')

    @mock.patch.object(readiness, 'urlopen')
    def test_public_configuration_uses_bounded_read_without_auth(self, urlopen):
        body = b'config_version_sha: abc\ntide: {}\n'
        urlopen.return_value.__enter__.return_value.read.return_value = body
        value, digest = readiness.deployed_config()
        self.assertEqual(value['tide'], {})
        self.assertEqual(len(digest), 64)
        urlopen.assert_called_once_with(readiness.PROW_URL, timeout=30)
        urlopen.return_value.__enter__.return_value.read.assert_called_once_with(
            8 * 1024 * 1024 + 1
        )


if __name__ == '__main__':
    unittest.main()

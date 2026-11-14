"""Candidate evidence must never become publication approval."""

import copy
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from typer.testing import CliRunner

from . import candidate
from .cli import app

SHA = 'a' * 40


class CandidateTest(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.output = Path(directory.name) / 'candidate'
        self.pr = {
            'number': 15618,
            'html_url': 'https://github.com/kubeflow/pipelines/pull/15618',
            'state': 'open',
            'merged': False,
            'base': {
                'ref': 'release-2.18',
                'repo': {'full_name': 'kubeflow/pipelines'},
            },
            'head': {'sha': SHA, 'repo': {'full_name': 'maintainer/pipelines'}},
        }
        self.runner = mock.Mock()
        self.runner.capture.side_effect = lambda command, **kwargs: (
            SHA if command[1] == 'rev-parse' else '2.18.1'
        )
        self.api = mock.patch.object(
            candidate, '_api', side_effect=lambda *args: copy.deepcopy(self.pr)
        ).start()
        self.coverage = mock.patch.object(
            candidate,
            'release_coverage',
            return_value=({('api', 'linux/amd64')}, {'3.9', '3.13'}),
        ).start()
        self.result = {'run': {'run_id': '123', 'run_attempt': 1}, 'scans': []}
        self.collect = mock.patch.object(
            candidate,
            'collect_ci',
            side_effect=lambda *args, **kwargs: copy.deepcopy(self.result),
        ).start()
        self.addCleanup(mock.patch.stopall)

    def scan(self, **kwargs):
        return candidate.scan_candidate(
            pr=15618,
            expected_head=SHA,
            version='2.18.1',
            output_dir=self.output,
            runner=self.runner,
            **kwargs,
        )

    def record(self):
        return json.loads((self.output / 'candidate.json').read_text())

    def test_complete_scan_is_candidate_only_and_never_touches_release_state(
        self,
    ):
        state = self.output.parent / 'release-state.json'
        state.write_text('{"release_sha":"unchanged","completed_steps":[]}')
        before = state.read_bytes()
        result = self.scan()
        self.assertEqual(result['status'], 'passed')
        self.assertTrue(result['passed'])
        self.assertNotIn('release_sha', result)
        self.assertNotIn('qualification_path', result)
        self.assertEqual(state.read_bytes(), before)
        self.assertFalse(hasattr(self.collect.call_args.args[0], 'state'))
        self.assertEqual(
            self.collect.call_args.args[1:4],
            (
                SHA,
                [{'name': 'api', 'platform': 'linux/amd64'}],
                ['3.13', '3.9'],
            ),
        )
        commands = [call.args[0] for call in self.runner.run.call_args_list]
        self.assertTrue(
            all(
                command[:2] in (['git', 'init'], ['git', 'fetch'])
                for command in commands
            )
        )
        self.assertEqual(self.api.call_count, 3)

    def test_invalid_pr_identities_never_dispatch(self):
        changes = [
            {'html_url': 'https://github.com/other/pipelines/pull/15618'},
            {
                'base': {
                    'ref': 'master',
                    'repo': {'full_name': 'kubeflow/pipelines'},
                }
            },
            {
                'base': {
                    'ref': 'release-2.18',
                    'repo': {'full_name': 'other/pipelines'},
                }
            },
            {'state': 'closed'},
            {'merged': True},
            {
                'head': {
                    'sha': 'b' * 40,
                    'repo': {'full_name': 'maintainer/pipelines'},
                }
            },
            {'head': {'sha': SHA, 'repo': None}},
        ]
        baseline = copy.deepcopy(self.pr)
        for changeset in changes:
            self.pr = {**baseline, **changeset}
            with self.subTest(change=changeset), self.assertRaises(ValueError):
                self.scan()
        self.collect.assert_not_called()

    def test_pr_move_during_collection_marks_evidence_stale(self):
        def collect(*args, **kwargs):
            self.pr['head']['sha'] = 'b' * 40
            return self.result

        self.collect.side_effect = collect
        with self.assertRaisesRegex(ValueError, 'head changed'):
            self.scan()
        self.assertEqual(self.record()['status'], 'stale')
        self.assertFalse(self.record()['passed'])
        self.assertTrue((self.output / 'evidence.json').exists())

    def test_fetched_head_race_fails_without_dispatch(self):
        self.runner.capture.return_value = 'b' * 40
        self.runner.capture.side_effect = None
        with self.assertRaisesRegex(ValueError, 'Fetched PR head changed'):
            self.scan()
        self.collect.assert_not_called()
        self.assertEqual(self.record()['status'], 'incomplete')

    def test_missing_coverage_retains_failure_instead_of_policy_success(self):
        self.collect.side_effect = ValueError('image inventory is incomplete')
        with self.assertRaisesRegex(ValueError, 'inventory is incomplete'):
            self.scan()
        self.assertEqual(self.record()['status'], 'incomplete')
        self.assertFalse((self.output / 'policy.json').exists())
        self.assertEqual(self.api.call_count, 3)

    def test_findings_do_not_require_or_create_exceptions(self):
        self.result['scans'] = [
            {
                'name': 'source',
                'findings': [
                    {
                        'id': 'GHSA-example',
                        'package': 'example',
                        'target': 'go.mod',
                        'installed_version': '1',
                        'fixed_versions': ['2'],
                        'fixable': True,
                    }
                ],
            }
        ]
        result = self.scan()
        self.assertEqual(result['status'], 'findings')
        self.assertEqual(result['blocking_count'], 1)
        self.assertFalse(result['passed'])
        policy = json.loads((self.output / 'policy.json').read_text())
        self.assertEqual(policy['accepted_count'], 0)

    def test_resume_rechecks_head_and_marks_prior_success_stale(self):
        self.scan()
        self.pr['head']['sha'] = 'b' * 40
        with self.assertRaisesRegex(ValueError, 'head changed'):
            self.scan(resume=True)
        self.assertEqual(self.record()['status'], 'stale')
        self.assertFalse(self.record()['passed'])
        self.assertEqual(self.collect.call_count, 1)

    def test_resume_requires_same_scope(self):
        self.scan()
        with self.assertRaisesRegex(ValueError, 'scope'):
            self.scan(resume=True, include_sdk=False)
        self.assertEqual(self.collect.call_count, 1)

    def test_resume_passes_explicit_request_without_release_state(self):
        self.scan()
        self.scan(resume=True)
        self.assertTrue(self.collect.call_args.kwargs['resume'])

    def test_empty_scope_and_nonimmutable_sha_fail(self):
        with self.assertRaisesRegex(ValueError, 'Select backend'):
            self.scan(include_backend=False, include_sdk=False)
        with self.assertRaisesRegex(ValueError, 'exact 40-character'):
            candidate.scan_candidate(
                pr=15618,
                expected_head='master',
                version='2.18.1',
                output_dir=self.output,
            )
        self.api.assert_not_called()


class CandidateCliTest(unittest.TestCase):
    def test_cli_separates_findings_from_collection_failures(self):
        args = [
            'scan-candidate',
            '--pr',
            '15618',
            '--expected-head',
            SHA,
            '--version',
            '2.18.1',
            '--output-dir',
            '/tmp/candidate-cli',
            '--no-include-sdk',
        ]
        with mock.patch.object(
            candidate,
            'scan_candidate',
            return_value={
                'passed': False,
                'blocking_count': 3,
                'residual_count': 4,
            },
        ) as scan:
            result = CliRunner().invoke(app, args)
        self.assertEqual(result.exit_code, 1, result.output)
        self.assertIn('3 fixable blocking', result.output)
        self.assertFalse(scan.call_args.kwargs['include_sdk'])
        with mock.patch.object(
            candidate,
            'scan_candidate',
            side_effect=ValueError('coverage missing'),
        ):
            result = CliRunner().invoke(app, args)
        self.assertEqual(result.exit_code, 2, result.output)
        self.assertIn('coverage missing', result.output)

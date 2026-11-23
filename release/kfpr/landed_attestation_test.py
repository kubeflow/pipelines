"""The release gate requires a trusted, current attestation for its exact
cut."""

import copy
import json
import unittest
from pathlib import Path
from unittest import mock

from kfpr import landed_attestation as attestation

REPO = 'kubeflow/pipelines'
SHA = 'a' * 40
MAIN = 'b' * 40


class LandedAttestationTest(unittest.TestCase):
    def setUp(self):
        self.context = mock.Mock()
        self.context.state.answers = {}
        self.run = {
            'id': 123,
            'path': attestation.WORKFLOW,
            'event': 'workflow_dispatch',
            'workflow_id': 42,
            'head_sha': MAIN,
            'head_branch': 'main',
            'head_repository': {'full_name': REPO},
            'repository': {'full_name': REPO},
            'display_title': f'Release landed CI #1 at {SHA}',
            'run_attempt': 2,
            'status': 'completed',
            'conclusion': 'success',
        }
        self.document = {
            'schema_version': 1,
            'repository': REPO,
            'pr': 1,
            'replay_pr': 2,
            'branch': 'release-2.18',
            'sha': SHA,
            'run_id': 123,
            'run_attempt': 2,
            'workflow_sha': MAIN,
            'evidence': [{'run': 45, 'attempt': 1}],
        }
        self.data = {
            '': {'default_branch': 'main'},
            'actions/runs/123': self.run,
            'actions/workflows/release-landed-ci.yml': {
                'path': attestation.WORKFLOW,
                'id': 42,
            },
            f'compare/{MAIN}...main?per_page=1': {
                'status': 'ahead',
                'merge_base_commit': {'sha': MAIN},
            },
            'actions/runs/123/attempts/2/jobs?per_page=100': {
                'total_count': 1,
                'jobs': [
                    {
                        'name': 'ci-passed',
                        'conclusion': 'success',
                        'status': 'completed',
                    }
                ],
            },
            'actions/runs/123/artifacts?name=landed-ci-123-2': {
                'total_count': 1,
                'artifacts': [{'name': 'landed-ci-123-2', 'expired': False}],
            },
        }
        self.api = self.enterContext(
            mock.patch.object(
                attestation,
                '_api',
                side_effect=lambda _, path: self.data[path],
            )
        )
        self.collect = self.enterContext(
            mock.patch(
                'kfpr.landed_ci.collect_evidence',
                return_value=(
                    'release-2.18',
                    copy.deepcopy(self.document['evidence']),
                ),
            )
        )

        def download(command, cwd):
            (Path(command[-1]) / 'landed-ci.json').write_text(
                json.dumps(self.document)
            )

        self.context.runner.run.side_effect = download

    def verify(self):
        attestation.verify_landed_attestation(
            self.context, 123, 1, 'release-2.18', SHA
        )

    def test_pins_attempt_and_revalidates_live_evidence(self):
        self.verify()
        self.assertEqual(self.context.state.answers['landed_ci_attempt'], 2)
        self.collect.assert_called_once_with(1, SHA, 2)
        self.collect.return_value = (
            'release-2.18',
            [{'run': 45, 'attempt': 2}],
        )
        with self.assertRaisesRegex(ValueError, 'differs'):
            self.verify()
        self.run['run_attempt'] = 3
        with self.assertRaisesRegex(ValueError, 'changed run attempt'):
            self.verify()

    def test_rejects_untrusted_or_unsuccessful_workflow(self):
        for key, value in (
            ('event', 'pull_request'),
            ('head_branch', 'release-2.18'),
            ('head_repository', {'full_name': 'other/pipelines'}),
            ('workflow_id', 99),
            ('path', '.github/workflows/other.yml'),
            ('display_title', f'Release landed CI #99 at {SHA}'),
            ('conclusion', 'failure'),
        ):
            with self.subTest(key=key), mock.patch.dict(self.run, {key: value}):
                with self.assertRaises(ValueError):
                    self.verify()
        self.data[f'compare/{MAIN}...main?per_page=1']['status'] = 'diverged'
        with self.assertRaisesRegex(ValueError, 'outside upstream main'):
            self.verify()
        self.context.runner.run.assert_not_called()

    def test_rejects_missing_or_stale_job_and_artifact(self):
        for path, change in (
            ('actions/runs/123/attempts/2/jobs?per_page=100', {'jobs': []}),
            (
                'actions/runs/123/artifacts?name=landed-ci-123-2',
                {'total_count': 0},
            ),
        ):
            with (
                self.subTest(path=path),
                mock.patch.dict(self.data[path], change),
            ):
                with self.assertRaises(ValueError):
                    self.verify()
        for key, value in (
            ('sha', MAIN),
            ('branch', 'main'),
            ('pr', 2),
            ('run_attempt', 1),
            ('run_id', 456),
            ('workflow_sha', SHA),
        ):
            with (
                self.subTest(key=key),
                mock.patch.dict(self.document, {key: value}),
            ):
                with self.assertRaisesRegex(ValueError, 'differs'):
                    self.verify()

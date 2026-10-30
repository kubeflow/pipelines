"""Bind promotion dispatch to the requested workflow commit and attempt."""

import copy
import unittest
from unittest import mock

from kfpr import promotion, workflow_runs


class PromotionDispatchTest(unittest.TestCase):
    def setUp(self):
        self.sha = 'a' * 40
        self.request = 'b' * 32
        self.run = {
            'id': 123,
            'repository': {'full_name': 'kubeflow/pipelines'},
            'display_title': f'KFP latest promotion 2.17.3 {self.request}',
            'head_sha': self.sha,
            'event': 'workflow_dispatch',
            'path': '.github/workflows/release-promote-latest.yml',
            'run_attempt': 1,
        }
        self.final = {
            **self.run,
            'conclusion': 'success',
            'status': 'completed',
        }
        self.runs = [self.run]

    def api(self, endpoint):
        if endpoint == 'commits/master':
            return {'sha': self.sha}
        if endpoint.startswith('actions/workflows/'):
            return {'workflow_runs': self.runs}
        if endpoint == 'actions/runs/123':
            return self.final
        raise AssertionError(endpoint)

    def invoke(self):
        with (
            mock.patch.object(promotion, 'discover') as discover,
            mock.patch.object(promotion, 'api', side_effect=self.api),
            mock.patch.object(promotion.uuid, 'uuid4') as request,
            mock.patch.object(promotion.subprocess, 'run') as run,
            mock.patch.object(workflow_runs.time, 'sleep'),
        ):
            request.return_value.hex = self.request
            promotion.dispatch('2.17.3')
        discover.assert_called_once_with('2.17.3')
        return run

    def test_dispatch_watches_matching_request_and_verifies_attempt(self):
        run = self.invoke()
        self.assertEqual(run.call_count, 2)
        self.assertEqual(
            run.call_args_list[0].args[0],
            [
                'gh',
                'workflow',
                'run',
                'release-promote-latest.yml',
                '--repo',
                'kubeflow/pipelines',
                '--ref',
                'master',
                '-f',
                'version=2.17.3',
                '-f',
                f'request_id={self.request}',
            ],
        )
        self.assertEqual(
            run.call_args_list[1].args[0],
            [
                'gh',
                'run',
                'watch',
                '123',
                '--repo',
                'kubeflow/pipelines',
                '--exit-status',
            ],
        )

    def test_wrong_workflow_identity_is_rejected(self):
        original = copy.deepcopy(self.run)
        for key, value in (
            ('repository', {'full_name': 'attacker/pipelines'}),
            ('head_sha', 'c' * 40),
            ('event', 'push'),
            ('path', '.github/workflows/other.yml'),
        ):
            with self.subTest(key=key):
                self.run.clear()
                self.run.update(original)
                self.run[key] = value
                with self.assertRaisesRegex(ValueError, 'provenance'):
                    self.invoke()

    def test_changed_attempt_or_failed_result_is_rejected(self):
        original = copy.deepcopy(self.final)
        for key, value in (
            ('repository', {'full_name': 'attacker/pipelines'}),
            ('head_sha', 'c' * 40),
            ('conclusion', 'failure'),
            ('status', 'in_progress'),
            ('event', 'push'),
            ('path', '.github/workflows/other.yml'),
            ('display_title', 'another request'),
            ('id', 456),
            ('run_attempt', 2),
        ):
            with self.subTest(key=key):
                self.final = {**original, key: value}
                with self.assertRaisesRegex(ValueError, 'expected attempt'):
                    self.invoke()

    def test_missing_or_duplicate_dispatch_is_rejected(self):
        self.runs = []
        with self.assertRaisesRegex(RuntimeError, 'not found'):
            self.invoke()
        self.runs = [self.run, self.run]
        with self.assertRaisesRegex(ValueError, 'Ambiguous'):
            self.invoke()

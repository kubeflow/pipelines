# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Diagnostic snapshots must expose omissions without granting acceptance."""
import contextlib
import io
import json
import unittest
from unittest import mock

import capture_run_diagnostics as diag
from kfp_http import CollectionError


class Client:

    def __init__(self, listed, direct):
        self.listed, self.direct = listed, direct
        self.calls = []

    def get(self, path, params=None):
        self.calls.append(path)
        if path == '/apis/v2beta1/runs':
            return dict(runs=self.listed, total_size=len(self.direct))
        if path.startswith('/apis/v2beta1/experiments/'):
            return dict(experiment_id='experiment', namespace='ns')
        value = self.direct[path.rsplit('/', 1)[1]]
        if isinstance(value, Exception):
            raise value
        return value


class RunDiagnosticsTests(unittest.TestCase):

    def setUp(self):
        self.case = dict(
            scenario='default',
            schedule_uid='schedule',
            service_account='runner',
            baseline_run_ids=[])
        self.run = dict(
            run_id='run',
            recurring_run_id='schedule',
            experiment_id='experiment',
            created_at='2026-01-01T00:01:00Z',
            service_account='runner',
            state='SUCCEEDED',
            pipeline_spec={'password': 'must-not-leak'},
            arbitrary='must-not-leak')
        self.start = diag.timestamp('2026-01-01T00:00:00Z')
        self.observed = dict(
            cases=[dict(schedule_uid='schedule', runs=[dict(run_id='run')])])

    def capture(self, client):
        return diag.collect(lambda: client, 'ns', [self.case], self.observed,
                            self.start)

    def test_get_proves_list_omission_without_acceptance(self):
        report = self.capture(Client([], {'run': self.run}))
        case = report['cases'][0]
        self.assertEqual(case['list']['returned_count'], 0)
        self.assertEqual(case['list']['pages'][0]['total_size'], 1)
        self.assertEqual(case['direct']['records'][0]['metadata']['state'],
                         'SUCCEEDED')
        self.assertTrue(case['direct']['records'][0]['namespace_verified'])
        self.assertNotEqual(report['outcome'], 'passed')
        self.assertNotIn('must-not-leak', json.dumps(report))

    def test_includes_ids_first_seen_during_drain(self):
        client = Client([], {
            'run': self.run,
            'late': dict(self.run, run_id='late')
        })
        report = diag.collect(lambda: client, 'ns', [self.case], self.observed,
                              self.start,
                              dict(known_run_ids={'schedule': ['late', 'run']}))
        direct = report['cases'][0]['direct']
        self.assertEqual(direct['known_id_count'], 2)
        self.assertEqual([r['requested_run_id'] for r in direct['records']],
                         ['late', 'run'])

    def test_timestamp_and_baseline_exclusions_are_distinct(self):
        self.case['baseline_run_ids'] = ['run']
        self.start = diag.timestamp('2026-01-01T00:02:00Z')
        output = self.capture(Client([self.run],
                                     {'run': self.run}))['cases'][0]['list']
        self.assertEqual(output['fresh_count'], 0)
        self.assertEqual(output['records'][0]['excluded_reasons'],
                         ['baseline_run', 'before_activation'])

    def test_direct_not_found_is_retained_without_raw_error(self):
        output = self.capture(Client([], {'run': CollectionError('not_found')}))
        self.assertEqual(output['cases'][0]['direct']['records'][0]['reason'],
                         'not_found')
        output = self.capture(Client([], {'run': ValueError('must-not-leak')}))
        self.assertNotIn('must-not-leak', json.dumps(output))

    def test_direct_identity_mismatch_is_not_accepted(self):
        other = dict(self.run, run_id='other')
        output = self.capture(Client(
            [], {'run': other}))['cases'][0]['direct']['records'][0]
        self.assertEqual(output['collection'], 'inconclusive')
        self.assertNotIn('metadata', output)

    def test_direct_sampling_is_bounded(self):
        runs = {
            f'run-{i:02}': dict(self.run, run_id=f'run-{i:02}')
            for i in range(25)
        }
        self.observed['cases'][0]['runs'] = [dict(run_id=uid) for uid in runs]
        output = self.capture(Client([], runs))['cases'][0]['direct']
        self.assertTrue(output['truncated'])
        self.assertEqual(output['known_id_count'], 25)
        self.assertEqual(len(output['records']), 20)

    def test_shared_deadline_stops_new_requests(self):
        client = Client([], {'run': self.run})
        with mock.patch.object(diag.time, 'monotonic', side_effect=[0, 0, 91]):
            report = self.capture(client)
        case = report['cases'][0]
        self.assertEqual(case['list']['collection'], 'complete')
        self.assertEqual(case['direct']['records'][0]['reason'],
                         'diagnostic_deadline_exceeded')
        self.assertEqual(client.calls, ['/apis/v2beta1/runs'])

    def test_cli_failure_emits_only_sanitized_json(self):
        args = [
            'capture_run_diagnostics.py', '--namespace', 'ns', '--baseline',
            'baseline', '--observed', 'observed', '--activation-start-file',
            'activation', '--endpoint', 'http://localhost', '--token-file',
            'token'
        ]
        output = io.StringIO()
        with mock.patch('sys.argv', args), mock.patch.object(
                diag, 'load', side_effect=ValueError(
                    'private-payload')), contextlib.redirect_stdout(output):
            diag.main()
        report = json.loads(output.getvalue())
        self.assertEqual(report['outcome'], 'inconclusive')
        self.assertEqual(report['reason'], 'invalid_or_incomplete_evidence')
        self.assertNotIn('private-payload', output.getvalue())

    def test_database_logs_retain_only_list_error_codes(self):
        logs = (
            "private /api.v2beta1.RunService/ListRuns call failed Error 1038 (HY001): sensitive\n"
            "private unrelated Error 1114 (HY000): sensitive\n")
        with mock.patch.object(diag, 'collect_logs', return_value=logs):
            report = diag.database_diagnostics('kind-kfp-readiness', self.start)
        self.assertEqual(report['mysql_error_code_counts'], {'1038': 1})
        self.assertNotIn('private', json.dumps(report))
        self.assertNotIn('sensitive', json.dumps(report))
        with mock.patch.object(
                diag, 'collect_logs', side_effect=ValueError('private')):
            report = diag.database_diagnostics('context', self.start)
        self.assertEqual(report['reason'], 'api_log_collection_failed')

    def test_input_read_is_bounded(self):
        with mock.patch.object(
                diag.Path,
                'open',
                return_value=io.BytesIO(b'x' * (diag.MAX_INPUT + 1))):
            with self.assertRaisesRegex(ValueError, 'input_limit_exceeded'):
                diag.load('oversized')

    def test_namespace_proof_failure_is_explicit(self):
        client = Client([], {'run': self.run})
        original = client.get

        def get(path, params=None):
            if '/experiments/' in path:
                return dict(experiment_id='experiment', namespace='foreign')
            return original(path, params)

        client.get = get
        item = self.capture(client)['cases'][0]['direct']['records'][0]
        self.assertFalse(item['namespace_verified'])


if __name__ == '__main__':
    unittest.main()

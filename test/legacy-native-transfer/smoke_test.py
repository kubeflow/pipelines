# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
import io
import json
import unittest
from pathlib import Path
from unittest import mock

import smoke


class SmokeTest(unittest.TestCase):
    @mock.patch.object(smoke.urllib.request, 'urlopen')
    def test_archive_bytes_are_uploaded_unchanged(self, urlopen):
        wire = b'{"number": 1.00, "escaped": "\\u0061"}\n'
        urlopen.return_value.__enter__.return_value = io.BytesIO(b'{}')
        smoke.request('transfer/import', wire, dry_run='true')
        self.assertEqual(wire, urlopen.call_args.args[0].data)

    @mock.patch.object(smoke.urllib.request, 'urlopen')
    def test_export_retains_original_bytes(self, urlopen):
        wire = b'{"number": 1.00}\n'
        urlopen.return_value.__enter__.return_value = io.BytesIO(wire)
        self.assertEqual(wire, smoke.request('transfer/export', {}, raw=True))

    @mock.patch.object(smoke, 'request')
    def test_recurring_run_wire_key_and_pagination(self, request):
        request.side_effect = [
            {
                'recurringRuns': [{'recurring_run_id': 'first'}],
                'next_page_token': 'next',
            },
            {'recurringRuns': [{'recurring_run_id': 'second'}]},
        ]
        self.assertEqual(
            [{'recurring_run_id': 'first'}, {'recurring_run_id': 'second'}],
            smoke.records('recurringruns'),
        )
        self.assertEqual('next', request.call_args.kwargs['page_token'])

    @mock.patch.object(
        smoke, 'request', return_value={'next_page_token': 'stuck'}
    )
    def test_repeating_pagination_is_bounded(self, request):
        with self.assertRaisesRegex(AssertionError, 'Unbounded'):
            smoke.records('runs')
        self.assertEqual(100, request.call_count)

    @mock.patch.object(smoke.time, 'sleep')
    @mock.patch.object(smoke, 'request')
    def test_terminal_run_does_not_require_mlmd_hydration(self, request, sleep):
        complete = {'state': 'SUCCEEDED', 'run_id': 'source'}
        request.side_effect = [{'state': 'RUNNING'}, complete]
        self.assertEqual(complete, smoke.wait_run('source'))
        sleep.assert_called_once_with(5)

    @mock.patch.object(
        smoke,
        'request',
        return_value={'error': {'message': 'invalid runtime config'}},
    )
    def test_run_api_conversion_error_fails_immediately(self, request):
        with self.assertRaisesRegex(
            AssertionError, 'API error.*invalid runtime config'
        ):
            smoke.wait_run('source')
        request.assert_called_once()

    @mock.patch.object(smoke, 'request', return_value={'state': 'FAILED'})
    def test_failed_run_is_not_retried(self, request):
        with self.assertRaisesRegex(AssertionError, 'ended FAILED'):
            smoke.wait_run('source')
        request.assert_called_once()

    @mock.patch.object(smoke.time, 'monotonic', side_effect=[0, 601])
    @mock.patch.object(smoke, 'request')
    def test_run_wait_is_bounded(self, request, monotonic):
        with self.assertRaisesRegex(AssertionError, '600s'):
            smoke.wait_run('source')
        request.assert_not_called()

    @mock.patch.object(smoke.time, 'sleep')
    @mock.patch.object(smoke, 'request')
    def test_export_waits_for_actual_legacy_graph_and_retains_wire(
        self, request, sleep
    ):
        fixture = (
            Path(__file__).resolve().parents[2]
            / 'backend/src/apiserver/history/testdata/legacy-218-mlmd-v2-export.json'
        )
        wire = fixture.read_bytes()
        archive = json.loads(wire)
        incomplete = dict(archive, metadata={})
        request.side_effect = [json.dumps(incomplete).encode(), wire]
        actual, parsed = smoke.wait_export(archive['runs'][0]['run']['UUID'])
        self.assertEqual(wire, actual)
        self.assertEqual(archive, parsed)
        sleep.assert_called_once_with(5)

    @mock.patch.object(smoke.time, 'monotonic', side_effect=[0, 181])
    @mock.patch.object(smoke, 'request')
    def test_export_wait_is_bounded(self, request, monotonic):
        with self.assertRaisesRegex(AssertionError, '180s'):
            smoke.wait_export('source')
        request.assert_not_called()

    def test_native_root_completion_matches_run_and_covers_children(self):
        run = {
            'finished_at': '2026-10-11T02:46:24Z',
            'tasks': [
                {'type': 'ROOT', 'end_time': '2026-10-11T02:46:24Z'},
                {'type': 'RUNTIME', 'end_time': '2026-10-11T02:46:18Z'},
            ],
        }
        smoke.verify_root_completion(run)
        run['tasks'][0]['end_time'] = '2026-10-11T02:45:23Z'
        with self.assertRaises(AssertionError):
            smoke.verify_root_completion(run)
        run['tasks'][0]['end_time'] = run['finished_at']
        run['tasks'][1]['end_time'] = '2026-10-11T02:46:25Z'
        with self.assertRaises(AssertionError):
            smoke.verify_root_completion(run)

    def test_artifact_references_deduplicate_inputs_and_outputs(self):
        self.assertEqual(
            {'s3://source/data'},
            smoke.artifact_uris(
                {
                    'tasks': [
                        {
                            'inputs': {
                                'artifacts': [{'uri': 's3://source/data'}]
                            }
                        },
                        {
                            'outputs': {
                                'artifacts': [{'uri': 's3://source/data'}]
                            }
                        },
                    ]
                }
            ),
        )


if __name__ == '__main__':
    unittest.main()

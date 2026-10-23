# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
from pathlib import Path
import unittest
from unittest import mock

import live_handoff_wait as live

TIMEOUT_BODY = dict(
    code=14,
    details=[{
        '@type': 'type.googleapis.com/google.rpc.ErrorInfo',
        'reason': 'SCHEDULE_HANDOFF_IN_PROGRESS',
        'domain': 'kubeflow.org'
    }])


class HandoffWaitTest(unittest.TestCase):

    @mock.patch.object(live.http.client, 'HTTPConnection')
    def test_503_is_returned_without_retry(self, constructor):
        connection = constructor.return_value
        connection.getresponse.return_value.status = 503
        connection.getresponse.return_value.read.return_value = b'{"code":14}'
        result = live.request(18888, 'test-token', 'POST',
                              '/apis/v2beta1/recurringruns',
                              {'mode': 'DISABLE'})
        self.assertEqual(result['status'], 503)
        self.assertEqual(result['attempts'], 1)
        connection.request.assert_called_once()
        connection.close.assert_called_once()

    @mock.patch.object(live.http.client, 'HTTPConnection')
    def test_cancellation_closes_connection_without_retry(self, constructor):
        connection = constructor.return_value
        connection.getresponse.side_effect = TimeoutError()
        with self.assertRaises(TimeoutError):
            live.request(
                18888,
                'test-token',
                'POST',
                '/apis/v2beta1/recurringruns',
                timeout=2)
        connection.request.assert_called_once()
        connection.close.assert_called_once()

    def test_timeout_contract_rejects_early_wrong_or_retried_results(self):
        valid = dict(status=503, body=TIMEOUT_BODY, elapsed=30, attempts=1)
        live.validate_timeout(valid)
        for field, value in [('status', 200), ('body', {
                'code': 4
        }), ('elapsed', 1), ('elapsed', 40), ('attempts', 2)]:
            with self.subTest(
                    field=field, value=value), self.assertRaises(ValueError):
                live.validate_timeout(dict(valid, **{field: value}))

    @mock.patch.object(live, 'kube')
    def test_release_allows_normal_controller_shutdown(self, kube):
        live.release('retiring-controller')
        kube.assert_called_once_with('-n', 'kubeflow', 'exec',
                                     'retiring-controller', '-c',
                                     'ml-pipeline-scheduledworkflow', '--',
                                     'touch', live.RELEASE_FILE)

    @mock.patch.object(live, 'ThreadPoolExecutor')
    @mock.patch.object(live.time, 'sleep')
    @mock.patch.object(live, 'inventory')
    @mock.patch.object(live, 'request')
    def test_exercise_checks_cancellation_timeout_then_released_success(
            self, request, inventory, sleep, executor):
        baseline = dict(jobs=[], schedules=[])
        created = dict(jobs=[{'UUID': 'new'}], schedules=[('new', False)])
        inventory.side_effect = [
            baseline, baseline, baseline, baseline, created, created
        ]
        request.side_effect = [
            TimeoutError(),
            dict(status=503, body=TIMEOUT_BODY, elapsed=30, attempts=1),
            dict(status=200, elapsed=0.1),
            dict(status=200, elapsed=0.1)
        ]
        future = executor.return_value.__enter__.return_value.submit.return_value
        future.done.return_value = False
        future.result.return_value = dict(status=200, elapsed=4, attempts=1)
        unblock = mock.Mock()
        result = live.exercise(18888, 'test-token', {'display_name': 'probe'},
                               unblock)
        unblock.assert_called_once()
        self.assertEqual(result['success']['attempts'], 1)
        self.assertEqual(request.call_count, 4)
        self.assertEqual(inventory.call_count, 6)

    def test_mutation_probe_follows_history_validation(self):
        script = (
            Path(__file__).resolve().parents[2] /
            '.github/resources/scripts/readiness-adoption.sh').read_text()
        self.assertLess(
            script.index('check drained'), script.index('live_handoff_wait.py'))
        self.assertLess(
            script.index('adoption-complete.json'),
            script.index('live_handoff_wait.py'))


if __name__ == '__main__':
    unittest.main()

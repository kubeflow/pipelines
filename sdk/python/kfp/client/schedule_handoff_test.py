# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

import json
import unittest
from unittest import mock

from kfp.client.client import Client
from kfp.client.schedule_handoff import _is_schedule_handoff
from kfp.client.schedule_handoff import retry_schedule_handoff
import kfp_server_api


def handoff_error(**changes):
    detail = {
        '@type': 'type.googleapis.com/google.rpc.ErrorInfo',
        'domain': 'kubeflow.org',
        'reason': 'SCHEDULE_HANDOFF_IN_PROGRESS',
    }
    detail.update(changes)
    error = kfp_server_api.ApiException(status=503)
    error.body = json.dumps({'code': 14, 'details': [detail]})
    return error


class ScheduleHandoffTest(unittest.TestCase):

    def test_requires_exact_structured_identity(self):
        error = handoff_error()
        self.assertTrue(_is_schedule_handoff(error))
        error.body = error.body.encode()
        self.assertTrue(_is_schedule_handoff(error))
        for changes in ({
                '@type': 'other'
        }, {
                'domain': 'other'
        }, {
                'reason': 'other'
        }):
            self.assertFalse(_is_schedule_handoff(handoff_error(**changes)))
        for status in (400, 500):
            error = handoff_error()
            error.status = status
            self.assertFalse(_is_schedule_handoff(error))
        error = handoff_error()
        error.body = error.body.replace('14', '9')
        self.assertFalse(_is_schedule_handoff(error))

    def test_malformed_or_message_only_error_is_not_retryable(self):
        for body in (None, b'\xff', '<html>unavailable</html>', 'null', '[]',
                     '{"code":14,"details":[null,1,"bad"]}',
                     '{"code":14,"details":{}}',
                     '{"code":14,"message":"SCHEDULE_HANDOFF_IN_PROGRESS"}'):
            with self.subTest(body=body):
                error = handoff_error()
                error.body = body
                self.assertFalse(_is_schedule_handoff(error))

    @mock.patch('kfp.client.schedule_handoff.time.sleep')
    def test_exhaustion_is_bounded_and_preserves_exception(self, sleep):
        error = handoff_error()
        request = mock.Mock(side_effect=error)
        with self.assertLogs(level='WARNING') as logs:
            with self.assertRaises(kfp_server_api.ApiException) as caught:
                retry_schedule_handoff(request)
        self.assertIs(caught.exception, error)
        self.assertEqual(request.call_count, 5)
        self.assertEqual(
            sleep.call_args_list,
            [mock.call(1),
             mock.call(2),
             mock.call(4),
             mock.call(8)])
        self.assertIn('request was not applied', logs.output[0])
        self.assertIn('verify the schedule state', logs.output[0])
        self.assertNotIn(error.body, logs.output[0])

    @mock.patch('kfp.client.schedule_handoff.time.sleep')
    def test_never_retries_ambiguous_error_even_after_handoff(self, sleep):
        for error in (TimeoutError('outcome unknown'),
                      kfp_server_api.ApiException(status=503)):
            for prior_handoff in (False, True):
                with self.subTest(
                        error=type(error).__name__, prior=prior_handoff):
                    sleep.reset_mock()
                    sequence = ([handoff_error()] if prior_handoff else [])
                    request = mock.Mock(side_effect=sequence + [error])
                    with self.assertRaises(type(error)) as caught:
                        retry_schedule_handoff(request)
                    self.assertIs(caught.exception, error)
                    self.assertEqual(request.call_count,
                                     2 if prior_handoff else 1)
                    self.assertEqual(sleep.call_count,
                                     1 if prior_handoff else 0)

    @mock.patch('kfp.client.schedule_handoff.time.sleep')
    def test_client_mutations_retry_identical_request_only(self, sleep):
        for operation in ('create', 'enable', 'disable'):
            with self.subTest(operation=operation):
                sdk = Client.__new__(Client)
                sdk._recurring_run_api = mock.Mock()
                api = getattr(
                    sdk._recurring_run_api,
                    f'recurring_run_service_{operation}_recurring_run')
                result = object()
                api.side_effect = [handoff_error(), handoff_error(), result]
                if operation == 'create':
                    got = sdk.create_recurring_run(
                        'experiment',
                        'nightly',
                        pipeline_id='pipeline',
                        interval_second=3600)
                    bodies = [
                        call.kwargs['recurring_run']
                        for call in api.call_args_list
                    ]
                    self.assertTrue(all(body is bodies[0] for body in bodies))
                else:
                    got = getattr(sdk, f'{operation}_recurring_run')('schedule')
                self.assertIs(got, result)
                self.assertEqual(api.call_count, 3)
                self.assertEqual(api.call_args_list, [api.call_args] * 3)

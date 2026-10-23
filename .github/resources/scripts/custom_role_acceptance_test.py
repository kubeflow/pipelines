# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Guard the live fixture's authorization evidence and grant boundaries."""
import importlib.util
import json
from pathlib import Path
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location(
    'custom_roles',
    Path(__file__).with_name('custom-role-acceptance.py'))
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class CustomRoleAcceptanceTest(unittest.TestCase):

    def test_server_failure_is_not_a_denial(self):
        matrix = MODULE.Matrix()
        with mock.patch.object(MODULE, 'request', return_value=(500, b'{}')):
            with self.assertRaisesRegex(RuntimeError, 'unexpected_status'):
                matrix.check(
                    'denied', 'author', 'GET', '/apis/test', expected=403)
        self.assertFalse(matrix.cases[0]['passed'])

    @staticmethod
    def handoff_body(**overrides):
        detail = {
            '@type': 'type.googleapis.com/google.rpc.ErrorInfo',
            'reason': 'SCHEDULE_HANDOFF_IN_PROGRESS',
            'domain': 'kubeflow.org',
        }
        detail.update(overrides)
        return json.dumps(dict(code=14, details=[detail])).encode()

    def test_handoff_retry_preserves_success_and_authorization_denial(self):
        for status in (200, 403):
            with self.subTest(status=status), mock.patch.object(
                    MODULE,
                    'request',
                    side_effect=[
                        (503, self.handoff_body()), (status, b'{}')
                    ]) as request, mock.patch.object(MODULE.time,
                                                     'sleep') as sleep:
                matrix = MODULE.Matrix()
                body = {'mode': 'DISABLE'}
                matrix.check(
                    'recurring',
                    'author',
                    'POST',
                    '/apis/v2beta1/recurringruns',
                    expected=status,
                    value=body)
                self.assertEqual(len(matrix.cases), 1)
                self.assertTrue(matrix.cases[0]['passed'])
                self.assertEqual(matrix.cases[0]['status'], status)
                self.assertEqual(matrix.cases[0]['handoff_retries'], 1)
                self.assertEqual(request.call_count, 2)
                self.assertEqual(request.call_args_list[0],
                                 request.call_args_list[1])
                sleep.assert_called_once_with(1)

    def test_handoff_retry_exhaustion_records_failure(self):
        with mock.patch.object(
                MODULE, 'request', return_value=(
                    503, self.handoff_body())) as request, mock.patch.object(
                        MODULE.time, 'sleep') as sleep:
            matrix = MODULE.Matrix()
            with self.assertRaisesRegex(RuntimeError, 'unexpected_status'):
                matrix.check('recurring', 'author', 'POST',
                             '/apis/v2beta1/recurringruns')
            self.assertEqual(request.call_count, 5)
            self.assertEqual(sleep.call_args_list,
                             [mock.call(n) for n in (1, 2, 4, 8)])
            self.assertFalse(matrix.cases[0]['passed'])
            self.assertEqual(matrix.cases[0]['status'], 503)
            self.assertEqual(matrix.cases[0]['handoff_retries'], 4)

    def test_unrelated_unavailable_response_is_not_retried(self):
        bodies = [
            b'{}',
            b'not json',
            b'[]',
            b'{"code":14,"message":"SCHEDULE_HANDOFF_IN_PROGRESS"}',
            self.handoff_body(domain='other.example'),
            self.handoff_body(reason='UNAVAILABLE'),
            self.handoff_body(**{'@type': 'other.ErrorInfo'}),
            self.handoff_body().replace(b'"code": 14', b'"code": 13'),
        ]
        for raw in bodies:
            with self.subTest(raw=raw), mock.patch.object(
                    MODULE, 'request',
                    return_value=(503, raw)) as request, mock.patch.object(
                        MODULE.time, 'sleep') as sleep:
                matrix = MODULE.Matrix()
                with self.assertRaisesRegex(RuntimeError, 'unexpected_status'):
                    matrix.check('recurring', 'author', 'POST',
                                 '/apis/v2beta1/recurringruns')
                request.assert_called_once()
                sleep.assert_not_called()
                self.assertFalse(matrix.cases[0]['passed'])

    def test_handoff_retry_is_limited_to_recurring_create(self):
        for method, path, status in [
            ('GET', '/apis/v2beta1/recurringruns', 503),
            ('POST', '/apis/v2beta1/runs', 503),
            ('POST', '/apis/v2beta1/recurringruns', 500)
        ]:
            with self.subTest(method=method, path=path, status=status), \
                    mock.patch.object(MODULE, 'request', return_value=(
                        status, self.handoff_body())) as request, \
                    mock.patch.object(MODULE.time, 'sleep') as sleep:
                with self.assertRaisesRegex(RuntimeError, 'unexpected_status'):
                    MODULE.Matrix().check('unavailable', 'author', method, path)
                request.assert_called_once()
                sleep.assert_not_called()

    def test_ambiguous_transport_failure_is_not_retried(self):
        with mock.patch.object(
                MODULE, 'request', side_effect=TimeoutError(
                    'response unavailable')) as request, mock.patch.object(
                        MODULE.time, 'sleep') as sleep:
            matrix = MODULE.Matrix()
            with self.assertRaises(TimeoutError):
                matrix.check('recurring', 'author', 'POST',
                             '/apis/v2beta1/recurringruns')
            request.assert_called_once()
            sleep.assert_not_called()
            self.assertFalse(any(case['passed'] for case in matrix.cases))

    def test_bad_request_requires_authorization_diagnostic(self):
        matrix = MODULE.Matrix()
        with mock.patch.object(
                MODULE,
                'request',
                return_value=(400, b'{"error_message":"malformed pipeline"}')):
            with self.assertRaisesRegex(RuntimeError,
                                        'missing_upload_authorization'):
                matrix.check(
                    'denied',
                    'author',
                    'POST',
                    '/apis/v2beta1/pipelines/upload',
                    expected=400)
        self.assertFalse(matrix.cases[0]['passed'])

    def test_valid_upload_denial(self):
        raw = json.dumps(
            dict(
                error_message='pipeline upload denied: permission to create pipelines.pipelines.kubeflow.org in namespace "team" is required. More guidance.'
            )).encode()
        with mock.patch.object(MODULE, 'request', return_value=(400, raw)):
            MODULE.Matrix().check(
                'denied',
                'author',
                'POST',
                '/apis/v2beta1/pipelines/upload',
                expected=400,
                expected_upload_scope=('pipeline', 'team'))

    def test_wrong_namespace_or_shared_scope_fails(self):
        for prefix, namespace in [('pipeline', 'kubeflow'),
                                  ('shared pipeline', 'team')]:
            raw = json.dumps(
                dict(
                    error_message=f'{prefix} upload denied: permission to create pipelines.pipelines.kubeflow.org in namespace "{namespace}" is required.'
                )).encode()
            with self.subTest(
                    prefix=prefix, namespace=namespace), mock.patch.object(
                        MODULE, 'request', return_value=(400, raw)):
                matrix = MODULE.Matrix()
                with self.assertRaisesRegex(RuntimeError,
                                            'missing_upload_authorization'):
                    matrix.check(
                        'denied',
                        'author',
                        'POST',
                        '/apis/v2beta1/pipelines/upload',
                        expected=400,
                        expected_upload_scope=('pipeline', 'team'))
                self.assertFalse(matrix.cases[0]['passed'])

    def test_viewer_failures_are_not_authorization_denials(self):
        with mock.patch.object(MODULE, 'request', return_value=(500, b'error')):
            with self.assertRaisesRegex(RuntimeError, 'unexpected_status'):
                MODULE.Matrix().ui('denied', 'viewer', 'DELETE', expected=401)

    def test_unrelated_ui_unauthorized_response_fails(self):
        with mock.patch.object(
                MODULE,
                'request',
                return_value=(401, b'authentication unavailable')):
            matrix = MODULE.Matrix()
            with self.assertRaisesRegex(RuntimeError,
                                        'missing_viewer_authorization'):
                matrix.ui('denied', 'viewer', 'DELETE', expected=401)
            self.assertFalse(matrix.cases[0]['passed'])

    def test_exact_ui_denial_is_retained(self):
        raw = b'User is not authorized to DELETE VIEWERS in namespace kfp-roles-team: Permission denied'
        with mock.patch.object(MODULE, 'request', return_value=(401, raw)):
            matrix = MODULE.Matrix()
            matrix.ui('denied', 'viewer', 'DELETE', expected=401)
            self.assertTrue(matrix.cases[0]['passed'])

    def test_end_users_have_only_namespaced_bindings(self):
        objects = MODULE.resources()
        for binding in objects:
            if binding['kind'].endswith('Binding'):
                for subject in binding['subjects']:
                    if subject['kind'] == 'User':
                        self.assertEqual(binding['kind'], 'RoleBinding')
        for role in objects:
            for rule in role.get('rules', []):
                self.assertNotIn('*', rule['verbs'])
                self.assertNotIn('*', rule['resources'])
                self.assertNotIn('use', rule['verbs'])

    def test_viewer_and_log_reader_remain_minimal(self):
        roles = {
            item['metadata']['name']: item
            for item in MODULE.resources()
            if item['kind'] == 'Role'
        }
        self.assertEqual(roles['custom-role-viewer']['rules'],
                         [MODULE.rule('kubeflow.org', ['viewers'], ['get'])])
        self.assertEqual(
            roles['custom-role-log-reader']['rules'],
            [MODULE.rule('pipelines.kubeflow.org', ['runs'], ['readLog'])])
        self.assertEqual(
            roles['custom-role-publisher']['metadata']['namespace'], 'kubeflow')


if __name__ == '__main__':
    unittest.main()

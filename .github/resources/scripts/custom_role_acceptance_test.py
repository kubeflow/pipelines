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
                error_message='pipeline upload denied: permission to create pipelines.pipelines.kubeflow.org in namespace "team" is required'
            )).encode()
        with mock.patch.object(MODULE, 'request', return_value=(400, raw)):
            MODULE.Matrix().check(
                'denied',
                'author',
                'POST',
                '/apis/v2beta1/pipelines/upload',
                expected=400)

    def test_viewer_failures_are_not_authorization_denials(self):
        with mock.patch.object(MODULE, 'request', return_value=(500, b'error')):
            with self.assertRaisesRegex(RuntimeError, 'unexpected_status'):
                MODULE.Matrix().ui('denied', 'viewer', 'DELETE', expected=401)

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

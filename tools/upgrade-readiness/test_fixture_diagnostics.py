# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Fixture diagnostics emit classifications, never raw execution data."""

import subprocess
import sys
import unittest
from unittest import mock

import fixture_diagnostics as diag


class DiagnosticsTests(unittest.TestCase):

    def pod(self, namespace=diag.NAMESPACE):
        return dict(
            metadata=dict(
                namespace=namespace,
                name='pod',
                uid='uid',
                labels={'workflows.argoproj.io/workflow': 'workflow'}),
            status=dict(containerStatuses=[
                dict(
                    name='main',
                    state=dict(
                        terminated=dict(
                            exitCode=1,
                            reason='Error',
                            message='private-token permission denied')))
            ]))

    def test_classifies_failed_container_without_raw_values(self):
        result = diag.container_diagnostics(
            [self.pod()], [dict(metadata=dict(name='workflow'))],
            logs=lambda *args: diag.categories(
                'private-password rpc error: code = Unavailable metadata-grpc'))
        record = result['containers'][0]
        self.assertEqual(record['exit_code'], 1)
        self.assertEqual(record['reason'], 'Error')
        self.assertEqual(record['log_categories'], ['connection', 'metadata'])
        self.assertNotIn('private', str(result))
        self.assertNotIn('pod', record)

    def test_log_read_scope_and_container_count_are_bounded(self):
        reader = mock.Mock(return_value=[])
        result = diag.container_diagnostics(
            [self.pod('other')], [dict(metadata=dict(name='workflow'))],
            logs=reader)
        self.assertEqual(result['containers'], [])
        reader.assert_not_called()
        result = diag.container_diagnostics(
            [self.pod()] * 20, [dict(metadata=dict(name='workflow'))],
            logs=reader)
        self.assertEqual(reader.call_count, diag.MAX_CONTAINERS)
        self.assertTrue(result['truncated'])

    def test_nodes_emit_only_known_types_and_templates(self):
        result = diag.node_diagnostics([
            dict(
                status=dict(
                    nodes={
                        'secret-id':
                            dict(
                                type='Pod',
                                phase='Error',
                                templateName='system-dag-driver',
                                message='secret'),
                        'other':
                            dict(
                                type='private',
                                phase='private',
                                templateName='private')
                    }))
        ])
        self.assertEqual(result['collection'], 'complete')
        self.assertNotIn('secret', str(result))
        self.assertNotIn('private', str(result))

    def test_log_byte_and_time_limits(self):
        popen = subprocess.Popen
        for script, budget in [("print('x' * 129)", 2),
                               ('import time; time.sleep(5)', .05)]:
            with mock.patch.object(diag, 'MAX_BYTES', 128), mock.patch.object(
                    diag, 'TIMEOUT', budget), mock.patch.object(
                        diag.subprocess,
                        'Popen',
                        side_effect=lambda command, **kwargs: popen(
                            [sys.executable, '-c', script], **kwargs)):
                with self.assertRaises(diag.CollectionError):
                    diag.log_categories('pod', 'main')

    def test_log_failure_never_includes_server_error(self):
        with mock.patch.object(
                diag.subprocess,
                'Popen',
                side_effect=OSError('private credentials')):
            with self.assertRaisesRegex(diag.CollectionError,
                                        '^fixture_log_unavailable$'):
                diag.log_categories('pod', 'main')


if __name__ == '__main__':
    unittest.main()

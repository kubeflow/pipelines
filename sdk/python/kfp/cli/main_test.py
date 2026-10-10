# Copyright 2018-2022 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import contextlib
import io
import json
import unittest
from unittest import mock

from kfp.cli import __main__
import kfp.server_api


class MainPaginationErrorTest(unittest.TestCase):

    def invoke(self, error):
        stderr = io.StringIO()
        with mock.patch.object(
                __main__.cli, 'cli', side_effect=error) as invoke:
            with contextlib.redirect_stderr(stderr):
                with self.assertRaises(SystemExit) as exit_error:
                    __main__.main()
            self.assertEqual(exit_error.exception.code, 1)
            invoke.assert_called_once_with(obj={}, auto_envvar_prefix='KFP')
        return stderr.getvalue()

    def test_restart_error_is_actionable_without_retry(self):
        error = kfp.server_api.ApiException(status=400)
        error.body = json.dumps({
            'code':
                9,
            'details': [{
                '@type': 'type.googleapis.com/google.rpc.ErrorInfo',
                'reason': 'PAGINATION_RESTART_REQUIRED',
                'domain': 'kubeflow.org',
            }],
        })
        output = self.invoke(error)
        self.assertIn('Remove --page-token', output)
        self.assertIn('Discard earlier output', output)
        self.assertIn('reconcile any actions already taken', output)
        self.assertNotIn('HTTP response', output)

    def test_unrelated_error_keeps_existing_output(self):
        output = self.invoke(ValueError('ordinary failure'))
        self.assertEqual(output, 'ordinary failure\n')

    def test_message_text_alone_does_not_trigger_restart(self):
        error = kfp.server_api.ApiException(status=400)
        error.body = '{"message": "PAGINATION_RESTART_REQUIRED"}'
        self.assertEqual(self.invoke(error), str(error) + '\n')

    def test_list_command_invokes_client_once_and_exits_nonzero(self):
        error = kfp.server_api.ApiException(status=400)
        error.body = json.dumps({
            'code':
                9,
            'details': [{
                '@type': 'type.googleapis.com/google.rpc.ErrorInfo',
                'reason': 'PAGINATION_RESTART_REQUIRED',
                'domain': 'kubeflow.org',
            }],
        })
        stderr = io.StringIO()
        with mock.patch.object(__main__.cli.client, 'Client') as client_class:
            client_class.return_value.list_runs.side_effect = error
            with mock.patch('sys.argv',
                            ['kfp', 'run', 'list', '--page-token', 'old']):
                with contextlib.redirect_stderr(stderr):
                    with self.assertRaises(SystemExit) as exit_error:
                        __main__.main()
            self.assertEqual(exit_error.exception.code, 1)
            client_class.return_value.list_runs.assert_called_once()
            self.assertEqual(
                client_class.return_value.list_runs.call_args
                .kwargs['page_token'], 'old')
        self.assertIn('Remove --page-token', stderr.getvalue())

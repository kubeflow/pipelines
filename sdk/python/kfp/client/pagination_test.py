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

import json
import unittest
from unittest import mock

from kfp.client import is_pagination_restart_required
import kfp.server_api


class PaginationRestartTest(unittest.TestCase):

    def error(self, **changes):
        detail = {
            '@type': 'type.googleapis.com/google.rpc.ErrorInfo',
            'reason': 'PAGINATION_RESTART_REQUIRED',
            'domain': 'kubeflow.org',
        }
        detail.update(changes)
        error = kfp.server_api.ApiException(status=400)
        error.body = json.dumps({'code': 9, 'details': [detail]})
        return error

    def test_recognizes_structured_error_without_message(self):
        error = self.error()
        for body in (error.body, error.body.encode('utf-8')):
            with self.subTest(body_type=type(body).__name__):
                error.body = body
                self.assertTrue(is_pagination_restart_required(error))
                self.assertEqual(error.body, body)

    def test_requires_exact_identity(self):
        for change in ({
                '@type': 'google.rpc.ErrorInfo'
        }, {
                'reason': 'OTHER_ERROR'
        }, {
                'domain': 'other.org'
        }):
            with self.subTest(change=change):
                self.assertFalse(
                    is_pagination_restart_required(self.error(**change)))

    def test_unrelated_details_do_not_hide_matching_error(self):
        error = self.error()
        body = json.loads(error.body)
        body['details'].insert(
            0, {'@type': 'type.googleapis.com/google.rpc.Status'})
        error.body = json.dumps(body)
        self.assertTrue(is_pagination_restart_required(error))

    def test_rejects_unstructured_and_malformed_responses(self):
        error = self.error()
        for body in (None, b'\xff', '', '<html>Bad Request</html>', 'null',
                     '[]', '{"code": 9, "details": {}}',
                     '{"code": 9, "details": [null, 9, "bad"]}',
                     '{"code": 9, "message": "PAGINATION_RESTART_REQUIRED"}'):
            with self.subTest(body=body):
                error.body = body
                self.assertFalse(is_pagination_restart_required(error))

    def test_rejects_other_status_and_code(self):
        error = self.error()
        error.status = 403
        self.assertFalse(is_pagination_restart_required(error))
        error = self.error()
        body = json.loads(error.body)
        body['code'] = 3
        error.body = json.dumps(body)
        self.assertFalse(is_pagination_restart_required(error))
        self.assertFalse(
            is_pagination_restart_required(
                ValueError('PAGINATION_RESTART_REQUIRED')))

    def test_generated_http_response_preserves_error_details(self):
        response = mock.Mock(status=400, reason='Bad Request')
        response.data = self.error().body.encode('utf-8')
        response.getheaders.return_value = {}
        error = kfp.server_api.ApiException(http_resp=response)
        self.assertTrue(is_pagination_restart_required(error))
        self.assertEqual(error.body, response.data)

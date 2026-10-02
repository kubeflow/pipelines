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
import io
from pathlib import Path
import tempfile
import unittest
from unittest import mock
import urllib.error
import urllib.request

import kfp_http


class Response(io.BytesIO):
    status = 200


class ClientTest(unittest.TestCase):

    def client(self, body=b'{}'):
        client = kfp_http.Client('http://127.0.0.1:8888/pipeline')
        client._opener = mock.Mock()
        client._opener.open.return_value = Response(body)
        return client

    def test_endpoint_validation(self):
        for endpoint in ('http://example.com', 'http://localhost:8888',
                         'file:///tmp/token', 'https://user:secret@host',
                         'https://host?secret', 'https://host#secret',
                         'https://host?', 'https://host:bad',
                         'https://host:99999', 'https://host/../other',
                         'https://host/\\evil', ' https://host',
                         'https://host/\npath'):
            with self.subTest(endpoint=endpoint), self.assertRaisesRegex(
                    kfp_http.CollectionError, '^invalid_endpoint$'):
                kfp_http.Client(endpoint)
        for endpoint in ('https://example.com/pipeline',
                         'http://127.0.0.1:8888', 'http://[::1]:8888'):
            kfp_http.Client(endpoint)

    def test_token_and_prefix_and_encoded_query(self):
        with tempfile.TemporaryDirectory() as directory:
            token = Path(directory) / 'token'
            token.write_text('private-token\n')
            client = kfp_http.Client(
                'https://example.com/pipeline/', token_file=token)
        client._opener = mock.Mock()
        client._opener.open.return_value = Response(b'{"items": []}')
        self.assertEqual({'items': []},
                         client.get('/apis/v2beta1/recurringruns',
                                    {'page_token': 'a&b'}))
        request = client._opener.open.call_args.args[0]
        self.assertEqual(
            'https://example.com/pipeline/apis/v2beta1/recurringruns?page_token=a%26b',
            request.full_url)
        self.assertEqual('Bearer private-token',
                         request.get_header('Authorization'))
        self.assertEqual('GET', request.method)
        self.assertEqual(20, client._opener.open.call_args.kwargs['timeout'])

    def test_invalid_tokens_and_ca_are_sanitized(self):
        with tempfile.TemporaryDirectory() as directory:
            token = Path(directory) / 'token'
            for value in ('', 'secret\r\nInjected: value', 'x' * 17000):
                token.write_text(value)
                with self.assertRaisesRegex(kfp_http.CollectionError,
                                            '^invalid_token_file$'):
                    kfp_http.Client('https://host', token_file=token)
        with self.assertRaisesRegex(kfp_http.CollectionError,
                                    '^invalid_ca_file$'):
            kfp_http.Client('https://host', ca_file='/missing/private/path')

    def test_no_environment_proxy_and_tls_verification(self):
        with mock.patch.dict(
                'os.environ',
            {'HTTPS_PROXY': 'http://untrusted:8080'
            }), mock.patch('kfp_http.urllib.request.build_opener') as build:
            kfp_http.Client('https://host')
        proxy, redirect, https = build.call_args.args
        self.assertEqual({}, proxy.proxies)
        self.assertIsInstance(redirect, kfp_http._NoRedirect)
        self.assertTrue(https._context.check_hostname)
        self.assertEqual(2, https._context.verify_mode)

    def test_path_cannot_change_origin_or_escape_api(self):
        client = self.client()
        for path in ('https://other/apis/v2beta1/runs', '//other/path',
                     '/apis/v1beta1/runs', '/apis/v2beta1/../secret',
                     '/apis/v2beta1/%2e%2e/secret',
                     '/apis/v2beta1/runs?token=private',
                     '/apis/v2beta1/runs#fragment', '/apis/v2beta1/\\other',
                     '/apis/v2beta1/%0a'):
            with self.subTest(path=path), self.assertRaisesRegex(
                    kfp_http.CollectionError, '^invalid_api_path$'):
                client.get(path)
        client._opener.open.assert_not_called()

    def test_redirect_handler_never_builds_redirect_request(self):
        handler = kfp_http._NoRedirect()
        for code in (301, 302, 303, 307, 308):
            body = Response(b'private response')
            with self.assertRaisesRegex(kfp_http.CollectionError,
                                        '^redirect_refused$'):
                handler.redirect_request(
                    urllib.request.Request('https://host'), body, code,
                    'private', {}, 'https://other/private')
            self.assertTrue(body.closed)

    def test_errors_are_sanitized(self):
        for error, reason in (
            (TimeoutError('private token'), 'request_timeout'),
            (urllib.error.URLError('private token'), 'request_failed'),
            (urllib.error.HTTPError('https://host/private', 403, 'secret', {},
                                    io.BytesIO()), 'access_denied'),
            (urllib.error.HTTPError('https://host/private', 302, 'secret', {},
                                    io.BytesIO()), 'redirect_refused'),
        ):
            client = self.client()
            client._opener.open.side_effect = error
            with self.assertRaisesRegex(kfp_http.CollectionError,
                                        '^' + reason + '$'):
                client.get('/apis/v2beta1/runs')

    def test_json_object_required(self):
        for body in (b'[]', b'null', b'private invalid json', b'\xff'):
            with self.assertRaisesRegex(kfp_http.CollectionError,
                                        '^invalid_json_object$'):
                self.client(body).get('/apis/v2beta1/runs')

    def test_embedded_api_error_is_sanitized(self):
        with self.assertRaisesRegex(kfp_http.CollectionError, '^api_error$'):
            self.client(b'{"error": {"message": "private detail"}}').get(
                '/apis/v2beta1/recurringruns')

    def test_response_and_cumulative_limits(self):
        with mock.patch.object(kfp_http, 'MAX_RESPONSE_BYTES', 4):
            with self.assertRaisesRegex(kfp_http.CollectionError,
                                        '^response_limit_exceeded$'):
                self.client(b'{"a": 1}').get('/apis/v2beta1/runs')
        with mock.patch.object(kfp_http, 'MAX_TOTAL_BYTES', 3):
            client = self.client()
            self.assertEqual({}, client.get('/apis/v2beta1/runs'))
            client._opener.open.return_value = Response(b'{}')
            with self.assertRaisesRegex(kfp_http.CollectionError,
                                        '^total_response_limit_exceeded$'):
                client.get('/apis/v2beta1/runs')

    def test_request_budget_includes_failures(self):
        client = self.client()
        client._opener.open.side_effect = OSError('private')
        with mock.patch.object(kfp_http, 'MAX_REQUESTS', 1):
            with self.assertRaises(kfp_http.CollectionError):
                client.get('/apis/v2beta1/runs')
            with self.assertRaisesRegex(kfp_http.CollectionError,
                                        '^request_budget_exceeded$'):
                client.get('/apis/v2beta1/runs')
        self.assertEqual(1, client._opener.open.call_count)

    def test_elapsed_deadline(self):
        with mock.patch.object(kfp_http.time, 'monotonic', side_effect=[0, 21]):
            with self.assertRaisesRegex(kfp_http.CollectionError,
                                        '^request_timeout$'):
                self.client().get('/apis/v2beta1/runs')


if __name__ == '__main__':
    unittest.main()

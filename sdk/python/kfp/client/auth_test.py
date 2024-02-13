# Copyright 2023 The Kubeflow Authors
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

import os
from unittest.mock import MagicMock
from unittest.mock import patch

from absl.testing import parameterized
from kfp.client import auth


class TestAuth(parameterized.TestCase):

    def test_is_ipython_return_false(self):
        mock = MagicMock()
        with patch.dict('sys.modules', IPython=mock):
            mock.get_ipython.return_value = None
            self.assertFalse(auth.is_ipython())

    def test_is_ipython_return_true(self):
        mock = MagicMock()
        with patch.dict('sys.modules', IPython=mock):
            mock.get_ipython.return_value = 'Something'
            self.assertTrue(auth.is_ipython())

    def test_is_ipython_should_raise_error(self):
        mock = MagicMock()
        with patch.dict('sys.modules', mock):
            mock.side_effect = ImportError
            self.assertFalse(auth.is_ipython())

    @patch('builtins.input', lambda *args:
           'https://oauth2.example.com/auth?code=4/P7q7W91a-oMsCeLvIaQm6bTrgtp7'
          )
    @patch('kfp.client.auth.is_ipython', lambda *args: True)
    @patch.dict(os.environ, dict(), clear=True)
    def test_get_auth_code_from_ipython(self):
        token, redirect_uri = auth.get_auth_code('sample-client-id')
        self.assertEqual(token, '4/P7q7W91a-oMsCeLvIaQm6bTrgtp7')
        self.assertEqual(redirect_uri, 'http://localhost:9901')

    @patch('builtins.input', lambda *args:
           'https://oauth2.example.com/auth?code=4/P7q7W91a-oMsCeLvIaQm6bTrgtp7'
          )
    @patch('kfp.client.auth.is_ipython', lambda *args: False)
    @patch.dict(os.environ, {'SSH_CONNECTION': 'ENABLED'}, clear=True)
    def test_get_auth_code_from_remote_connection(self):
        token, redirect_uri = auth.get_auth_code('sample-client-id')
        self.assertEqual(token, '4/P7q7W91a-oMsCeLvIaQm6bTrgtp7')
        self.assertEqual(redirect_uri, 'http://localhost:9901')

    @patch('builtins.input', lambda *args:
           'https://oauth2.example.com/auth?code=4/P7q7W91a-oMsCeLvIaQm6bTrgtp7'
          )
    @patch('kfp.client.auth.is_ipython', lambda *args: False)
    @patch.dict(os.environ, {'SSH_CLIENT': 'ENABLED'}, clear=True)
    def test_get_auth_code_from_remote_client(self):
        token, redirect_uri = auth.get_auth_code('sample-client-id')
        self.assertEqual(token, '4/P7q7W91a-oMsCeLvIaQm6bTrgtp7')
        self.assertEqual(redirect_uri, 'http://localhost:9901')

    @patch('builtins.input', lambda *args: 'https://oauth2.example.com/auth')
    @patch('kfp.client.auth.is_ipython', lambda *args: False)
    @patch.dict(os.environ, {'SSH_CLIENT': 'ENABLED'}, clear=True)
    def test_get_auth_code_from_remote_client_missing_code(self):
        self.assertRaises(KeyError, auth.get_auth_code, 'sample-client-id')

    @patch('kfp.client.auth.get_auth_response_local', lambda *args:
           'https://oauth2.example.com/auth?code=4/P7q7W91a-oMsCeLvIaQm6bTrgtp7'
          )
    @patch('kfp.client.auth.is_ipython', lambda *args: False)
    @patch.dict(os.environ, dict(), clear=True)
    def test_get_auth_code_from_local(self):
        token, redirect_uri = auth.get_auth_code('sample-client-id')
        self.assertEqual(token, '4/P7q7W91a-oMsCeLvIaQm6bTrgtp7')
        self.assertEqual(redirect_uri, 'http://localhost:9901')

    @patch('kfp.client.auth.get_auth_response_local', lambda *args: None)
    @patch('kfp.client.auth.is_ipython', lambda *args: False)
    @patch.dict(os.environ, dict(), clear=True)
    def test_get_auth_code_from_local_empty_response(self):
        self.assertRaises(ValueError, auth.get_auth_code, 'sample-client-id')

    @patch('kfp.client.auth.get_auth_response_local',
           lambda *args: 'this-is-an-invalid-response')
    @patch('kfp.client.auth.is_ipython', lambda *args: False)
    @patch.dict(os.environ, dict(), clear=True)
    def test_get_auth_code_from_local_invalid_response(self):
        self.assertRaises(KeyError, auth.get_auth_code, 'sample-client-id')

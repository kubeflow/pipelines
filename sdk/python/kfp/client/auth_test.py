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
from unittest.mock import ANY
from unittest.mock import MagicMock
from unittest.mock import patch

from absl.testing import parameterized
import google.auth.compute_engine.credentials
import google.oauth2.credentials
import google.oauth2.service_account
from kfp.client import auth


class TestAuth(parameterized.TestCase):

    @patch('kfp.client.auth.google.auth.default')
    def test_user_credentials_skip_service_account_auth(self, mock_default):
        credentials = MagicMock(spec=google.oauth2.credentials.Credentials)
        mock_default.return_value = (credentials, 'project')

        self.assertIsNone(auth.get_service_account_credentials('audience'))
        credentials.refresh.assert_not_called()

    @parameterized.named_parameters(('compute_engine', True),
                                    ('service_account', False))
    @patch('kfp.client.auth.google.auth.iam.Signer')
    @patch('kfp.client.auth.google.auth.default')
    def test_service_account_credentials_use_modern_signer(
            self, compute_engine, mock_default, mock_iam_signer):
        credential_type = (
            google.auth.compute_engine.credentials.Credentials
            if compute_engine else google.oauth2.service_account.Credentials)
        credentials = MagicMock(spec=credential_type)
        credentials.service_account_email = None

        def refresh(request):
            credentials.service_account_email = 'test@example.iam.gserviceaccount.com'

        credentials.refresh.side_effect = refresh
        mock_default.return_value = (credentials, 'project')
        result = auth.get_service_account_credentials('audience')

        mock_default.assert_called_once_with(scopes=[auth.IAM_SCOPE])
        credentials.refresh.assert_called_once()
        self.assertEqual(result.service_account_email,
                         'test@example.iam.gserviceaccount.com')
        self.assertEqual(result._additional_claims,
                         {'target_audience': 'audience'})
        if compute_engine:
            mock_iam_signer.assert_called_once_with(
                ANY, credentials, 'test@example.iam.gserviceaccount.com')
            self.assertIs(result.signer, mock_iam_signer.return_value)
        else:
            mock_iam_signer.assert_not_called()
            self.assertIs(result.signer, credentials.signer)

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

    @patch(
        'builtins.input', lambda *args:
        'https://oauth2.example.com/auth?code=4/P7q7W91a-oMsCeLvIaQm6bTrgtp7')
    @patch('kfp.client.auth.is_ipython', lambda *args: True)
    @patch.dict(os.environ, dict(), clear=True)
    def test_get_auth_code_from_ipython(self):
        token, redirect_uri = auth.get_auth_code('sample-client-id')
        self.assertEqual(token, '4/P7q7W91a-oMsCeLvIaQm6bTrgtp7')
        self.assertEqual(redirect_uri, 'http://localhost:9901')

    @patch(
        'builtins.input', lambda *args:
        'https://oauth2.example.com/auth?code=4/P7q7W91a-oMsCeLvIaQm6bTrgtp7')
    @patch('kfp.client.auth.is_ipython', lambda *args: False)
    @patch.dict(os.environ, {'SSH_CONNECTION': 'ENABLED'}, clear=True)
    def test_get_auth_code_from_remote_connection(self):
        token, redirect_uri = auth.get_auth_code('sample-client-id')
        self.assertEqual(token, '4/P7q7W91a-oMsCeLvIaQm6bTrgtp7')
        self.assertEqual(redirect_uri, 'http://localhost:9901')

    @patch(
        'builtins.input', lambda *args:
        'https://oauth2.example.com/auth?code=4/P7q7W91a-oMsCeLvIaQm6bTrgtp7')
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

    @patch(
        'kfp.client.auth.get_auth_response_local', lambda *args:
        'https://oauth2.example.com/auth?code=4/P7q7W91a-oMsCeLvIaQm6bTrgtp7')
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

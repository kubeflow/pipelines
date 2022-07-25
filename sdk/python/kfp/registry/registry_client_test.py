# Copyright 2022 The Kubeflow Authors
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
"""Tests for KFP Registry RegistryClient."""

import json
import os
from unittest import mock

from absl.testing import parameterized
from kfp.registry import ApiAuth
from kfp.registry import RegistryClient
import requests

_DEFAULT_HOST = 'https://us-central1-kfp.pkg.dev/proj/repo'
_KFP_CONFIG_FILE = os.path.join(
    os.path.dirname(__file__), 'context/kfp_pkg_dev.json')


class RegistryClientTest(parameterized.TestCase):

    def setUp(self):
        super(RegistryClientTest, self).setUp()
        with open(_KFP_CONFIG_FILE, 'r') as f:
            read_value = f.read()
        self._mock_open = self.enter_context(
            mock.patch(
                'builtins.open',
                side_effect=[
                    mock.mock_open(read_data=read_value).return_value,
                    mock.mock_open(read_data='file_content').return_value
                ]))

    def test_is_ar_host_kfp(self):
        client = RegistryClient(
            host='https://us-central1-kfp.pkg.dev/proj/repo', auth=ApiAuth(''))
        self.assertEqual(client._is_ar_host(), True)

    def test_is_ar_host_false(self):
        client = RegistryClient(
            host='https://hub.docker.com/r/google/cloud-sdk', auth=ApiAuth(''))
        self.assertFalse(client._is_ar_host())

    def test_load_config(self):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        expected_config = {
            'host':
                host,
            'upload_url':
                host,
            'download_version_url':
                f'{host}/{{package_name}}/{{version}}',
            'download_tag_url':
                f'{host}/{{package_name}}/{{tag}}',
            'get_package_url':
                ('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}'),
            'list_packages_url':
                ('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages'),
            'delete_package_url':
                ('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}'),
            'get_tag_url':
                ('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}/tags/{tag}'),
            'list_tags_url':
                ('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}/tags'),
            'delete_tag_url':
                ('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}/tags/{tag}'),
            'create_tag_url':
                ('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}/tags?tagId={tag}'),
            'update_tag_url':
                ('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}/tags/{tag}?updateMask=version'),
            'get_version_url':
                ('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}/versions/{version}?view=FULL'),
            'list_versions_url':
                ('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}/versions?view=FULL'),
            'delete_version_url':
                ('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}/versions/{version}'),
            'package_format':
                ('projects/proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}'),
            'tag_format': ('projects/proj/locations/us-central1/repositories'
                           '/repo/packages/{package_name}/tags/{tag}'),
            'version_format':
                ('projects/proj/locations/us-central1/repositories'
                 '/repo/packages/{package_name}/versions/{version}')
        }
        self.assertEqual(self._mock_open.call_args_list[0][0],
                         (_KFP_CONFIG_FILE, 'r'))
        self.assertEqual(expected_config, client._config)

    @parameterized.parameters(
        {
            'version':
                'sha256:abcde12345',
            'tag':
                None,
            'file_name':
                None,
            'expected_url':
                'https://us-central1-kfp.pkg.dev/proj/repo/pack/sha256:abcde12345',
            'expected_file_name':
                'pack_abcde12345.yaml'
        },
        {
            'version':
                None,
            'tag':
                'tag1',
            'file_name':
                None,
            'expected_url':
                'https://us-central1-kfp.pkg.dev/proj/repo/pack/tag1',
            'expected_file_name':
                'pack_tag1.yaml'
        },
        {
            'version':
                None,
            'tag':
                'tag1',
            'file_name':
                'pipeline.yaml',
            'expected_url':
                'https://us-central1-kfp.pkg.dev/proj/repo/pack/tag1',
            'expected_file_name':
                'pipeline.yaml'
        },
    )
    @mock.patch('requests.get', autospec=True)
    def test_download_pipeline(self, mock_get, version, tag, file_name,
                               expected_url, expected_file_name):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.download_pipeline(
            package_name='pack', version=version, tag=tag, file_name=file_name)
        mock_get.assert_called_once_with(
            url=expected_url, data='', headers=None, auth=mock.ANY)
        self.assertEqual(self._mock_open.call_args_list[1][0],
                         (expected_file_name, 'wb'))

    def test_download_pipeline_version_error(self):
        client = RegistryClient(host=_DEFAULT_HOST, auth=ApiAuth(''))
        with self.assertRaises(ValueError):
            client.download_pipeline(package_name='pack', version='abcde12345')

    def test_download_pipeline_tag_error(self):
        client = RegistryClient(host=_DEFAULT_HOST, auth=ApiAuth(''))
        with self.assertRaises(ValueError):
            client.download_pipeline(
                package_name='pack', tag='sha256:abcde12345')

    @parameterized.parameters(
        {
            'tags': 'tag1',
            'expected_tags': 'tag1'
        },
        {
            'tags': ['tag1', 'tag2'],
            'expected_tags': 'tag1,tag2'
        },
    )
    @mock.patch('requests.post', autospec=True)
    def test_upload_pipeline(self, mock_post, tags, expected_tags):
        mock_post.return_value.text = 'package_name/sha256:abcde12345'
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        package_name, version = client.upload_pipeline(
            file_name='pipeline.yaml',
            tags=tags,
            extra_headers={'description': 'nothing'})
        mock_post.assert_called_once_with(
            url=host,
            data={'tags': expected_tags},
            headers={'description': 'nothing'},
            files={'content': mock.ANY},
            auth=mock.ANY)
        self.assertEqual(self._mock_open.call_args_list[1][0],
                         ('pipeline.yaml', 'rb'))
        self.assertEqual(package_name, 'package_name')
        self.assertEqual(version, 'sha256:abcde12345')

    @mock.patch('requests.get', autospec=True)
    def test_get_package(self, mock_get):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.get_package('pack')
        mock_get.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.get', autospec=True)
    def test_list_packages(self, mock_get):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.list_packages()
        mock_get.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.get', autospec=True)
    def test_list_packages_empty(self, mock_get):
        host = _DEFAULT_HOST
        mock_response = requests.Response()
        mock_response.status_code = 200
        mock_response._content = json.dumps({}).encode('utf-8')
        mock_get.return_value = mock_response
        client = RegistryClient(host=host, auth=ApiAuth(''))
        packages = client.list_packages()
        self.assertEqual(packages, {})
        mock_get.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.delete', autospec=True)
    def test_delete_package(self, mock_delete):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.delete_package('pack')
        mock_delete.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.get', autospec=True)
    def test_get_version(self, mock_get):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.get_version('pack', 'sha256:abcde12345')
        mock_get.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack/versions/sha256:abcde12345?view=FULL'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.get', autospec=True)
    def test_list_versions(self, mock_get):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.list_versions('pack')
        mock_get.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack/versions?view=FULL'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.get', autospec=True)
    def test_list_versions_empty(self, mock_get):
        host = _DEFAULT_HOST
        mock_response = requests.Response()
        mock_response.status_code = 200
        mock_response._content = json.dumps({}).encode('utf-8')
        mock_get.return_value = mock_response
        client = RegistryClient(host=host, auth=ApiAuth(''))
        versions = client.list_versions('pack')
        self.assertEqual(versions, {})
        mock_get.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack/versions?view=FULL'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.delete', autospec=True)
    def test_delete_version(self, mock_delete):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.delete_version('pack', 'sha256:abcde12345')
        mock_delete.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack/versions/sha256:abcde12345'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.get', autospec=True)
    def test_get_tag(self, mock_get):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.get_tag('pack', 'tag1')
        mock_get.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack/tags/tag1'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.get', autospec=True)
    def test_list_tags(self, mock_get):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.list_tags('pack')
        mock_get.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack/tags'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.get', autospec=True)
    def test_list_tags_empty(self, mock_get):
        host = _DEFAULT_HOST
        mock_response = requests.Response()
        mock_response.status_code = 200
        mock_response._content = json.dumps({}).encode('utf-8')
        mock_get.return_value = mock_response
        client = RegistryClient(host=host, auth=ApiAuth(''))
        tags = client.list_tags('pack')
        self.assertEqual(tags, {})
        mock_get.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack/tags'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.delete', autospec=True)
    def test_delete_tag(self, mock_delete):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.delete_tag('pack', 'tag1')
        mock_delete.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack/tags/tag1'),
            data='',
            headers=None,
            auth=mock.ANY)

    @mock.patch('requests.post', autospec=True)
    def test_create_tag(self, mock_post):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.create_tag('pack', 'sha256:abcde12345', 'tag1')
        expected_data = json.dumps({
            'name':
                '',
            'version': ('projects/proj/locations/us-central1/repositories'
                        '/repo/packages/pack/versions/sha256:abcde12345')
        })
        mock_post.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack/tags?tagId=tag1'),
            data=expected_data,
            headers={
                'Content-type': 'application/json',
            },
            auth=mock.ANY)

    @mock.patch('requests.patch', autospec=True)
    def test_update_tag(self, mock_patch):
        host = _DEFAULT_HOST
        client = RegistryClient(host=host, auth=ApiAuth(''))
        client.update_tag('pack', 'sha256:abcde12345', 'tag1')
        expected_data = json.dumps({
            'name':
                '',
            'version': ('projects/proj/locations/us-central1/repositories'
                        '/repo/packages/pack/versions/sha256:abcde12345')
        })
        mock_patch.assert_called_once_with(
            url=('https://artifactregistry.googleapis.com/v1/projects/'
                 'proj/locations/us-central1/repositories'
                 '/repo/packages/pack/tags/tag1?updateMask=version'),
            data=expected_data,
            headers={
                'Content-type': 'application/json',
            },
            auth=mock.ANY)

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
"""Class for KFP Registry Client."""

import logging

import google.auth
import json
import requests
import re
from google.auth.credentials import Credentials
from typing import Any, Optional, List, Tuple, Union

_KNOWN_HOSTS_REGEX = {
    "kfp_pkg_dev":
        r'(^https\:\/\/(?P<location>[\w\-]+)\-kfp\.pkg\.dev\/(?P<project_id>.*)\/(?P<repo_id>.*))',
}

_DEFAULT_JSON_HEADER = {
    'Content-type': 'application/json',
}


class _SafeDict(dict):

    def __missing__(self, key: str):
        return '{' + key + '}'


class ApiAuth(requests.auth.AuthBase):

    def __init__(self, token: str):
        self._token = token

    def __call__(self, request: requests.Request):
        request.headers['authorization'] = 'Bearer ' + self._token
        return request


class RegistryClient:

    def __init__(self,
                 host: str,
                 auth: Optional[Union[requests.auth.AuthBase,
                                      Credentials]] = None):
        self._host = host.rstrip('/')
        self._known_host_key = ""
        for key in _KNOWN_HOSTS_REGEX.keys():
            if re.match(_KNOWN_HOSTS_REGEX[key], self._host):
                self._known_host_key = key
                break
        self._config = self.load_config()
        if auth:
            self._auth = auth
        elif self._is_ar_host():
            self._auth, _ = google.auth.default()

    def _request(self,
                 request_url: str,
                 request_body: str = '',
                 http_request: str = 'get',
                 extra_headers: dict = None) -> Any:
        """Call the HTTP request"""
        self._refresh_creds()
        auth = self._get_auth()
        http_request_fn = getattr(requests, http_request)

        response = http_request_fn(
            url=request_url,
            data=request_body,
            headers=extra_headers,
            auth=auth)
        response.raise_for_status()

        return response

    def _is_ar_host(self):
        return self._known_host_key == "kfp_pkg_dev"

    def _is_known_host(self) -> bool:
        return bool(self._known_host_key)

    def load_config(self):
        config = {}
        if self._is_ar_host():
            repo_resource_format = ''
            try:
                matched = re.match(_KNOWN_HOSTS_REGEX[self._known_host_key],
                                   self._host)
                repo_resource_format = ('projects/'
                                        '{project_id}/locations/{location}/'
                                        'repositories/{repo_id}'.format_map(
                                            _SafeDict(matched.groupdict())))
            except AttributeError:
                raise ValueError('Invalid host URL')
            registry_endpoint = 'https://artifactregistry.googleapis.com/v1'
            api_endpoint = f'{registry_endpoint}/{repo_resource_format}'
            package_endpoint = f'{api_endpoint}/packages'
            package_name_endpoint = f'{package_endpoint}/{{package_name}}'
            tags_endpoint = f'{package_name_endpoint}/tags'
            versions_endpoint = f'{package_name_endpoint}/versions'
            config = {
                'host':
                    self._host,
                'upload_url':
                    self._host,
                'download_version_url':
                    f'{self._host}/{{package_name}}/{{version}}',
                'download_tag_url':
                    f'{self._host}/{{package_name}}/{{tag}}',
                'get_package_url':
                    f'{package_name_endpoint}',
                'list_packages_url':
                    package_endpoint,
                'delete_package_url':
                    f'{package_name_endpoint}',
                'get_tag_url':
                    f'{tags_endpoint}/{{tag}}',
                'list_tags_url':
                    f'{tags_endpoint}',
                'delete_tag_url':
                    f'{tags_endpoint}/{{tag}}',
                'create_tag_url':
                    f'{tags_endpoint}?tagId={{tag}}',
                'update_tag_url':
                    f'{tags_endpoint}/{{tag}}?updateMask=version',
                'get_version_url':
                    f'{versions_endpoint}/{{version}}',
                'list_versions_url':
                    f'{versions_endpoint}',
                'delete_version_url':
                    f'{versions_endpoint}/{{version}}',
                'package_format':
                    f'{repo_resource_format}/packages/{{package_name}}',
                'tag_format':
                    f'{repo_resource_format}/packages/{{package_name}}/tags/{{tag}}',
                'version_format':
                    f'{repo_resource_format}/packages/{{package_name}}/versions/{{version}}',
            }
        else:
            logging.info(f'load_config not implemented for host: {self._host}')
        return config

    def _get_auth(self):
        auth = self._auth
        if isinstance(auth, Credentials):
            auth = ApiAuth(auth.token)
        return auth

    def _refresh_creds(self):
        if self._is_ar_host() and isinstance(
                self._auth, Credentials) and not self._auth.valid:
            self._auth.refresh(google.auth.transport.requests.Request())

    def upload_pipeline(self, file_name: str, tags: Optional[Union[str,
                                                                   List[str]]],
                        extra_headers: Optional[dict]) -> Tuple[str, str]:
        url = self._config['upload_url']
        self._refresh_creds()
        auth = self._get_auth()
        request_body = {}
        if tags:
            if isinstance(tags, str):
                request_body = {'tags': tags}
            elif isinstance(tags, List):
                request_body = {'tags': ','.join(tags)}

        files = {'content': open(file_name, 'rb')}
        response = requests.post(
            url=url,
            data=request_body,
            headers=extra_headers,
            files=files,
            auth=auth)
        response.raise_for_status()
        [package_name, version] = response.text.split('/')

        return (package_name, version)

    def _get_download_url(self,
                          package_name: str,
                          version: Optional[str] = None,
                          tag: Optional[str] = None) -> str:
        if (not version) and (not tag):
            raise ValueError('Either version or tag must be specified.')
        if version:
            url = self._config['download_version_url'].format(
                package_name=package_name, version=version)
        if tag:
            if version:
                logging.info(
                    'Both version and tag are specified, using version only.')
            else:
                url = self._config['download_tag_url'].format(
                    package_name=package_name, tag=tag)
        return url

    def download_pipeline(self,
                          package_name: str,
                          version: Optional[str] = None,
                          tag: Optional[str] = None,
                          file_name: str = None) -> str:
        url = self._get_download_url(package_name, version, tag)
        response = self._request(request_url=url)

        if not file_name:
            file_name = package_name + '_'
            if version:
                file_name += version[len('sha256:'):]
            elif tag:
                file_name += tag
            file_name += '.yaml'

        with open(file_name, 'wb') as f:
            f.write(response.content)

        return file_name

    def get_package(self, package_name: str) -> dict:
        url = self._config['get_package_url'].format(package_name=package_name)
        response = self._request(request_url=url)

        return response.json()

    def list_packages(self) -> List[dict]:
        url = self._config['list_packages_url']
        response = self._request(request_url=url)

        return response.json()

    def delete_package(self, package_name: str) -> bool:
        url = self._config['delete_package_url'].format(
            package_name=package_name)
        response = self._request(request_url=url, http_request='delete')
        response_json = response.json()

        return response_json['done']

    def get_version(self, package_name: str, version: str) -> dict:
        url = self._config['get_version_url'].format(
            package_name=package_name, version=version)
        response = self._request(request_url=url)

        return response.json()

    def list_versions(self, package_name: str) -> List[dict]:
        url = self._config['list_versions_url'].format(
            package_name=package_name)
        response = self._request(request_url=url)

        return response.json()

    def delete_version(self, package_name: str, version: str) -> bool:
        url = self._config['delete_version_url'].format(
            package_name=package_name, version=version)
        response = self._request(request_url=url, http_request='delete')
        response_json = response.json()

        return response_json['done']

    def create_tag(self, package_name: str, version: str, tag: str) -> dict:
        url = self._config['create_tag_url'].format(
            package_name=package_name, tag=tag)
        new_tag = {
            'name':
                '',
            'version':
                self._config['version_format'].format(
                    package_name=package_name, version=version)
        }
        response = self._request(
            request_url=url,
            request_body=json.dumps(new_tag),
            http_request='post',
            extra_headers=_DEFAULT_JSON_HEADER)

        return response.json()

    def get_tag(self, package_name: str, tag: str) -> dict:
        url = self._config['get_tag_url'].format(
            package_name=package_name, tag=tag)
        response = self._request(request_url=url)

        return response.json()

    def update_tag(self, package_name: str, version: str, tag: str) -> dict:
        url = self._config['update_tag_url'].format(
            package_name=package_name, tag=tag)
        new_tag = {
            'name':
                '',
            'version':
                self._config['version_format'].format(
                    package_name=package_name, version=version)
        }
        response = self._request(
            request_url=url,
            request_body=json.dumps(new_tag),
            http_request='patch',
            extra_headers=_DEFAULT_JSON_HEADER)

        return response.json()

    def list_tags(self, package_name: str) -> List[dict]:
        url = self._config['list_tags_url'].format(package_name=package_name)
        response = self._request(request_url=url)

        return response.json()

    def delete_tag(self, package_name: str, tag: str) -> bool:
        url = self._config['delete_tag_url'].format(
            package_name=package_name, tag=tag)
        response = self._request(request_url=url, http_request='delete')

        return response.json()

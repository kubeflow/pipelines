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
from kfp.components import base_component, load_component_from_text
from typing import Any
from google.protobuf import json_format

_AR_HOST_TEMPLATE_GENERIC = r'(^https\:\/\/[\w\-]+\-kfp\.pkg\.dev\/.*$)'
_AR_HOST_TEMPLATE_GROUPS = r'(https://(?P<location>.*)-kfp.pkg.dev/(?P<project_id>.*)/(?P<repo_id>.*))'
AR_HOST_SUFFIX = 'pkg.dev'

class _SafeDict(dict):
    def __missing__(self, key):
        return '{' + key + '}'

class Client:
    def __init__(self,
        host: str,
        credentials: Optional[Any] = None
    ):
        # host_split = self._host.split('/')
        # self._repo_id = host_split[-1]
        # self._project_id = host_split[-2]
        self._config = self.load_config(host)
        if credentials:
            self._credentials = credentials
        elif self._is_ar_host(host):
            logger = logging.getLogger("google.auth._default")
            logging_warning_filter = utils.LoggingFilter(logging.WARNING)
            logger.addFilter(logging_warning_filter)
            self._creds, _ = google.auth.default()
            logger.removeFilter(logging_warning_filter)

    def _is_ar_host(self, host: str):
        return bool(re.match(_AR_HOST_TEMPLATE_GENERIC, host.rstrip('/')))

    def _request(self, request_url: str, request_body: str = '',
                http_request: str = 'get', extra_headers: str = '') -> Any:
        """Call the HTTP request"""
        if not self._creds.valid:
            self._creds.refresh(google.auth.transport.requests.Request())
        headers = {
            'Content-type': 'application/json',
            'Authorization': 'Bearer ' + self._creds.token,
        }

        http_request_fn = getattr(requests, http_request)
        response = http_request_fn(
            url=request_url, data=request_body, headers=headers).json()
        response.raise_for_status()

        return response

    def load_config(self, host: str):
        config = {}
        host = host.rstrip('/')
        if self._is_ar_host(host):
            try:
                matched = re.match(_AR_HOST_TEMPLATE_GROUPS, host)
                registry_endpoint = 'https://artifactregistry.googleapis.com/v1/'
                repo_resource_format = ('projects/'
                    '{project_id}/locations/{location}/'
                    'repositories/{repo_id}'.format_map(
                        _SafeDict(matched.groupdict())))
                api_endpoint = f'{registry_endpoint}/{repo_resource_format}'
                package_endpoint = f'{api_endpoint}/packages'
                package_name_endpoint = f'{package_endpoint}/{{package_name}}'
                tags_endpoint = f'{package_name_endpoint}/tags'
                versions_endpoint = f'{package_name_endpoint}/versions'
                config = {
                    'host': host,
                    'upload_url': host,
                    'download_version_url': f'{host}/{{package_name}}/sha256:{{version}}',
                    'download_tag_url': f'{host}/{{package_name}}/{{tag}}',
                    'get_package_url': f'{package_name_endpoint}',
                    'list_packages_url': f'{package_endpoint}/',
                    'delete_package_url': f'{package_name_endpoint}',
                    'get_tag_url': f'{tags_endpoint}/{{tag}}',
                    'list_tags_url': f'{tags_endpoint}/',
                    'delete_tag_url': f'{tags_endpoint}/{{tag}}',
                    'create_tag_url': f'{tags_endpoint}?tagId={{tag}}',
                    'update_tag_url': f'{tags_endpoint}/{{tag}}?updateMask=version',
                    'get_version_url': f'{versions_endpoint}/{{version}}',
                    'list_versions_url': f'{versions_endpoint}/',
                    'delete_version_url': f'{versions_endpoint}/{{version}}',
                    'package_format': f'{repo_resource_format}/packages/{{package_name}}',
                    'tag_format': f'{repo_resource_format}/packages/{{package_name}}/tags/{{tag}}',
                    'version_format': f'{repo_resource_format}/packages/{{package_name}}/versions/{{version}}',
                }
            except AttributeError as err:
                raise ValueError('Invalid host URL')
        else:
            raise ValueError(f'load_config not implemented for host: {host}')
        return config

    def upload_component_template(self, file_name: str, tags: Optional[List[str]],    
                    extra_headers: Optional[dict]) -> Tuple[str, str]:
        url = self._config['upload_url']
        if not self._creds.valid:
            self._creds.refresh(google.auth.transport.requests.Request())
        headers = {
            'Authorization': 'Bearer ' + self._creds.token,
            **extra_headers,
        }
        request_body = {}
        if tags:
            request_body = {'tags': ','.join(tags)}

        files = {'content': open(file_name, 'rb')}
        response = requests.post(
            url=url, data=request_body, headers=headers, files=files).json()
        response.raise_for_status()

        return response

    def _get_download_url(package_name: str,
                      version: Optional[str] = None,
                      tag: Optional[str] = None) -> str:
        if (not version) and (not tag):
            raise ValueError('Either version or tag must be specified.')
        if version:
            if version.startswith('sha256:'):
                version = version[len('sha256:'):]
            url = self._config['download_version_url'].format(**locals())
        if tag:
            if version:
                logging.info(
                    'Both version and tag are specified, using version only.')
            else:
                url = self._config['download_tag_url'].format(**locals())
        return url

    def download_component_template(self, package_name: str,
                      version: Optional[str] = None,
                      tag: Optional[str] = None,
                      file_name: str = None) -> str:
        url = self._get_download_url(package_name, version, tag)
        response = self._request(request_url=url)

        if not file_name:
            file_name = package_name + '_'
            if version:
                if version.startswith('sha256:'):
                    file_name += version[len('sha256:'):]
                else:
                    file_name += version
            elif tag:
                file_name += tag
            file_name += '.yaml'

        with open(file_name, 'w') as f:
            f.write(response.content)

        return file_name


    def load_component_template_from_registry(self, package_name: str,
                                version: Optional[str] = None,
                                tag: Optional[str] = None
                                ) -> base_component.BaseComponent:
        """Loads component template from Registry.
        """
        url = self._get_download_url(package_name, version, tag)
        response = self._request(request_url=url)

        return load_component_from_text(response.content)

    def get_package(self, package_name: str) -> package_pb2.Package:
        url = self._config['get_package_url'].format(**locals())
        response = self._request(request_url=url)

        return package_pb2.Package.from_json(response.text)

    def list_packages(self, package_name: str) -> List[package_pb2.Package]:
        url = self._config['list_packages_url'].format(**locals())
        response = self._request(request_url=url)

        return [package_pb2.Package.from_json(json.dumps(package)) for package in response.json()]

    def delete_package(self, package_name: str) -> bool:
        url = self._config['delete_package_url'].format(**locals())
        response = self._request(request_url=url, http_request='delete')
        response_json = response.json()

        return response_json['done']

    def get_version(self, package_name: str, version: str) -> version_pb2.Version:
        url = self._config['get_version_url'].format(**locals())
        response = self._request(request_url=url)

        return version_pb2.Version.from_json(response.text)

    def list_versions(self, package_name: str) -> List[version_pb2.Version]:
        url = self._config['list_versions_url'].format(**locals())
        response = self._request(request_url=url)

        return [version_pb2.Version.from_json(json.dumps(version)) for version in response.json()]

    def delete_version(self, package_name: str, version: str) -> bool:
        url = self._config['delete_version_url'].format(**locals())
        response = self._request(request_url=url, http_request='delete')
        response_json = response.json()

        return response_json['done']

    def create_tag(self, package_name: str, version: str, tag: str) -> tag_pb2.Tag:
        url = self._config['update_tag_url'].format(**locals())
        new_tag = tag_pb2.Tag(
            name=self._config['tag_resource_format'].format(**locals()),
            version=self._config['version_resource_format'].format(**locals())
        )
        response = self._request(
            request_url=url,
            request_body=tag_pb2.Tag.to_dict(new_tag),
            http_request='patch'
        )

        return tag_pb2.Tag.from_json(response.text)

    def get_tag(self, package_name: str, tag: str) -> tag_pb2.Tag:
        url = self._config['get_tag_url'].format(**locals())
        response = self._request(request_url=url)

        return tag_pb2.Tag.from_json(response.text)

    def update_tag(self, package_name: str, version: str, tag: str) -> tag_pb2.Tag:
        url = self._config['update_tag_url'].format(**locals())
        new_tag = tag_pb2.Tag(
            name=self._config['tag_resource_format'].format(**locals()),
            version=''
        )
        response = self._request(
            request_url=url,
            request_body=tag_pb2.Tag.to_dict(new_tag),
            http_request='post'
        )

        return tag_pb2.Tag.from_json(response.text)

    def list_tags(self, package_name: str, tag: str) -> List[tag_pb2.Tag]:
        url = self._config['list_tags_url'].format(**locals())
        response = self._request(request_url=url)

        return tag_pb2.Tag.from_json(response.text)

    def delete_tag(self, package_name: str, tag: str) -> bool:
        url = self._config['delete_tag_url'].format(**locals())
        response = self._request(request_url=url, http_request='delete')
        response_json = response.json()

        return response_json['done']

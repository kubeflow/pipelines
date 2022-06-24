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

import json
import logging
import re
from typing import Any, Dict, List, Optional, Tuple, Union

import google.auth
import requests
from google.auth import credentials

_KNOWN_HOSTS_REGEX = {
    'kfp_pkg_dev':
        r'(^https\:\/\/(?P<location>[\w\-]+)\-kfp\.pkg\.dev\/(?P<project_id>.*)\/(?P<repo_id>.*))',
}

_DEFAULT_JSON_HEADER = {
    'Content-type': 'application/json',
}

_VERSION_PREFIX = 'sha256:'


class _SafeDict(dict):
    """Class for safely handling missing keys in .format_map."""

    def __missing__(self, key: str) -> str:
        """Handle missing keys by adding them back.

        Args:
            key: The key itself.

        Returns:
            The key with curly braces.
        """
        return '{' + key + '}'


class ApiAuth(requests.auth.AuthBase):
    """Class for authentication using API token."""

    def __init__(self, token: str) -> None:
        self._token = token

    def __call__(self,
                 request: requests.PreparedRequest) -> requests.PreparedRequest:
        request.headers['authorization'] = 'Bearer ' + self._token
        return request


class RegistryClient:
    """Registry Client class for communicating with registry hosts."""

    def __init__(
        self,
        host: str,
        auth: Optional[Union[requests.auth.AuthBase,
                             credentials.Credentials]] = None
    ) -> None:
        """Initializes the RegistryClient.

        Args:
            host: The address of the registry host.
            auth: Authentication using python requests or google.auth credentials.
        """
        self._host = host.rstrip('/')
        self._known_host_key = ''
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
                 request_body: Optional[str] = '',
                 http_request: Optional[str] = 'get',
                 extra_headers: Optional[dict] = None) -> requests.Response:
        """Calls the HTTP request.

        Args:
            request_url: The address of the API endpoint to send the request to.
            request_body: Body of the request.
            http_request: Type of HTTP request (post, get, delete etc, defaults to get).
            extra_headers: Any extra headers required.

        Returns:
            Response from the request.
        """
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

    def _is_ar_host(self) -> bool:
        """Checks if the host is on Artifact Registry.

        Returns:
            Whether the host is on Artifact Registry.
        """
        # TODO: Handle multiple known hosts.
        return self._known_host_key == 'kfp_pkg_dev'

    def _is_known_host(self) -> bool:
        """Checks if the host is a known host.

        Returns:
            Whether the host is a known host.
        """
        return bool(self._known_host_key)

    def _validate_version(self, version: str) -> None:
        """Validates the version.

        Args:
            version: Version of the package.
        """
        if not version.startswith(_VERSION_PREFIX):
            raise ValueError('Version should start with \"sha256:\".')

    def _validate_tag(self, tag: str) -> None:
        """Validates the tag.

        Args:
            tag: Tag attached to the package.
        """
        if tag.startswith(_VERSION_PREFIX):
            raise ValueError('Tag should not start with \"sha256:\".')

    def load_config(self) -> dict:
        """Loads the config.

        Returns:
            The loaded config if it's a known host, otherwise None.
        """
        # TODO: Move config file code to its own file.
        config = {}
        if self._is_ar_host():
            repo_resource_format = ''
            matched = re.match(_KNOWN_HOSTS_REGEX[self._known_host_key],
                               self._host)
            if matched:
                repo_resource_format = ('projects/'
                                        '{project_id}/locations/{location}/'
                                        'repositories/{repo_id}'.format_map(
                                            _SafeDict(matched.groupdict())))
            else:
                raise ValueError(f'Invalid host URL: {self._host}.')
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

    def _get_auth(self) -> requests.auth.AuthBase:
        """Helper function to convert google credentials to AuthBase class if
        needed.

        Returns:
            An instance of the AuthBase class
        """
        auth = self._auth
        if isinstance(auth, credentials.Credentials):
            auth = ApiAuth(auth.token)
        return auth

    def _refresh_creds(self) -> None:
        """Helper function to refresh google credentials if needed."""
        if self._is_ar_host() and isinstance(
                self._auth, credentials.Credentials) and not self._auth.valid:
            self._auth.refresh(google.auth.transport.requests.Request())

    def upload_pipeline(
            self,
            file_name: str,
            tags: Optional[Union[str, List[str]]] = None,
            extra_headers: Optional[dict] = None) -> Tuple[str, str]:
        """Uploads the pipeline.

        Args:
            file_name: The name of the file to be uploaded.
            tags: Tags to be attached to the uploaded pipeline.
            extra_headers: Any extra headers required.

        Returns:
            A tuple representing the package name and the version
        """
        url = self._config['upload_url']
        self._refresh_creds()
        auth = self._get_auth()
        request_body = {}
        if tags:
            if isinstance(tags, str):
                request_body = {'tags': tags}
            elif isinstance(tags, List):
                request_body = {'tags': ','.join(tags)}

        with open(file_name, 'rb') as f:
            files = {'content': f}
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
        """Gets the download url based on version or tag (either one must be
        specified).

        Args:
            package_name: Name of the package.
            version: Version of the package.
            tag: Tag attached to the package.

        Returns:
            A url for downloading the package.
        """
        if (not version) and (not tag):
            raise ValueError('Either version or tag must be specified.')
        if version:
            self._validate_version(version)
            url = self._config['download_version_url'].format(
                package_name=package_name, version=version)
        if tag:
            if version:
                logging.info(
                    'Both version and tag are specified, using version only.')
            else:
                self._validate_tag(tag)
                url = self._config['download_tag_url'].format(
                    package_name=package_name, tag=tag)
        return url

    def download_pipeline(self,
                          package_name: str,
                          version: Optional[str] = None,
                          tag: Optional[str] = None,
                          file_name: Optional[str] = None) -> str:
        """Downloads a pipeline - either version or tag must be specified.

        Args:
            package_name: Name of the package.
            version: Version of the package.
            tag: Tag attached to the package.
            file_name: File name to be saved as. If not specified, the 
                file name will be based on the package name and version/tag.

        Returns:
            The file name of the downloaded pipeline.
        """
        url = self._get_download_url(package_name, version, tag)
        response = self._request(request_url=url)

        if not file_name:
            file_name = package_name + '_'
            if version:
                self._validate_version(version)
                file_name += version[len(_VERSION_PREFIX):]
            elif tag:
                self._validate_tag(tag)
                file_name += tag
            file_name += '.yaml'

        with open(file_name, 'wb') as f:
            f.write(response.content)

        return file_name

    def get_package(self, package_name: str) -> Dict[str, Any]:
        """Gets package metadata.

        Args:
            package_name: Name of the package.

        Returns:
            The package metadata.
        """
        url = self._config['get_package_url'].format(package_name=package_name)
        response = self._request(request_url=url)

        return response.json()

    def list_packages(self) -> List[dict]:
        """Lists packages.

        Returns:
            List of packages in the repository.
        """
        url = self._config['list_packages_url']
        response = self._request(request_url=url)
        response_json = response.json()

        return response_json.get('packages', {})

    def delete_package(self, package_name: str) -> bool:
        """Deletes a package.

        Args:
            package_name: Name of the package.

        Returns:
            Whether the package was deleted successfully.
        """
        url = self._config['delete_package_url'].format(
            package_name=package_name)
        response = self._request(request_url=url, http_request='delete')
        response_json = response.json()

        return response_json['done']

    def get_version(self, package_name: str, version: str) -> Dict[str, Any]:
        """Gets package version metadata.

        Args:
            package_name: Name of the package.
            version: Version of the package.

        Returns:
            The version metadata.
        """
        self._validate_version(version)
        url = self._config['get_version_url'].format(
            package_name=package_name, version=version)
        response = self._request(request_url=url)

        return response.json()

    def list_versions(self, package_name: str) -> List[dict]:
        """Lists package versions.

        Args:
            package_name: Name of the package.

        Returns:
            List of package versions.
        """
        url = self._config['list_versions_url'].format(
            package_name=package_name)
        response = self._request(request_url=url)
        response_json = response.json()

        return response_json.get('versions', {})

    def delete_version(self, package_name: str, version: str) -> bool:
        """Deletes package version.

        Args:
            package_name: Name of the package.
            version: Version of the package.

        Returns:
            Whether the version was deleted successfully.
        """
        self._validate_version(version)
        url = self._config['delete_version_url'].format(
            package_name=package_name, version=version)
        response = self._request(request_url=url, http_request='delete')
        response_json = response.json()

        return response_json['done']

    def create_tag(self, package_name: str, version: str,
                   tag: str) -> Dict[str, Any]:
        """Creates a tag on a package version.

        Args:
            package_name: Name of the package.
            version: Version of the package.
            tag: Tag to be attached to the package version.

        Returns:
            The metadata for the created tag.
        """
        self._validate_version(version)
        self._validate_tag(tag)
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

    def get_tag(self, package_name: str, tag: str) -> Dict[str, Any]:
        """Gets tag metadata.

        Args:
            package_name: Name of the package.
            tag: Tag attached to the package version.

        Returns:
            The metadata for the tag.
        """
        self._validate_tag(tag)
        url = self._config['get_tag_url'].format(
            package_name=package_name, tag=tag)
        response = self._request(request_url=url)

        return response.json()

    def update_tag(self, package_name: str, version: str,
                   tag: str) -> Dict[str, Any]:
        """Updates a tag to another package version.

        Args:
            package_name: Name of the package.
            version: Version of the package.
            tag: Tag to be attached to the new package version.

        Returns:
            The metadata for the updated tag.
        """
        self._validate_version(version)
        self._validate_tag(tag)
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
        """Lists package tags.

        Args:
            package_name: Name of the package.

        Returns:
            List of tags.
        """
        url = self._config['list_tags_url'].format(package_name=package_name)
        response = self._request(request_url=url)
        response_json = response.json()

        return response_json.get('tags', {})

    def delete_tag(self, package_name: str, tag: str) -> Dict[str, Any]:
        """Deletes package tag.

        Args:
            package_name: Name of the package.
            tag: Tag to be deleted.

        Returns:
            Response from the delete request.
        """
        self._validate_tag(tag)
        url = self._config['delete_tag_url'].format(
            package_name=package_name, tag=tag)
        response = self._request(request_url=url, http_request='delete')

        return response.json()

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
import os
import re
from typing import Any, Dict, List, Optional, Tuple, Union

import google.auth
from google.auth import credentials
import requests

_KNOWN_HOSTS_REGEX = {
    'kfp_pkg_dev': (
        r'(^https\:\/\/(?P<location>[\w\-]+)\-kfp\.pkg\.dev\/(?P<project_id>.*)\/(?P<repo_id>.*))',
        os.path.join(os.path.dirname(__file__), 'context/kfp_pkg_dev.json'))
}

_DEFAULT_JSON_HEADER = {
    'Content-type': 'application/json',
}

_VERSION_PREFIX = 'sha256:'

LOCAL_REGISTRY_CREDENTIAL = os.path.expanduser(
    '~/.config/kfp/registry_credentials.json')
LOCAL_REGISTRY_CONTEXT = os.path.expanduser(
    '~/.config/kfp/registry_context.json')
DEFAULT_REGISTRY_CONTEXT = os.path.join(
    os.path.dirname(__file__), 'context/default_pkg_dev.json')


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
    """Class for registry authentication using an API token.

    Args:
        token: The API token.

    Example:
      ::

        client = RegistryClient(
            host='https://us-central1-kfp.pkg.dev/proj/repo', auth=ApiAuth('my_token'))
    """

    def __init__(self, token: str) -> None:
        """Initializes the ApiAuth object."""
        self._token = token

    def __call__(self,
                 request: requests.PreparedRequest) -> requests.PreparedRequest:
        request.headers['authorization'] = 'Bearer ' + self._token
        return request


class RegistryClient:
    """Class for communicating with registry hosts.

    Args:
        host: The address of the registry host. The host needs to be specified here or in the config file.
        auth: Authentication using ``requests.auth.AuthBase`` or ``google.auth.credentials.Credentials``.
        config_file: The location of the local config file. If not specified, defaults to ``'~/.config/kfp/context.json'`` (if it exists).
        auth_file: The location of the local config file that contains the authentication token. If not specified, defaults to ``'~/.config/kfp/registry_credentials.json'`` (if it exists).
    """

    def __init__(self,
                 host: Optional[str],
                 auth: Optional[Union[requests.auth.AuthBase,
                                      credentials.Credentials]] = None,
                 config_file: Optional[str] = None,
                 auth_file: Optional[str] = None) -> None:
        """Initializes the RegistryClient."""
        self._host = ''
        self._known_host_key = ''
        self._config = self._load_config(host, config_file)
        self._auth = self._load_auth(auth, auth_file)

    def _request(self,
                 request_url: str,
                 request_body: Optional[str] = '',
                 http_request: Optional[str] = None,
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
        http_request = http_request or 'get'
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

    def _load_auth(
        self,
        auth: Optional[Union[requests.auth.AuthBase,
                             credentials.Credentials]] = None,
        auth_file: Optional[str] = None
    ) -> Optional[Union[requests.auth.AuthBase, credentials.Credentials]]:
        """Loads the credentials for authentication.

        Args:
            auth: Authentication using ``requests.auth.AuthBase`` or ``google.auth.credentials.Credentials``.
            auth_file: The location of the local config file that contains the
                authentication token. If not specified, defaults to
                ``'~/.config/kfp/registry_credentials.json'`` (if it exists).

        Returns:
            The loaded authentication token.
        """
        if auth:
            return auth
        elif self._is_ar_host():
            auth, _ = google.auth.default()
            auth_scopes = self._config.get('auth_scopes')
            auth_scopes = auth_scopes.split(',') if auth_scopes else None
            return credentials.with_scopes_if_required(auth, auth_scopes)
        elif auth_file:
            if os.path.exists(auth_file):
                # Fetch auth token using the locally stored credentials.
                with open(auth_file, 'r') as f:
                    auth_token = json.load(f)
                    return ApiAuth(auth_token)
            else:
                raise ValueError(f'Auth file not found: {auth_file}.')
        elif os.path.exists(LOCAL_REGISTRY_CREDENTIAL):
            # Fetch auth token using the locally stored credentials.
            with open(LOCAL_REGISTRY_CREDENTIAL, 'r') as f:
                auth_token = json.load(f)
                return ApiAuth(auth_token)
        return None

    def _load_config(self, host: Optional[str],
                     config_file: Optional[str]) -> dict:
        """Loads the config.

        Args:
            host: The address of the registry host.
            config_file: The location of the local config file. If not specified,
                defaults to ``'~/.config/kfp/context.json'`` (if it exists).

        Returns:
            The loaded config.
        """
        if host:
            self._host = host.rstrip('/')
        else:
            # Check config file exists
            config_file_path = ''
            if config_file:
                if os.path.exists(config_file):
                    config_file_path = config_file
                else:
                    raise ValueError(f'Config file not found: {config_file}.')
            elif os.path.exists(LOCAL_REGISTRY_CONTEXT):
                config_file_path = LOCAL_REGISTRY_CONTEXT
            # Try loading host from config file
            if config_file_path:
                with open(config_file_path, 'r') as f:
                    data = json.load(f)
                    if 'host' in data:
                        self._host = data['host']
        if not self._host:
            raise ValueError('No host found.')

        # Check if it's a known host
        for key in _KNOWN_HOSTS_REGEX.keys():
            if re.match(_KNOWN_HOSTS_REGEX[key][0], self._host):
                self._known_host_key = key
                break

        # Try loading config from known contexts or local context
        if self._is_known_host():
            config = self._load_context(
                _KNOWN_HOSTS_REGEX[self._known_host_key][1])
        elif os.path.exists(LOCAL_REGISTRY_CONTEXT):
            config = self._load_context(LOCAL_REGISTRY_CONTEXT)
        else:
            config = self._load_context(DEFAULT_REGISTRY_CONTEXT)

        # If config file is specified, add/override any extra context info needed
        if config_file and os.path.exists(config_file):
            config = self._load_context(config_file, config)

        matched = None
        if self._is_known_host():
            matched = re.match(_KNOWN_HOSTS_REGEX[self._known_host_key][0],
                               self._host)
        elif 'regex' in config:
            matched = re.match(config['regex'], self._host)

        if matched:
            map_dict = _SafeDict(**matched.groupdict(), host=self._host)
        else:
            map_dict = _SafeDict(host=self._host)

        # Replace all currently known variables with values
        for config_key in config:
            config[config_key] = config[config_key].format_map(map_dict)

        return config

    def _load_context(self,
                      config_file: str,
                      config: Optional[dict] = None) -> dict:
        """Loads the context from the given config_file.

        Args:
            config_file: The location of the config file.
            config: An existing config to set as the default config.

        Returns:
            The loaded config.
        """
        if not os.path.exists(config_file):
            raise ValueError(f'Config file not found: {config_file}.')
        with open(config_file, 'r') as f:
            loaded_config = json.load(f)
        if config:
            config.update(loaded_config)
            return config
        return loaded_config

    def _get_auth(
        self
    ) -> Optional[Union[requests.auth.AuthBase, credentials.Credentials]]:
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
            A tuple of the package name and the version.
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
        """Downloads a pipeline. Either version or tag must be specified.

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
            Metadata for the created tag.
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

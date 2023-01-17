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
"""The SDK client for Kubeflow Pipelines API."""

import copy
import datetime
import json
import logging
import os
import re
import tarfile
import tempfile
import time
from types import ModuleType
from typing import Any, Dict, List, Optional
import warnings
import zipfile

from kfp import compiler
from kfp.client import auth
from kfp.client import set_volume_credentials
from kfp.components import base_component
import kfp_server_api
import yaml

# Operators on scalar values. Only applies to one of |int_value|,
# |long_value|, |string_value| or |timestamp_value|.
_FILTER_OPERATIONS = {
    'UNKNOWN': 0,
    'EQUALS': 1,
    'NOT_EQUALS': 2,
    'GREATER_THAN': 3,
    'GREATER_THAN_EQUALS': 5,
    'LESS_THAN': 6,
    'LESS_THAN_EQUALS': 7
}

KF_PIPELINES_ENDPOINT_ENV = 'KF_PIPELINES_ENDPOINT'
KF_PIPELINES_UI_ENDPOINT_ENV = 'KF_PIPELINES_UI_ENDPOINT'
KF_PIPELINES_DEFAULT_EXPERIMENT_NAME = 'KF_PIPELINES_DEFAULT_EXPERIMENT_NAME'
KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME = 'KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME'
KF_PIPELINES_IAP_OAUTH2_CLIENT_ID_ENV = 'KF_PIPELINES_IAP_OAUTH2_CLIENT_ID'
KF_PIPELINES_APP_OAUTH2_CLIENT_ID_ENV = 'KF_PIPELINES_APP_OAUTH2_CLIENT_ID'
KF_PIPELINES_APP_OAUTH2_CLIENT_SECRET_ENV = 'KF_PIPELINES_APP_OAUTH2_CLIENT_SECRET'


class JobConfig:

    def __init__(
            self, spec: kfp_server_api.ApiPipelineSpec,
            resource_references: kfp_server_api.ApiResourceReference) -> None:
        self.spec = spec
        self.resource_references = resource_references


class RunPipelineResult:

    def __init__(self, client: 'Client',
                 run_info: kfp_server_api.ApiRun) -> None:
        self._client = client
        self.run_info = run_info
        self.run_id = run_info.id

    def wait_for_run_completion(self, timeout=None):
        timeout = timeout or datetime.timedelta.max
        return self._client.wait_for_run_completion(self.run_id, timeout)

    def __repr__(self):
        return f'RunPipelineResult(run_id={self.run_id})'


class Client:
    """The KFP SDK client for the Kubeflow Pipelines backend API.

    Args:
        host: Host name to use to talk to Kubeflow Pipelines. If not set,
            the in-cluster service DNS name will be used, which only works if
            the current environment is a pod in the same cluster (such as a
            Jupyter instance spawned by Kubeflow's JupyterHub). (`More information on connecting. <https://www.kubeflow.org/docs/components/pipelines/sdk/connect-api/>`_)
        client_id: Client ID used by Identity-Aware Proxy.
        namespace: Kubernetes namespace to use. Used for multi-user deployments. For single-user deployments, this should be left as ``None``.
        other_client_id: Client ID used to obtain the auth codes and refresh
            tokens (`reference <https://cloud.google.com/iap/docs/authentication-howto#authenticating_from_a_desktop_app>`_).
        other_client_secret: Client secret used to obtain the auth codes and
            refresh tokens.
        existing_token: Authentication token to pass in directly. Used in cases where the token is
            generated from outside the SDK.
        cookies: CookieJar object containing cookies that will be passed to the
            Pipelines API.
        proxy: HTTP or HTTPS proxy server.
        ssl_ca_cert: Certification for proxy.
        kube_context: kubectl context to use. Must be a context listed in the kubeconfig file. Defaults to the current-context set within kubeconfig.
        credentials: ``TokenCredentialsBase`` object which provides the logic to
            populate the requests with credentials to authenticate against the
            API server.
        ui_host: Base URL to use to open the Kubeflow Pipelines UI. This is used
            when running the client from a notebook to generate and print links.
        verify_ssl: Whether to verify the server's TLS certificate.
    """

    # in-cluster DNS name of the pipeline service
    _IN_CLUSTER_DNS_NAME = 'ml-pipeline.{}.svc.cluster.local:8888'
    _KUBE_PROXY_PATH = 'api/v1/namespaces/{}/services/ml-pipeline:http/proxy/'

    # Auto populated path in pods
    # https://kubernetes.io/docs/tasks/access-application-cluster/access-cluster/#accessing-the-api-from-a-pod
    # https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#serviceaccount-admission-controller
    _NAMESPACE_PATH = '/var/run/secrets/kubernetes.io/serviceaccount/namespace'

    _LOCAL_KFP_CONTEXT = os.path.expanduser('~/.config/kfp/context.json')

    # TODO: Wrap the configurations for different authentication methods.
    def __init__(
        self,
        host: Optional[str] = None,
        client_id: Optional[str] = None,
        namespace: str = 'kubeflow',
        other_client_id: Optional[str] = None,
        other_client_secret: Optional[str] = None,
        existing_token: Optional[str] = None,
        cookies: Optional[str] = None,
        proxy: Optional[str] = None,
        ssl_ca_cert: Optional[str] = None,
        kube_context: Optional[str] = None,
        credentials: Optional[str] = None,
        ui_host: Optional[str] = None,
        verify_ssl: Optional[bool] = None,
    ) -> None:
        """Create a new instance of kfp client."""
        warnings.warn(
            'This client only works with Kubeflow Pipeline v2.0.0-alpha.0 '
            'and later versions.',
            category=FutureWarning)

        host = host or os.environ.get(KF_PIPELINES_ENDPOINT_ENV)
        self._uihost = os.environ.get(KF_PIPELINES_UI_ENDPOINT_ENV, ui_host or
                                      host)
        client_id = client_id or os.environ.get(
            KF_PIPELINES_IAP_OAUTH2_CLIENT_ID_ENV)
        other_client_id = other_client_id or os.environ.get(
            KF_PIPELINES_APP_OAUTH2_CLIENT_ID_ENV)
        other_client_secret = other_client_secret or os.environ.get(
            KF_PIPELINES_APP_OAUTH2_CLIENT_SECRET_ENV)
        config = self._load_config(host, client_id, namespace, other_client_id,
                                   other_client_secret, existing_token, proxy,
                                   ssl_ca_cert, kube_context, credentials,
                                   verify_ssl)
        # Save the loaded API client configuration, as a reference if update is
        # needed.
        self._load_context_setting_or_default()

        # If custom namespace provided, overwrite the loaded or default one in
        # context settings for current client instance
        if namespace != 'kubeflow':
            self._context_setting['namespace'] = namespace

        self._existing_config = config
        if cookies is None:
            cookies = self._context_setting.get('client_authentication_cookie')
        api_client = kfp_server_api.ApiClient(
            config,
            cookie=cookies,
            header_name=self._context_setting.get(
                'client_authentication_header_name'),
            header_value=self._context_setting.get(
                'client_authentication_header_value'))
        _add_generated_apis(self, kfp_server_api, api_client)
        self._job_api = kfp_server_api.JobServiceApi(api_client)
        self._run_api = kfp_server_api.RunServiceApi(api_client)
        self._experiment_api = kfp_server_api.ExperimentServiceApi(api_client)
        self._pipelines_api = kfp_server_api.PipelineServiceApi(api_client)
        self._upload_api = kfp_server_api.PipelineUploadServiceApi(api_client)
        self._healthz_api = kfp_server_api.HealthzServiceApi(api_client)
        if not self._context_setting['namespace'] and self.get_kfp_healthz(
        ).multi_user is True:
            try:
                with open(Client._NAMESPACE_PATH, 'r') as f:
                    current_namespace = f.read()
                    self.set_user_namespace(current_namespace)
            except FileNotFoundError:
                logging.info(
                    'Failed to automatically set namespace.', exc_info=False)

    def _load_config(
        self,
        host: Optional[str],
        client_id: Optional[str],
        namespace: str,
        other_client_id: Optional[str],
        other_client_secret: Optional[str],
        existing_token: Optional[str],
        proxy: Optional[str],
        ssl_ca_cert: Optional[str],
        kube_context: Optional[str],
        credentials: Optional[str],
        verify_ssl: Optional[bool],
    ) -> kfp_server_api.Configuration:
        config = kfp_server_api.Configuration()

        if proxy:
            # https://github.com/kubeflow/pipelines/blob/c6ac5e0b1fd991e19e96419f0f508ec0a4217c29/backend/api/python_http_client/kfp_server_api/rest.py#L100
            config.proxy = proxy
        if verify_ssl is not None:
            config.verify_ssl = verify_ssl

        if ssl_ca_cert:
            config.ssl_ca_cert = ssl_ca_cert

        host = host or ''

        # Defaults to 'https' if host does not contain 'http' or 'https' protocol.
        if host and not host.startswith('http'):
            warnings.warn(
                f'The host {host} does not contain the "http" or "https" protocol. Defaults to "https".'
            )
            host = 'https://' + host

        # Preprocess the host endpoint to prevent some common user mistakes.
        if not client_id:
            # always preserving the protocol (http://localhost requires it)
            host = host.rstrip('/')

        if host:
            config.host = host

        token = None

        # "existing_token" is designed to accept token generated outside of SDK.
        #
        # https://cloud.google.com/functions/docs/securing/function-identity
        # https://cloud.google.com/endpoints/docs/grpc/service-account-authentication
        #
        # Here is an example.
        #
        # import requests
        # import kfp
        #
        # def get_access_token():
        #     url = 'http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token'
        #     r = requests.get(url, headers={'Metadata-Flavor': 'Google'})
        #     r.raise_for_status()
        #     access_token = r.json()['access_token']
        #     return access_token
        #
        # client = kfp.Client(host='<KFPHost>', existing_token=get_access_token())
        #
        if existing_token:
            token = existing_token
            self._is_refresh_token = False
        elif client_id:
            token, self._is_refresh_token = auth.get_auth_token(
                client_id, other_client_id, other_client_secret)
        elif self._is_inverse_proxy_host(host):
            token = auth.get_gcp_access_token()
            self._is_refresh_token = False
        elif credentials:
            config.api_key['authorization'] = 'placeholder'
            config.api_key_prefix['authorization'] = 'Bearer'
            config.refresh_api_key_hook = credentials.refresh_api_key_hook

        if token:
            config.api_key['authorization'] = token
            config.api_key_prefix['authorization'] = 'Bearer'
            return config

        if host:
            # if host is explicitly set with auth token, it's probably a port
            # forward address.
            return config

        import kubernetes as k8s
        in_cluster = True
        try:
            k8s.config.load_incluster_config()
        except:
            in_cluster = False

        if in_cluster:
            config.host = Client._IN_CLUSTER_DNS_NAME.format(namespace)
            config = self._get_config_with_default_credentials(config)
            return config

        try:
            k8s.config.load_kube_config(
                client_configuration=config, context=kube_context)
        except:
            print('Failed to load kube config.')
            return config

        if config.host:
            config.host = config.host + '/' + Client._KUBE_PROXY_PATH.format(
                namespace)
        return config

    def _is_inverse_proxy_host(self, host: str) -> bool:
        return bool(re.match(r'\S+.googleusercontent.com/{0,1}$', host))

    def _is_ipython(self) -> bool:
        """Returns whether we are running in notebook."""
        try:
            import IPython
            ipy = IPython.get_ipython()
            if ipy is None:
                return False
        except ImportError:
            return False

        return True

    def _get_url_prefix(self) -> str:
        if self._uihost:
            # User's own connection.
            if self._uihost.startswith('http://') or self._uihost.startswith(
                    'https://'):
                return self._uihost
            else:
                return 'http://' + self._uihost

        # In-cluster pod. We could use relative URL.
        return '/pipeline'

    def _load_context_setting_or_default(self) -> None:
        if os.path.exists(Client._LOCAL_KFP_CONTEXT):
            with open(Client._LOCAL_KFP_CONTEXT, 'r') as f:
                self._context_setting = json.load(f)
        else:
            self._context_setting = {
                'namespace': '',
            }

    def _refresh_api_client_token(self) -> None:
        """Refreshes the existing token associated with the kfp_api_client."""
        if getattr(self, '_is_refresh_token', None):
            return

        new_token = auth.get_gcp_access_token()
        self._existing_config.api_key['authorization'] = new_token

    def _get_config_with_default_credentials(
            self, config: kfp_server_api.Configuration
    ) -> kfp_server_api.Configuration:
        """Apply default credentials to the configuration object.

        This method accepts a Configuration object and extends it with
        some default credentials interface.
        """
        # XXX: The default credentials are audience-based service account tokens
        # projected by the kubelet (ServiceAccountTokenVolumeCredentials). As we
        # implement more and more credentials, we can have some heuristic and
        # choose from a number of options.
        # See https://github.com/kubeflow/pipelines/pull/5287#issuecomment-805654121
        credentials = set_volume_credentials.ServiceAccountTokenVolumeCredentials(
        )
        config_copy = copy.deepcopy(config)

        try:
            credentials.refresh_api_key_hook(config_copy)
        except Exception:
            logging.warning('Failed to set up default credentials. Proceeding'
                            ' without credentials...')
            return config

        config.refresh_api_key_hook = credentials.refresh_api_key_hook
        config.api_key_prefix['authorization'] = 'Bearer'
        config.refresh_api_key_hook(config)
        return config

    def set_user_namespace(self, namespace: str) -> None:
        """Sets the namespace in the Kuberenetes cluster to use.

        This function should only be used when Kubeflow Pipelines is in the
        multi-user mode.

        Args:
            namespace: Namespace to use within the Kubernetes cluster (namespace containing the Kubeflow Pipelines deployment).
        """
        self._context_setting['namespace'] = namespace
        if not os.path.exists(os.path.dirname(Client._LOCAL_KFP_CONTEXT)):
            os.makedirs(os.path.dirname(Client._LOCAL_KFP_CONTEXT))
        with open(Client._LOCAL_KFP_CONTEXT, 'w') as f:
            json.dump(self._context_setting, f)

    def get_kfp_healthz(
            self,
            sleep_duration: int = 5) -> kfp_server_api.ApiGetHealthzResponse:
        """Gets healthz info for KFP deployment.

        Args:
            sleep_duration: Time in seconds between retries.

        Returns:
            JSON response from the healthz endpoint.
        """
        count = 0
        response = None
        max_attempts = 5
        while not response:
            count += 1
            if count > max_attempts:
                raise TimeoutError(
                    f'Failed getting healthz endpoint after {max_attempts} attempts.'
                )

            try:
                return self._healthz_api.get_healthz()
            # ApiException, including network errors, is the only type that may
            # recover after retry.
            except kfp_server_api.ApiException:
                # logging.exception also logs detailed info about the ApiException
                logging.exception(
                    f'Failed to get healthz info attempt {count} of {max_attempts}.'
                )
                time.sleep(sleep_duration)

    def get_user_namespace(self) -> str:
        """Gets user namespace in context config.

        Returns:
            Kubernetes namespace from the local context file or empty if it
            wasn't set.
        """
        return self._context_setting['namespace']

    def create_experiment(
            self,
            name: str,
            description: str = None,
            namespace: str = None) -> kfp_server_api.ApiExperiment:
        """Creates a new experiment.

        Args:
            name: Name of the experiment.
            description: Description of the experiment.
            namespace: Kubernetes namespace to use. Used for multi-user deployments. For single-user deployments, this should be left as ``None``.

        Returns:
            ``ApiExperiment`` object.
        """
        namespace = namespace or self.get_user_namespace()
        experiment = None
        try:
            experiment = self.get_experiment(
                experiment_name=name, namespace=namespace)
        except ValueError as error:
            # Ignore error if the experiment does not exist.
            if not str(error).startswith('No experiment is found with name'):
                raise error

        if not experiment:
            logging.info(f'Creating experiment {name}.')

            resource_references = []
            if namespace is not None:
                key = kfp_server_api.ApiResourceKey(
                    id=namespace, type=kfp_server_api.ApiResourceType.NAMESPACE)
                reference = kfp_server_api.ApiResourceReference(
                    key=key, relationship=kfp_server_api.ApiRelationship.OWNER)
                resource_references.append(reference)

            experiment = kfp_server_api.ApiExperiment(
                name=name,
                description=description,
                resource_references=resource_references)
            experiment = self._experiment_api.create_experiment(body=experiment)

        link = f'{self._get_url_prefix()}/#/experiments/details/{experiment.id}'
        if self._is_ipython():
            import IPython
            html = f'<a href="{link}" target="_blank" >Experiment details</a>.'
            IPython.display.display(IPython.display.HTML(html))
        else:
            print(f'Experiment details: {link}')

        return experiment

    def get_pipeline_id(self, name: str) -> Optional[str]:
        """Gets the ID of a pipeline by its name.

        Args:
            name: Pipeline name.

        Returns:
            The pipeline ID if a pipeline with the name exists.
        """
        pipeline_filter = json.dumps({
            'predicates': [{
                'op': _FILTER_OPERATIONS['EQUALS'],
                'key': 'name',
                'stringValue': name,
            }]
        })
        result = self._pipelines_api.list_pipelines(filter=pipeline_filter)
        if result.pipelines is None:
            return None
        if len(result.pipelines) == 1:
            return result.pipelines[0].id
        elif len(result.pipelines) > 1:
            raise ValueError(
                f'Multiple pipelines with the name: {name} found, the name needs to be unique'
            )
        return None

    def list_experiments(
        self,
        page_token: str = '',
        page_size: int = 10,
        sort_by: str = '',
        namespace: Optional[str] = None,
        filter: Optional[str] = None
    ) -> kfp_server_api.ApiListExperimentsResponse:
        """Lists experiments.

        Args:
            page_token: Page token for obtaining page from paginated response.
            page_size: Size of the page.
            sort_by: Sort string of format ``'[field_name]', '[field_name] desc'``. For example, ``'name desc'``.
            namespace: Kubernetes namespace to use. Used for multi-user deployments. For single-user deployments, this should be left as ``None``.
            filter: A url-encoded, JSON-serialized Filter protocol buffer
                (see `filter.proto message <https://github.com/kubeflow/pipelines/blob/777c98153daf3dfae82730e14ff37bdddc334c4d/backend/api/filter.proto>`_). For a list of all filter operations ``'op'``, see `here <https://github.com/kubeflow/pipelines/blob/777c98153daf3dfae82730e14ff37bdddc334c4d/sdk/python/kfp/client/client.py#L37-L45>`_. Example:

                  ::

                    json.dumps({
                        "predicates": [{
                            "op": _FILTER_OPERATIONS["EQUALS"],
                            "key": "name",
                            "stringValue": "my-name",
                        }]
                    })

        Returns:
            ``ApiListExperimentsResponse`` object.
        """
        namespace = namespace or self.get_user_namespace()
        return self._experiment_api.list_experiment(
            page_token=page_token,
            page_size=page_size,
            sort_by=sort_by,
            resource_reference_key_type=kfp_server_api.ApiResourceType
            .NAMESPACE,
            resource_reference_key_id=namespace,
            filter=filter)

    def get_experiment(
            self,
            experiment_id: Optional[str] = None,
            experiment_name: Optional[str] = None,
            namespace: Optional[str] = None) -> kfp_server_api.ApiExperiment:
        """Gets details of an experiment.

        Either ``experiment_id`` or ``experiment_name`` is required.

        Args:
            experiment_id: ID of the experiment.
            experiment_name: Name of the experiment.
            namespace: Kubernetes namespace to use. Used for multi-user deployments. For single-user deployments, this should be left as ``None``.

        Returns:
            ``ApiExperiment`` object.
        """
        namespace = namespace or self.get_user_namespace()
        if experiment_id is None and experiment_name is None:
            raise ValueError(
                'Either experiment_id or experiment_name is required')
        if experiment_id is not None:
            return self._experiment_api.get_experiment(id=experiment_id)
        experiment_filter = json.dumps({
            'predicates': [{
                'op': _FILTER_OPERATIONS['EQUALS'],
                'key': 'name',
                'stringValue': experiment_name,
            }]
        })
        if namespace is not None:
            result = self._experiment_api.list_experiment(
                filter=experiment_filter,
                resource_reference_key_type=kfp_server_api.ApiResourceType
                .NAMESPACE,
                resource_reference_key_id=namespace)
        else:
            result = self._experiment_api.list_experiment(
                filter=experiment_filter)
        if not result.experiments:
            raise ValueError(
                f'No experiment is found with name {experiment_name}.')
        if len(result.experiments) > 1:
            raise ValueError(
                f'Multiple experiments is found with name {experiment_name}.')
        return result.experiments[0]

    def archive_experiment(self, experiment_id: str) -> dict:
        """Archives an experiment.

        Args:
            experiment_id: ID of the experiment.

        Returns:
            Empty dictionary.
        """
        return self._experiment_api.archive_experiment(id=experiment_id)

    def unarchive_experiment(self, experiment_id: str) -> dict:
        """Unarchives an experiment.

        Args:
            experiment_id: ID of the experiment.

        Returns:
            Empty dictionary.
        """
        return self._experiment_api.unarchive_experiment(id=experiment_id)

    def delete_experiment(self, experiment_id: str) -> dict:
        """Delete experiment.

        Args:
            experiment_id: ID of the experiment.

        Returns:
            Empty dictionary.
        """
        return self._experiment_api.delete_experiment(id=experiment_id)

    def _extract_pipeline_yaml(self, package_file: str) -> dict:

        def _choose_pipeline_file(file_list: List[str]) -> str:
            pipeline_files = [
                file for file in file_list if file.endswith('.yaml')
            ]
            if not pipeline_files:
                raise ValueError(
                    'Invalid package. Missing pipeline yaml file in the package.'
                )

            if 'pipeline.yaml' in pipeline_files:
                return 'pipeline.yaml'
            elif len(pipeline_files) == 1:
                return pipeline_files[0]
            else:
                raise ValueError(
                    'Invalid package. There is no pipeline.json file or there '
                    'are multiple yaml files.')

        if package_file.endswith('.tar.gz') or package_file.endswith('.tgz'):
            with tarfile.open(package_file, 'r:gz') as tar:
                file_names = [member.name for member in tar if member.isfile()]
                pipeline_file = _choose_pipeline_file(file_names)
                with tar.extractfile(
                        tar.getmember(pipeline_file)) as f:  # type: ignore
                    return yaml.safe_load(f)
        elif package_file.endswith('.zip'):
            with zipfile.ZipFile(package_file, 'r') as zip:
                pipeline_file = _choose_pipeline_file(zip.namelist())
                with zip.open(pipeline_file) as f:
                    return yaml.safe_load(f)
        elif package_file.endswith('.yaml') or package_file.endswith('.yml'):
            with open(package_file, 'r') as f:
                return yaml.safe_load(f)
        else:
            raise ValueError(
                f'The package_file {package_file} should end with one of the '
                'following formats: [.tar.gz, .tgz, .zip, .yaml, .yml].')

    def _override_caching_options(self, pipeline_obj: dict,
                                  enable_caching: bool) -> None:
        """Overrides caching options.

        Args:
            pipeline_obj: Dict object parsed from the yaml file.
            enable_caching: Overrides options, one of 'True', 'False'.
        """
        for _, task in pipeline_obj['root']['dag']['tasks'].items():
            if 'cachingOptions' in task:
                task['cachingOptions']['enableCache'] = enable_caching
            else:
                task['cachingOptions'] = {'enableCache': enable_caching}

    def list_pipelines(
        self,
        page_token: str = '',
        page_size: int = 10,
        sort_by: str = '',
        filter: Optional[str] = None
    ) -> kfp_server_api.ApiListPipelinesResponse:
        """Lists pipelines.

        Args:
            page_token: Page token for obtaining page from paginated response.
            page_size: Size of the page.
            sort_by: Sort string of format ``'[field_name]', '[field_name] desc'``. For example, ``'name desc'``.
            filter: A url-encoded, JSON-serialized Filter protocol buffer
                (see `filter.proto message <https://github.com/kubeflow/pipelines/blob/777c98153daf3dfae82730e14ff37bdddc334c4d/backend/api/filter.proto>`_). For a list of all filter operations ``'op'``, see `here <https://github.com/kubeflow/pipelines/blob/777c98153daf3dfae82730e14ff37bdddc334c4d/sdk/python/kfp/client/client.py#L37-L45>`_. Example:

                  ::

                    json.dumps({
                        "predicates": [{
                            "op": _FILTER_OPERATIONS["EQUALS"],
                            "key": "name",
                            "stringValue": "my-name",
                        }]
                    })

        Returns:
            ``ApiListPipelinesResponse`` object.
        """
        return self._pipelines_api.list_pipelines(
            page_token=page_token,
            page_size=page_size,
            sort_by=sort_by,
            filter=filter)

    # TODO: provide default namespace, similar to kubectl default namespaces.
    def run_pipeline(
        self,
        experiment_id: str,
        job_name: str,
        pipeline_package_path: Optional[str] = None,
        params: Optional[Dict[str, Any]] = None,
        pipeline_id: Optional[str] = None,
        version_id: Optional[str] = None,
        pipeline_root: Optional[str] = None,
        enable_caching: Optional[bool] = None,
        service_account: Optional[str] = None,
    ) -> kfp_server_api.ApiRun:
        """Runs a specified pipeline.

        Args:
            experiment_id: ID of an experiment.
            job_name: Name of the job.
            pipeline_package_path: Local path of the pipeline package (the
                filename should end with one of the following .tar.gz, .tgz,
                .zip, .json).
            params: Arguments to the pipeline function provided as a dict.
            pipeline_id: ID of the pipeline.
            version_id: ID of the pipeline version to run.
                If both pipeline_id and version_id are specified, version_id
                will take precendence.
                If only pipeline_id is specified, the default version of this
                pipeline is used to create the run.
            pipeline_root: Root path of the pipeline outputs.
            enable_caching: Whether or not to enable caching for the
                run. If not set, defaults to the compile-time settings, which
                is ``True`` for all tasks by default. If set, the
                setting applies to all tasks in the pipeline (overrides the
                compile time settings).
            service_account: Specifies which Kubernetes service
                account to use for this run.

        Returns:
            ``ApiRun`` object.
        """
        if params is None:
            params = {}

        job_config = self._create_job_config(
            experiment_id=experiment_id,
            params=params,
            pipeline_package_path=pipeline_package_path,
            pipeline_id=pipeline_id,
            version_id=version_id,
            enable_caching=enable_caching,
            pipeline_root=pipeline_root,
        )
        run_body = kfp_server_api.ApiRun(
            pipeline_spec=job_config.spec,
            resource_references=job_config.resource_references,
            name=job_name,
            service_account=service_account)

        response = self._run_api.create_run(body=run_body)

        link = f'{self._get_url_prefix()}/#/runs/details/{response.run.id}'
        if self._is_ipython():
            import IPython
            html = (f'<a href="{link}" target="_blank" >Run details</a>.')
            IPython.display.display(IPython.display.HTML(html))
        else:
            print(f'Run details: {link}')

        return response.run

    def archive_run(self, run_id: str) -> dict:
        """Archives a run.

        Args:
            run_id: ID of the run.

        Returns:
            Empty dictionary.
        """
        return self._run_api.archive_run(id=run_id)

    def unarchive_run(self, run_id: str) -> dict:
        """Restores an archived run.

        Args:
            run_id: ID of the run.

        Returns:
            Empty dictionary.
        """
        return self._run_api.unarchive_run(id=run_id)

    def delete_run(self, run_id: str) -> dict:
        """Deletes a run.

        Args:
            run_id: ID of the run.

        Returns:
            Empty dictionary.
        """
        return self._run_api.delete_run(id=run_id)

    def terminate_run(self, run_id: str) -> dict:
        """Terminates a run.

        Args:
            run_id: ID of the run.

        Returns:
            Empty dictionary.
        """
        return self._run_api.terminate_run(run_id=run_id)

    def create_recurring_run(
        self,
        experiment_id: str,
        job_name: str,
        description: Optional[str] = None,
        start_time: Optional[str] = None,
        end_time: Optional[str] = None,
        interval_second: Optional[int] = None,
        cron_expression: Optional[str] = None,
        max_concurrency: Optional[int] = 1,
        no_catchup: Optional[bool] = None,
        params: Optional[dict] = None,
        pipeline_package_path: Optional[str] = None,
        pipeline_id: Optional[str] = None,
        version_id: Optional[str] = None,
        enabled: bool = True,
        pipeline_root: Optional[str] = None,
        enable_caching: Optional[bool] = None,
        service_account: Optional[str] = None,
    ) -> kfp_server_api.ApiJob:
        """Creates a recurring run.

        Args:
            experiment_id: ID of the experiment.
            job_name: Name of the job.
            description: Description of the job.
            start_time: RFC3339 time string of the time when to start the
                job.
            end_time: RFC3339 time string of the time when to end the job.
            interval_second: Integer indicating the seconds between two
                recurring runs in for a periodic schedule.
            cron_expression: Cron expression representing a set of times,
                using 6 space-separated fields (e.g., ``'0 0 9 ? * 2-6'``). See `cron format
                <https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format>`_.
            max_concurrency: Integer indicating how many jobs can be run in
                parallel.
            no_catchup: Whether the recurring run should catch up if behind
                schedule. For example, if the recurring run is paused for a
                while and re-enabled afterwards. If ``no_catchup=False``, the
                scheduler will catch up on (backfill) each missed interval.
                Otherwise, it only schedules the latest interval if more than
                one interval is ready to be scheduled. Usually, if your pipeline
                handles backfill internally, you should turn catchup off to
                avoid duplicate backfill.
            pipeline_package_path: Local path of the pipeline package (the
                filename should end with one of the following .tar.gz, .tgz,
                .zip, .json).
            params: Arguments to the pipeline function provided as a dict.
            pipeline_id: ID of a pipeline.
            version_id: ID of a pipeline version.
                If both ``pipeline_id`` and ``version_id`` are specified, ``version_id``
                will take precedence.
                If only ``pipeline_id`` is specified, the default version of this
                pipeline is used to create the run.
            enabled: Whether to enable or disable the recurring run.
            pipeline_root: Root path of the pipeline outputs.
            enable_caching: Whether or not to enable caching for the
                run. If not set, defaults to the compile time settings, which
                is ``True`` for all tasks by default, while users may specify
                different caching options for individual tasks. If set, the
                setting applies to all tasks in the pipeline (overrides the
                compile time settings).
            service_account: Specifies which Kubernetes service
                account this recurring run uses.
        Returns:
            ``ApiJob`` object.
        """

        job_config = self._create_job_config(
            experiment_id=experiment_id,
            params=params,
            pipeline_package_path=pipeline_package_path,
            pipeline_id=pipeline_id,
            version_id=version_id,
            enable_caching=enable_caching,
            pipeline_root=pipeline_root,
        )

        if all([interval_second, cron_expression
               ]) or not any([interval_second, cron_expression]):
            raise ValueError(
                'Either interval_second or cron_expression is required')
        if interval_second is not None:
            trigger = kfp_server_api.ApiTrigger(
                periodic_schedule=kfp_server_api.ApiPeriodicSchedule(
                    start_time=start_time,
                    end_time=end_time,
                    interval_second=interval_second))
        if cron_expression is not None:
            trigger = kfp_server_api.ApiTrigger(
                cron_schedule=kfp_server_api.ApiCronSchedule(
                    start_time=start_time,
                    end_time=end_time,
                    cron=cron_expression))

        job_body = kfp_server_api.ApiJob(
            enabled=enabled,
            pipeline_spec=job_config.spec,
            resource_references=job_config.resource_references,
            name=job_name,
            description=description,
            no_catchup=no_catchup,
            trigger=trigger,
            max_concurrency=max_concurrency,
            service_account=service_account)
        return self._job_api.create_job(body=job_body)

    def _create_job_config(
        self,
        experiment_id: str,
        params: Optional[Dict[str, Any]],
        pipeline_package_path: Optional[str],
        pipeline_id: Optional[str],
        version_id: Optional[str],
        enable_caching: Optional[bool],
        pipeline_root: Optional[str],
    ) -> JobConfig:
        """Creates a JobConfig with spec and resource_references.

        Args:
            experiment_id: ID of an experiment.
            pipeline_package_path: Local path of the pipeline package (the
                filename should end with one of the following .tar.gz, .tgz,
                .zip, .yaml, .yml).
            params: A dictionary with key as param name and value as param value.
            pipeline_id: ID of a pipeline.
            version_id: ID of a pipeline version.
                If both pipeline_id and version_id are specified, version_id
                will take precedence. If only pipeline_id is specified, the
                default version of this pipeline is used to create the run.
            enable_caching: Whether or not to enable caching for the
                run. If not set, defaults to the compile time settings, which
                is ``True`` for all tasks by default, while users may specify
                different caching options for individual tasks. If set, the
                setting applies to all tasks in the pipeline (overrides the
                compile time settings).
            pipeline_root: Root path of the pipeline outputs.

        Returns:
            A JobConfig object with attributes .spec and .resource_reference.
        """

        params = params or {}
        pipeline_yaml_string = None
        if pipeline_package_path:
            pipeline_obj = self._extract_pipeline_yaml(pipeline_package_path)

            # Caching option set at submission time overrides the compile time
            # settings.
            if enable_caching is not None:
                self._override_caching_options(pipeline_obj, enable_caching)

            pipeline_yaml_string = yaml.dump(pipeline_obj, sort_keys=True)

        runtime_config = kfp_server_api.PipelineSpecRuntimeConfig(
            pipeline_root=pipeline_root,
            parameters=params,
        )
        resource_references = []
        key = kfp_server_api.ApiResourceKey(
            id=experiment_id, type=kfp_server_api.ApiResourceType.EXPERIMENT)
        reference = kfp_server_api.ApiResourceReference(
            key=key, relationship=kfp_server_api.ApiRelationship.OWNER)
        resource_references.append(reference)

        if version_id:
            key = kfp_server_api.ApiResourceKey(
                id=version_id,
                type=kfp_server_api.ApiResourceType.PIPELINE_VERSION)
            reference = kfp_server_api.ApiResourceReference(
                key=key, relationship=kfp_server_api.ApiRelationship.CREATOR)
            resource_references.append(reference)

        spec = kfp_server_api.ApiPipelineSpec(
            pipeline_id=pipeline_id,
            pipeline_manifest=pipeline_yaml_string,
            runtime_config=runtime_config,
        )
        return JobConfig(spec=spec, resource_references=resource_references)

    def create_run_from_pipeline_func(
        self,
        pipeline_func: base_component.BaseComponent,
        arguments: Optional[Dict[str, Any]] = None,
        run_name: Optional[str] = None,
        experiment_name: Optional[str] = None,
        namespace: Optional[str] = None,
        pipeline_root: Optional[str] = None,
        enable_caching: Optional[bool] = None,
        service_account: Optional[str] = None,
    ) -> RunPipelineResult:
        """Runs pipeline on KFP-enabled Kubernetes cluster.

        This command compiles the pipeline function, creates or gets an
        experiment, then submits the pipeline for execution.

        Args:
            pipeline_func: Pipeline function constructed with ``@kfp.dsl.pipeline`` decorator.
            arguments: Arguments to the pipeline function provided as a dict.
            run_name: Name of the run to be shown in the UI.
            experiment_name: Name of the experiment to add the run to.
            namespace: Kubernetes namespace to use. Used for multi-user deployments. For single-user deployments, this should be left as ``None``.
            pipeline_root: Root path of the pipeline outputs.
            enable_caching: Whether or not to enable caching for the
                run. If not set, defaults to the compile time settings, which
                is ``True`` for all tasks by default, while users may specify
                different caching options for individual tasks. If set, the
                setting applies to all tasks in the pipeline (overrides the
                compile time settings).
            service_account: Specifies which Kubernetes service
                account to use for this run.

        Returns:
            ``RunPipelineResult`` object containing information about the pipeline run.
        """
        #TODO: Check arguments against the pipeline function
        pipeline_name = pipeline_func.name
        run_name = run_name or pipeline_name + ' ' + datetime.datetime.now(
        ).strftime('%Y-%m-%d %H-%M-%S')

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_package_path = os.path.join(tmpdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_func,
                package_path=pipeline_package_path,
            )

            return self.create_run_from_pipeline_package(
                pipeline_file=pipeline_package_path,
                arguments=arguments,
                run_name=run_name,
                experiment_name=experiment_name,
                namespace=namespace,
                pipeline_root=pipeline_root,
                enable_caching=enable_caching,
                service_account=service_account,
            )

    def create_run_from_pipeline_package(
        self,
        pipeline_file: str,
        arguments: Optional[Dict[str, Any]] = None,
        run_name: Optional[str] = None,
        experiment_name: Optional[str] = None,
        namespace: Optional[str] = None,
        pipeline_root: Optional[str] = None,
        enable_caching: Optional[bool] = None,
        service_account: Optional[str] = None,
    ) -> RunPipelineResult:
        """Runs pipeline on KFP-enabled Kubernetes cluster.

        This command takes a local pipeline package, creates or gets an
        experiment, then submits the pipeline for execution.

        Args:
            pipeline_file: A compiled pipeline package file.
            arguments: Arguments to the pipeline function provided as a dict.
            run_name:  Name of the run to be shown in the UI.
            experiment_name: Name of the experiment to add the run to.
            namespace: Kubernetes namespace to use. Used for multi-user deployments. For single-user deployments, this should be left as ``None``.
            pipeline_root: Root path of the pipeline outputs.
            enable_caching: Whether or not to enable caching for the
                run. If not set, defaults to the compile time settings, which
                is ``True`` for all tasks by default, while users may specify
                different caching options for individual tasks. If set, the
                setting applies to all tasks in the pipeline (overrides the
                compile time settings).
            service_account: Specifies which Kubernetes service
                account to use for this run.

        Returns:
            ``RunPipelineResult`` object containing information about the pipeline run.
        """

        #TODO: Check arguments against the pipeline function
        pipeline_name = os.path.basename(pipeline_file)
        experiment_name = experiment_name or os.environ.get(
            KF_PIPELINES_DEFAULT_EXPERIMENT_NAME, None)
        overridden_experiment_name = os.environ.get(
            KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME, experiment_name)
        if overridden_experiment_name != experiment_name:
            warnings.warn(
                f'Changing experiment name from "{experiment_name}" to "{overridden_experiment_name}".'
            )
        experiment_name = overridden_experiment_name or 'Default'
        run_name = run_name or (
            pipeline_name + ' ' +
            datetime.datetime.now().strftime('%Y-%m-%d %H-%M-%S'))
        experiment = self.create_experiment(
            name=experiment_name, namespace=namespace)
        run_info = self.run_pipeline(
            experiment_id=experiment.id,
            job_name=run_name,
            pipeline_package_path=pipeline_file,
            params=arguments,
            pipeline_root=pipeline_root,
            enable_caching=enable_caching,
            service_account=service_account,
        )
        return RunPipelineResult(self, run_info)

    def delete_job(self, job_id: str) -> dict:
        """Deletes a job (recurring run).

        Args:
            job_id: ID of the job.

        Returns:
            Empty dictionary.
        """
        return self._job_api.delete_job(id=job_id)

    def disable_job(self, job_id: str) -> dict:
        """Disables a job (recurring run).

        Args:
            job_id: ID of the job.

        Returns:
            Empty dictionary.
        """
        return self._job_api.disable_job(id=job_id)

    def enable_job(self, job_id: str) -> dict:
        """Enables a job (recurring run).

        Args:
            job_id: ID of the job.

        Returns:
            Empty dictionary.
        """
        return self._job_api.enable_job(id=job_id)

    def list_runs(
            self,
            page_token: str = '',
            page_size: int = 10,
            sort_by: str = '',
            experiment_id: Optional[str] = None,
            namespace: Optional[str] = None,
            filter: Optional[str] = None) -> kfp_server_api.ApiListRunsResponse:
        """List runs.

        Args:
            page_token: Page token for obtaining page from paginated response.
            page_size: Size of the page.
            sort_by: Sort string of format ``'[field_name]', '[field_name] desc'``. For example, ``'name desc'``.
            experiment_id: Experiment ID to filter upon
            namespace: Kubernetes namespace to use. Used for multi-user deployments. For single-user deployments, this should be left as ``None``.
            filter: A url-encoded, JSON-serialized Filter protocol buffer
                (see `filter.proto message <https://github.com/kubeflow/pipelines/blob/777c98153daf3dfae82730e14ff37bdddc334c4d/backend/api/filter.proto>`_). For a list of all filter operations ``'op'``, see `here <https://github.com/kubeflow/pipelines/blob/777c98153daf3dfae82730e14ff37bdddc334c4d/sdk/python/kfp/client/client.py#L37-L45>`_. Example:

                  ::

                    json.dumps({
                        "predicates": [{
                            "op": _FILTER_OPERATIONS["EQUALS"],
                            "key": "name",
                            "stringValue": "my-name",
                        }]
                    })

          Returns:
            ``ApiListRunsResponse`` object.
        """
        namespace = namespace or self.get_user_namespace()
        if experiment_id is not None:
            return self._run_api.list_runs(
                page_token=page_token,
                page_size=page_size,
                sort_by=sort_by,
                resource_reference_key_type=kfp_server_api.ApiResourceType
                .EXPERIMENT,
                resource_reference_key_id=experiment_id,
                filter=filter)

        elif namespace is not None:
            return self._run_api.list_runs(
                page_token=page_token,
                page_size=page_size,
                sort_by=sort_by,
                resource_reference_key_type=kfp_server_api.ApiResourceType
                .NAMESPACE,
                resource_reference_key_id=namespace,
                filter=filter)

        else:
            return self._run_api.list_runs(
                page_token=page_token,
                page_size=page_size,
                sort_by=sort_by,
                filter=filter)

    def list_recurring_runs(
            self,
            page_token: str = '',
            page_size: int = 10,
            sort_by: str = '',
            experiment_id: Optional[str] = None,
            filter: Optional[str] = None) -> kfp_server_api.ApiListJobsResponse:
        """Lists recurring runs.

        Args:
            page_token: Page token for obtaining page from paginated response.
            page_size: Size of the page.
            sort_by: Sort string of format ``'[field_name]', '[field_name] desc'``. For example, ``'name desc'``.
            experiment_id: Experiment ID to filter upon.
            filter: A url-encoded, JSON-serialized Filter protocol buffer
                (see `filter.proto message <https://github.com/kubeflow/pipelines/blob/777c98153daf3dfae82730e14ff37bdddc334c4d/backend/api/filter.proto>`_). For a list of all filter operations ``'op'``, see `here <https://github.com/kubeflow/pipelines/blob/777c98153daf3dfae82730e14ff37bdddc334c4d/sdk/python/kfp/client/client.py#L37-L45>`_. Example:

                  ::

                    json.dumps({
                        "predicates": [{
                            "op": _FILTER_OPERATIONS["EQUALS"],
                            "key": "name",
                            "stringValue": "my-name",
                        }]
                    })

        Returns:
            ``ApiListJobsResponse`` object.
        """
        if experiment_id is not None:
            return self._job_api.list_jobs(
                page_token=page_token,
                page_size=page_size,
                sort_by=sort_by,
                resource_reference_key_type=kfp_server_api.ApiResourceType
                .EXPERIMENT,
                resource_reference_key_id=experiment_id,
                filter=filter)

        else:
            return self._job_api.list_jobs(
                page_token=page_token,
                page_size=page_size,
                sort_by=sort_by,
                filter=filter)

    def get_recurring_run(self, job_id: str) -> kfp_server_api.ApiJob:
        """Gets recurring run (job) details.

        Args:
            job_id: ID of the recurring run (job).

        Returns:
            ``ApiJob`` object.
        """
        return self._job_api.get_job(id=job_id)

    def get_run(self, run_id: str) -> kfp_server_api.ApiRun:
        """Gets run details.

        Args:
            run_id: ID of the run.

        Returns:
            ``ApiRun`` object.
        """
        return self._run_api.get_run(run_id=run_id)

    def wait_for_run_completion(
            self,
            run_id: str,
            timeout: int,
            sleep_duration: int = 5) -> kfp_server_api.ApiRun:
        """Waits for a run to complete.

        Args:
            run_id: ID of the run.
            timeout: Timeout after which the client should stop waiting for run completion (seconds).
            sleep_duration: Time in seconds between retries.

        Returns:
            ``ApiRun`` object.
        """
        status = 'Running:'
        start_time = datetime.datetime.now()
        if isinstance(timeout, datetime.timedelta):
            timeout = timeout.total_seconds()
        is_valid_token = False
        while (status is None or status.lower()
               not in ['succeeded', 'failed', 'skipped', 'error']):
            try:
                get_run_response = self._run_api.get_run(run_id=run_id)
                is_valid_token = True
            except kfp_server_api.ApiException as api_ex:
                # if the token is valid but receiving 401 Unauthorized error
                # then refresh the token
                if is_valid_token and api_ex.status == 401:
                    logging.info('Access token has expired !!! Refreshing ...')
                    self._refresh_api_client_token()
                    continue
                else:
                    raise api_ex
            status = get_run_response.run.status
            elapsed_time = (datetime.datetime.now() -
                            start_time).total_seconds()
            logging.info('Waiting for the job to complete...')
            if elapsed_time > timeout:
                raise TimeoutError('Run timeout')
            time.sleep(sleep_duration)
        return get_run_response

    def _get_workflow_json(self, run_id: str) -> dict:
        """Gets the workflow json.

        Args:
            run_id: run id, returned from run_pipeline.

        Returns:
            Workflow JSON.
        """
        get_run_response = self._run_api.get_run(run_id=run_id)
        workflow = get_run_response.pipeline_runtime.workflow_manifest
        return json.loads(workflow)

    def upload_pipeline(
        self,
        pipeline_package_path: str,
        pipeline_name: Optional[str] = None,
        description: Optional[str] = None,
    ) -> kfp_server_api.ApiPipeline:
        """Uploads a pipeline.

        Args:
            pipeline_package_path: Local path to the pipeline package.
            pipeline_name: Name of the pipeline to be shown in the UI.
            description: Description of the pipeline to be shown in
                the UI.

        Returns:
            ``ApiPipeline`` object.
        """
        if pipeline_name is None:
            pipeline_name = os.path.splitext(
                os.path.basename('something/file.txt'))[0]

        validate_pipeline_resource_name(pipeline_name)
        response = self._upload_api.upload_pipeline(
            pipeline_package_path, name=pipeline_name, description=description)
        link = f'{self._get_url_prefix()}/#/pipelines/details/{response.id}'
        if self._is_ipython():
            import IPython
            html = f'<a href="{link}" target="_blank" >Pipeline details</a>.'
            IPython.display.display(IPython.display.HTML(html))
        else:
            print(f'Pipeline details: {link}')

        return response

    def upload_pipeline_version(
        self,
        pipeline_package_path: str,
        pipeline_version_name: str,
        pipeline_id: Optional[str] = None,
        pipeline_name: Optional[str] = None,
        description: Optional[str] = None,
    ) -> kfp_server_api.ApiPipelineVersion:
        """Uploads a new version of the pipeline.

        Args:
            pipeline_package_path: Local path to the pipeline package.
            pipeline_version_name:  Name of the pipeline version to be shown in
                the UI.
            pipeline_id: ID of the pipeline.
            pipeline_name: Name of the pipeline.
            description: Description of the pipeline version to show in the UI.

        Returns:
            ``ApiPipelineVersion`` object.
        """

        if all([pipeline_id, pipeline_name
               ]) or not any([pipeline_id, pipeline_name]):
            raise ValueError('Either pipeline_id or pipeline_name is required')

        if pipeline_name:
            pipeline_id = self.get_pipeline_id(pipeline_name)
        kwargs = dict(
            name=pipeline_version_name,
            pipelineid=pipeline_id,
        )

        if description:
            kwargs['description'] = description

        response = self._upload_api.upload_pipeline_version(
            pipeline_package_path, **kwargs)

        link = f'{self._get_url_prefix()}/#/pipelines/details/{response.id}'
        if self._is_ipython():
            import IPython
            html = f'<a href="{link}" target="_blank" >Pipeline details</a>.'
            IPython.display.display(IPython.display.HTML(html))
        else:
            print(f'Pipeline details: {link}')

        return response

    def get_pipeline(self, pipeline_id: str) -> kfp_server_api.ApiPipeline:
        """Gets pipeline details.

        Args:
            pipeline_id: ID of the pipeline.

        Returns:
            ``ApiPipeline`` object.
        """
        return self._pipelines_api.get_pipeline(id=pipeline_id)

    def delete_pipeline(self, pipeline_id: str) -> dict:
        """Deletes a pipeline.

        Args:
            pipeline_id: ID of the pipeline.

        Returns:
            Empty dictionary.
        """
        return self._pipelines_api.delete_pipeline(id=pipeline_id)

    def list_pipeline_versions(
        self,
        pipeline_id: str,
        page_token: str = '',
        page_size: int = 10,
        sort_by: str = '',
        filter: Optional[str] = None
    ) -> kfp_server_api.ApiListPipelineVersionsResponse:
        """Lists pipeline versions.

        Args:
            pipeline_id: ID of the pipeline for which to list versions.
            page_token: Page token for obtaining page from paginated response.
            page_size: Size of the page.
            sort_by: Sort string of format ``'[field_name]', '[field_name] desc'``. For example, ``'name desc'``.
            filter: A url-encoded, JSON-serialized Filter protocol buffer
                (see `filter.proto message <https://github.com/kubeflow/pipelines/blob/777c98153daf3dfae82730e14ff37bdddc334c4d/backend/api/filter.proto>`_). For a list of all filter operations ``'op'``, see `here <https://github.com/kubeflow/pipelines/blob/777c98153daf3dfae82730e14ff37bdddc334c4d/sdk/python/kfp/client/client.py#L37-L45>`_. Example:

                  ::

                    json.dumps({
                        "predicates": [{
                            "op": _FILTER_OPERATIONS["EQUALS"],
                            "key": "name",
                            "stringValue": "my-name",
                        }]
                    })

        Returns:
            ``ApiListPipelineVersionsResponse`` object.
        """

        return self._pipelines_api.list_pipeline_versions(
            page_token=page_token,
            page_size=page_size,
            sort_by=sort_by,
            resource_key_type=kfp_server_api.ApiResourceType.PIPELINE,
            resource_key_id=pipeline_id,
            filter=filter)

    def get_pipeline_version(
            self, version_id: str) -> kfp_server_api.ApiPipelineVersion:
        """Gets a pipeline version.

        Args:
            version_id: ID of the pipeline version.

        Returns:
            ``ApiPipelineVersion`` object.
        """
        return self._pipelines_api.get_pipeline_version(version_id=version_id)

    def delete_pipeline_version(self, version_id: str) -> dict:
        """Deletes a pipeline version.

        Args:
            version_id: ID of the pipeline version.

        Returns:
            Empty dictionary.
        """
        return self._pipelines_api.delete_pipeline_version(
            version_id=version_id)


def _add_generated_apis(target_struct: Any, api_module: ModuleType,
                        api_client: kfp_server_api.ApiClient) -> None:
    """Initializes a hierarchical API object based on the generated API module.

    PipelineServiceApi.create_pipeline becomes
    target_struct.pipelines.create_pipeline
    """
    Struct = type('Struct', (), {})

    def camel_case_to_snake_case(name: str) -> str:
        return re.sub('([a-z0-9])([A-Z])', r'\1_\2', name).lower()

    for api_name in dir(api_module):
        if not api_name.endswith('ServiceApi'):
            continue

        short_api_name = camel_case_to_snake_case(
            api_name[0:-len('ServiceApi')]) + 's'
        api_struct = Struct()
        setattr(target_struct, short_api_name, api_struct)
        service_api = getattr(api_module.api, api_name)
        initialized_service_api = service_api(api_client)
        for member_name in dir(initialized_service_api):
            if member_name.startswith('_') or member_name.endswith(
                    '_with_http_info'):
                continue

            bound_member = getattr(initialized_service_api, member_name)
            setattr(api_struct, member_name, bound_member)
    models_struct = Struct()
    for member_name in dir(api_module.models):
        if not member_name[0].islower():
            setattr(models_struct, member_name,
                    getattr(api_module.models, member_name))
    target_struct.api_models = models_struct


def validate_pipeline_resource_name(name: str) -> None:
    REGEX = r'[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*'

    if re.fullmatch(REGEX, name) is None:
        raise ValueError(
            f'Invalid pipeline name: "{name}". Pipeline name must conform to the regex: "{REGEX}".'
        )

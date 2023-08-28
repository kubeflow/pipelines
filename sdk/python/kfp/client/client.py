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
import dataclasses
import datetime
import json
import logging
import os
import re
import tarfile
import tempfile
import time
from types import ModuleType
from typing import Any, Dict, List, Optional, TextIO
import warnings
import zipfile

from google.protobuf import json_format
from kfp import compiler
from kfp.client import auth
from kfp.client import set_volume_credentials
from kfp.dsl import base_component
from kfp.pipeline_spec import pipeline_spec_pb2
import kfp_server_api
import yaml

# Operators on scalar values. Only applies to one of |int_value|,
# |long_value|, |string_value| or |timestamp_value|.
_FILTER_OPERATIONS = {
    'EQUALS': 1,
    'NOT_EQUALS': 2,
    'GREATER_THAN': 3,
    'GREATER_THAN_EQUALS': 5,
    'LESS_THAN': 6,
    'LESS_THAN_EQUALS': 7,
    'IN': 8,
    'IS_SUBSTRING': 9,
}

KF_PIPELINES_ENDPOINT_ENV = 'KF_PIPELINES_ENDPOINT'
KF_PIPELINES_UI_ENDPOINT_ENV = 'KF_PIPELINES_UI_ENDPOINT'
KF_PIPELINES_DEFAULT_EXPERIMENT_NAME = 'KF_PIPELINES_DEFAULT_EXPERIMENT_NAME'
KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME = 'KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME'
KF_PIPELINES_IAP_OAUTH2_CLIENT_ID_ENV = 'KF_PIPELINES_IAP_OAUTH2_CLIENT_ID'
KF_PIPELINES_APP_OAUTH2_CLIENT_ID_ENV = 'KF_PIPELINES_APP_OAUTH2_CLIENT_ID'
KF_PIPELINES_APP_OAUTH2_CLIENT_SECRET_ENV = 'KF_PIPELINES_APP_OAUTH2_CLIENT_SECRET'


@dataclasses.dataclass
class _PipelineDoc:
    pipeline_spec: pipeline_spec_pb2.PipelineSpec
    platform_spec: pipeline_spec_pb2.PlatformSpec

    def to_dict(self) -> dict:
        if self.platform_spec == pipeline_spec_pb2.PlatformSpec():
            return json_format.MessageToDict(self.pipeline_spec)
        else:
            return {
                'pipeline_spec': json_format.MessageToDict(self.pipeline_spec),
                'platform_spec': json_format.MessageToDict(self.platform_spec),
            }


@dataclasses.dataclass
class _JobConfig:
    pipeline_spec: dict
    pipeline_version_reference: kfp_server_api.V2beta1PipelineVersionReference
    runtime_config: kfp_server_api.V2beta1RuntimeConfig


class RunPipelineResult:

    def __init__(self, client: 'Client',
                 run_info: kfp_server_api.V2beta1Run) -> None:
        self._client = client
        self.run_info = run_info
        self.run_id = run_info.run_id

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
            'This client only works with Kubeflow Pipeline v2.0.0-beta.2 '
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
        self._recurring_run_api = kfp_server_api.RecurringRunServiceApi(
            api_client)
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
        sleep_duration: int = 5,
    ) -> kfp_server_api.V2beta1GetHealthzResponse:
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
        namespace: str = None,
    ) -> kfp_server_api.V2beta1Experiment:
        """Creates a new experiment.

        Args:
            name: Name of the experiment.
            description: Description of the experiment.
            namespace: Kubernetes namespace to use. Used for multi-user deployments. For single-user deployments, this should be left as ``None``.

        Returns:
            ``V2beta1Experiment`` object.
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

            experiment = kfp_server_api.V2beta1Experiment(
                display_name=name,
                description=description,
                namespace=namespace,
            )
            experiment = self._experiment_api.create_experiment(body=experiment)

        link = f'{self._get_url_prefix()}/#/experiments/details/{experiment.experiment_id}'
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
                'operation': _FILTER_OPERATIONS['EQUALS'],
                'key': 'display_name',
                'stringValue': name,
            }]
        })
        result = self._pipelines_api.list_pipelines(filter=pipeline_filter)
        if result.pipelines is None:
            return None
        if len(result.pipelines) == 1:
            return result.pipelines[0].pipeline_id
        elif len(result.pipelines) > 1:
            raise ValueError(
                f'Multiple pipelines with the name: {name} found, the name needs to be unique.'
            )
        return None

    def list_experiments(
        self,
        page_token: str = '',
        page_size: int = 10,
        sort_by: str = '',
        namespace: Optional[str] = None,
        filter: Optional[str] = None,
    ) -> kfp_server_api.V2beta1ListExperimentsResponse:
        """Lists experiments.

        Args:
            page_token: Page token for obtaining page from paginated response.
            page_size: Size of the page.
            sort_by: Sort string of format ``'[field_name]', '[field_name] desc'``. For example, ``'display_name desc'``.
            namespace: Kubernetes namespace to use. Used for multi-user deployments. For single-user deployments, this should be left as ``None``.
            filter: A url-encoded, JSON-serialized Filter protocol buffer
                (see `filter.proto message <https://github.com/kubeflow/pipelines/blob/cb7d9a87c999eb1d2280959e5afbeee9e270ef3d/backend/api/v2beta1/filter.proto>`_). Example:

                  ::

                    json.dumps({
                        "predicates": [{
                            "operation": "EQUALS",
                            "key": "display_name",
                            "stringValue": "my-name",
                        }]
                    })

        Returns:
            ``V2beta1ListExperimentsResponse`` object.
        """
        namespace = namespace or self.get_user_namespace()
        return self._experiment_api.list_experiments(
            page_token=page_token,
            page_size=page_size,
            sort_by=sort_by,
            filter=filter,
            namespace=namespace,
        )

    def get_experiment(
        self,
        experiment_id: Optional[str] = None,
        experiment_name: Optional[str] = None,
        namespace: Optional[str] = None,
    ) -> kfp_server_api.V2beta1Experiment:
        """Gets details of an experiment.

        Either ``experiment_id`` or ``experiment_name`` is required.

        Args:
            experiment_id: ID of the experiment.
            experiment_name: Name of the experiment.
            namespace: Kubernetes namespace to use. Used for multi-user deployments.
                For single-user deployments, this should be left as ``None``.

        Returns:
            ``V2beta1Experiment`` object.
        """
        namespace = namespace or self.get_user_namespace()
        if experiment_id is None and experiment_name is None:
            raise ValueError(
                'Either experiment_id or experiment_name is required.')
        if experiment_id is not None:
            return self._experiment_api.get_experiment(
                experiment_id=experiment_id)
        experiment_filter = json.dumps({
            'predicates': [{
                'operation': _FILTER_OPERATIONS['EQUALS'],
                'key': 'display_name',
                'stringValue': experiment_name,
            }]
        })
        if namespace is not None:
            result = self._experiment_api.list_experiments(
                filter=experiment_filter, namespace=namespace)
        else:
            result = self._experiment_api.list_experiments(
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
        return self._experiment_api.archive_experiment(
            experiment_id=experiment_id)

    def unarchive_experiment(self, experiment_id: str) -> dict:
        """Unarchives an experiment.

        Args:
            experiment_id: ID of the experiment.

        Returns:
            Empty dictionary.
        """
        return self._experiment_api.unarchive_experiment(
            experiment_id=experiment_id)

    def delete_experiment(self, experiment_id: str) -> dict:
        """Delete experiment.

        Args:
            experiment_id: ID of the experiment.

        Returns:
            Empty dictionary.
        """
        return self._experiment_api.delete_experiment(
            experiment_id=experiment_id)

    def list_pipelines(
        self,
        page_token: str = '',
        page_size: int = 10,
        sort_by: str = '',
        filter: Optional[str] = None,
        namespace: Optional[str] = None,
    ) -> kfp_server_api.V2beta1ListPipelinesResponse:
        """Lists pipelines.

        Args:
            page_token: Page token for obtaining page from paginated response.
            page_size: Size of the page.
            sort_by: Sort string of format ``'[field_name]', '[field_name] desc'``. For example, ``'display_name desc'``.
            filter: A url-encoded, JSON-serialized Filter protocol buffer
                (see `filter.proto message <https://github.com/kubeflow/pipelines/blob/cb7d9a87c999eb1d2280959e5afbeee9e270ef3d/backend/api/v2beta1/filter.proto>`_). Example:

                  ::

                    json.dumps({
                        "predicates": [{
                            "operation": "EQUALS",
                            "key": "display_name",
                            "stringValue": "my-name",
                        }]
                    })

        Returns:
            ``V2beta1ListPipelinesResponse`` object.
        """
        return self._pipelines_api.list_pipelines(
            namespace=namespace,
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
    ) -> kfp_server_api.V2beta1Run:
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
            ``V2beta1Run`` object.
        """
        job_config = self._create_job_config(
            params=params,
            pipeline_package_path=pipeline_package_path,
            pipeline_id=pipeline_id,
            version_id=version_id,
            enable_caching=enable_caching,
            pipeline_root=pipeline_root,
        )

        run_body = kfp_server_api.V2beta1Run(
            experiment_id=experiment_id,
            display_name=job_name,
            pipeline_spec=job_config.pipeline_spec,
            pipeline_version_reference=job_config.pipeline_version_reference,
            runtime_config=job_config.runtime_config,
            service_account=service_account)

        response = self._run_api.create_run(body=run_body)

        link = f'{self._get_url_prefix()}/#/runs/details/{response.run_id}'
        if self._is_ipython():
            import IPython
            html = (f'<a href="{link}" target="_blank" >Run details</a>.')
            IPython.display.display(IPython.display.HTML(html))
        else:
            print(f'Run details: {link}')

        return response

    def archive_run(self, run_id: str) -> dict:
        """Archives a run.

        Args:
            run_id: ID of the run.

        Returns:
            Empty dictionary.
        """
        return self._run_api.archive_run(run_id=run_id)

    def unarchive_run(self, run_id: str) -> dict:
        """Restores an archived run.

        Args:
            run_id: ID of the run.

        Returns:
            Empty dictionary.
        """
        return self._run_api.unarchive_run(run_id=run_id)

    def delete_run(self, run_id: str) -> dict:
        """Deletes a run.

        Args:
            run_id: ID of the run.

        Returns:
            Empty dictionary.
        """
        return self._run_api.delete_run(run_id=run_id)

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
    ) -> kfp_server_api.V2beta1RecurringRun:
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
            ``V2beta1RecurringRun`` object.
        """

        job_config = self._create_job_config(
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
                'Either interval_second or cron_expression is required.')
        if interval_second is not None:
            trigger = kfp_server_api.V2beta1Trigger(
                periodic_schedule=kfp_server_api.V2beta1PeriodicSchedule(
                    start_time=start_time,
                    end_time=end_time,
                    interval_second=interval_second))
        if cron_expression is not None:
            trigger = kfp_server_api.V2beta1Trigger(
                cron_schedule=kfp_server_api.V2beta1CronSchedule(
                    start_time=start_time,
                    end_time=end_time,
                    cron=cron_expression))

        mode = kfp_server_api.RecurringRunMode.DISABLE
        if enabled:
            mode = kfp_server_api.RecurringRunMode.ENABLE

        job_body = kfp_server_api.V2beta1RecurringRun(
            experiment_id=experiment_id,
            mode=mode,
            pipeline_spec=job_config.pipeline_spec,
            pipeline_version_reference=job_config.pipeline_version_reference,
            runtime_config=job_config.runtime_config,
            display_name=job_name,
            description=description,
            no_catchup=no_catchup,
            trigger=trigger,
            max_concurrency=max_concurrency,
            service_account=service_account)
        return self._recurring_run_api.create_recurring_run(body=job_body)

    def _create_job_config(
        self,
        params: Optional[Dict[str, Any]],
        pipeline_package_path: Optional[str],
        pipeline_id: Optional[str],
        version_id: Optional[str],
        enable_caching: Optional[bool],
        pipeline_root: Optional[str],
    ) -> _JobConfig:
        """Creates a JobConfig with spec and resource_references.

        Args:
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
            A _JobConfig object with attributes .pipeline_spec,
                .pipeline_version_reference, and .runtime_config.
        """
        from_spec = pipeline_package_path is not None
        from_template = pipeline_id is not None or version_id is not None
        if from_spec == from_template:
            raise ValueError(
                'Must specify either `pipeline_pacakge_path` or both `pipeline_id` and `version_id`.'
            )
        if (pipeline_id is None) != (version_id is None):
            raise ValueError(
                'To run a pipeline from an existing template, both `pipeline_id` and `version_id` are required.'
            )

        if params is None:
            params = {}

        pipeline_spec = None
        if pipeline_package_path:
            pipeline_doc = _extract_pipeline_yaml(pipeline_package_path)

            # Caching option set at submission time overrides the compile time
            # settings.
            if enable_caching is not None:
                _override_caching_options(pipeline_doc.pipeline_spec,
                                          enable_caching)
            pipeline_spec = pipeline_doc.to_dict()

        pipeline_version_reference = None
        if pipeline_id is not None and version_id is not None:
            pipeline_version_reference = kfp_server_api.V2beta1PipelineVersionReference(
                pipeline_id=pipeline_id, pipeline_version_id=version_id)

        runtime_config = kfp_server_api.V2beta1RuntimeConfig(
            pipeline_root=pipeline_root,
            parameters=params,
        )
        return _JobConfig(
            pipeline_spec=pipeline_spec,
            pipeline_version_reference=pipeline_version_reference,
            runtime_config=runtime_config,
        )

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
        experiment_id: Optional[str] = None,
    ) -> RunPipelineResult:
        """Runs pipeline on KFP-enabled Kubernetes cluster.

        This command compiles the pipeline function, creates or gets an
        experiment, then submits the pipeline for execution.

        Args:
            pipeline_func: Pipeline function constructed with ``@kfp.dsl.pipeline`` decorator.
            arguments: Arguments to the pipeline function provided as a dict.
            run_name: Name of the run to be shown in the UI.
            experiment_name: Name of the experiment to add the run to. You cannot specify both experiment_name and experiment_id.
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
            experiment_id: ID of the experiment to add the run to. You cannot specify both experiment_id and experiment_name.

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
                experiment_id=experiment_id,
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
        experiment_id: Optional[str] = None,
    ) -> RunPipelineResult:
        """Runs pipeline on KFP-enabled Kubernetes cluster.

        This command takes a local pipeline package, creates or gets an
        experiment, then submits the pipeline for execution.

        Args:
            pipeline_file: A compiled pipeline package file.
            arguments: Arguments to the pipeline function provided as a dict.
            run_name:  Name of the run to be shown in the UI.
            experiment_name: Name of the experiment to add the run to. You cannot specify both experiment_name and experiment_id.
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
            experiment_id: ID of the experiment to add the run to. You cannot specify both experiment_id and experiment_name.

        Returns:
            ``RunPipelineResult`` object containing information about the pipeline run.
        """

        #TODO: Check arguments against the pipeline function
        pipeline_name = os.path.basename(pipeline_file)

        if (experiment_name is not None) and (experiment_id is not None):
            raise ValueError(
                'You cannot specify both experiment_name and experiment_id.')

        if not experiment_id:
            experiment_name = experiment_name or os.environ.get(
                KF_PIPELINES_DEFAULT_EXPERIMENT_NAME, None)
            overridden_experiment_name = os.environ.get(
                KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME, experiment_name)
            if overridden_experiment_name != experiment_name:
                warnings.warn(
                    f'Changing experiment name from "{experiment_name}" to "{overridden_experiment_name}".'
                )
            experiment_name = overridden_experiment_name or 'Default'
            experiment = self.create_experiment(
                name=experiment_name, namespace=namespace)
            experiment_id = experiment.experiment_id

        run_name = run_name or (
            pipeline_name + ' ' +
            datetime.datetime.now().strftime('%Y-%m-%d %H-%M-%S'))

        run_info = self.run_pipeline(
            experiment_id=experiment_id,
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
        warnings.warn(
            '`delete_job` is deprecated. Please use `delete_recurring_run` instead.'
            f'\nReroute to calling `delete_recurring_run(recurring_run_id="{job_id}")`',
            category=DeprecationWarning,
            stacklevel=2)
        return self.delete_recurring_run(recurring_run_id=job_id)

    def delete_recurring_run(self, recurring_run_id: str) -> dict:
        """Deletes a recurring run.

        Args:
            recurring_run_id: ID of the recurring_run.

        Returns:
            Empty dictionary.
        """
        return self._recurring_run_api.delete_recurring_run(
            recurring_run_id=recurring_run_id)

    def disable_job(self, job_id: str) -> dict:
        """Disables a job (recurring run).

        Args:
            job_id: ID of the job.

        Returns:
            Empty dictionary.
        """
        warnings.warn(
            '`disable_job` is deprecated. Please use `disable_recurring_run` instead.'
            f'\nReroute to calling `disable_recurring_run(recurring_run_id="{job_id}")`',
            category=DeprecationWarning,
            stacklevel=2)
        return self.disable_recurring_run(recurring_run_id=job_id)

    def disable_recurring_run(self, recurring_run_id: str) -> dict:
        """Disables a recurring run.

        Args:
            recurring_run_id: ID of the recurring_run.

        Returns:
            Empty dictionary.
        """
        return self._recurring_run_api.disable_recurring_run(
            recurring_run_id=recurring_run_id)

    def enable_job(self, job_id: str) -> dict:
        """Enables a job (recurring run).

        Args:
            job_id: ID of the job.

        Returns:
            Empty dictionary.
        """
        warnings.warn(
            '`enable_job` is deprecated. Please use `enable_recurring_run` instead.'
            f'\nReroute to calling `enable_recurring_run(recurring_run_id="{job_id}")`',
            category=DeprecationWarning,
            stacklevel=2)
        return self.enable_recurring_run(recurring_run_id=job_id)

    def enable_recurring_run(self, recurring_run_id: str) -> dict:
        """Enables a recurring run.

        Args:
            recurring_run_id: ID of the recurring_run.

        Returns:
            Empty dictionary.
        """
        return self._recurring_run_api.enable_recurring_run(
            recurring_run_id=recurring_run_id)

    def list_runs(
        self,
        page_token: str = '',
        page_size: int = 10,
        sort_by: str = '',
        experiment_id: Optional[str] = None,
        namespace: Optional[str] = None,
        filter: Optional[str] = None,
    ) -> kfp_server_api.V2beta1ListRunsResponse:
        """List runs.

        Args:
            page_token: Page token for obtaining page from paginated response.
            page_size: Size of the page.
            sort_by: Sort string of format ``'[field_name]', '[field_name] desc'``. For example, ``'display_name desc'``.
            experiment_id: Experiment ID to filter upon
            namespace: Kubernetes namespace to use. Used for multi-user deployments. For single-user deployments, this should be left as ``None``.
            filter: A url-encoded, JSON-serialized Filter protocol buffer
                (see `filter.proto message <https://github.com/kubeflow/pipelines/blob/cb7d9a87c999eb1d2280959e5afbeee9e270ef3d/backend/api/v2beta1/filter.proto>`_). For a list of all filter operations ``'opertion'``, see `here <https://github.com/kubeflow/pipelines/blob/777c98153daf3dfae82730e14ff37bdddc334c4d/sdk/python/kfp/client/client.py#L37-L45>`_. Example:

                  ::

                    json.dumps({
                        "predicates": [{
                            "operation": "EQUALS",
                            "key": "display_name",
                            "stringValue": "my-name",
                        }]
                    })

          Returns:
            ``V2beta1ListRunsResponse`` object.
        """
        namespace = namespace or self.get_user_namespace()
        if experiment_id is not None:
            return self._run_api.list_runs(
                page_token=page_token,
                page_size=page_size,
                sort_by=sort_by,
                experiment_id=experiment_id,
                filter=filter)

        elif namespace is not None:
            return self._run_api.list_runs(
                page_token=page_token,
                page_size=page_size,
                sort_by=sort_by,
                namespace=namespace,
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
        namespace: Optional[str] = None,
        filter: Optional[str] = None,
    ) -> kfp_server_api.V2beta1ListRecurringRunsResponse:
        """Lists recurring runs.

        Args:
            page_token: Page token for obtaining page from paginated response.
            page_size: Size of the page.
            sort_by: Sort string of format ``'[field_name]', '[field_name] desc'``. For example, ``'display_name desc'``.
            experiment_id: Experiment ID to filter upon.
            namespace: Kubernetes namespace to use. Used for multi-user deployments. For single-user deployments, this should be left as ``None``.
            filter: A url-encoded, JSON-serialized Filter protocol buffer
                (see `filter.proto message <https://github.com/kubeflow/pipelines/blob/cb7d9a87c999eb1d2280959e5afbeee9e270ef3d/backend/api/v2beta1/filter.proto>`_). Example:

                  ::

                    json.dumps({
                        "predicates": [{
                            "operation": "EQUALS",
                            "key": "display_name",
                            "stringValue": "my-name",
                        }]
                    })

        Returns:
            ``V2beta1ListRecurringRunsResponse`` object.
        """
        if experiment_id is not None:
            return self._recurring_run_api.list_recurring_runs(
                page_token=page_token,
                page_size=page_size,
                sort_by=sort_by,
                experiment_id=experiment_id,
                filter=filter)

        elif namespace is not None:
            return self._recurring_run_api.list_recurring_runs(
                page_token=page_token,
                page_size=page_size,
                sort_by=sort_by,
                namespace=namespace,
                filter=filter)

        else:
            return self._recurring_run_api.list_recurring_runs(
                page_token=page_token,
                page_size=page_size,
                sort_by=sort_by,
                filter=filter)

    def get_recurring_run(
        self,
        recurring_run_id: str,
        job_id: Optional[str] = None,
    ) -> kfp_server_api.V2beta1RecurringRun:
        """Gets recurring run details.

        Args:
            recurring_run_id: ID of the recurring run.
            job_id: Deprecated. Use `recurring_run_id` instead.

        Returns:
            ``V2beta1RecurringRun`` object.
        """
        if job_id is not None:
            warnings.warn(
                '`job_id` is deprecated. Please use `recurring_run_id` instead.',
                category=DeprecationWarning,
                stacklevel=2)
            recurring_run_id = recurring_run_id or job_id

        return self._recurring_run_api.get_recurring_run(
            recurring_run_id=recurring_run_id)

    def get_run(self, run_id: str) -> kfp_server_api.V2beta1Run:
        """Gets run details.

        Args:
            run_id: ID of the run.

        Returns:
            ``V2beta1Run`` object.
        """
        return self._run_api.get_run(run_id=run_id)

    def wait_for_run_completion(
        self,
        run_id: str,
        timeout: int,
        sleep_duration: int = 5,
    ) -> kfp_server_api.V2beta1Run:
        """Waits for a run to complete.

        Args:
            run_id: ID of the run.
            timeout: Timeout after which the client should stop waiting for run completion (seconds).
            sleep_duration: Time in seconds between retries.

        Returns:
            ``V2beta1Run`` object.
        """
        state = 'Running:'
        start_time = datetime.datetime.now()
        if isinstance(timeout, datetime.timedelta):
            timeout = timeout.total_seconds()
        is_valid_token = False
        finish_states = ['succeeded', 'failed', 'skipped', 'error']
        while True:
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
            state = get_run_response.state
            elapsed_time = (datetime.datetime.now() -
                            start_time).total_seconds()
            logging.info('Waiting for the job to complete...')
            if elapsed_time > timeout:
                raise TimeoutError('Run timeout')
            if state is not None and state.lower() in finish_states:
                return get_run_response
            time.sleep(sleep_duration)

    def upload_pipeline(
        self,
        pipeline_package_path: str,
        pipeline_name: Optional[str] = None,
        description: Optional[str] = None,
        namespace: Optional[str] = None,
    ) -> kfp_server_api.V2beta1Pipeline:
        """Uploads a pipeline.

        Args:
            pipeline_package_path: Local path to the pipeline package.
            pipeline_name: Name of the pipeline to be shown in the UI.
            description: Description of the pipeline to be shown in the UI.
            namespace: Optional. Kubernetes namespace where the pipeline should
                be uploaded. For single user deployment, leave it as None; For
                multi user, input a namespace where the user is authorized.

        Returns:
            ``V2beta1Pipeline`` object.
        """
        if pipeline_name is None:
            pipeline_doc = _extract_pipeline_yaml(pipeline_package_path)
            pipeline_name = pipeline_doc.pipeline_spec.pipeline_info.name
        validate_pipeline_display_name(pipeline_name)
        response = self._upload_api.upload_pipeline(
            pipeline_package_path,
            name=pipeline_name,
            description=description,
            namespace=namespace)
        link = f'{self._get_url_prefix()}/#/pipelines/details/{response.pipeline_id}'
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
    ) -> kfp_server_api.V2beta1PipelineVersion:
        """Uploads a new version of the pipeline.

        Args:
            pipeline_package_path: Local path to the pipeline package.
            pipeline_version_name:  Name of the pipeline version to be shown in
                the UI.
            pipeline_id: ID of the pipeline.
            pipeline_name: Name of the pipeline.
            description: Description of the pipeline version to show in the UI.

        Returns:
            ``V2beta1PipelineVersion`` object.
        """

        if all([pipeline_id, pipeline_name
               ]) or not any([pipeline_id, pipeline_name]):
            raise ValueError('Either pipeline_id or pipeline_name is required.')

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

        link = f'{self._get_url_prefix()}/#/pipelines/details/{response.pipeline_id}/version/{response.pipeline_version_id}'
        if self._is_ipython():
            import IPython
            html = f'<a href="{link}" target="_blank" >Pipeline details</a>.'
            IPython.display.display(IPython.display.HTML(html))
        else:
            print(f'Pipeline details: {link}')

        return response

    def get_pipeline(self, pipeline_id: str) -> kfp_server_api.V2beta1Pipeline:
        """Gets pipeline details.

        Args:
            pipeline_id: ID of the pipeline.

        Returns:
            ``V2beta1Pipeline`` object.
        """
        return self._pipelines_api.get_pipeline(pipeline_id=pipeline_id)

    def delete_pipeline(self, pipeline_id: str) -> dict:
        """Deletes a pipeline.

        Args:
            pipeline_id: ID of the pipeline.

        Returns:
            Empty dictionary.
        """
        return self._pipelines_api.delete_pipeline(pipeline_id=pipeline_id)

    def list_pipeline_versions(
        self,
        pipeline_id: str,
        page_token: str = '',
        page_size: int = 10,
        sort_by: str = '',
        filter: Optional[str] = None,
    ) -> kfp_server_api.V2beta1ListPipelineVersionsResponse:
        """Lists pipeline versions.

        Args:
            pipeline_id: ID of the pipeline for which to list versions.
            page_token: Page token for obtaining page from paginated response.
            page_size: Size of the page.
            sort_by: Sort string of format ``'[field_name]', '[field_name] desc'``. For example, ``'display_name desc'``.
            filter: A url-encoded, JSON-serialized Filter protocol buffer
                (see `filter.proto message <https://github.com/kubeflow/pipelines/blob/cb7d9a87c999eb1d2280959e5afbeee9e270ef3d/backend/api/v2beta1/filter.proto>`_). Example:

                  ::

                    json.dumps({
                        "predicates": [{
                            "operation": "EQUALS",
                            "key": "display_name",
                            "stringValue": "my-name",
                        }]
                    })

        Returns:
            ``V2beta1ListPipelineVersionsResponse`` object.
        """

        return self._pipelines_api.list_pipeline_versions(
            page_token=page_token,
            page_size=page_size,
            sort_by=sort_by,
            pipeline_id=pipeline_id,
            filter=filter)

    def get_pipeline_version(
        self,
        pipeline_id: str,
        pipeline_version_id: str,
    ) -> kfp_server_api.V2beta1PipelineVersion:
        """Gets a pipeline version.

        Args:
            pipeline_id: ID of the pipeline.
            pipeline_version_id: ID of the pipeline version.

        Returns:
            ``V2beta1PipelineVersion`` object.
        """
        return self._pipelines_api.get_pipeline_version(
            pipeline_id=pipeline_id,
            pipeline_version_id=pipeline_version_id,
        )

    def delete_pipeline_version(
        self,
        pipeline_id: str,
        pipeline_version_id: str,
    ) -> dict:
        """Deletes a pipeline version.p.

        Args:
            pipeline_id: ID of the pipeline.
            pipeline_version_id: ID of the pipeline version.

        Returns:
            Empty dictionary.
        """
        return self._pipelines_api.delete_pipeline_version(
            pipeline_id=pipeline_id,
            pipeline_version_id=pipeline_version_id,
        )


def _add_generated_apis(
    target_struct: Any,
    api_module: ModuleType,
    api_client: kfp_server_api.ApiClient,
) -> None:
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


def validate_pipeline_display_name(name: str) -> None:
    if not name or name.isspace():
        raise ValueError(
            f'Invalid pipeline name. Pipeline name cannot be empty or contain only whitespace.'
        )


def _extract_pipeline_yaml(package_file: str) -> _PipelineDoc:

    def _choose_pipeline_file(file_list: List[str]) -> str:
        pipeline_files = [file for file in file_list if file.endswith('.yaml')]
        if not pipeline_files:
            raise ValueError(
                'Invalid package. Missing pipeline yaml file in the package.')

        if 'pipeline.yaml' in pipeline_files:
            return 'pipeline.yaml'
        elif len(pipeline_files) == 1:
            return pipeline_files[0]
        else:
            raise ValueError(
                'Invalid package. There is no pipeline.json file or there '
                'are multiple yaml files.')

    def _safe_load_yaml(stream: TextIO) -> _PipelineDoc:
        docs = yaml.safe_load_all(stream)
        pipeline_spec_dict = None
        platform_spec_dict = {}
        for doc in docs:
            if pipeline_spec_dict is None:
                pipeline_spec_dict = doc
            else:
                platform_spec_dict.update(doc)

        return _PipelineDoc(
            pipeline_spec=json_format.ParseDict(
                pipeline_spec_dict, pipeline_spec_pb2.PipelineSpec()),
            platform_spec=json_format.ParseDict(
                platform_spec_dict, pipeline_spec_pb2.PlatformSpec()))

    if package_file.endswith('.tar.gz') or package_file.endswith('.tgz'):
        with tarfile.open(package_file, 'r:gz') as tar:
            file_names = [member.name for member in tar if member.isfile()]
            pipeline_file = _choose_pipeline_file(file_names)
            with tar.extractfile(
                    tar.getmember(pipeline_file)) as f:  # type: ignore
                return _safe_load_yaml(f)
    elif package_file.endswith('.zip'):
        with zipfile.ZipFile(package_file, 'r') as zip:
            pipeline_file = _choose_pipeline_file(zip.namelist())
            with zip.open(pipeline_file) as f:
                return _safe_load_yaml(f)
    elif package_file.endswith('.yaml') or package_file.endswith('.yml'):
        with open(package_file, 'r') as f:
            return _safe_load_yaml(f)
    else:
        raise ValueError(
            f'The package_file {package_file} should end with one of the '
            'following formats: [.tar.gz, .tgz, .zip, .yaml, .yml].')


def _override_caching_options(
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    enable_caching: bool,
) -> None:
    """Overrides caching options.

    Args:
        pipeline_spec: The PipelineSpec object to update in-place.
        enable_caching: Overrides options, one of True, False.
    """
    for _, task_spec in pipeline_spec.root.dag.tasks.items():
        task_spec.caching_options.enable_cache = enable_caching

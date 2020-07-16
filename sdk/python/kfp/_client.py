# Copyright 2018 Google LLC
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

import time
import logging
import json
import os
import re
import tarfile
import tempfile
import warnings
import yaml
import zipfile
import datetime
from typing import Mapping, Callable

import kfp
import kfp_server_api

from kfp.compiler import compiler
from kfp.compiler._k8s_helper import sanitize_k8s_name

from kfp._auth import get_auth_token, get_gcp_access_token

# TTL of the access token associated with the client. This is needed because
# `gcloud auth print-access-token` generates a token with TTL=1 hour, after
# which the authentication expires. This TTL is needed for kfp.Client()
# initialized with host=<inverse proxy endpoint>.
# Set to 55 mins to provide some safe margin.
_GCP_ACCESS_TOKEN_TIMEOUT = datetime.timedelta(minutes=55)
# Operators on scalar values. Only applies to one of |int_value|,
# |long_value|, |string_value| or |timestamp_value|.
_FILTER_OPERATIONS = {"UNKNOWN": 0,
    "EQUALS" : 1,
    "NOT_EQUALS" : 2,
    "GREATER_THAN": 3,
    "GREATER_THAN_EQUALS": 5,
    "LESS_THAN": 6,
    "LESS_THAN_EQUALS": 7}

def _add_generated_apis(target_struct, api_module, api_client):
  """Initializes a hierarchical API object based on the generated API module.
  PipelineServiceApi.create_pipeline becomes target_struct.pipelines.create_pipeline
  """
  Struct = type('Struct', (), {})

  def camel_case_to_snake_case(name):
      import re
      return re.sub('([a-z0-9])([A-Z])', r'\1_\2', name).lower()

  for api_name in dir(api_module):
      if not api_name.endswith('ServiceApi'):
          continue

      short_api_name = camel_case_to_snake_case(api_name[0:-len('ServiceApi')]) + 's'
      api_struct = Struct()
      setattr(target_struct, short_api_name, api_struct)
      service_api = getattr(api_module.api, api_name)
      initialized_service_api = service_api(api_client)
      for member_name in dir(initialized_service_api):
          if member_name.startswith('_') or member_name.endswith('_with_http_info'):
              continue

          bound_member = getattr(initialized_service_api, member_name)
          setattr(api_struct, member_name, bound_member)
  models_struct = Struct()
  for member_name in dir(api_module.models):
      if not member_name[0].islower():
          setattr(models_struct, member_name, getattr(api_module.models, member_name))
  target_struct.api_models = models_struct


KF_PIPELINES_ENDPOINT_ENV = 'KF_PIPELINES_ENDPOINT'
KF_PIPELINES_UI_ENDPOINT_ENV = 'KF_PIPELINES_UI_ENDPOINT'
KF_PIPELINES_DEFAULT_EXPERIMENT_NAME = 'KF_PIPELINES_DEFAULT_EXPERIMENT_NAME'
KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME = 'KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME'


class Client(object):
  """API Client for KubeFlow Pipeline.

  Args:
      host: The host name to use to talk to Kubeflow Pipelines. If not set, the in-cluster
        service DNS name will be used, which only works if the current environment is a pod
        in the same cluster (such as a Jupyter instance spawned by Kubeflow's
        JupyterHub). If you have a different connection to cluster, such as a kubectl
        proxy connection, then set it to something like :code:`127.0.0.1:8080/pipeline`.
        If you connect to an IAP enabled cluster, set it to
        :code:`https://<your-deployment>.endpoints.<your-project>.cloud.goog/pipeline`.
      client_id: The client ID used by Identity-Aware Proxy.
      namespace: The namespace where the kubeflow pipeline system is run.
      other_client_id: The client ID used to obtain the auth codes and refresh tokens.
          Reference: https://cloud.google.com/iap/docs/authentication-howto#authenticating_from_a_desktop_app.
      other_client_secret: The client secret used to obtain the auth codes and refresh tokens.
      existing_token: Pass in token directly, it's used for cases better get token outside of SDK, e.x. GCP
          Cloud Functions or caller already has a token
      cookies: CookieJar object containing cookies that will be passed to the pipelines API.
  """

  # in-cluster DNS name of the pipeline service
  IN_CLUSTER_DNS_NAME = 'ml-pipeline.{}.svc.cluster.local:8888'
  KUBE_PROXY_PATH = 'api/v1/namespaces/{}/services/ml-pipeline:http/proxy/'

  LOCAL_KFP_CONTEXT = os.path.expanduser('~/.config/kfp/context.json')

  # TODO: Wrap the configurations for different authentication methods.
  def __init__(self, host=None, client_id=None, namespace='kubeflow', other_client_id=None, other_client_secret=None, existing_token=None, cookies=None):
    """Create a new instance of kfp client.
    """
    host = host or os.environ.get(KF_PIPELINES_ENDPOINT_ENV)
    self._uihost = os.environ.get(KF_PIPELINES_UI_ENDPOINT_ENV, host)
    config = self._load_config(host, client_id, namespace, other_client_id, other_client_secret, existing_token)
    # Save the loaded API client configuration, as a reference if update is
    # needed.
    self._existing_config = config
    api_client = kfp_server_api.api_client.ApiClient(config, cookie=cookies)
    _add_generated_apis(self, kfp_server_api, api_client)
    self._job_api = kfp_server_api.api.job_service_api.JobServiceApi(api_client)
    self._run_api = kfp_server_api.api.run_service_api.RunServiceApi(api_client)
    self._experiment_api = kfp_server_api.api.experiment_service_api.ExperimentServiceApi(api_client)
    self._pipelines_api = kfp_server_api.api.pipeline_service_api.PipelineServiceApi(api_client)
    self._upload_api = kfp_server_api.api.PipelineUploadServiceApi(api_client)
    self._load_context_setting_or_default()

  def _load_config(self, host, client_id, namespace, other_client_id, other_client_secret, existing_token):
    config = kfp_server_api.configuration.Configuration()

    host = host or ''
    # Preprocess the host endpoint to prevent some common user mistakes.
    # This should only be done for non-IAP cases (when client_id is None). IAP requires preserving the protocol.
    if not client_id:
      host = re.sub(r'^(http|https)://', '', host).rstrip('/')

    if host:
      config.host = host

    token = None

    # "existing_token" is designed to accept token generated outside of SDK. Here is an example.
    #
    # https://cloud.google.com/functions/docs/securing/function-identity
    # https://cloud.google.com/endpoints/docs/grpc/service-account-authentication
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
      token = get_auth_token(client_id, other_client_id, other_client_secret)
      self._is_refresh_token = True
    elif self._is_inverse_proxy_host(host):
      token = get_gcp_access_token()
      self._is_refresh_token = False

    if token:
      config.api_key['authorization'] = token
      config.api_key_prefix['authorization'] = 'Bearer'
      return config

    if host:
      # if host is explicitly set with auth token, it's probably a port forward address.
      return config

    import kubernetes as k8s
    in_cluster = True
    try:
      k8s.config.load_incluster_config()
    except:
      in_cluster = False
      pass

    if in_cluster:
      config.host = Client.IN_CLUSTER_DNS_NAME.format(namespace)
      return config

    try:
      k8s.config.load_kube_config(client_configuration=config)
    except:
      print('Failed to load kube config.')
      return config

    if config.host:
      config.host = config.host + '/' + Client.KUBE_PROXY_PATH.format(namespace)
    return config

  def _is_inverse_proxy_host(self, host):
    if host:
      return re.match(r'\S+.googleusercontent.com/{0,1}$', host)
    if re.match(r'\w+', host):
      warnings.warn(
          'The received host is %s, please include the full endpoint address '
          '(with ".(pipelines/notebooks).googleusercontent.com")' % host)
    return False

  def _is_ipython(self):
    """Returns whether we are running in notebook."""
    try:
      import IPython
      ipy = IPython.get_ipython()
      if ipy is None:
        return False
    except ImportError:
      return False

    return True

  def _get_url_prefix(self):
    if self._uihost:
      # User's own connection.
      if self._uihost.startswith('http://') or self._uihost.startswith('https://'):
        return self._uihost
      else:
        return 'http://' + self._uihost

    # In-cluster pod. We could use relative URL.
    return '/pipeline'

  def _load_context_setting_or_default(self):
    if os.path.exists(Client.LOCAL_KFP_CONTEXT):
      with open(Client.LOCAL_KFP_CONTEXT, 'r') as f:
        self._context_setting = json.load(f)
    else:
      self._context_setting = {
        'namespace': '',
      }
      
  def _refresh_api_client_token(self):
    """Refreshes the existing token associated with the kfp_api_client."""
    if getattr(self, '_is_refresh_token', None):
      return

    new_token = get_gcp_access_token()
    self._existing_config.api_key['authorization'] = new_token

  def set_user_namespace(self, namespace):
    """Set user namespace into local context setting file.
    
    This function should only be used when Kubeflow Pipelines is in the multi-user mode.

    Args:
      namespace: kubernetes namespace the user has access to.
    """
    self._context_setting['namespace'] = namespace
    with open(Client.LOCAL_KFP_CONTEXT, 'w') as f:
      json.dump(self._context_setting, f)

  def get_user_namespace(self):
    """Get user namespace in context config.

    Returns:
      namespace: kubernetes namespace from the local context file or empty if it wasn't set.
    """
    return self._context_setting['namespace']

  def create_experiment(self, name, description=None, namespace=None):
    """Create a new experiment.

    Args:
      name: The name of the experiment.
      description: Description of the experiment.
      namespace: Kubernetes namespace where the experiment should be created.
        For single user deployment, leave it as None;
        For multi user, input a namespace where the user is authorized.

    Returns:
      An Experiment object. Most important field is id.
    """
    namespace = namespace or self.get_user_namespace()
    experiment = None
    try:
      experiment = self.get_experiment(experiment_name=name, namespace=namespace)
    except:
      # Ignore error if the experiment does not exist.
      pass

    if not experiment:
      logging.info('Creating experiment {}.'.format(name))

      resource_references = []
      if namespace:
        key = kfp_server_api.models.ApiResourceKey(id=namespace, type=kfp_server_api.models.ApiResourceType.NAMESPACE)
        reference = kfp_server_api.models.ApiResourceReference(key=key, relationship=kfp_server_api.models.ApiRelationship.OWNER)
        resource_references.append(reference)

      experiment = kfp_server_api.models.ApiExperiment(
        name=name,
        description=description,
        resource_references=resource_references)
      experiment = self._experiment_api.create_experiment(body=experiment)

    if self._is_ipython():
      import IPython
      html = \
          ('Experiment link <a href="%s/#/experiments/details/%s" target="_blank" >here</a>'
          % (self._get_url_prefix(), experiment.id))
      IPython.display.display(IPython.display.HTML(html))
    return experiment

  def get_pipeline_id(self, name):
    """Find the id of a pipeline by name.

    Args:
      name: Pipeline name.

    Returns:
      Returns the pipeline id if a pipeline with the name exists.
    """
    pipeline_filter = json.dumps({
      "predicates": [
        {
          "op":  _FILTER_OPERATIONS["EQUALS"],
          "key": "name",
          "stringValue": name,
        }
      ]
    })
    result = self._pipelines_api.list_pipelines(filter=pipeline_filter)
    if len(result.pipelines)==1:
      return result.pipelines[0].id
    elif len(result.pipelines)>1:
      raise ValueError("Multiple pipelines with the name: {} found, the name needs to be unique".format(name))
    return None

  def list_experiments(self, page_token='', page_size=10, sort_by='', namespace=None):
    """List experiments.

    Args:
      page_token: Token for starting of the page.
      page_size: Size of the page.
      sort_by: Can be '[field_name]', '[field_name] des'. For example, 'name desc'.
      namespace: Kubernetes namespace where the experiment was created.
        For single user deployment, leave it as None;
        For multi user, input a namespace where the user is authorized.
  
    Returns:
      A response object including a list of experiments and next page token.
    """
    namespace = namespace or self.get_user_namespace()
    response = self._experiment_api.list_experiment(
      page_token=page_token,
      page_size=page_size,
      sort_by=sort_by,
      resource_reference_key_type=kfp_server_api.models.api_resource_type.ApiResourceType.NAMESPACE,
      resource_reference_key_id=namespace)
    return response

  def get_experiment(self, experiment_id=None, experiment_name=None, namespace=None):
    """Get details of an experiment

    Either experiment_id or experiment_name is required

    Args:
      experiment_id: Id of the experiment. (Optional)
      experiment_name: Name of the experiment. (Optional)
      namespace: Kubernetes namespace where the experiment was created.
        For single user deployment, leave it as None;
        For multi user, input the namespace where the user is authorized.

    Returns:
      A response object including details of a experiment.

    Throws:
      Exception if experiment is not found or None of the arguments is provided
    """
    namespace = namespace or self.get_user_namespace()
    if experiment_id is None and experiment_name is None:
      raise ValueError('Either experiment_id or experiment_name is required')
    if experiment_id is not None:
      return self._experiment_api.get_experiment(id=experiment_id)
    next_page_token = ''
    while next_page_token is not None:
      list_experiments_response = self.list_experiments(page_size=100, page_token=next_page_token, namespace=namespace)
      next_page_token = list_experiments_response.next_page_token
      for experiment in list_experiments_response.experiments or []:
        if experiment.name == experiment_name:
          return self._experiment_api.get_experiment(id=experiment.id)
    raise ValueError('No experiment is found with name {}.'.format(experiment_name))

  def _extract_pipeline_yaml(self, package_file):
    def _choose_pipeline_yaml_file(file_list) -> str:
      yaml_files = [file for file in file_list if file.endswith('.yaml')]
      if len(yaml_files) == 0:
        raise ValueError('Invalid package. Missing pipeline yaml file in the package.')

      if 'pipeline.yaml' in yaml_files:
        return 'pipeline.yaml'
      else:
        if len(yaml_files) == 1:
          return yaml_files[0]
        raise ValueError('Invalid package. There is no pipeline.yaml file and there are multiple yaml files.')

    if package_file.endswith('.tar.gz') or package_file.endswith('.tgz'):
      with tarfile.open(package_file, "r:gz") as tar:
        file_names = [member.name for member in tar if member.isfile()]
        pipeline_yaml_file = _choose_pipeline_yaml_file(file_names)
        with tar.extractfile(tar.getmember(pipeline_yaml_file)) as f:
          return yaml.safe_load(f)
    elif package_file.endswith('.zip'):
      with zipfile.ZipFile(package_file, 'r') as zip:
        pipeline_yaml_file = _choose_pipeline_yaml_file(zip.namelist())
        with zip.open(pipeline_yaml_file) as f:
          return yaml.safe_load(f)
    elif package_file.endswith('.yaml') or package_file.endswith('.yml'):
      with open(package_file, 'r') as f:
        return yaml.safe_load(f)
    else:
      raise ValueError('The package_file '+ package_file + ' should end with one of the following formats: [.tar.gz, .tgz, .zip, .yaml, .yml]')

  def list_pipelines(self, page_token='', page_size=10, sort_by=''):
    """List pipelines.

    Args:
      page_token: Token for starting of the page.
      page_size: Size of the page.
      sort_by: one of 'field_name', 'field_name desc'. For example, 'name desc'.

    Returns:
      A response object including a list of pipelines and next page token.
    """
    return self._pipelines_api.list_pipelines(page_token=page_token, page_size=page_size, sort_by=sort_by)

  def list_pipeline_versions(self, pipeline_id: str, page_token='', page_size=10, sort_by=''):
    """List all versions of a given pipeline.

    Args:
      pipeline_id: The id of a pipeline.
      page_token: Token for starting of the page.
      page_size: Size of the page.
        sort_by: one of 'field_name', 'field_name desc'. For example, 'name desc'.

    Returns:
      A response object including a list of pipeline versions and next page token.
    """
    return self._pipelines_api.list_pipeline_versions(
        resource_key_type="PIPELINE",
        resource_key_id=pipeline_id,
        page_token=page_token,
        page_size=page_size,
        sort_by=sort_by
    )

  # TODO: provide default namespace, similar to kubectl default namespaces.
  def run_pipeline(self, experiment_id, job_name, pipeline_package_path=None, params={}, pipeline_id=None, version_id=None):
    """Run a specified pipeline.

    Args:
      experiment_id: The id of an experiment.
      job_name: Name of the job.
      pipeline_package_path: Local path of the pipeline package(the filename should end with one of the following .tar.gz, .tgz, .zip, .yaml, .yml).
      params: A dictionary with key (string) as param name and value (string) as as param value.
      pipeline_id: The id of a pipeline.
      version_id: The id of a pipeline version.
        If both pipeline_id and version_id are specified, version_id will take precendence.
        If only pipeline_id is specified, the default version of this pipeline is used to create the run.

    Returns:
      A run object. Most important field is id.
    """
    job_config = self._create_job_config(
      experiment_id=experiment_id,
      params=params,
      pipeline_package_path=pipeline_package_path,
      pipeline_id=pipeline_id,
      version_id=version_id)
    run_body = kfp_server_api.models.ApiRun(
        pipeline_spec=job_config.spec, resource_references=job_config.resource_references, name=job_name)

    response = self._run_api.create_run(body=run_body)

    if self._is_ipython():
      import IPython
      html = ('Run link <a href="%s/#/runs/details/%s" target="_blank" >here</a>'
              % (self._get_url_prefix(), response.run.id))
      IPython.display.display(IPython.display.HTML(html))
    return response.run

  def create_recurring_run(self, experiment_id, job_name, description=None, start_time=None, end_time=None, interval_second=None, cron_expression=None, max_concurrency=1, no_catchup=None, params={}, pipeline_package_path=None, pipeline_id=None, version_id=None, enabled=True):
    """Create a recurring run.

    Args:
      experiment_id: The string id of an experiment.
      job_name: Name of the job.
      description: An optional job description.
      start_time: The RFC3339 time string of the time when to start the job.
      end_time: The RFC3339 time string of the time when to end the job.
      interval_second: Integer indicating the seconds between two recurring runs in for a periodic schedule.
      cron_expression: A cron expression representing a set of times, using 5 space-separated fields, e.g. "0 0 9 ? * 2-6".
      max_concurrency: Integer indicating how many jobs can be run in parallel.
      no_catchup: Whether the recurring run should catch up if behind schedule.
        For example, if the recurring run is paused for a while and re-enabled
        afterwards. If no_catchup=False, the scheduler will catch up on (backfill) each
        missed interval. Otherwise, it only schedules the latest interval if more than one interval
        is ready to be scheduled.
        Usually, if your pipeline handles backfill internally, you should turn catchup
        off to avoid duplicate backfill. (default: {False})
      pipeline_package_path: Local path of the pipeline package(the filename should end with one of the following .tar.gz, .tgz, .zip, .yaml, .yml).
      params: A dictionary with key (string) as param name and value (string) as param value.
      pipeline_id: The string ID of a pipeline.
      version_id: The string ID of a pipeline version. 
        If both pipeline_id and version_id are specified, pipeline_id will take precendence
        This will change in a future version, so it is recommended to use version_id by itself.
      enabled: A bool indicating whether the recurring run is enabled or disabled.

    Returns:
      A Job object. Most important field is id.
    """
    job_config = self._create_job_config(
      experiment_id=experiment_id,
      params=params,
      pipeline_package_path=pipeline_package_path,
      pipeline_id=pipeline_id,
      version_id=version_id)

    if all([interval_second, cron_expression]) or not any([interval_second, cron_expression]):
      raise ValueError('Either interval_second or cron_expression is required')
    if interval_second is not None:
      trigger = kfp_server_api.models.ApiTrigger(
        periodic_schedule=kfp_server_api.models.ApiPeriodicSchedule(
          start_time=start_time, end_time=end_time, interval_second=interval_second)
      )
    if cron_expression is not None:
      trigger = kfp_server_api.models.ApiTrigger(
        cron_schedule=kfp_server_api.models.ApiCronSchedule(
        start_time=start_time, end_time=end_time, cron=cron_expression)
      )

    job_body = kfp_server_api.models.ApiJob(
        enabled=enabled,
        pipeline_spec=job_config.spec,
        resource_references=job_config.resource_references,
        name=job_name,
        description=description,
        no_catchup=no_catchup,
        trigger=trigger,
        max_concurrency=max_concurrency)
    return self._job_api.create_job(body=job_body)

  def _create_job_config(self, experiment_id, params, pipeline_package_path, pipeline_id, version_id):
    """Create a JobConfig with spec and resource_references.

    Args:
      experiment_id: The id of an experiment.
      pipeline_package_path: Local path of the pipeline package(the filename should end with one of the following .tar.gz, .tgz, .zip, .yaml, .yml).
      params: A dictionary with key (string) as param name and value (string) as param value.
      pipeline_id: The id of a pipeline.
      version_id: The id of a pipeline version. 
        If both pipeline_id and version_id are specified, pipeline_id will take precendence
        This will change in a future version, so it is recommended to use version_id by itself.

    Returns:
      A JobConfig object with attributes spec and resource_reference.
    """
    
    class JobConfig:
      def __init__(self, spec, resource_references):
        self.spec = spec
        self.resource_references = resource_references

    pipeline_json_string = None
    if pipeline_package_path:
      pipeline_obj = self._extract_pipeline_yaml(pipeline_package_path)
      pipeline_json_string = json.dumps(pipeline_obj)
    api_params = [kfp_server_api.ApiParameter(
        name=sanitize_k8s_name(name=k, allow_capital_underscore=True),
        value=str(v)) for k,v in params.items()]
    resource_references = []
    key = kfp_server_api.models.ApiResourceKey(id=experiment_id,
                                        type=kfp_server_api.models.ApiResourceType.EXPERIMENT)
    reference = kfp_server_api.models.ApiResourceReference(key=key,
                                                           relationship=kfp_server_api.models.ApiRelationship.OWNER)
    resource_references.append(reference)

    if version_id:
      key = kfp_server_api.models.ApiResourceKey(id=version_id,
                                                 type=kfp_server_api.models.ApiResourceType.PIPELINE_VERSION)
      reference = kfp_server_api.models.ApiResourceReference(key=key,
                                                             relationship=kfp_server_api.models.ApiRelationship.CREATOR)
      resource_references.append(reference)

    spec = kfp_server_api.models.ApiPipelineSpec(
        pipeline_id=pipeline_id,
        workflow_manifest=pipeline_json_string,
        parameters=api_params)
    return JobConfig(spec=spec, resource_references=resource_references)

  def create_run_from_pipeline_func(self, pipeline_func: Callable, arguments: Mapping[str, str], run_name=None, experiment_name=None, pipeline_conf: kfp.dsl.PipelineConf = None, namespace=None):
    """Runs pipeline on KFP-enabled Kubernetes cluster.

    This command compiles the pipeline function, creates or gets an experiment and submits the pipeline for execution.

    Args:
      pipeline_func: A function that describes a pipeline by calling components and composing them into execution graph.
      arguments: Arguments to the pipeline function provided as a dict.
      run_name: Optional. Name of the run to be shown in the UI.
      experiment_name: Optional. Name of the experiment to add the run to.
      namespace: Kubernetes namespace where the pipeline runs are created.
        For single user deployment, leave it as None;
        For multi user, input a namespace where the user is authorized
    """
    #TODO: Check arguments against the pipeline function
    pipeline_name = pipeline_func.__name__
    run_name = run_name or pipeline_name + ' ' + datetime.datetime.now().strftime('%Y-%m-%d %H-%M-%S')
    with tempfile.TemporaryDirectory() as tmpdir:
      pipeline_package_path = os.path.join(tmpdir, 'pipeline.yaml')
      compiler.Compiler().compile(pipeline_func, pipeline_package_path, pipeline_conf=pipeline_conf)
      return self.create_run_from_pipeline_package(pipeline_package_path, arguments, run_name, experiment_name, namespace)

  def create_run_from_pipeline_package(self, pipeline_file: str, arguments: Mapping[str, str], run_name=None, experiment_name=None, namespace=None):
    """Runs pipeline on KFP-enabled Kubernetes cluster.

    This command compiles the pipeline function, creates or gets an experiment and submits the pipeline for execution.

    Args:
      pipeline_file: A compiled pipeline package file.
      arguments: Arguments to the pipeline function provided as a dict.
      run_name: Optional. Name of the run to be shown in the UI.
      experiment_name: Optional. Name of the experiment to add the run to.
      namespace: Kubernetes namespace where the pipeline runs are created.
        For single user deployment, leave it as None;
        For multi user, input a namespace where the user is authorized
    """

    class RunPipelineResult:
      def __init__(self, client, run_info):
        self._client = client
        self.run_info = run_info
        self.run_id = run_info.id

      def wait_for_run_completion(self, timeout=None):
        timeout = timeout or datetime.timedelta.max
        return self._client.wait_for_run_completion(self.run_id, timeout)

      def __repr__(self):
        return 'RunPipelineResult(run_id={})'.format(self.run_id)

    #TODO: Check arguments against the pipeline function
    pipeline_name = os.path.basename(pipeline_file)
    experiment_name = experiment_name or os.environ.get(KF_PIPELINES_DEFAULT_EXPERIMENT_NAME, None)
    overridden_experiment_name = os.environ.get(KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME, experiment_name)
    if overridden_experiment_name != experiment_name:
      import warnings
      warnings.warn('Changing experiment name from "{}" to "{}".'.format(experiment_name, overridden_experiment_name))
    experiment_name = overridden_experiment_name or 'Default'
    run_name = run_name or (pipeline_name + ' ' +
                            datetime.datetime.now().strftime(
                                '%Y-%m-%d %H-%M-%S'))
    experiment = self.create_experiment(name=experiment_name, namespace=namespace)
    run_info = self.run_pipeline(experiment.id, run_name, pipeline_file, arguments)
    return RunPipelineResult(self, run_info)

  def list_runs(self, page_token='', page_size=10, sort_by='', experiment_id=None, namespace=None):
    """List runs, optionally can be filtered by experiment or namespace.

    Args:
      page_token: Token for starting of the page.
      page_size: Size of the page.
      sort_by: One of 'field_name', 'field_name desc'. For example, 'name desc'.
      experiment_id: Experiment id to filter upon
      namespace: Kubernetes namespace to filter upon.
        For single user deployment, leave it as None;
        For multi user, input a namespace where the user is authorized.

    Returns:
      A response object including a list of experiments and next page token.
    """
    namespace = namespace or self.get_user_namespace()
    if experiment_id is not None:
      response = self._run_api.list_runs(page_token=page_token, page_size=page_size, sort_by=sort_by, resource_reference_key_type=kfp_server_api.models.api_resource_type.ApiResourceType.EXPERIMENT, resource_reference_key_id=experiment_id)
    elif namespace:
      response = self._run_api.list_runs(page_token=page_token, page_size=page_size, sort_by=sort_by, resource_reference_key_type=kfp_server_api.models.api_resource_type.ApiResourceType.NAMESPACE, resource_reference_key_id=namespace)
    else:
      response = self._run_api.list_runs(page_token=page_token, page_size=page_size, sort_by=sort_by)
    return response

  def list_recurring_runs(self, page_token='', page_size=10, sort_by='', experiment_id=None):
    """List recurring runs.

    Args:
      page_token: Token for starting of the page.
      page_size: Size of the page.
      sort_by: One of 'field_name', 'field_name desc'. For example, 'name desc'.
      experiment_id: Experiment id to filter upon.

    Returns:
      A response object including a list of recurring_runs and next page token.
    """
    if experiment_id is not None:
      response = self._job_api.list_jobs(page_token=page_token, page_size=page_size, sort_by=sort_by, resource_reference_key_type=kfp_server_api.models.api_resource_type.ApiResourceType.EXPERIMENT, resource_reference_key_id=experiment_id)
    else:
      response = self._job_api.list_jobs(page_token=page_token, page_size=page_size, sort_by=sort_by)
    return response

  def get_recurring_run(self, job_id):
    """Get recurring_run details.

    Args:
      job_id: id of the recurring_run.

    Returns:
      A response object including details of a recurring_run.

    Throws:
      Exception if recurring_run is not found.
    """
    return self._job_api.get_job(id=job_id)


  def get_run(self, run_id):
    """Get run details.

    Args:
      run_id: id of the run.

    Returns:
      A response object including details of a run.

    Throws:
      Exception if run is not found.
    """
    return self._run_api.get_run(run_id=run_id)

  def wait_for_run_completion(self, run_id, timeout):
    """Waits for a run to complete.

    Args:
      run_id: Run id, returned from run_pipeline.
      timeout: Timeout in seconds.

    Returns:
      A run detail object: Most important fields are run and pipeline_runtime.

    Raises:
      TimeoutError: if the pipeline run failed to finish before the specified timeout.
    """
    status = 'Running:'
    start_time = datetime.datetime.now()
    last_token_refresh_time = datetime.datetime.now()
    while (status is None or
           status.lower() not in ['succeeded', 'failed', 'skipped', 'error']):
      # Refreshes the access token before it hits the TTL.
      if (datetime.datetime.now() - last_token_refresh_time
          > _GCP_ACCESS_TOKEN_TIMEOUT):
        self._refresh_api_client_token()
        last_token_refresh_time = datetime.datetime.now()
        
      get_run_response = self._run_api.get_run(run_id=run_id)
      status = get_run_response.run.status
      elapsed_time = (datetime.datetime.now() - start_time).seconds
      logging.info('Waiting for the job to complete...')
      if elapsed_time > timeout:
        raise TimeoutError('Run timeout')
      time.sleep(5)
    return get_run_response

  def _get_workflow_json(self, run_id):
    """Get the workflow json.

    Args:
      run_id: run id, returned from run_pipeline.

    Returns:
      workflow: Json workflow
    """
    get_run_response = self._run_api.get_run(run_id=run_id)
    workflow = get_run_response.pipeline_runtime.workflow_manifest
    workflow_json = json.loads(workflow)
    return workflow_json

  def upload_pipeline(
    self,
    pipeline_package_path: str = None,
    pipeline_name: str = None,
    description: str = None,
  ):
    """Uploads the pipeline to the Kubeflow Pipelines cluster.

    Args:
      pipeline_package_path: Local path to the pipeline package.
      pipeline_name: Optional. Name of the pipeline to be shown in the UI.
      description: Optional. Description of the pipeline to be shown in the UI.

    Returns:
      Server response object containing pipleine id and other information.
    """

    response = self._upload_api.upload_pipeline(pipeline_package_path, name=pipeline_name, description=description)
    if self._is_ipython():
      import IPython
      html = 'Pipeline link <a href=%s/#/pipelines/details/%s>here</a>' % (self._get_url_prefix(), response.id)
      IPython.display.display(IPython.display.HTML(html))
    return response

  def get_pipeline(self, pipeline_id):
    """Get pipeline details.

    Args:
      pipeline_id: id of the pipeline.

    Returns:
      A response object including details of a pipeline.

    Throws:
      Exception if pipeline is not found.
    """
    return self._pipelines_api.get_pipeline(id=pipeline_id)

  def delete_pipeline(self, pipeline_id):
    """Delete pipeline.

    Args:
      pipeline_id: id of the pipeline.

    Returns:
      Object. If the method is called asynchronously, returns the request thread.

    Throws:
      Exception if pipeline is not found.
    """
    return self._pipelines_api.delete_pipeline(id=pipeline_id)

  def list_pipeline_versions(self, pipeline_id, page_token='', page_size=10, sort_by=''):
    """Lists pipeline versions.

    Args:
      pipeline_id: Id of the pipeline to list versions
      page_token: Token for starting of the page.
      page_size: Size of the page.
      sort_by: One of 'field_name', 'field_name des'. For example, 'name des'.

    Returns:
      A response object including a list of versions and next page token.
    """

    return self._pipelines_api.list_pipeline_versions(page_token=page_token, page_size=page_size, sort_by=sort_by, resource_key_type=kfp_server_api.models.api_resource_type.ApiResourceType.PIPELINE, resource_key_id=pipeline_id)

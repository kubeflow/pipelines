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
import tarfile
import tempfile
import zipfile
import yaml
from datetime import datetime
from typing import Mapping, Callable

import kfp_server_api

from kfp.compiler import compiler
from kfp.compiler import _k8s_helper

from kfp._auth import get_auth_token



def _add_generated_apis(target_struct, api_module, api_client):
  '''Initializes a hierarchical API object based on the generated API module.
  PipelineServiceApi.create_pipeline becomes target_struct.pipelines.create_pipeline
  '''
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
      service_api = getattr(api_module, api_name)
      initialized_service_api = service_api(api_client)
      for member_name in dir(initialized_service_api):
          if member_name.startswith('_') or member_name.endswith('_with_http_info'):
              continue

          bound_member = getattr(initialized_service_api, member_name)
          setattr(api_struct, member_name, bound_member)


KF_PIPELINES_ENDPOINT_ENV = 'KF_PIPELINES_ENDPOINT'
KF_PIPELINES_UI_ENDPOINT_ENV = 'KF_PIPELINES_UI_ENDPOINT'

class Client(object):
  """ API Client for KubeFlow Pipeline.
  """

  # in-cluster DNS name of the pipeline service
  IN_CLUSTER_DNS_NAME = 'ml-pipeline.{}.svc.cluster.local:8888'
  KUBE_PROXY_PATH = 'api/v1/namespaces/{}/services/ml-pipeline:http/proxy/'

  def __init__(self, host=None, client_id=None, namespace='kubeflow'):
    """Create a new instance of kfp client.

    Args:
      host: the host name to use to talk to Kubeflow Pipelines. If not set, the in-cluster
          service DNS name will be used, which only works if the current environment is a pod
          in the same cluster (such as a Jupyter instance spawned by Kubeflow's
          JupyterHub). If you have a different connection to cluster, such as a kubectl
          proxy connection, then set it to something like "127.0.0.1:8080/pipeline.
          If you connect to an IAP enabled cluster, set it to
          https://<your-deployment>.endpoints.<your-project>.cloud.goog/pipeline".
      client_id: The client ID used by Identity-Aware Proxy.
    """

    self._uihost = os.environ.get(KF_PIPELINES_UI_ENDPOINT_ENV, host)
    config = self._load_config(host, client_id, namespace)
    api_client = kfp_server_api.api_client.ApiClient(config)
    _add_generated_apis(self, kfp_server_api.api, api_client)
    self._run_api = kfp_server_api.api.run_service_api.RunServiceApi(api_client)
    self._experiment_api = kfp_server_api.api.experiment_service_api.ExperimentServiceApi(api_client)
    self._pipelines_api = kfp_server_api.api.pipeline_service_api.PipelineServiceApi(api_client)
    self._upload_api = kfp_server_api.api.PipelineUploadServiceApi(api_client)

  def _load_config(self, host, client_id, namespace):
    config = kfp_server_api.configuration.Configuration()
    host = host or os.environ.get(KF_PIPELINES_ENDPOINT_ENV)
    if host:
      config.host = host

    token = None
    if host and client_id:
      # fetch IAP auth token
      token = get_auth_token(client_id)

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
      config.host = os.path.join(config.host, Client.KUBE_PROXY_PATH.format(namespace))
    return config

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

  def create_experiment(self, name):
    """Create a new experiment.
    Args:
      name: the name of the experiment.
    Returns:
      An Experiment object. Most important field is id.
    """

    experiment = None
    try:
      experiment = self.get_experiment(experiment_name=name)
    except:
      # Ignore error if the experiment does not exist.
      pass

    if not experiment:
      logging.info('Creating experiment {}.'.format(name))
      experiment = kfp_server_api.models.ApiExperiment(name=name)
      experiment = self._experiment_api.create_experiment(body=experiment)

    if self._is_ipython():
      import IPython
      html = \
          ('Experiment link <a href="%s/#/experiments/details/%s" target="_blank" >here</a>'
          % (self._get_url_prefix(), experiment.id))
      IPython.display.display(IPython.display.HTML(html))
    return experiment

  def list_experiments(self, page_token='', page_size=10, sort_by=''):
    """List experiments.
    Args:
      page_token: token for starting of the page.
      page_size: size of the page.
      sort_by: can be '[field_name]', '[field_name] des'. For example, 'name des'.
    Returns:
      A response object including a list of experiments and next page token.
    """
    response = self._experiment_api.list_experiment(
        page_token=page_token, page_size=page_size, sort_by=sort_by)
    return response

  def get_experiment(self, experiment_id=None, experiment_name=None):
    """Get details of an experiment
    Either experiment_id or experiment_name is required
    Args:
      experiment_id: id of the experiment. (Optional)
      experiment_name: name of the experiment. (Optional)
    Returns:
      A response object including details of a experiment.
    Throws:
      Exception if experiment is not found or None of the arguments is provided
    """
    if experiment_id is None and experiment_name is None:
      raise ValueError('Either experiment_id or experiment_name is required')
    if experiment_id is not None:
      return self._experiment_api.get_experiment(id=experiment_id)
    next_page_token = ''
    while next_page_token is not None:
      list_experiments_response = self.list_experiments(page_size=100, page_token=next_page_token)
      next_page_token = list_experiments_response.next_page_token
      for experiment in list_experiments_response.experiments:
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
      raise ValueError('The package_file '+ package_file + ' should ends with one of the following formats: [.tar.gz, .tgz, .zip, .yaml, .yml]')

  def list_pipelines(self, page_token='', page_size=10, sort_by=''):
    """List pipelines.
    Args:
      page_token: token for starting of the page.
      page_size: size of the page.
      sort_by: one of 'field_name', 'field_name des'. For example, 'name des'.
    Returns:
      A response object including a list of pipelines and next page token.
    """
    return self._pipelines_api.list_pipelines(page_token=page_token, page_size=page_size, sort_by=sort_by)

  def run_pipeline(self, experiment_id, job_name, pipeline_package_path=None, params={}, pipeline_id=None):
    """Run a specified pipeline.

    Args:
      experiment_id: The string id of an experiment.
      job_name: name of the job.
      pipeline_package_path: local path of the pipeline package(the filename should end with one of the following .tar.gz, .tgz, .zip, .yaml, .yml).
      params: a dictionary with key (string) as param name and value (string) as as param value.
      pipeline_id: the string ID of a pipeline.

    Returns:
      A run object. Most important field is id.
    """

    pipeline_json_string = None
    if pipeline_package_path:
      pipeline_obj = self._extract_pipeline_yaml(pipeline_package_path)
      pipeline_json_string = json.dumps(pipeline_obj)
    api_params = [kfp_server_api.ApiParameter(name=_k8s_helper.K8sHelper.sanitize_k8s_name(k), value=str(v))
                  for k,v in params.items()]
    key = kfp_server_api.models.ApiResourceKey(id=experiment_id,
                                        type=kfp_server_api.models.ApiResourceType.EXPERIMENT)
    reference = kfp_server_api.models.ApiResourceReference(key, kfp_server_api.models.ApiRelationship.OWNER)
    spec = kfp_server_api.models.ApiPipelineSpec(
        pipeline_id=pipeline_id,
        workflow_manifest=pipeline_json_string,
        parameters=api_params)
    run_body = kfp_server_api.models.ApiRun(
        pipeline_spec=spec, resource_references=[reference], name=job_name)

    response = self._run_api.create_run(body=run_body)

    if self._is_ipython():
      import IPython
      html = ('Run link <a href="%s/#/runs/details/%s" target="_blank" >here</a>'
              % (self._get_url_prefix(), response.run.id))
      IPython.display.display(IPython.display.HTML(html))
    return response.run

  def create_run_from_pipeline_func(self, pipeline_func: Callable, arguments: Mapping[str, str], run_name=None, experiment_name=None):
    '''Runs pipeline on KFP-enabled Kubernetes cluster.
    This command compiles the pipeline function, creates or gets an experiment and submits the pipeline for execution.

    Args:
      pipeline_func: A function that describes a pipeline by calling components and composing them into execution graph.
      arguments: Arguments to the pipeline function provided as a dict.
      run_name: Optional. Name of the run to be shown in the UI.
      experiment_name: Optional. Name of the experiment to add the run to.
    '''
    #TODO: Check arguments against the pipeline function
    pipeline_name = pipeline_func.__name__
    run_name = run_name or pipeline_name + ' ' + datetime.now().strftime('%Y-%m-%d %H-%M-%S')
    try:
      (_, pipeline_package_path) = tempfile.mkstemp(suffix='.zip')
      compiler.Compiler().compile(pipeline_func, pipeline_package_path)
      return self.create_run_from_pipeline_package(pipeline_package_path, arguments, run_name, experiment_name)
    finally:
      os.remove(pipeline_package_path)

  def create_run_from_pipeline_package(self, pipeline_file: str, arguments: Mapping[str, str], run_name=None, experiment_name=None):
    '''Runs pipeline on KFP-enabled Kubernetes cluster.
    This command compiles the pipeline function, creates or gets an experiment and submits the pipeline for execution.

    Args:
      pipeline_file: A compiled pipeline package file.
      arguments: Arguments to the pipeline function provided as a dict.
      run_name: Optional. Name of the run to be shown in the UI.
      experiment_name: Optional. Name of the experiment to add the run to.
    '''

    class RunPipelineResult:
      def __init__(self, client, run_info):
        self._client = client
        self.run_info = run_info
        self.run_id = run_info.id

      def wait_for_run_completion(self, timeout=None):
        timeout = timeout or datetime.datetime.max - datetime.datetime.min
        return self._client.wait_for_run_completion(timeout)

      def __str__(self):
        return '<RunPipelineResult(run_id={})>'.format(self.run_id)

    #TODO: Check arguments against the pipeline function
    pipeline_name = os.path.basename(pipeline_file)
    experiment_name = experiment_name or 'Default'
    run_name = run_name or pipeline_name + ' ' + datetime.now().strftime('%Y-%m-%d %H-%M-%S')
    experiment = self.create_experiment(name=experiment_name)
    run_info = self.run_pipeline(experiment.id, run_name, pipeline_file, arguments)
    return RunPipelineResult(self, run_info)

  def list_runs(self, page_token='', page_size=10, sort_by='', experiment_id=None):
    """List runs.
    Args:
      page_token: token for starting of the page.
      page_size: size of the page.
      sort_by: one of 'field_name', 'field_name des'. For example, 'name des'.
      experiment_id: experiment id to filter upon
    Returns:
      A response object including a list of experiments and next page token.
    """
    if experiment_id is not None:
      response = self._run_api.list_runs(page_token=page_token, page_size=page_size, sort_by=sort_by, resource_reference_key_type=kfp_server_api.models.api_resource_type.ApiResourceType.EXPERIMENT, resource_reference_key_id=experiment_id)
    else:
      response = self._run_api.list_runs(page_token=page_token, page_size=page_size, sort_by=sort_by)
    return response

  def get_run(self, run_id):
    """Get run details.
    Args:
      id of the run.
    Returns:
      A response object including details of a run.
    Throws:
      Exception if run is not found.
    """
    return self._run_api.get_run(run_id=run_id)

  def wait_for_run_completion(self, run_id, timeout):
    """Wait for a run to complete.
    Args:
      run_id: run id, returned from run_pipeline.
      timeout: timeout in seconds.
    Returns:
      A run detail object: Most important fields are run and pipeline_runtime
    """
    status = 'Running:'
    start_time = datetime.now()
    while status is None or status.lower() not in ['succeeded', 'failed', 'skipped', 'error']:
      get_run_response = self._run_api.get_run(run_id=run_id)
      status = get_run_response.run.status
      elapsed_time = (datetime.now() - start_time).seconds
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
      workflow: json workflow
    """
    get_run_response = self._run_api.get_run(run_id=run_id)
    workflow = get_run_response.pipeline_runtime.workflow_manifest
    workflow_json = json.loads(workflow)
    return workflow_json

  def upload_pipeline(self, pipeline_package_path, pipeline_name=None):
    """Uploads the pipeline to the Kubeflow Pipelines cluster.
    Args:
      pipeline_package_path: Local path to the pipeline package.
      pipeline_name: Optional. Name of the pipeline to be shown in the UI.
    Returns:
      Server response object containing pipleine id and other information.
    """

    response = self._upload_api.upload_pipeline(pipeline_package_path, name=pipeline_name)
    if self._is_ipython():
      import IPython
      html = 'Pipeline link <a href=%s/#/pipelines/details/%s>here</a>' % (self._get_url_prefix(), response.id)
      IPython.display.display(IPython.display.HTML(html))
    return response

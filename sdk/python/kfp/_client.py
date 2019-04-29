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
import zipfile
import yaml
from datetime import datetime

import kfp_server_api

from .compiler import compiler
from .compiler import _k8s_helper

from ._auth import get_auth_token

class Client(object):
  """ API Client for KubeFlow Pipeline.
  """

  # in-cluster DNS name of the pipeline service
  IN_CLUSTER_DNS_NAME = 'ml-pipeline.kubeflow.svc.cluster.local:8888'

  def __init__(self, host=None, client_id=None):
    """Create a new instance of kfp client.

    Args:
      host: the host name to use to talk to Kubeflow Pipelines. If not set, the in-cluster
          service DNS name will be used, which only works if the current environment is a pod
          in the same cluster (such as a Jupyter instance spawned by Kubeflow's
          JupyterHub). If you have a different connection to cluster, such as a kubectl
          proxy connection, then set it to something like "127.0.0.1:8080/pipeline".
      client_id: The client ID used by Identity-Aware Proxy.
    """

    self._host = host

    token = None
    if host and client_id:
      token = get_auth_token(client_id)
  
    config = kfp_server_api.configuration.Configuration()
    config.host = host if host else Client.IN_CLUSTER_DNS_NAME
    self._configure_auth(config, token)
    api_client = kfp_server_api.api_client.ApiClient(config)
    self._run_api = kfp_server_api.api.run_service_api.RunServiceApi(api_client)
    self._experiment_api = kfp_server_api.api.experiment_service_api.ExperimentServiceApi(api_client)

  def _configure_auth(self, config, token):
    if token:
      config.api_key['authorization'] = token
      config.api_key_prefix['authorization'] = 'Bearer'

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
    if self._host:
      # User's own connection.
      if self._host.startswith('http://'):
        return self._host
      else:
        return 'http://' + self._host

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
    if package_file.endswith('.tar.gz') or package_file.endswith('.tgz'):
      with tarfile.open(package_file, "r:gz") as tar:
        all_yaml_files = [m for m in tar if m.isfile() and
            (os.path.splitext(m.name)[-1] == '.yaml' or os.path.splitext(m.name)[-1] == '.yml')]
        if len(all_yaml_files) == 0:
          raise ValueError('Invalid package. Missing pipeline yaml file in the package.')
        
        if len(all_yaml_files) > 1:
          raise ValueError('Invalid package. Multiple yaml files in the package.')
        
        with tar.extractfile(all_yaml_files[0]) as f:
          return yaml.safe_load(f)
    elif package_file.endswith('.zip'):
      with zipfile.ZipFile(package_file, 'r') as zip:
        all_yaml_files = [m for m in zip.namelist() if
                          (os.path.splitext(m)[-1] == '.yaml' or os.path.splitext(m)[-1] == '.yml')]
        if len(all_yaml_files) == 0:
          raise ValueError('Invalid package. Missing pipeline yaml file in the package.')

        if len(all_yaml_files) > 1:
          raise ValueError('Invalid package. Multiple yaml files in the package.')

        with zip.open(all_yaml_files[0]) as f:
          return yaml.safe_load(f)
    elif package_file.endswith('.yaml') or package_file.endswith('.yml'):
      with open(package_file, 'r') as f:
        return yaml.safe_load(f)
    else:
      raise ValueError('The package_file '+ package_file + ' should ends with one of the following formats: [.tar.gz, .tgz, .zip, .yaml, .yml]')

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

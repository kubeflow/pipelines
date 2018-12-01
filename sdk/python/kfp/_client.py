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
import yaml
from datetime import datetime


class Client(object):
  """ API Client for KubeFlow Pipeline.
  """

  def __init__(self, namespace='kubeflow'):
    """Create a new instance of kfp client.

    Args:
      namespace: the namespace where pipelines are deployed. Default is kubeflow
        TODO: check if it works outside of the cluster.
    """

    try:
      import kfp_experiment
    except ImportError:
      raise Exception('This module requires installation of kfp_experiment')

    try:
      import kfp_run
    except ImportError:
      raise Exception('This module requires installation of kfp_run')

    host='ml-pipeline.' + namespace + '.svc.cluster.local:8888'
    config = kfp_run.configuration.Configuration()
    config.host = host
    api_client = kfp_run.api_client.ApiClient(config)
    self._run_api = kfp_run.api.run_service_api.RunServiceApi(api_client)

    config = kfp_experiment.configuration.Configuration()
    config.host = host
    api_client = kfp_experiment.api_client.ApiClient(config)
    self._experiment_api = \
        kfp_experiment.api.experiment_service_api.ExperimentServiceApi(api_client)

  def _is_ipython(self):
    """Returns whether we are running in notebook."""
    try:
      import IPython
    except ImportError:
      return False

    return True

  def create_experiment(self, name):
    """Create a new experiment.
    Args:
      name: the name of the experiment.
    Returns:
      An Experiment object. Most important field is id.
    """
    import kfp_experiment

    exp = kfp_experiment.models.ApiExperiment(name=name)
    response = self._experiment_api.create_experiment(body=exp)
    
    if self._is_ipython():
      import IPython
      html = \
          ('Experiment link <a href="/pipeline/#/experiments/details/%s" target="_blank" >here</a>'
          % response.id)
      IPython.display.display(IPython.display.HTML(html))
    return response

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

  def get_experiment(self, experiment_id):
    """Get details of an experiment
    Args:
      id of the experiment.
    Returns:
      A response object including details of a experiment.
    Throws:
      Exception if experiment is not found.        
    """
    return self._experiment_api.get_experiment(id=experiment_id)

  def _extract_pipeline_yaml(self, tar_file):
    with tarfile.open(tar_file, "r:gz") as tar:
      all_yaml_files = [m for m in tar if m.isfile() and 
          (os.path.splitext(m.name)[-1] == '.yaml' or os.path.splitext(m.name)[-1] == '.yml')]
      if len(all_yaml_files) == 0:
        raise ValueError('Invalid package. Missing pipeline yaml file in the package.')
        
      if len(all_yaml_files) > 1:
        raise ValueError('Invalid package. Multiple yaml files in the package.')
        
      with tar.extractfile(all_yaml_files[0]) as f:
        return yaml.load(f)

  def run_pipeline(self, experiment_id, job_name, pipeline_package_path, params={}):
    """Run a specified pipeline.

    Args:
      experiment_id: The string id of an experiment.
      job_name: name of the job.
      pipeline_package_path: local path of the pipeline package(tar.gz file).
      params: a dictionary with key (string) as param name and value (string) as as param value.

    Returns:
      A run object. Most important field is id.
    """
    import kfp_run

    pipeline_obj = self._extract_pipeline_yaml(pipeline_package_path)
    pipeline_json_string = json.dumps(pipeline_obj)
    api_params = [kfp_run.ApiParameter(name=k, value=str(v)) for k,v in params.items()]
    key = kfp_run.models.ApiResourceKey(id=experiment_id,
                                        type=kfp_run.models.ApiResourceType.EXPERIMENT)
    reference = kfp_run.models.ApiResourceReference(key, kfp_run.models.ApiRelationship.OWNER)
    spec = kfp_run.models.ApiPipelineSpec(
        workflow_manifest=pipeline_json_string, parameters=api_params)
    run_body = kfp_run.models.ApiRun(
        pipeline_spec=spec, resource_references=[reference], name=job_name)

    response = self._run_api.create_run(body=run_body)
    
    if self._is_ipython():
      import IPython
      html = ('Run link <a href="/pipeline/#/runs/details/%s" target="_blank" >here</a>'
              % response.run.id)
      IPython.display.display(IPython.display.HTML(html))
    return response.run

  def list_runs(self, page_token='', page_size=10, sort_by=''):
    """List runs.
    Args:
      page_token: token for starting of the page.
      page_size: size of the page.
      sort_by: one of 'field_name', 'field_name des'. For example, 'name des'.
    Returns:
      A response object including a list of experiments and next page token.
    """
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

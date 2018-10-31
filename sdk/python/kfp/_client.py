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


import six
import time
import logging
import json

class Client(object):
  """ API Client for KubeFlow Pipeline.
  """

  def __init__(self, host='ml-pipeline.kubeflow.svc.cluster.local:8888'):
    """Create a new instance of kfp client.

    Args:
      host: the API host. If running inside the cluster as a Pod, default value should work.
    """

    try:
      import kfp_upload
    except ImportError:
      raise Exception('This module requires installation of kfp_upload')

    try:
      import kfp_job
    except ImportError:
      raise Exception('This module requires installation of kfp_job')

    try:
      import kfp_run
    except ImportError:
      raise Exception('This module requires installation of kfp_run')

    config = kfp_upload.configuration.Configuration()
    config.host = host
    api_client = kfp_upload.api_client.ApiClient(config)
    self._upload_api = kfp_upload.api.pipeline_upload_service_api.PipelineUploadServiceApi(api_client)

    config = kfp_job.configuration.Configuration()
    config.host = host
    api_client = kfp_job.api_client.ApiClient(config)
    self._job_api = kfp_job.api.job_service_api.JobServiceApi(api_client)

    config = kfp_run.configuration.Configuration()
    config.host = host
    api_client = kfp_run.api_client.ApiClient(config)
    self._run_api = kfp_run.api.run_service_api.RunServiceApi(api_client)

  def _is_ipython(self):
    """Returns whether we are running in notebook."""
    try:
      import IPython
    except ImportError:
      return False

    return True

  def upload_pipeline(self, pipeline_path):
    """Upload a pipeline package to local catalog.

    Args:
      pipeline_path: a local path to the pipeline package.
    Returns:
      A pipeline object. Most important field is id.
    """

    response = self._upload_api.upload_pipeline(pipeline_path)
    if self._is_ipython():
      import IPython
      html = 'Pipeline link <a href=/pipeline/pipeline?id=%s>here</a>' % response.id
      IPython.display.display(IPython.display.HTML(html))
    return response

  def run_pipeline(self, job_name, pipeline_id, params={}):
    """Run a specified pipeline.

    Args:
      job_name: name of the job.
      pipeline_id: a string returned by upload_pipeline.
      params: a dictionary with key (string) as param name and value (string) as as param value.

    Returns:
      A job object. Most important field is id.
    """

    import kfp_job

    api_params = [kfp_job.ApiParameter(name=k, value=str(v)) for k,v in six.iteritems(params)]
    body = kfp_job.ApiJob(pipeline_id=pipeline_id, name=job_name, parameters=api_params,
                            max_concurrency=10, enabled = True)
    response = self._job_api.create_job(body)
    if self._is_ipython():
      import IPython
      html = 'Job link <a href=/pipeline/job?id=%s>here</a>' % response.id
      IPython.display.display(IPython.display.HTML(html))
    return response

  def _wait_for_job_completion(self, job_id, timeout):
    """Wait for a job to complete.

    Args:
      job_id: job id, returned from run_pipeline.
      timeout: timeout in seconds.

    Returns:
      success: boolean value of whether the job fails or succeeds
      elapsed_time: elapsed time in seconds
    """
    #TODO: need to check the status of the runs because job status is always 'succeeded:'
    #finished_at timestamp is only available in the runtime argo yaml(workflow json): workflow_json['status']['startedAt'] and workflow_json['status']['finishedAt']
    status = 'Running:'
    elapsed_time = 0
    while status.lower() not in ['succeeded:', 'failed:']:
      get_job_response = self._job_api.get_job(id=job_id)
      status = get_job_response.status
      created_at = get_job_response.created_at
      updated_at = get_job_response.updated_at
      elapsed_time = (updated_at - created_at).seconds
      logging.getLogger(__name__).info('Waiting for the job to complete...')
      if elapsed_time > timeout:
        logging.getLogger(__name__).error('Job timeout')
        return False, elapsed_time
      time.sleep(5)
    return (status.lower()=='succeeded'), elapsed_time

  def _get_workflow_json(self, job_id, run_index):
    """Get the workflow json.

    Args:
      job_id: job id, returned from run_pipeline.
      run_index: the index of the run

    Returns:
      workflow: json workflow
    """
    #TODO: run_index should be changed to a more intuitive value.
    list_job_response = self._job_api.list_job_runs(job_id=job_id)
    run_id = list_job_response.runs[run_index].id
    get_run_response = self._run_service_api.get_run(job_id, run_id)
    workflow = get_run_response.workflow
    workflow_json = json.loads(workflow)
    return workflow_json


  def _delete_job(self, job_id):
    """Delete the job.

    Args:
      job_id: job id, returned from run_pipeline."""
    self._job_api.delete_job(id=job_id)
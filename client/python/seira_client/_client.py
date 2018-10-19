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


class Client(object):
  """ API Client for Seira.
  """

  def __init__(self, host='ml-pipeline.kubeflow.svc.cluster.local:8888'):
    """Create a new instance of Seira client.

    Args:
      host: the API host. If running inside the cluster as a Pod, default value should work.
    """

    try:
      import seira_upload
    except ImportError:
      raise Exception('This module requires installation of seira_upload')

    try:
      import seira_job
    except ImportError:
      raise Exception('This module requires installation of seira_job')

    config = seira_upload.configuration.Configuration()
    config.host = host
    api_client = seira_upload.api_client.ApiClient(config)
    self._upload_api = seira_upload.api.pipeline_upload_service_api.PipelineUploadServiceApi(api_client)

    config = seira_job.configuration.Configuration()
    config.host = host
    api_client = seira_job.api_client.ApiClient(config)
    self._job_api = seira_job.api.job_service_api.JobServiceApi(api_client)

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

  def run_pipeline(self, job_name, pipeline_id, params):
    """Run a specified pipeline.

    Args:
      job_name: name of the job.
      pipeline_id: a string returned by upload_pipeline.
      params: a dictionary with key (string) as param name and value (string) as as param value.

    Returns:
      A job object. Most important field is id.
    """

    import seira_job

    api_params = [seira_job.ApiParameter(name=k, value=v) for k,v in six.iteritems(params)]
    body = seira_job.ApiJob(pipeline_id=pipeline_id, name=job_name, parameters=api_params,
                            max_concurrency=10, enabled = True)
    response = self._job_api.create_job(body)
    if self._is_ipython():
      import IPython
      html = 'Job link <a href=/pipeline/job?id=%s>here</a>' % response.id
      IPython.display.display(IPython.display.HTML(html))
    return response

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

    config = kfp_upload.configuration.Configuration()
    config.host = host
    api_client = kfp_upload.api_client.ApiClient(config)
    self._upload_api = kfp_upload.api.pipeline_upload_service_api.PipelineUploadServiceApi(api_client)

    config = kfp_job.configuration.Configuration()
    config.host = host
    api_client = kfp_job.api_client.ApiClient(config)
    self._job_api = kfp_job.api.job_service_api.JobServiceApi(api_client)

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

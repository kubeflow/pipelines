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
import json

from kubernetes import client as k8s_client
from kubernetes import config as k8s_config

import swagger_job
import swagger_run
import swagger_pipeline_upload
from swagger_pipeline.configuration import Configuration
from swagger_pipeline_upload.api_client import ApiClient as PipelineUploadApiClient
from swagger_pipeline_upload.api.pipeline_upload_service_api import PipelineUploadServiceApi
from swagger_job.api_client import ApiClient as JobApiClient
from swagger_job.api.job_service_api import JobServiceApi
from swagger_run.api_client import ApiClient as RunApiClient
from swagger_run.api.run_service_api import RunServiceApi

# Initialize the configuration that Service handler uses to connect to the API Service
def init_config():
  k8s_config.load_incluster_config()
  k8s_api_client = k8s_client.ApiClient()

  mlPipelineAPIServerBase = '/api/v1/namespaces/default/services/ml-pipeline:8888/proxy'
  config = Configuration()
  config.host = k8s_api_client.configuration.host + mlPipelineAPIServerBase
  config.api_key = k8s_api_client.configuration.api_key
  config.ssl_ca_cert = k8s_api_client.configuration.ssl_ca_cert

  return config

def init_pipeline_upload_api(config):
  pipeline_upload_api_client = PipelineUploadApiClient(config)
  pipeline_upload_service_api = PipelineUploadServiceApi(pipeline_upload_api_client)
  return pipeline_upload_service_api

def init_job_api(config):
  job_api_client = JobApiClient(config)
  job_service_api = JobServiceApi(job_api_client)
  return job_service_api

def init_run_api(config):
  run_api_client = RunApiClient(config)
  run_service_api = RunServiceApi(run_api_client)
  return run_service_api

###### utility functions of the API Server ######
def create_job(job_service_api, pipeline_id, params, job_name):
  print('Creating the job')
  body = swagger_job.ApiJob(pipeline_id=pipeline_id, name=job_name, parameters=params,
                            max_concurrency=10, enabled = True)
  job_id = ''
  try:
    create_job_response = job_service_api.create_job(body)
    print(create_job_response)
    job_id = create_job_response.id
  except swagger_job.rest.ApiException as e:
    print("Exception when creating the job: %s\n" % e)
    return job_id, False
  return job_id, True

def delete_job(job_service_api, job_id):
  try:
    job_service_api.delete_job(id=job_id)
  except swagger_job.rest.ApiException as e:
    print("Exception when deleting the job: %s\n" % e)
    return False
  return True

def upload_pipeline_yaml(pipeline_upload_service_api, yaml_path):
  pipeline_id = ''
  try:
    upload_pipeline_response = pipeline_upload_service_api.upload_pipeline(yaml_path)
    pipeline_id = upload_pipeline_response.id
  except swagger_pipeline_upload.rest.ApiException as e:
    print("Exception when uploading the pipeline yaml: %s\n", e)
    return pipeline_id, False
  return pipeline_id, True

# wait_for_job_completion
#   timeout: in seconds.
def wait_for_job_completion(job_service_api, job_id, timeout):
  status = 'Running:'
  elapsed_time = 0
  while status.lower() not in ['succeeded:', 'failed:']:
    try:
      get_job_response = job_service_api.get_job(id=job_id)
      status = get_job_response.status
      created_at = get_job_response.created_at
      completed_at = get_job_response.updated_at
      elapsed_time = (completed_at - created_at).seconds
      print('Waiting for the job to complete')
      if elapsed_time > timeout:
        return 'Not Finished', elapsed_time
      time.sleep(5)
    except swagger_job.rest.ApiException as e:
      print("Exception when creating the job: %s\n" % e)
      return 'Exception', elapsed_time
  return status, elapsed_time

def get_workflow_json(job_service_api, run_service_api, job_id):
  workflow_json = ""
  try:
    list_job_response = job_service_api.list_job_runs(job_id=job_id)
    run_id = list_job_response.runs[0].id
    get_run_response = run_service_api.get_run(job_id, run_id)
    workflow = get_run_response.workflow
    workflow_json = json.loads(workflow)
  except swagger_job.rest.ApiException as e:
    print("Exception when obtaining the run id: %s\n", e)
    return workflow_json, False
  except swagger_run.rest.ApiException as e:
    print("Exception when obtaining the workflow: %s\n", e)
    return workflow_json, False
  return workflow_json, True

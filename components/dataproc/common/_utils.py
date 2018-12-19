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


import datetime
import googleapiclient.discovery as discovery
import os
import subprocess
import time


def get_client():
    """Builds a client to the dataproc API."""
    dataproc = discovery.build('dataproc', 'v1')
    return dataproc

  
def create_cluster(api, project, region, cluster_name, init_file_url):
  """Create a DataProc clsuter."""
  cluster_data = {
    'projectId': project,
    'clusterName': cluster_name,
    'config': {
        'gceClusterConfig': {
        },
        'initializationActions': {
             'executableFile': init_file_url
        }
    }
  }
  result = api.projects().regions().clusters().create(
      projectId=project,
      region=region,
      body=cluster_data).execute()
  return result


def delete_cluster(api, project, region, cluster):
  result = api.projects().regions().clusters().delete(
      projectId=project,
      region=region,
      clusterName=cluster).execute()
  return result


def wait_for_operation(api, job_name):
  """Waiting for a long running operation by polling it."""
  while True:
    result = api.projects().regions().operations().get(name=job_name).execute()
    if result.get('done'):
      if result['metadata']['status']['state'] == 'DONE':
        return result
      else:
        raise Exception(result)
    time.sleep(5)


def wait_for_job(api, project, region, job_id):
  """Waiting for a job by polling it."""
  while True:
    result = api.projects().regions().jobs().get(
        projectId=project,
        region=region,
        jobId=job_id).execute()

    if result['status']['state'] == 'ERROR':
      raise Exception(result['status']['details'])
    elif result['status']['state'] == 'DONE':
      return result
    time.sleep(5)


def submit_pyspark_job(api, project, region, cluster_name, filepath, args):
  """Submits the Pyspark job to the cluster"""
  job_details = {
      'projectId': project,
      'job': {
          'placement': {
              'clusterName': cluster_name
          },
          'pysparkJob': {
              'mainPythonFileUri': filepath,
              'args': args
          }
      }
  }
  result = api.projects().regions().jobs().submit(
      projectId=project,
      region=region,
      body=job_details).execute()
  job_id = result['reference']['jobId']
  return job_id  


def submit_spark_job(api, project, region, cluster_name, jar_files, main_class, args):
  """Submits the spark job to the cluster"""

  job_details = {
      'projectId': project,
      'job': {
          'placement': {
              'clusterName': cluster_name
          },
          'sparkJob': {
              'jarFileUris': jar_files,
              'mainClass': main_class,
              'args': args,
          }
      }
  }
  result = api.projects().regions().jobs().submit(
      projectId=project,
      region=region,
      body=job_details).execute()
  job_id = result['reference']['jobId']
  return job_id 


def copy_resources_to_gcs(file_paths, gcs_path):
  """Copy a local resources to a GCS location."""
  tmpdir = datetime.datetime.now().strftime('xgb_%y%m%d_%H%M%S')

  dest_files = []
  for file_name in file_paths:
    dest_file = os.path.join(gcs_path, tmpdir, os.path.basename(file_name))
    subprocess.call(['gcloud', 'auth', 'activate-service-account', '--key-file', os.environ['GOOGLE_APPLICATION_CREDENTIALS']])
    subprocess.call(['gsutil', 'cp', file_name, dest_file])
    dest_files.append(dest_file)

  return dest_files


def remove_resources_from_gcs(file_paths):
  """Remove staged resources from a GCS location."""
  subprocess.call(['gsutil', '-m', 'rm'] + file_paths)


def delete_directory_from_gcs(dir_path):
  """Delete a GCS dir recursively. Ignore errors."""
  try:
    subprocess.call(['gsutil', '-m', 'rm', '-r', dir_path])
  except:
    pass

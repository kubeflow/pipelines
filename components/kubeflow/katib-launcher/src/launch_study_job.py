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
import argparse
import datetime
import json
import os
import logging
import requests
import subprocess
import six
import time
import yaml
import grpc

import api_pb2
import api_pb2_grpc

from kubernetes import client as k8s_client
from kubernetes.client import rest
from kubernetes import config

STUDY_JOB_GROUP = "kubeflow.org"
STUDY_JOB_PLURAL = "studyjobs"
STUDY_JOB_KIND = "StudyJob"

TIMEOUT = 120

def wait_for_condition(client,
                       namespace,
                       name,
                       expected_condition,
                       version="v1alpha1",
                       timeout=datetime.timedelta(minutes=10),
                       polling_interval=datetime.timedelta(seconds=30),
                       status_callback=None):
  """Waits until any of the specified conditions occur.

  Args:
    client: K8s api client.
    namespace: namespace for the job.
    name: Name of the job.
    expected_condition: A list of conditions. Function waits until any of the
      supplied conditions is reached.
    timeout: How long to wait for the job.
    polling_interval: How often to poll for the status of the job.
    status_callback: (Optional): Callable. If supplied this callable is
      invoked after we poll the job. Callable takes a single argument which
      is the job.
  """
  crd_api = k8s_client.CustomObjectsApi(client)
  end_time = datetime.datetime.now() + timeout
  while True:
    # By setting async_req=True ApiClient returns multiprocessing.pool.AsyncResult
    # If we don't set async_req=True then it could potentially block forever.
    thread = crd_api.get_namespaced_custom_object(
      STUDY_JOB_GROUP, version, namespace, STUDY_JOB_PLURAL, name, async_req=True)

    # Try to get the result but timeout.
    results = None
    try:
      results = thread.get(TIMEOUT)
    except multiprocessing.TimeoutError:
      logging.error("Timeout trying to get TFJob.")
    except Exception as e:
      logging.error("There was a problem waiting for Job %s.%s; Exception; %s",
                    name, name, e)
      raise

    if results:
      if status_callback:
        status_callback(results)

      # If we poll the CRD quick enough status won't have been set yet.
      condition = results.get("status", {}).get("condition")
      #  might have a value of None in status.
      if condition in expected_condition:
          return results

    if datetime.datetime.now() + polling_interval > end_time:
      raise util.JobTimeoutError(
        "Timeout waiting for job {0} in namespace {1} to enter one of the "
        "conditions {2}.".format(name, namespace, expected_condition), results)

    time.sleep(polling_interval.seconds)

def create_study_job(client, spec, version="v1alpha1"):
  """Create a studyJob.

  Args:
    client: A K8s api client.
    spec: The spec for the job.
  """
  crd_api = k8s_client.CustomObjectsApi(client)
  try:
    # Create a Resource
    namespace = spec["metadata"].get("namespace", "default")
    thread = crd_api.create_namespaced_custom_object(
      STUDY_JOB_GROUP, version, namespace, STUDY_JOB_PLURAL, spec, async_req=True)
    api_response = thread.get(TIMEOUT)
    logging.info("Created job %s", api_response["metadata"]["name"])
    return api_response
  except rest.ApiException as e:
    message = ""
    if e.message:
      message = e.message
    if e.body:
      try:
        body = json.loads(e.body)
      except ValueError:
        # There was a problem parsing the body of the response as json.
        logging.error(
          ("Exception when calling DefaultApi->"
           "apis_fqdn_v1_namespaces_namespace_resource_post. body: %s"), e.body)
        raise
      message = body.get("message")

    logging.error(("Exception when calling DefaultApi->"
                   "apis_fqdn_v1_namespaces_namespace_resource_post: %s"),
                  message)
    raise e

def delete_study_job(client, name, namespace, version="v1alpha1"):
  crd_api = k8s_client.CustomObjectsApi(client)
  try:
    body = {
      # Set garbage collection so that job won't be deleted until all
      # owned references are deleted.
      "propagationPolicy": "Foreground",
    }
    logging.info("Deleting job %s.%s", namespace, name)
    thread = crd_api.delete_namespaced_custom_object(
      STUDY_JOB_GROUP,
      version,
      namespace,
      STUDY_JOB_PLURAL,
      name,
      body,
      async_req=True)
    api_response = thread.get(TIMEOUT)
    logging.info("Deleting job %s.%s returned: %s", namespace, name,
                 api_response)
    return api_response
  except rest.ApiException as e:
    message = ""
    if e.message:
      message = e.message
    if e.body:
      try:
        body = json.loads(e.body)
      except ValueError:
        # There was a problem parsing the body of the response as json.
        logging.error(
          ("Exception when calling DefaultApi->"
           "apis_fqdn_v1_namespaces_namespace_resource_post. body: %s"), e.body)
        raise
      message = body.get("message")

    logging.error(("Exception when calling DefaultApi->"
                   "apis_fqdn_v1_namespaces_namespace_resource_post: %s"),
                  message)
    raise e


def yamlOrJsonStr(str):
    if str == "" or str == None:
        return None
    try:
        return json.loads(str)
    except:
        return yaml.load(str)

def strToList(str):
    return str.split(",")

def _update_or_pop(spec, name, value):
    if value:
        spec[name] = value
    else:
        spec.pop(name)

def _generate_studyjob_yaml(src_filename, name, namespace, optimizationtype, objectivevaluename, optimizationgoal, requestcount,
                            metricsnames, parameterconfigs, nasConfig, workertemplatepath, mcollectortemplatepath, suggestionspec):
  """_generate_studyjob_yaml generates studyjob yaml file based on hp.template.yaml"""
  with open(src_filename, 'r') as f:
    content = yaml.load(f)

  content['metadata']['name'] = name
  content['metadata']['namespace'] = namespace
  content['spec']['studyName'] = name
  content['spec']['optimizationtype'] = optimizationtype
  content['spec']['objectivevaluename'] = objectivevaluename
  content['spec']['optimizationgoal'] = optimizationgoal
  content['spec']['requestcount'] = requestcount

  _update_or_pop(content['spec'], 'parameterconfigs', parameterconfigs)
  _update_or_pop(content['spec'], 'nasConfig', nasConfig)
  _update_or_pop(content['spec'], 'metricsnames', metricsnames)
  _update_or_pop(content['spec'], 'suggestionSpec', suggestionspec)

  if workertemplatepath:
    content['spec']['workerSpec']['goTemplate']['templatePath'] = workertemplatepath
  else:
    content['spec'].pop('workerSpec')

  if mcollectortemplatepath:
    content['spec']['metricsCollectorSpec']['goTemplate']['templatePath'] = mcollectortemplatepath
  else :
    content['spec'].pop('metricsCollectorSpec')

  return content

def get_best_trial(trial_id):
    vizier_core = "vizier-core.kubeflow:6789"
    with grpc.insecure_channel(vizier_core) as channel:
        stub = api_pb2_grpc.ManagerStub(channel)
        response = stub.GetTrial(api_pb2.GetTrialRequest(trial_id=trial_id))
        return response.trial

def main(argv=None):
  parser = argparse.ArgumentParser(description='Kubeflow StudyJob launcher')
  parser.add_argument('--name', type=str,
                      help='StudyJob name.')
  parser.add_argument('--namespace', type=str,
                      default='kubeflow',
                      help='StudyJob namespace.')
  parser.add_argument('--optimizationtype', type=str,
                      default='minimize',
                      help='Direction of optimization. minimize or maximize.')
  parser.add_argument('--objectivevaluename', type=str,
                      help='Objective value name which trainer optimizes.')
  parser.add_argument('--optimizationgoal', type=float,
                      help='Stop studying once objectivevaluename value ' +
                           'exceeds optimizationgoal')
  parser.add_argument('--requestcount', type=int,
                      default=1,
                      help='The times asking request to suggestion service.')
  parser.add_argument('--metricsnames', type=strToList,
                      help='StudyJob metrics name list.')
  parser.add_argument('--parameterconfigs', type=yamlOrJsonStr,
                      default={},
                      help='StudyJob parameterconfigs.')
  parser.add_argument('--nasConfig',type=yamlOrJsonStr,
                      default={},
                      help='StudyJob nasConfig.')
  parser.add_argument('--workertemplatepath', type=str,
                      default="",
                      help='StudyJob worker spec.')
  parser.add_argument('--mcollectortemplatepath', type=str,
                      default="",
                      help='StudyJob worker spec.')
  parser.add_argument('--suggestionspec', type=yamlOrJsonStr,
                      default={},
                      help='StudyJob suggestion spec.')
  parser.add_argument('--studyjobtimeoutminutes', type=int,
                      default=10,
                      help='Time in minutes to wait for the StudyJob to complete')
  
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)


  logging.info('Generating studyjob template.')
  template_file = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'hp.template.yaml')
  content_yaml = _generate_studyjob_yaml(template_file, args.name, args.namespace, args.optimizationtype, args.objectivevaluename,
                                         args.optimizationgoal, args.requestcount, args.metricsnames, args.parameterconfigs,
                                         args.nasConfig, args.workertemplatepath, args.mcollectortemplatepath, args.suggestionspec)

  config.load_incluster_config()
  api_client = k8s_client.ApiClient()
  create_response = create_study_job(api_client, content_yaml)
  job_name = create_response['metadata']['name']
  job_namespace = create_response['metadata']['namespace']

  expected_condition = ["Completed", "Failed"]
  wait_response = wait_for_condition(
      api_client, job_namespace, job_name, expected_condition, 
      timeout=datetime.timedelta(minutes=args.studyjobtimeoutminutes))
  succ = False
  if wait_response.get("status", {}).get("condition") == "Completed":
    succ = True
    trial = get_best_trial(wait_response["status"]["bestTrialId"])
    with open('/output.txt', 'w') as f:
      ps_dict = {}
      for ps in trial.parameter_set:
          ps_dict[ps.name] = ps.value
      f.write(json.dumps(ps_dict))
  if succ:
    logging.info('Study success.')

  delete_study_job(api_client, job_name, job_namespace)

if __name__== "__main__":
  main()

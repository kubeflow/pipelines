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
from distutils.util import strtobool
import json
import os
import logging
import requests
import subprocess
import yaml
import grpc

import api_pb2
import api_pb2_grpc

from kubernetes import client as k8s_client
from kubernetes import config
import study_job_client

def yamlOrJsonStr(str):
    if str == "" or str == None:
        return None
    try:
        return json.loads(str)
    except:
        return yaml.safe_load(str)

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
    content = yaml.safe_load(f)

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
  parser.add_argument('--outputfile', type=str,
                      default='/output.txt',
                      help='The file which stores the best trial of the studyJob.')
  parser.add_argument('--deleteAfterDone', type=strtobool,
                      default=True,
                      help='When studyjob done, delete the studyjob automatically if it is True.')
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
  create_response = study_job_client.create_study_job(api_client, content_yaml)
  job_name = create_response['metadata']['name']
  job_namespace = create_response['metadata']['namespace']

  expected_condition = ["Completed", "Failed"]
  wait_response = study_job_client.wait_for_condition(
      api_client, job_namespace, job_name, expected_condition, 
      timeout=datetime.timedelta(minutes=args.studyjobtimeoutminutes))
  succ = False
  if wait_response.get("status", {}).get("condition") == "Completed":
    succ = True
    trial = get_best_trial(wait_response["status"]["bestTrialId"])
    if not os.path.exists(os.path.dirname(args.outputfile)):
      os.makedirs(os.path.dirname(args.outputfile))
    with open(args.outputfile, 'w') as f:
      ps_dict = {}
      for ps in trial.parameter_set:
          ps_dict[ps.name] = ps.value
      f.write(json.dumps(ps_dict))
  if succ:
    logging.info('Study success.')
  if args.deleteAfterDone:
    study_job_client.delete_study_job(api_client, job_name, job_namespace)

if __name__== "__main__":
  main()

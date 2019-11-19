# Copyright 2019 kubeflow.org.
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
import yaml
import launch_crd

from kubernetes import client as k8s_client
from kubernetes import config

def yamlOrJsonStr(str):
    if str == "" or str == None:
        return None
    try:
        return json.loads(str)
    except:
        return yaml.safe_load(str)

ExperimentGroup = "kubeflow.org"
ExperimentPlural = "experiments"

class Experiment(launch_crd.K8sCR):
  def __init__(self, version="v1alpha3", client=None):
    super(Experiment, self).__init__(ExperimentGroup, ExperimentPlural, version, client)

  def is_expected_conditions(self, inst, expected_conditions):
    conditions = inst.get('status', {}).get("conditions")
    if not conditions:
      return False, ""
    if conditions[-1]["type"] in expected_conditions:
      return True, conditions[-1]["type"]
    else:
      return False, conditions[-1]["type"]

def main(argv=None):
  parser = argparse.ArgumentParser(description='Kubeflow Experiment launcher')
  parser.add_argument('--name', type=str,
                      help='Experiment name.')
  parser.add_argument('--namespace', type=str,
                      default='kubeflow',
                      help='Experiment namespace.')
  parser.add_argument('--version', type=str,
                      default='v1alpha3',
                      help='Experiment version.')
  parser.add_argument('--maxTrialCount', type=int,
                      help='How many trial will be created for the experiment at most.')
  parser.add_argument('--maxFailedTrialCount', type=int,
                      help='Stop the experiment when $maxFailedTrialCount trials failed.')
  parser.add_argument('--parallelTrialCount', type=int,
                      default=3,
                      help='How many trials can be running at most.')
  parser.add_argument('--objectiveConfig', type=yamlOrJsonStr,
                      default={},
                      help='Experiment objective.')
  parser.add_argument('--algorithmConfig', type=yamlOrJsonStr,
                      default={},
                      help='Experiment algorithm.')
  parser.add_argument('--trialTemplate', type=yamlOrJsonStr,
                      default={},
                      help='Experiment trialTemplate.')
  parser.add_argument('--parameters', type=yamlOrJsonStr,
                      default=[],
                      help='Experiment parameters.')
  parser.add_argument('--metricsCollector', type=yamlOrJsonStr,
                      default={},
                      help='Experiment metricsCollectorSpec.')
  parser.add_argument('--outputFile', type=str,
                      default='/output.txt',
                      help='The file which stores the best trial of the experiment.')
  parser.add_argument('--deleteAfterDone', type=strtobool,
                      default=True,
                      help='When experiment done, delete the experiment automatically if it is True.')
  parser.add_argument('--experimentTimeoutMinutes', type=int,
                      default=60*24,
                      help='Time in minutes to wait for the Experiment to reach end')

  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)

  logging.info('Generating experiment template.')

  config.load_incluster_config()
  api_client = k8s_client.ApiClient()
  experiment = Experiment(version=args.version, client=api_client)
  inst = {
    "apiVersion": "%s/%s" % (ExperimentGroup, args.version),
    "kind": "Experiment",
    "metadata": {
      "name": args.name,
      "namespace": args.namespace,
    },
    "spec": {
      "algorithm": args.algorithmConfig,
      "maxFailedTrialCount": args.maxFailedTrialCount,
      "maxTrialCount": args.maxTrialCount,
      "metricsCollectorSpec": args.metricsCollector,
      "objective": args.objectiveConfig,
      "parallelTrialCount": args.parallelTrialCount,
      "parameters": args.parameters,
      "trialTemplate": args.trialTemplate,
    },
  }
  create_response = experiment.create(inst)

  expected_conditions = ["Succeeded", "Failed"]
  current_inst = experiment.wait_for_condition(
      args.namespace, args.name, expected_conditions,
      timeout=datetime.timedelta(minutes=args.experimentTimeoutMinutes))
  expected, conditon = experiment.is_expected_conditions(current_inst, ["Succeeded"])
  if expected:
    paramAssignments = current_inst["status"]["currentOptimalTrial"]["parameterAssignments"]
    if not os.path.exists(os.path.dirname(args.outputFile)):
      os.makedirs(os.path.dirname(args.outputFile))
    with open(args.outputFile, 'w') as f:
      f.write(json.dumps(paramAssignments))
  if args.deleteAfterDone:
    experiment.delete(args.name, args.namespace)

if __name__== "__main__":
  main()

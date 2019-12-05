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
    return yaml.safe_load(str)

TFJobGroup = "kubeflow.org"
TFJobPlural = "tfjobs"

class TFJob(launch_crd.K8sCR):
    def __init__(self, version="v1", client=None):
        super(TFJob, self).__init__(TFJobGroup, TFJobPlural, version, client)

    def is_expected_conditions(self, inst, expected_conditions):
        conditions = inst.get('status', {}).get("conditions")
        if not conditions:
            return False, ""
        if conditions[-1]["type"] in expected_conditions and conditions[-1]["status"] == "True":
            return True, conditions[-1]["type"]
        else:
            return False, conditions[-1]["type"]

def main(argv=None):
    parser = argparse.ArgumentParser(description='Kubeflow TFJob launcher')
    parser.add_argument('--name', type=str,
                        help='TFJob name.')
    parser.add_argument('--namespace', type=str,
                        default='kubeflow',
                        help='TFJob namespace.')
    parser.add_argument('--version', type=str,
                        default='v1',
                        help='TFJob version.')
    parser.add_argument('--activeDeadlineSeconds', type=int,
                        default=-1,
                        help='Specifies the duration (in seconds) since startTime during which the job can remain active before it is terminated. Must be a positive integer. This setting applies only to pods where restartPolicy is OnFailure or Always.')
    parser.add_argument('--backoffLimit', type=int,
                        default=-1,
                        help='Number of retries before marking this job as failed.')
    parser.add_argument('--cleanPodPolicy', type=str,
                        default="Running",
                        help='Defines the policy for cleaning up pods after the TFJob completes.')
    parser.add_argument('--ttlSecondsAfterFinished', type=int,
                        default=-1,
                        help='Defines the TTL for cleaning up finished TFJobs.')
    parser.add_argument('--psSpec', type=yamlOrJsonStr,
                        default={},
                        help='TFJob ps replicaSpecs.')
    parser.add_argument('--workerSpec', type=yamlOrJsonStr,
                        default={},
                        help='TFJob worker replicaSpecs.')
    parser.add_argument('--chiefSpec', type=yamlOrJsonStr,
                        default={},
                        help='TFJob chief replicaSpecs.')
    parser.add_argument('--evaluatorSpec', type=yamlOrJsonStr,
                        default={},
                        help='TFJob evaluator replicaSpecs.')
    parser.add_argument('--deleteAfterDone', type=strtobool,
                        default=True,
                        help='When tfjob done, delete the tfjob automatically if it is True.')
    parser.add_argument('--tfjobTimeoutMinutes', type=int,
                        default=60*24,
                        help='Time in minutes to wait for the TFJob to reach end')

    args = parser.parse_args()

    logging.getLogger().setLevel(logging.INFO)

    logging.info('Generating tfjob template.')

    config.load_incluster_config()
    api_client = k8s_client.ApiClient()
    tfjob = TFJob(version=args.version, client=api_client)
    inst = {
        "apiVersion": "%s/%s" % (TFJobGroup, args.version),
        "kind": "TFJob",
        "metadata": {
            "name": args.name,
            "namespace": args.namespace,
        },
        "spec": {
            "cleanPodPolicy": args.cleanPodPolicy,
            "tfReplicaSpecs": {
            },
        },
    }
    if args.ttlSecondsAfterFinished >=0:
        inst["spec"]["ttlSecondsAfterFinished"] = args.ttlSecondsAfterFinished
    if args.backoffLimit >= 0:
        inst["spec"]["backoffLimit"] = args.backoffLimit
    if args.activeDeadlineSeconds >=0:
        inst["spec"]["activeDeadlineSecond"] = args.activeDeadlineSeconds
    if args.psSpec:
        inst["spec"]["tfReplicaSpecs"]["PS"] = args.psSpec
    if args.chiefSpec:
        inst["spec"]["tfReplicaSpecs"]["Chief"] = args.chiefSpec
    if args.workerSpec:
        inst["spec"]["tfReplicaSpecs"]["Worker"] = args.workerSpec
    if args.evaluatorSpec:
        inst["spec"]["tfReplicaSpecs"]["Evaluator"] = args.evaluatorSpec
   
    create_response = tfjob.create(inst)

    expected_conditions = ["Succeeded", "Failed"]
    tfjob.wait_for_condition(
        args.namespace, args.name, expected_conditions,
        timeout=datetime.timedelta(minutes=args.tfjobTimeoutMinutes))
    if args.deleteAfterDone:
        tfjob.delete(args.name, args.namespace)

if __name__== "__main__":
    main()

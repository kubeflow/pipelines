# Copyright 2020 The Kubeflow Authors.
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
import time

from kubernetes.client import V1ObjectMeta

from kubeflow.katib import KatibClient
from kubeflow.katib import ApiClient
from kubeflow.katib import V1beta1Experiment

logger = logging.getLogger()
logging.basicConfig(level=logging.INFO)

FINISH_CONDITIONS = ["Succeeded", "Failed"]


class JSONObject(object):
    """ This class is needed to deserialize input JSON.
    Katib API client expects JSON under .data attribute.
    """

    def __init__(self, json):
        self.data = json


def wait_experiment_finish(katib_client, experiment, timeout):
    polling_interval = datetime.timedelta(seconds=30)
    end_time = datetime.datetime.now() + datetime.timedelta(minutes=timeout)
    experiment_name = experiment.metadata.name
    experiment_namespace = experiment.metadata.namespace
    while True:
        current_status = None
        try:
            current_status = katib_client.get_experiment_status(name=experiment_name, namespace=experiment_namespace)
        except Exception as e:
            logger.info("Unable to get current status for the Experiment: {} in namespace: {}. Exception: {}".format(
                experiment_name, experiment_namespace, e))
        # If Experiment has reached complete condition, exit the loop.
        if current_status in FINISH_CONDITIONS:
            logger.info("Experiment: {} in namespace: {} has reached the end condition: {}".format(
                experiment_name, experiment_namespace, current_status))
            return
        # Print the current condition.
        logger.info("Current condition for Experiment: {} in namespace: {} is: {}".format(
            experiment_name, experiment_namespace, current_status))
        # If timeout has been reached, rise an exception.
        if datetime.datetime.now() > end_time:
            raise Exception("Timout waiting for Experiment: {} in namespace: {} "
                            "to reach one of these conditions: {}".format(
                                experiment_name, experiment_namespace, FINISH_CONDITIONS))
        # Sleep for poll interval.
        time.sleep(polling_interval.seconds)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Katib Experiment launcher')
    parser.add_argument('--experiment-name', type=str,
                        help='Experiment name')
    parser.add_argument('--experiment-namespace', type=str, default='anonymous',
                        help='Experiment namespace')
    parser.add_argument('--experiment-spec', type=str, default='',
                        help='Experiment specification')
    parser.add_argument('--experiment-timeout-minutes', type=int, default=60*24,
                        help='Time in minutes to wait for the Experiment to complete')
    parser.add_argument('--delete-after-done', type=strtobool, default=True,
                        help='Whether to delete the Experiment after it is finished')

    parser.add_argument('--output-file', type=str, default='/output.txt',
                        help='The file which stores the best hyperparameters of the Experiment')

    args = parser.parse_args()

    experiment_name = args.experiment_name
    experiment_namespace = args.experiment_namespace

    logger.info("Creating Experiment: {} in namespace: {}".format(experiment_name, experiment_namespace))

    # Create JSON object from experiment spec
    experiment_spec = JSONObject(args.experiment_spec)
    # Deserialize JSON to ExperimentSpec
    experiment_spec = ApiClient().deserialize(experiment_spec, "V1beta1ExperimentSpec")

    # Create Experiment object.
    experiment = V1beta1Experiment(
        api_version="kubeflow.org/v1beta1",
        kind="Experiment",
        metadata=V1ObjectMeta(
            name=experiment_name,
            namespace=experiment_namespace
        ),
        spec=experiment_spec
    )

    # Create Katib client.
    katib_client = KatibClient()
    # Create Experiment in Kubernetes cluster.
    output = katib_client.create_experiment(experiment, namespace=experiment_namespace)

    # Wait until Experiment is created.
    end_time = datetime.datetime.now() + datetime.timedelta(minutes=args.experiment_timeout_minutes)
    while True:
        current_status = None
        # Try to get Experiment status.
        try:
            current_status = katib_client.get_experiment_status(name=experiment_name, namespace=experiment_namespace)
        except Exception:
            logger.info("Waiting until Experiment is created...")
        # If current status is set, exit the loop.
        if current_status is not None:
            break
        # If timeout has been reached, rise an exception.
        if datetime.datetime.now() > end_time:
            raise Exception("Timout waiting for Experiment: {} in namespace: {} to be created".format(
                experiment_name, experiment_namespace))
        time.sleep(1)

    logger.info("Experiment is created")

    # Wait for Experiment finish.
    wait_experiment_finish(katib_client, experiment, args.experiment_timeout_minutes)

    # Check if Experiment is successful.
    if katib_client.is_experiment_succeeded(name=experiment_name, namespace=experiment_namespace):
        logger.info("Experiment: {} in namespace: {} is successful".format(
            experiment_name, experiment_namespace))

        optimal_hp = katib_client.get_optimal_hyperparameters(
            name=experiment_name, namespace=experiment_namespace)
        logger.info("Optimal hyperparameters:\n{}".format(optimal_hp))

        # Create dir if it doesn't exist.
        if not os.path.exists(os.path.dirname(args.output_file)):
            os.makedirs(os.path.dirname(args.output_file))
        # Save HyperParameters to the file.
        with open(args.output_file, 'w') as f:
            f.write(json.dumps(optimal_hp))
    else:
        logger.info("Experiment: {} in namespace: {} is failed".format(
            experiment_name, experiment_namespace))
        # Print Experiment if it is failed.
        experiment = katib_client.get_experiment(name=experiment_name, namespace=experiment_namespace)
        logger.info(experiment)

    # Delete Experiment if it is needed.
    if args.delete_after_done:
        katib_client.delete_experiment(name=experiment_name, namespace=experiment_namespace)
        logger.info("Experiment: {} in namespace: {} has been deleted".format(
            experiment_name, experiment_namespace))

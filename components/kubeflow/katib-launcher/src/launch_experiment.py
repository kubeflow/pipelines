# Copyright 2020 kubeflow.org.
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
from kubernetes.client import V1Container
from kubernetes.client import V1HTTPGetAction

from kubeflow.katib import KatibClient
from kubeflow.katib import V1beta1Experiment
from kubeflow.katib import V1beta1ExperimentSpec
from kubeflow.katib import V1beta1ObjectiveSpec
from kubeflow.katib import V1beta1MetricStrategy
from kubeflow.katib import V1beta1AlgorithmSpec
from kubeflow.katib import V1beta1AlgorithmSetting
from kubeflow.katib import V1beta1EarlyStoppingSpec
from kubeflow.katib import V1beta1EarlyStoppingSetting
from kubeflow.katib import V1beta1ParameterSpec
from kubeflow.katib import V1beta1FeasibleSpace
from kubeflow.katib import V1beta1MetricsCollectorSpec
from kubeflow.katib import V1beta1CollectorSpec
from kubeflow.katib import V1beta1SourceSpec
from kubeflow.katib import V1beta1FileSystemPath
from kubeflow.katib import V1beta1FilterSpec
from kubeflow.katib import V1beta1TrialTemplate
from kubeflow.katib import V1beta1TrialParameterSpec

logger = logging.getLogger()
logging.basicConfig(level=logging.INFO)

FINISH_CONDITIONS = ["Succeeded", "Failed"]


def remove_none_from_dict(obj_dict):
    """ Returns dict without "None" values """
    if isinstance(obj_dict, (list, tuple, set)):
        return type(obj_dict)(remove_none_from_dict(x) for x in obj_dict if x is not None)
    elif isinstance(obj_dict, dict):
        return type(obj_dict)((remove_none_from_dict(k), remove_none_from_dict(v))
                              for k, v in obj_dict.items() if k is not None and v is not None)
    else:
        return obj_dict


def convert_dict_to_experiment_spec(experiment_spec_dict):
    """ Returns V1beta1ExperimentSpec for the given dict """

    experiment_spec = V1beta1ExperimentSpec()
    # Set Trials count.
    if "max_trial_count" in experiment_spec_dict:
        experiment_spec.max_trial_count = experiment_spec_dict["max_trial_count"]
    if "max_failed_trial_count" in experiment_spec_dict:
        experiment_spec.max_failed_trial_count = experiment_spec_dict["max_failed_trial_count"]
    if "parallel_trial_count" in experiment_spec_dict:
        experiment_spec.parallel_trial_count = experiment_spec_dict["parallel_trial_count"]

    # Set ResumePolicy.
    if "resume_policy" in experiment_spec_dict:
        experiment_spec.resume_policy = experiment_spec_dict["resume_policy"]

    # Set Objective.
    if "objective" in experiment_spec_dict:
        experiment_spec.objective = V1beta1ObjectiveSpec(
            **experiment_spec_dict["objective"]
        )
        # Convert metrics strategies.
        if experiment_spec.objective.metric_strategies is not None:
            experiment_spec.objective.metric_strategies = []
            for strategy in experiment_spec_dict["objective"]["metric_strategies"]:
                experiment_spec.objective.metric_strategies.append(
                    V1beta1MetricStrategy(
                        name=strategy["name"],
                        value=strategy["value"]
                    )
                )

    # Set Algorithm.
    if "algorithm" in experiment_spec_dict:
        experiment_spec.algorithm = V1beta1AlgorithmSpec(
            **experiment_spec_dict["algorithm"]
        )
        # Convert algorithm settings.
        if experiment_spec.algorithm.algorithm_settings is not None:
            experiment_spec.algorithm.algorithm_settings = []
            for setting in experiment_spec_dict["algorithm"]["algorithm_settings"]:
                experiment_spec.algorithm.algorithm_settings.append(
                    V1beta1AlgorithmSetting(
                        name=setting["name"],
                        value=setting["value"]
                    )
                )

    # Set Early Stopping.
    if "early_stopping" in experiment_spec_dict:
        experiment_spec.early_stopping = V1beta1EarlyStoppingSpec(
            **experiment_spec_dict["early_stopping"]
        )
        # Convert early stopping settings
        if experiment_spec.early_stopping.algorithm_settings is not None:
            experiment_spec.early_stopping.algorithm_settings = []
            for setting in experiment_spec_dict["early_stopping"]["algorithm_settings"]:
                experiment_spec.early_stopping.algorithm_settings.append(
                    V1beta1EarlyStoppingSetting(
                        name=setting["name"],
                        value=setting["value"]
                    )
                )

    # Set parameters.
    if "parameters" in experiment_spec_dict:
        experiment_spec.parameters = []
        for parameter in experiment_spec_dict["parameters"]:
            experiment_spec.parameters.append(
                V1beta1ParameterSpec(
                    name=parameter["name"],
                    parameter_type=parameter["parameter_type"],
                    feasible_space=V1beta1FeasibleSpace(
                        **parameter["feasible_space"]
                    )
                )
            )

    # Set Metrics Collector spec.
    if "metrics_collector_spec" in experiment_spec_dict:
        mc_spec = experiment_spec_dict["metrics_collector_spec"]
        collector_spec = V1beta1CollectorSpec()
        source_spec = V1beta1SourceSpec()

        # Convert collector.
        if "collector" in mc_spec:
            collector_spec.kind = mc_spec["kind"]
            # Convert custom collector container.
            if "custom_collector" in mc_spec["collector"]:
                collector_spec.custom_collector = V1Container(
                    **mc_spec["collector"]["custom_collector"]
                )

        # Convert Source.
        if "source" in mc_spec:
            # Convert file system path.
            if "file_system_path" in mc_spec["source"]:
                source_spec.file_system_path = V1beta1FileSystemPath(
                    kind=mc_spec["source"]["file_system_path"]["kind"],
                    path=mc_spec["source"]["file_system_path"]["path"]
                )
            # Convert filters.
            if "filter" in mc_spec["source"]:
                source_spec.spec.filter = V1beta1FilterSpec(
                    metrics_format=mc_spec["source"]["filter"]["metricsFormat"]
                )
            # Convert HTTP get.
            if "http_get" in mc_spec["source"]:
                source_spec.spec.http_get = V1HTTPGetAction(
                    **mc_spec["source"]["http_get"]
                )
        experiment_spec.metrics_collector_spec = V1beta1MetricsCollectorSpec(
            collector=collector_spec,
            source=source_spec
        )

    # Set Trial template parameters.
    if "trial_template" in experiment_spec_dict:
        experiment_spec.trial_template = V1beta1TrialTemplate(
            **experiment_spec_dict["trial_template"]
        )
        # Convert Trial parameters.
        if experiment_spec.trial_template.trial_parameters is not None:
            experiment_spec.trial_template.trial_parameters = []
            for parameter in experiment_spec_dict["trial_template"]["trial_parameters"]:
                experiment_spec.trial_template.trial_parameters.append(
                    V1beta1TrialParameterSpec(
                        **parameter
                    )
                )
    return experiment_spec


def wait_experiment_finish(katib_client, experiment, timeout):
    polling_interval = datetime.timedelta(seconds=30)
    end_time = datetime.datetime.now() + datetime.timedelta(minutes=timeout)
    experiment_name = experiment.metadata.name
    experiment_namespace = experiment.metadata.namespace
    while True:
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
    parser.add_argument('--experiment-spec', type=json.loads, default='',
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

    # Remove "None" values from the dict.
    experiment_spec = remove_none_from_dict(args.experiment_spec)
    # Convert Dict to V1beta1ExperimentSpec
    experiment_spec = convert_dict_to_experiment_spec(experiment_spec)

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

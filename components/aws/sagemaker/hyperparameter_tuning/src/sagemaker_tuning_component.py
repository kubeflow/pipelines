"""SageMaker component for tuning."""
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import logging
from typing import Dict
from sagemaker.image_uris import retrieve

from train.src.built_in_algos import BUILT_IN_ALGOS
from hyperparameter_tuning.src.sagemaker_tuning_spec import (
    SageMakerTuningSpec,
    SageMakerTuningInputs,
    SageMakerTuningOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)


@ComponentMetadata(
    name="SageMaker - Hyperparameter Tuning",
    description="Hyperparameter Tuning Jobs in SageMaker",
    spec=SageMakerTuningSpec,
)
class SageMakerTuningComponent(SageMakerComponent):
    """SageMaker component for tuning."""

    def Do(self, spec: SageMakerTuningSpec):
        self._tuning_job_name = (
            spec.inputs.job_name
            if spec.inputs.job_name
            else self._generate_unique_timestamped_id(prefix="HPOJob")
        )
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        response = self._sm_client.describe_hyper_parameter_tuning_job(
            HyperParameterTuningJobName=self._tuning_job_name
        )
        status = response["HyperParameterTuningJobStatus"]

        if status == "Completed":
            return SageMakerJobStatus(
                is_completed=True, has_error=False, raw_status=status
            )
        if status == "Failed":
            message = response["FailureReason"]
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=message,
                raw_status=status,
            )

        return SageMakerJobStatus(is_completed=False, raw_status=status)

    def _after_job_complete(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerTuningInputs,
        outputs: SageMakerTuningOutputs,
    ):
        (
            best_job,
            best_hyperparameters,
        ) = self._get_best_training_job_and_hyperparameters()
        model_artifact_url = self._get_model_artifacts_from_job(best_job)
        image = self._get_image_from_job(best_job)

        outputs.best_hyperparameters = best_hyperparameters
        outputs.best_job_name = best_job
        outputs.model_artifact_url = model_artifact_url
        outputs.training_image = image
        outputs.hpo_job_name = self._tuning_job_name

    def _on_job_terminated(self):
        self._sm_client.stop_hyper_parameter_tuning_job(
            HyperParameterTuningJobName=self._tuning_job_name
        )

    def _create_job_request(
        self, inputs: SageMakerTuningInputs, outputs: SageMakerTuningOutputs,
    ) -> Dict:
        ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_hyper_parameter_tuning_job
        request = self._get_request_template("hpo")

        ### Create a hyperparameter tuning job
        request["HyperParameterTuningJobName"] = self._tuning_job_name

        request["HyperParameterTuningJobConfig"]["Strategy"] = inputs.strategy
        request["HyperParameterTuningJobConfig"]["HyperParameterTuningJobObjective"][
            "Type"
        ] = inputs.metric_type
        request["HyperParameterTuningJobConfig"]["HyperParameterTuningJobObjective"][
            "MetricName"
        ] = inputs.metric_name
        request["HyperParameterTuningJobConfig"]["ResourceLimits"][
            "MaxNumberOfTrainingJobs"
        ] = inputs.max_num_jobs
        request["HyperParameterTuningJobConfig"]["ResourceLimits"][
            "MaxParallelTrainingJobs"
        ] = inputs.max_parallel_jobs
        request["HyperParameterTuningJobConfig"]["ParameterRanges"][
            "IntegerParameterRanges"
        ] = inputs.integer_parameters
        request["HyperParameterTuningJobConfig"]["ParameterRanges"][
            "ContinuousParameterRanges"
        ] = inputs.continuous_parameters
        request["HyperParameterTuningJobConfig"]["ParameterRanges"][
            "CategoricalParameterRanges"
        ] = inputs.categorical_parameters
        request["HyperParameterTuningJobConfig"][
            "TrainingJobEarlyStoppingType"
        ] = inputs.early_stopping_type

        request["TrainingJobDefinition"][
            "StaticHyperParameters"
        ] = self._validate_hyperparameters(inputs.static_parameters)
        request["TrainingJobDefinition"]["AlgorithmSpecification"][
            "TrainingInputMode"
        ] = inputs.training_input_mode

        ### Update training image (for BYOC) or algorithm resource name
        if not inputs.image and not inputs.algorithm_name:
            logging.error("Please specify training image or algorithm name.")
            raise Exception("Could not create job request")
        if inputs.image and inputs.algorithm_name:
            logging.error(
                "Both image and algorithm name inputted, only one should be specified. Proceeding with image."
            )

        if inputs.image:
            request["TrainingJobDefinition"]["AlgorithmSpecification"][
                "TrainingImage"
            ] = inputs.image
            request["TrainingJobDefinition"]["AlgorithmSpecification"].pop(
                "AlgorithmName"
            )
        else:
            # TODO: Adjust this implementation to account for custom algorithm resources names that are the same as built-in algorithm names
            algo_name = inputs.algorithm_name.lower().strip()
            if algo_name in BUILT_IN_ALGOS.keys():
                request["TrainingJobDefinition"]["AlgorithmSpecification"][
                    "TrainingImage"
                ] = retrieve(BUILT_IN_ALGOS[algo_name], inputs.region)
                request["TrainingJobDefinition"]["AlgorithmSpecification"].pop(
                    "AlgorithmName"
                )
                logging.warning(
                    "Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm."
                )
            # To give the user more leeway for built-in algorithm name inputs
            elif algo_name in BUILT_IN_ALGOS.values():
                request["TrainingJobDefinition"]["AlgorithmSpecification"][
                    "TrainingImage"
                ] = retrieve(algo_name, inputs.region)
                request["TrainingJobDefinition"]["AlgorithmSpecification"].pop(
                    "AlgorithmName"
                )
                logging.warning(
                    "Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm."
                )
            else:
                request["TrainingJobDefinition"]["AlgorithmSpecification"][
                    "AlgorithmName"
                ] = inputs.algorithm_name
                request["TrainingJobDefinition"]["AlgorithmSpecification"].pop(
                    "TrainingImage"
                )

        ### Update metric definitions
        if inputs.metric_definitions:
            for key, val in inputs.metric_definitions.items():
                request["TrainingJobDefinition"]["AlgorithmSpecification"][
                    "MetricDefinitions"
                ].append({"Name": key, "Regex": val})
        else:
            request["TrainingJobDefinition"]["AlgorithmSpecification"].pop(
                "MetricDefinitions"
            )

        ### Update or pop VPC configs
        if inputs.vpc_security_group_ids and inputs.vpc_subnets:
            request["TrainingJobDefinition"]["VpcConfig"][
                "SecurityGroupIds"
            ] = inputs.vpc_security_group_ids.split(",")
            request["TrainingJobDefinition"]["VpcConfig"][
                "Subnets"
            ] = inputs.vpc_subnets.split(",")
        else:
            request["TrainingJobDefinition"].pop("VpcConfig")

        ### Update input channels, must have at least one specified
        if len(inputs.channels) > 0:
            request["TrainingJobDefinition"]["InputDataConfig"] = inputs.channels
        else:
            logging.error("Must specify at least one input channel.")
            raise Exception("Could not make job request")

        request["TrainingJobDefinition"]["OutputDataConfig"][
            "S3OutputPath"
        ] = inputs.output_location
        request["TrainingJobDefinition"]["OutputDataConfig"][
            "KmsKeyId"
        ] = inputs.output_encryption_key
        request["TrainingJobDefinition"]["ResourceConfig"][
            "InstanceType"
        ] = inputs.instance_type
        request["TrainingJobDefinition"]["ResourceConfig"][
            "VolumeKmsKeyId"
        ] = inputs.resource_encryption_key
        request["TrainingJobDefinition"][
            "EnableNetworkIsolation"
        ] = inputs.network_isolation
        request["TrainingJobDefinition"][
            "EnableInterContainerTrafficEncryption"
        ] = inputs.traffic_encryption
        request["TrainingJobDefinition"]["RoleArn"] = inputs.role

        ### Update InstanceCount, VolumeSizeInGB, and MaxRuntimeInSeconds if input is non-empty and > 0, otherwise use default values
        if inputs.instance_count:
            request["TrainingJobDefinition"]["ResourceConfig"][
                "InstanceCount"
            ] = inputs.instance_count

        if inputs.volume_size:
            request["TrainingJobDefinition"]["ResourceConfig"][
                "VolumeSizeInGB"
            ] = inputs.volume_size

        if inputs.max_run_time:
            request["TrainingJobDefinition"]["StoppingCondition"][
                "MaxRuntimeInSeconds"
            ] = inputs.max_run_time

        ### Update or pop warm start configs
        if inputs.warm_start_type and inputs.parent_hpo_jobs:
            request["WarmStartConfig"]["WarmStartType"] = inputs.warm_start_type
            parent_jobs = [n.strip() for n in inputs.parent_hpo_jobs.split(",")]
            for i in range(len(parent_jobs)):
                request["WarmStartConfig"]["ParentHyperParameterTuningJobs"].append(
                    {"HyperParameterTuningJobName": parent_jobs[i]}
                )
        else:
            if inputs.warm_start_type or inputs.parent_hpo_jobs:
                if not inputs.warm_start_type:
                    logging.error(
                        'Must specify warm start type as either "IdenticalDataAndAlgorithm" or "TransferLearning".'
                    )
                if not inputs.parent_hpo_jobs:
                    logging.error(
                        "Must specify at least one parent hyperparameter tuning job"
                    )
                raise Exception("Could not make job request")
            request.pop("WarmStartConfig")

        self._enable_spot_instance_support(request["TrainingJobDefinition"], inputs)
        self._enable_tag_support(request, inputs)

        return request

    def _submit_job_request(self, request: Dict) -> object:
        return self._sm_client.create_hyper_parameter_tuning_job(**request)

    def _after_submit_job_request(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerTuningInputs,
        outputs: SageMakerTuningOutputs,
    ):
        logging.info(
            "Created Hyperparameter Training Job with name: " + self._tuning_job_name
        )
        logging.info(
            "HPO job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/hyper-tuning-jobs/{}".format(
                inputs.region, inputs.region, self._tuning_job_name
            )
        )
        logging.info(
            "CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/TrainingJobs;prefix={};streamFilter=typeLogStreamPrefix".format(
                inputs.region, inputs.region, self._tuning_job_name
            )
        )

    def _get_best_training_job_and_hyperparameters(self):
        """Gets the best rated training job and accompanying hyperparameters.

        Returns:
            tuple: (Name of the best training job, training job hyperparameters).
        """
        ### Get and return best training job and its hyperparameters, without the objective metric
        info = self._sm_client.describe_hyper_parameter_tuning_job(
            HyperParameterTuningJobName=self._tuning_job_name
        )
        best_job = info["BestTrainingJob"]["TrainingJobName"]
        training_info = self._sm_client.describe_training_job(TrainingJobName=best_job)
        train_hyperparameters = training_info["HyperParameters"]
        train_hyperparameters.pop("_tuning_objective_metric")
        return best_job, train_hyperparameters


if __name__ == "__main__":
    import sys

    spec = SageMakerTuningSpec(sys.argv[1:])

    component = SageMakerTuningComponent()
    component.Do(spec)

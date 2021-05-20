"""Specification for the SageMaker tuning component."""
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

from dataclasses import dataclass

from typing import List
from common.sagemaker_component_spec import (
    SageMakerComponentSpec,
    SageMakerComponentBaseOutputs,
)
from common.spec_input_parsers import SpecInputParsers
from common.common_inputs import (
    COMMON_INPUTS,
    SPOT_INSTANCE_INPUTS,
    SageMakerComponentCommonInputs,
    SageMakerComponentInput as Input,
    SageMakerComponentOutput as Output,
    SageMakerComponentInputValidator as InputValidator,
    SageMakerComponentOutputValidator as OutputValidator,
    SpotInstanceInputs,
)


@dataclass(frozen=True)
class SageMakerTuningInputs(SageMakerComponentCommonInputs, SpotInstanceInputs):
    """Defines the set of inputs for the tuning component."""

    job_name: Input
    role: Input
    image: Input
    algorithm_name: Input
    training_input_mode: Input
    metric_definitions: Input
    strategy: Input
    metric_name: Input
    metric_type: Input
    early_stopping_type: Input
    static_parameters: Input
    integer_parameters: Input
    continuous_parameters: Input
    categorical_parameters: Input
    channels: Input
    output_location: Input
    output_encryption_key: Input
    instance_type: Input
    instance_count: Input
    volume_size: Input
    max_num_jobs: Input
    max_parallel_jobs: Input
    resource_encryption_key: Input
    vpc_security_group_ids: Input
    vpc_subnets: Input
    network_isolation: Input
    traffic_encryption: Input
    warm_start_type: Input
    parent_hpo_jobs: Input


@dataclass
class SageMakerTuningOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the tuning component."""

    hpo_job_name: Output
    model_artifact_url: Output
    best_job_name: Output
    best_hyperparameters: Output
    training_image: Output


class SageMakerTuningSpec(
    SageMakerComponentSpec[SageMakerTuningInputs, SageMakerTuningOutputs]
):
    INPUTS: SageMakerTuningInputs = SageMakerTuningInputs(
        job_name=InputValidator(
            input_type=str,
            required=False,
            description="The name of the tuning job. Must be unique within the same AWS account and AWS region.",
        ),
        role=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.",
        ),
        image=InputValidator(
            input_type=str,
            required=False,
            description="The registry path of the Docker image that contains the training algorithm.",
            default="",
        ),
        algorithm_name=InputValidator(
            input_type=str,
            required=False,
            description="The name of the resource algorithm to use for the hyperparameter tuning job.",
            default="",
        ),
        training_input_mode=InputValidator(
            choices=["File", "Pipe"],
            input_type=str,
            required=False,
            description="The input mode that the algorithm supports. File or Pipe.",
            default="File",
        ),
        metric_definitions=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            required=False,
            description="The dictionary of name-regex pairs specify the metrics that the algorithm emits.",
            default={},
        ),
        strategy=InputValidator(
            choices=["Bayesian", "Random"],
            input_type=str,
            required=False,
            description="How hyperparameter tuning chooses the combinations of hyperparameter values to use for the training job it launches.",
            default="Bayesian",
        ),
        metric_name=InputValidator(
            input_type=str,
            required=True,
            description="The name of the metric to use for the objective metric.",
        ),
        metric_type=InputValidator(
            choices=["Maximize", "Minimize"],
            input_type=str,
            required=True,
            description="Whether to minimize or maximize the objective metric.",
        ),
        early_stopping_type=InputValidator(
            choices=["Off", "Auto"],
            input_type=str,
            required=False,
            description="Whether to minimize or maximize the objective metric.",
            default="Off",
        ),
        static_parameters=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            required=False,
            description="The values of hyperparameters that do not change for the tuning job.",
            default={},
        ),
        integer_parameters=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="The array of IntegerParameterRange objects that specify ranges of integer hyperparameters that you want to search.",
            default=[],
        ),
        continuous_parameters=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="The array of ContinuousParameterRange objects that specify ranges of continuous hyperparameters that you want to search.",
            default=[],
        ),
        categorical_parameters=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="The array of CategoricalParameterRange objects that specify ranges of categorical hyperparameters that you want to search.",
            default=[],
        ),
        channels=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=True,
            description="A list of dicts specifying the input channels. Must have at least one.",
        ),
        output_location=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon S3 path where you want Amazon SageMaker to store the results of the transform job.",
        ),
        output_encryption_key=InputValidator(
            input_type=str,
            required=False,
            description="The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.",
            default="",
        ),
        instance_type=InputValidator(
            input_type=str,
            required=False,
            description="The ML compute instance type.",
            default="ml.m4.xlarge",
        ),
        instance_count=InputValidator(
            input_type=int,
            required=False,
            description="The number of ML compute instances to use in each training job.",
            default=1,
        ),
        volume_size=InputValidator(
            input_type=int,
            required=False,
            description="The size of the ML storage volume that you want to provision.",
            default=30,
        ),
        max_num_jobs=InputValidator(
            input_type=int,
            required=True,
            description="The maximum number of training jobs that a hyperparameter tuning job can launch.",
        ),
        max_parallel_jobs=InputValidator(
            input_type=int,
            required=True,
            description="The maximum number of concurrent training jobs that a hyperparameter tuning job can launch.",
        ),
        resource_encryption_key=InputValidator(
            input_type=str,
            required=False,
            description="The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).",
            default="",
        ),
        vpc_security_group_ids=InputValidator(
            input_type=str,
            required=False,
            description="The VPC security group IDs, in the form sg-xxxxxxxx.",
        ),
        vpc_subnets=InputValidator(
            input_type=str,
            required=False,
            description="The ID of the subnets in the VPC to which you want to connect your hpo job.",
        ),
        network_isolation=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            required=False,
            description="Isolates the training container.",
            default=True,
        ),
        traffic_encryption=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            required=False,
            description="Encrypts all communications between ML compute instances in distributed training.",
            default=False,
        ),
        warm_start_type=InputValidator(
            choices=["IdenticalDataAndAlgorithm", "TransferLearning", ""],
            input_type=str,
            required=False,
            description='Specifies either "IdenticalDataAndAlgorithm" or "TransferLearning"',
        ),
        parent_hpo_jobs=InputValidator(
            input_type=str,
            required=False,
            description="List of previously completed or stopped hyperparameter tuning jobs to be used as a starting point.",
            default="",
        ),
        **vars(COMMON_INPUTS),
        **vars(SPOT_INSTANCE_INPUTS)
    )

    OUTPUTS = SageMakerTuningOutputs(
        hpo_job_name=OutputValidator(
            description="The name of the hyper parameter tuning job."
        ),
        model_artifact_url=OutputValidator(
            description="The output model artifacts S3 url."
        ),
        best_job_name=OutputValidator(
            description="Best training job in the hyper parameter tuning job."
        ),
        best_hyperparameters=OutputValidator(
            description="The resulting tuned hyperparameters."
        ),
        training_image=OutputValidator(
            description="The registry path of the Docker image that contains the training algorithm."
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(arguments, SageMakerTuningInputs, SageMakerTuningOutputs)

    @property
    def inputs(self) -> SageMakerTuningInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerTuningOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerTuningOutputs:
        return self._output_paths

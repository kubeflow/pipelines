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

"""Specification for the SageMaker - TrainingJob"""

from dataclasses import dataclass

from typing import List
from commonv2.sagemaker_component_spec import (
    SageMakerComponentSpec,
    SageMakerComponentBaseOutputs,
)
from commonv2.spec_input_parsers import SpecInputParsers
from commonv2.common_inputs import (
    COMMON_INPUTS,
    SageMakerComponentCommonInputs,
    SageMakerComponentInput as Input,
    SageMakerComponentOutput as Output,
    SageMakerComponentInputValidator as InputValidator,
    SageMakerComponentOutputValidator as OutputValidator,
)


@dataclass(frozen=False)
class SageMakerTrainingJobInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the TrainingJob component."""

    algorithm_specification: Input
    checkpoint_config: Input
    debug_hook_config: Input
    debug_rule_configurations: Input
    enable_inter_container_traffic_encryption: Input
    enable_managed_spot_training: Input
    enable_network_isolation: Input
    environment: Input
    experiment_config: Input
    hyper_parameters: Input
    input_data_config: Input
    output_data_config: Input
    profiler_config: Input
    profiler_rule_configurations: Input
    resource_config: Input
    retry_strategy: Input
    role_arn: Input
    stopping_condition: Input
    tags: Input
    tensor_board_output_config: Input
    training_job_name: Input
    vpc_config: Input


@dataclass
class SageMakerTrainingJobOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the TrainingJob component."""

    ack_resource_metadata: Output
    conditions: Output
    creation_time: Output
    debug_rule_evaluation_statuses: Output
    failure_reason: Output
    last_modified_time: Output
    model_artifacts: Output
    profiler_rule_evaluation_statuses: Output
    profiling_status: Output
    secondary_status: Output
    training_job_status: Output
    warm_pool_status: Output


class SageMakerTrainingJobSpec(
    SageMakerComponentSpec[SageMakerTrainingJobInputs, SageMakerTrainingJobOutputs]
):
    INPUTS: SageMakerTrainingJobInputs = SageMakerTrainingJobInputs(
        algorithm_specification=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="The registry path of the Docker image that contains the training algorithm and algorithm-specific me",
            required=True,
        ),
        checkpoint_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Contains information about the output location for managed spot training checkpoint data.",
            required=False,
        ),
        debug_hook_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Configuration information for the Amazon SageMaker Debugger hook parameters, metric and tensor colle",
            required=False,
        ),
        debug_rule_configurations=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="Configuration information for Amazon SageMaker Debugger rules for debugging output tensors.",
            required=False,
        ),
        enable_inter_container_traffic_encryption=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            description="To encrypt all communications between ML compute instances in distributed training, choose True. Enc",
            required=False,
        ),
        enable_managed_spot_training=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            description="To train models using managed spot training, choose True. Managed spot training provides a fully man",
            required=False,
        ),
        enable_network_isolation=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            description="Isolates the training container. No inbound or outbound network calls can be made, except for calls",
            required=False,
        ),
        environment=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="The environment variables to set in the Docker container.",
            required=False,
        ),
        experiment_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Associates a SageMaker job as a trial component with an experiment and trial. Specified when you cal",
            required=False,
        ),
        hyper_parameters=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Algorithm-specific parameters that influence the quality of the model. You set hyperparameters befor",
            required=False,
        ),
        input_data_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="An array of Channel objects. Each channel is a named input source. InputDataConfig describes the inp",
            required=False,
        ),
        output_data_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Specifies the path to the S3 location where you want to store model artifacts. SageMaker creates sub",
            required=True,
        ),
        profiler_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Configuration information for Amazon SageMaker Debugger system monitoring, framework profiling, and",
            required=False,
        ),
        profiler_rule_configurations=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="Configuration information for Amazon SageMaker Debugger rules for profiling system and framework met",
            required=False,
        ),
        resource_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="The resources, including the ML compute instances and ML storage volumes, to use for model training.",
            required=True,
        ),
        retry_strategy=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="The number of times to retry the job when the job fails due to an InternalServerError.",
            required=False,
        ),
        role_arn=InputValidator(
            input_type=str,
            description="The Amazon Resource Name (ARN) of an IAM role that SageMaker can assume to perform tasks on your beh",
            required=True,
        ),
        stopping_condition=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Specifies a limit to how long a model training job can run. It also specifies how long a managed Spo",
            required=True,
        ),
        tags=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="An array of key-value pairs. You can use tags to categorize your Amazon Web Services resources in di",
            required=False,
        ),
        tensor_board_output_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Configuration of storage locations for the Amazon SageMaker Debugger TensorBoard output data.",
            required=False,
        ),
        training_job_name=InputValidator(
            input_type=str,
            description="The name of the training job. The name must be unique within an Amazon Web Services Region in an Ama",
            required=True,
        ),
        vpc_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="A VpcConfig object that specifies the VPC that you want your training job to connect to. Control acc",
            required=False,
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = SageMakerTrainingJobOutputs(
        ack_resource_metadata=OutputValidator(
            description="All CRs managed by ACK have a common `Status.ACKResourceMetadata` member that is used to contain res",
        ),
        conditions=OutputValidator(
            description="All CRS managed by ACK have a common `Status.Conditions` member that contains a collection of `ackv1",
        ),
        creation_time=OutputValidator(
            description="A timestamp that indicates when the training job was created.",
        ),
        debug_rule_evaluation_statuses=OutputValidator(
            description="Evaluation status of Amazon SageMaker Debugger rules for debugging on a training job.",
        ),
        failure_reason=OutputValidator(
            description="If the training job failed, the reason it failed.",
        ),
        last_modified_time=OutputValidator(
            description="A timestamp that indicates when the status of the training job was last modified.",
        ),
        model_artifacts=OutputValidator(
            description="Information about the Amazon S3 location that is configured for storing model artifacts.",
        ),
        profiler_rule_evaluation_statuses=OutputValidator(
            description="Evaluation status of Amazon SageMaker Debugger rules for profiling on a training job.",
        ),
        profiling_status=OutputValidator(
            description="Profiling status of a training job.",
        ),
        secondary_status=OutputValidator(
            description="Provides detailed information about the state of the training job. For detailed information on the s",
        ),
        training_job_status=OutputValidator(
            description="The status of the training job.   SageMaker provides the following training job statuses:   * InProg",
        ),
        warm_pool_status=OutputValidator(
            description="The status of the warm pool associated with the training job.",
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(
            arguments, SageMakerTrainingJobInputs, SageMakerTrainingJobOutputs
        )

    @property
    def inputs(self) -> SageMakerTrainingJobInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerTrainingJobOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerTrainingJobOutputs:
        return self._output_paths

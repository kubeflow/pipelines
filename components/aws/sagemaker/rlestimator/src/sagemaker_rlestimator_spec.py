"""Specification for the SageMaker RLEstimator component."""
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
    SageMakerComponentCommonInputs,
    SpotInstanceInputs,
    SPOT_INSTANCE_INPUTS,
    SageMakerComponentInput as Input,
    SageMakerComponentOutput as Output,
    SageMakerComponentInputValidator as InputValidator,
    SageMakerComponentOutputValidator as OutputValidator,
)


@dataclass(frozen=True)
class SageMakerRLEstimatorInputs(SageMakerComponentCommonInputs, SpotInstanceInputs):
    """Defines the set of inputs for the rlestimator component."""

    job_name: Input
    role: Input
    image: Input
    entry_point: Input
    source_dir: Input
    toolkit: Input
    toolkit_version: Input
    framework: Input
    metric_definitions: Input
    training_input_mode: Input
    hyperparameters: Input
    instance_type: Input
    instance_count: Input
    volume_size: Input
    max_run: Input
    model_artifact_path: Input
    vpc_security_group_ids: Input
    vpc_subnets: Input
    network_isolation: Input
    traffic_encryption: Input
    debug_hook_config: Input
    debug_rule_config: Input


@dataclass
class SageMakerRLEstimatorOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the rlestimator component."""

    model_artifact_url: Output
    job_name: Output
    training_image: Output


class SageMakerRLEstimatorSpec(
    SageMakerComponentSpec[SageMakerRLEstimatorInputs, SageMakerRLEstimatorOutputs]
):
    INPUTS: SageMakerRLEstimatorInputs = SageMakerRLEstimatorInputs(
        job_name=InputValidator(
            input_type=str,
            required=False,
            description="Training job name.",
            default="",
        ),
        role=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.",
        ),
        image=InputValidator(
            input_type=str,
            required=False,
            description="An ECR url. If specified, the estimator will use this image for training and hosting",
            default=None,
        ),
        entry_point=InputValidator(
            input_type=str,
            required=True,
            description="Path (absolute or relative) to the Python source file which should be executed as the entry point to training.",
            default="",
        ),
        source_dir=InputValidator(
            input_type=str,
            required=False,
            description="Path (S3 URI) to a directory with any other training source code dependencies aside from the entry point file.",
            default="",
        ),
        toolkit=InputValidator(
            input_type=str,
            choices=["coach", "ray", ""],
            required=False,
            description="RL toolkit you want to use for executing your model training code.",
            default="",
        ),
        toolkit_version=InputValidator(
            input_type=str,
            required=False,
            description="RL toolkit version you want to be use for executing your model training code.",
            default=None,
        ),
        framework=InputValidator(
            input_type=str,
            choices=["tensorflow", "mxnet", "pytorch", ""],
            required=False,
            description="Framework (MXNet, TensorFlow or PyTorch) you want to be used as a toolkit backed for reinforcement learning training.",
            default="",
        ),
        metric_definitions=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="The dictionary of name-regex pairs specify the metrics that the algorithm emits.",
            default=[],
        ),
        training_input_mode=InputValidator(
            choices=["File", "Pipe"],
            input_type=str,
            description="The input mode that the algorithm supports. File or Pipe.",
            default="File",
        ),
        hyperparameters=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            required=False,
            description="Hyperparameters that will be used for training.",
            default={},
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
            description="The number of ML compute instances to use in the training job.",
            default=1,
        ),
        volume_size=InputValidator(
            input_type=int,
            required=True,
            description="The size of the ML storage volume that you want to provision.",
            default=30,
        ),
        max_run=InputValidator(
            input_type=int,
            required=False,
            description="Timeout in seconds for training (default: 24 * 60 * 60).",
            default=24 * 60 * 60,
        ),
        model_artifact_path=InputValidator(
            input_type=str,
            required=True,
            description="Identifies the S3 path where you want Amazon SageMaker to store the model artifacts.",
        ),
        vpc_security_group_ids=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="The VPC security group IDs, in the form sg-xxxxxxxx.",
            default=[],
        ),
        vpc_subnets=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="The ID of the subnets in the VPC to which you want to connect your hpo job.",
            default=[],
        ),
        network_isolation=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            description="Isolates the training container.",
            default=False,
        ),
        traffic_encryption=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            description="Encrypts all communications between ML compute instances in distributed training.",
            default=False,
        ),
        debug_hook_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            required=False,
            description="Configuration information for the debug hook parameters, collection configuration, and storage paths.",
            default={},
        ),
        debug_rule_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="Configuration information for debugging rules.",
            default=[],
        ),
        **vars(COMMON_INPUTS),
        **vars(SPOT_INSTANCE_INPUTS)
    )

    OUTPUTS = SageMakerRLEstimatorOutputs(
        model_artifact_url=OutputValidator(description="The model artifacts URL."),
        job_name=OutputValidator(description="The training job name."),
        training_image=OutputValidator(
            description="The registry path of the Docker image that contains the training algorithm."
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(
            arguments, SageMakerRLEstimatorInputs, SageMakerRLEstimatorOutputs
        )

    @property
    def inputs(self) -> SageMakerRLEstimatorInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerRLEstimatorOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerRLEstimatorOutputs:
        return self._output_paths

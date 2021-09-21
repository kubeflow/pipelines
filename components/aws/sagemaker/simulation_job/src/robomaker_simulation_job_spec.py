"""Specification for the RoboMaker create simulation job component."""
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
from common.sagemaker_component_spec import SageMakerComponentSpec
from common.spec_input_parsers import SpecInputParsers
from common.common_inputs import (
    COMMON_INPUTS,
    SageMakerComponentCommonInputs,
    SageMakerComponentInput as Input,
    SageMakerComponentOutput as Output,
    SageMakerComponentBaseOutputs,
    SageMakerComponentInputValidator as InputValidator,
    SageMakerComponentOutputValidator as OutputValidator,
)


@dataclass(frozen=True)
class RoboMakerSimulationJobInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the create simulation job component."""

    role: Input
    output_bucket: Input
    output_path: Input
    max_run: Input
    failure_behavior: Input
    sim_app_arn: Input
    sim_app_version: Input
    sim_app_launch_config: Input
    sim_app_world_config: Input
    robot_app_arn: Input
    robot_app_version: Input
    robot_app_launch_config: Input
    data_sources: Input
    vpc_security_group_ids: Input
    vpc_subnets: Input
    use_public_ip: Input
    sim_unit_limit: Input
    record_ros_topics: Input


@dataclass
class RoboMakerSimulationJobOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the create simulation job component."""

    arn: Output
    output_artifacts: Output
    job_id: Output


class RoboMakerSimulationJobSpec(
    SageMakerComponentSpec[RoboMakerSimulationJobInputs, RoboMakerSimulationJobOutputs]
):
    INPUTS: RoboMakerSimulationJobInputs = RoboMakerSimulationJobInputs(
        role=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon Resource Name (ARN) that Amazon RoboMaker assumes to perform tasks on your behalf.",
        ),
        output_bucket=InputValidator(
            input_type=str,
            required=True,
            description="The bucket to place outputs from the simulation job.",
            default="",
        ),
        output_path=InputValidator(
            input_type=str,
            required=True,
            description="The S3 key where outputs from the simulation job are placed.",
            default="",
        ),
        max_run=InputValidator(
            input_type=int,
            required=True,
            description="Timeout in seconds for simulation job (default: 8 * 60 * 60).",
            default=8 * 60 * 60,
        ),
        failure_behavior=InputValidator(
            input_type=str,
            required=False,
            description="The failure behavior the simulation job (Continue|Fail).",
            default="Fail",
        ),
        sim_app_arn=InputValidator(
            input_type=str,
            required=False,
            description="The application ARN for the simulation application.",
            default="",
        ),
        sim_app_version=InputValidator(
            input_type=str,
            required=False,
            description="The application version for the simulation application.",
            default="",
        ),
        sim_app_launch_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            required=False,
            description="The launch configuration for the simulation application.",
            default={},
        ),
        sim_app_world_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="A list of world configurations.",
            default=[],
        ),
        robot_app_arn=InputValidator(
            input_type=str,
            required=False,
            description="The application ARN for the robot application.",
            default="",
        ),
        robot_app_version=InputValidator(
            input_type=str,
            required=False,
            description="The application version for the robot application.",
            default="",
        ),
        robot_app_launch_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            required=False,
            description="The launch configuration for the robot application.",
            default={},
        ),
        data_sources=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="Specify data sources to mount read-only files from S3 into your simulation.",
            default=[],
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
            description="The ID of the subnets in the VPC to which you want to connect your simulation job.",
            default=[],
        ),
        use_public_ip=InputValidator(
            input_type=bool,
            description="A boolean indicating whether to assign a public IP address.",
            default=False,
        ),
        sim_unit_limit=InputValidator(
            input_type=int,
            required=False,
            description="The simulation unit limit.",
            default=15,
        ),
        record_ros_topics=InputValidator(
            input_type=bool,
            description="A boolean indicating whether to record all ROS topics. Used for logging.",
            default=False,
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = RoboMakerSimulationJobOutputs(
        arn=OutputValidator(
            description="The Amazon Resource Name (ARN) of the simulation job."
        ),
        output_artifacts=OutputValidator(
            description="The simulation job artifacts URL."
        ),
        job_id=OutputValidator(description="The simulation job id."),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(
            arguments, RoboMakerSimulationJobInputs, RoboMakerSimulationJobOutputs,
        )

    @property
    def inputs(self) -> RoboMakerSimulationJobInputs:
        return self._inputs

    @property
    def outputs(self) -> RoboMakerSimulationJobOutputs:
        return self._outputs

    @property
    def output_paths(self) -> RoboMakerSimulationJobOutputs:
        return self._output_paths

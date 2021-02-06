"""Specification for the RoboMaker create simulation application component."""
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
class RoboMakerCreateSimulationAppInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the create simulation application component."""

    app_name: Input
    sources: Input
    simulation_software_name: Input
    simulation_software_version: Input
    robot_software_name: Input
    robot_software_version: Input
    rendering_engine_name: Input
    rendering_engine_version: Input


@dataclass
class RoboMakerCreateSimulationAppOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the create simulation application component."""

    arn: Output
    app_name: Output
    version: Output
    revision_id: Output


class RoboMakerCreateSimulationAppSpec(
    SageMakerComponentSpec[
        RoboMakerCreateSimulationAppInputs, RoboMakerCreateSimulationAppOutputs
    ]
):
    INPUTS: RoboMakerCreateSimulationAppInputs = RoboMakerCreateSimulationAppInputs(
        app_name=InputValidator(
            input_type=str,
            required=True,
            description="The name of the simulation application.",
            default="",
        ),
        sources=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=True,
            description="The code sources of the simulation application.",
            default={},
        ),
        simulation_software_name=InputValidator(
            input_type=str,
            required=True,
            description="The simulation software used by the simulation application.",
            default="",
        ),
        simulation_software_version=InputValidator(
            input_type=str,
            required=True,
            description="The simulation software version used by the simulation application.",
            default="",
        ),
        robot_software_name=InputValidator(
            input_type=str,
            required=True,
            description="The robot software used by the simulation application.",
            default="",
        ),
        robot_software_version=InputValidator(
            input_type=str,
            required=True,
            description="The robot software version used by the simulation application.",
            default="",
        ),
        rendering_engine_name=InputValidator(
            input_type=str,
            required=False,
            description="The rendering engine for the simulation application.",
            default="",
        ),
        rendering_engine_version=InputValidator(
            input_type=str,
            required=False,
            description="The rendering engine version for the simulation application.",
            default="",
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = RoboMakerCreateSimulationAppOutputs(
        arn=OutputValidator(
            description="The Amazon Resource Name (ARN) of the simulation application."
        ),
        app_name=OutputValidator(description="The name of the simulation application."),
        version=OutputValidator(
            description="The version of the simulation application."
        ),
        revision_id=OutputValidator(
            description="The revision id of the simulation application."
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(
            arguments,
            RoboMakerCreateSimulationAppInputs,
            RoboMakerCreateSimulationAppOutputs,
        )

    @property
    def inputs(self) -> RoboMakerCreateSimulationAppInputs:
        return self._inputs

    @property
    def outputs(self) -> RoboMakerCreateSimulationAppOutputs:
        return self._outputs

    @property
    def output_paths(self) -> RoboMakerCreateSimulationAppOutputs:
        return self._output_paths

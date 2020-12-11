"""Specification for the RoboMaker delete. simulation application component."""
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
class RoboMakerDeleteSimulationAppInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the delete simulation application component."""

    arn: Input
    version: Input


@dataclass
class RoboMakerDeleteSimulationAppOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the create simulation application component."""

    arn: Output


class RoboMakerDeleteSimulationAppSpec(
    SageMakerComponentSpec[
        RoboMakerDeleteSimulationAppInputs, RoboMakerDeleteSimulationAppOutputs
    ]
):
    INPUTS: RoboMakerDeleteSimulationAppInputs = RoboMakerDeleteSimulationAppInputs(
        arn=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon Resource Name (ARN) of the simulation application.",
            default="",
        ),
        version=InputValidator(
            input_type=str,
            required=False,
            description="The version of the simulation application.",
            default=None,
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = RoboMakerDeleteSimulationAppOutputs(
        arn=OutputValidator(
            description="The Amazon Resource Name (ARN) of the simulation application."
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(
            arguments,
            RoboMakerDeleteSimulationAppInputs,
            RoboMakerDeleteSimulationAppOutputs,
        )

    @property
    def inputs(self) -> RoboMakerDeleteSimulationAppInputs:
        return self._inputs

    @property
    def outputs(self) -> RoboMakerDeleteSimulationAppOutputs:
        return self._outputs

    @property
    def output_paths(self) -> RoboMakerDeleteSimulationAppOutputs:
        return self._output_paths

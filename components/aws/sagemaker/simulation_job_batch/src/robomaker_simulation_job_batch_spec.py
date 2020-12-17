"""Specification for the RoboMaker create simulation job batch component."""
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
class RoboMakerSimulationJobBatchInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the create simulation job batch component."""

    role: Input
    timeout_in_secs: Input
    max_concurrency: Input
    simulation_job_requests: Input
    sim_app_arn: Input


@dataclass
class RoboMakerSimulationJobBatchOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the create simulation job batch component."""

    arn: Output
    batch_job_id: Output


class RoboMakerSimulationJobBatchSpec(
    SageMakerComponentSpec[
        RoboMakerSimulationJobBatchInputs, RoboMakerSimulationJobBatchOutputs
    ]
):
    INPUTS: RoboMakerSimulationJobBatchInputs = RoboMakerSimulationJobBatchInputs(
        role=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon Resource Name (ARN) that Amazon RoboMaker assumes to perform tasks on your behalf.",
        ),
        timeout_in_secs=InputValidator(
            input_type=int,
            required=False,
            description="The amount of time, in seconds, to wait for the batch to complete.",
            default=0,
        ),
        max_concurrency=InputValidator(
            input_type=int,
            required=False,
            description="The number of active simulation jobs create as part of the batch that can be in an active state at the same time.",
            default=0,
        ),
        simulation_job_requests=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=True,
            description="A list of simulation job requests to create in the batch.",
            default=[],
        ),
        sim_app_arn=InputValidator(
            input_type=str,
            required=False,
            description="The application ARN for the simulation application.",
            default="",
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = RoboMakerSimulationJobBatchOutputs(
        arn=OutputValidator(
            description="The Amazon Resource Name (ARN) of the simulation job."
        ),
        batch_job_id=OutputValidator(description="The simulation job batch id."),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(
            arguments,
            RoboMakerSimulationJobBatchInputs,
            RoboMakerSimulationJobBatchOutputs,
        )

    @property
    def inputs(self) -> RoboMakerSimulationJobBatchInputs:
        return self._inputs

    @property
    def outputs(self) -> RoboMakerSimulationJobBatchOutputs:
        return self._outputs

    @property
    def output_paths(self) -> RoboMakerSimulationJobBatchOutputs:
        return self._output_paths

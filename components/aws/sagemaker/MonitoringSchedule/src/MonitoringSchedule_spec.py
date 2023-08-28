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

"""Specification for the SageMaker - MonitoringSchedule"""

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
class SageMakerMonitoringScheduleInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the MonitoringSchedule component."""

    monitoring_schedule_config: Input
    monitoring_schedule_name: Input
    tags: Input


@dataclass
class SageMakerMonitoringScheduleOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the MonitoringSchedule component."""

    ack_resource_metadata: Output
    conditions: Output
    creation_time: Output
    failure_reason: Output
    last_modified_time: Output
    last_monitoring_execution_summary: Output
    monitoring_schedule_status: Output
    sagemaker_resource_name: Output


class SageMakerMonitoringScheduleSpec(
    SageMakerComponentSpec[
        SageMakerMonitoringScheduleInputs, SageMakerMonitoringScheduleOutputs
    ]
):
    INPUTS: SageMakerMonitoringScheduleInputs = SageMakerMonitoringScheduleInputs(
        monitoring_schedule_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="The configuration object that specifies the monitoring schedule and defines the monitoring job.",
            required=True,
        ),
        monitoring_schedule_name=InputValidator(
            input_type=str,
            description="The name of the monitoring schedule.",
            required=True,
        ),
        tags=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="(Optional) An array of key-value pairs.",
            required=False,
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = SageMakerMonitoringScheduleOutputs(
        ack_resource_metadata=OutputValidator(
            description="All CRs managed by ACK have a common `Status.",
        ),
        conditions=OutputValidator(
            description="All CRS managed by ACK have a common `Status.",
        ),
        creation_time=OutputValidator(
            description="The time at which the monitoring job was created.",
        ),
        failure_reason=OutputValidator(
            description="A string, up to one KB in size, that contains the reason a monitoring job failed, if it failed.",
        ),
        last_modified_time=OutputValidator(
            description="The time at which the monitoring job was last modified.",
        ),
        last_monitoring_execution_summary=OutputValidator(
            description="Describes metadata on the last execution to run, if there was one.",
        ),
        monitoring_schedule_status=OutputValidator(
            description="The status of an monitoring job.",
        ),
        sagemaker_resource_name=OutputValidator(
            description="Resource name on Sagemaker",
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(
            arguments,
            SageMakerMonitoringScheduleInputs,
            SageMakerMonitoringScheduleOutputs,
        )

    @property
    def inputs(self) -> SageMakerMonitoringScheduleInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerMonitoringScheduleOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerMonitoringScheduleOutputs:
        return self._output_paths

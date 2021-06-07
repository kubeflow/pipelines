"""Specification for the SageMaker workteam component."""
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
    SageMakerComponentInput as Input,
    SageMakerComponentOutput as Output,
    SageMakerComponentInputValidator as InputValidator,
    SageMakerComponentOutputValidator as OutputValidator,
)


@dataclass(frozen=True)
class SageMakerWorkteamInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the workteam component."""

    team_name: Input
    description: Input
    user_pool: Input
    user_groups: Input
    client_id: Input
    sns_topic: Input


@dataclass
class SageMakerWorkteamOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the workteam component."""

    workteam_arn: Output


class SageMakerWorkteamSpec(
    SageMakerComponentSpec[SageMakerWorkteamInputs, SageMakerWorkteamOutputs]
):
    INPUTS: SageMakerWorkteamInputs = SageMakerWorkteamInputs(
        team_name=InputValidator(
            input_type=str, required=True, description="The name of your work team."
        ),
        description=InputValidator(
            input_type=str, required=True, description="A description of the work team."
        ),
        user_pool=InputValidator(
            input_type=str,
            required=False,
            description="An identifier for a user pool. The user pool must be in the same region as the service that you are calling.",
        ),
        user_groups=InputValidator(
            input_type=str,
            required=False,
            description="A list of identifiers for user groups separated by commas.",
            default="",
        ),
        client_id=InputValidator(
            input_type=str,
            required=False,
            description="An identifier for an application client. You must create the app client ID using Amazon Cognito.",
        ),
        sns_topic=InputValidator(
            input_type=str,
            required=False,
            description="The ARN for the SNS topic to which notifications should be published.",
            default="",
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = SageMakerWorkteamOutputs(
        workteam_arn=OutputValidator(description="The ARN of the workteam."),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(arguments, SageMakerWorkteamInputs, SageMakerWorkteamOutputs)

    @property
    def inputs(self) -> SageMakerWorkteamInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerWorkteamOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerWorkteamOutputs:
        return self._output_paths

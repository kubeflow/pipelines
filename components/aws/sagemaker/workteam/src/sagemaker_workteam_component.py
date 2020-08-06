"""SageMaker component for workteam."""
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

from workteam.src.sagemaker_workteam_spec import (
    SageMakerWorkteamSpec,
    SageMakerWorkteamInputs,
    SageMakerWorkteamOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)


@ComponentMetadata(
    name="SageMaker - Private Workforce",
    description="Private workforce in SageMaker",
    spec=SageMakerWorkteamSpec,
)
class SageMakerWorkteamComponent(SageMakerComponent):
    """SageMaker component for workteam."""

    def Do(self, spec: SageMakerWorkteamSpec):
        self._workteam_name = spec.inputs.team_name
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        return SageMakerJobStatus(is_completed=True, raw_status="")

    def _after_job_complete(
        self,
        job: Dict,
        request: Dict,
        inputs: SageMakerWorkteamInputs,
        outputs: SageMakerWorkteamOutputs,
    ):
        outputs.workteam_arn = job["WorkteamArn"]

    def _create_job_request(
        self, inputs: SageMakerWorkteamInputs, outputs: SageMakerWorkteamOutputs,
    ) -> Dict:
        ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_workteam
        request = self._get_request_template("workteam")

        request["WorkteamName"] = self._workteam_name
        request["Description"] = inputs.description

        if inputs.sns_topic:
            request["NotificationConfiguration"][
                "NotificationTopicArn"
            ] = inputs.sns_topic
        else:
            request.pop("NotificationConfiguration")

        for group in [n.strip() for n in inputs.user_groups.split(",")]:
            request["MemberDefinitions"].append(
                {
                    "CognitoMemberDefinition": {
                        "UserPool": inputs.user_pool,
                        "UserGroup": group,
                        "ClientId": inputs.client_id,
                    }
                }
            )

        self._enable_tag_support(request, inputs)

        return request

    def _submit_job_request(self, request: Dict) -> object:
        return self._sm_client.create_workteam(**request)

    def _after_submit_job_request(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerWorkteamInputs,
        outputs: SageMakerWorkteamOutputs,
    ):
        portal = self._get_portal_domain()
        logging.info(f"Labeling portal: {portal}")

    def _get_portal_domain(self):
        """Gets the domain for the newly created workteam.

        Returns:
            str: The portal domain URL.
        """
        return self._sm_client.describe_workteam(WorkteamName=self._workteam_name)[
            "Workteam"
        ]["SubDomain"]


if __name__ == "__main__":
    import sys

    spec = SageMakerWorkteamSpec(sys.argv[1:])

    component = SageMakerWorkteamComponent()
    component.Do(spec)

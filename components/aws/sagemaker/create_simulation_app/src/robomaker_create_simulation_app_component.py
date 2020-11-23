"""RoboMaker component for creating a simulation application."""
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
from create_simulation_app.src.robomaker_create_simulation_app_spec import (
    RoboMakerCreateSimulationAppSpec,
    RoboMakerCreateSimulationAppInputs,
    RoboMakerCreateSimulationAppOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)
from common.boto3_manager import Boto3Manager
from common.common_inputs import SageMakerComponentCommonInputs


@ComponentMetadata(
    name="RoboMaker - Create Simulation Application",
    description="Creates a simulation application.",
    spec=RoboMakerCreateSimulationAppSpec,
)
class RoboMakerCreateSimulationAppComponent(SageMakerComponent):
    """RoboMaker component for creating a simulation application."""

    def Do(self, spec: RoboMakerCreateSimulationAppSpec):
        self._app_name = (
            spec.inputs.app_name
            if spec.inputs.app_name
            else RoboMakerCreateSimulationAppComponent._generate_unique_timestamped_id(
                prefix="SimulationApplication"
            )
        )
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        try:
            response = self._rm_client.describe_simulation_application(
                application=self._arn
            )
            status = response["arn"]

            if status is not None:
                return SageMakerJobStatus(is_completed=True, raw_status=status)
            else:
                return SageMakerJobStatus(
                    is_completed=True,
                    has_error=True,
                    error_message="No ARN present",
                    raw_status=status,
                )
        except Exception as ex:
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=str(ex),
                raw_status=str(ex),
            )

    def _configure_aws_clients(self, inputs: SageMakerComponentCommonInputs):
        """Configures the internal AWS clients for the component.

        Args:
            inputs: A populated list of user inputs.
        """
        self._rm_client = Boto3Manager.get_robomaker_client(
            self._get_component_version(),
            inputs.region,
            endpoint_url=inputs.endpoint_url,
            assume_role_arn=inputs.assume_role,
        )
        self._cw_client = Boto3Manager.get_cloudwatch_client(
            inputs.region, assume_role_arn=inputs.assume_role
        )

    def _after_job_complete(
        self,
        job: Dict,
        request: Dict,
        inputs: RoboMakerCreateSimulationAppInputs,
        outputs: RoboMakerCreateSimulationAppOutputs,
    ):
        outputs.app_name = self._app_name
        outputs.arn = job["arn"]
        outputs.version = job["version"]
        outputs.revision_id = job["revisionId"]
        logging.info(
            "Simulation Application in RoboMaker: https://{}.console.aws.amazon.com/robomaker/home?region={}#/simulationApplications/{}".format(
                inputs.region, inputs.region, str(outputs.arn).split("/", 1)[1]
            )
        )

    def _on_job_terminated(self):
        self._rm_client.delete_simulation_application(application=self._arn)

    def _create_job_request(
        self,
        inputs: RoboMakerCreateSimulationAppInputs,
        outputs: RoboMakerCreateSimulationAppOutputs,
    ) -> Dict:
        """
        Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/robomaker.html#RoboMaker.Client.create_simulation_application
        """
        request = self._get_request_template("robomaker.create.simulation.app")

        request["name"] = self._app_name
        request["sources"] = inputs.sources
        request["simulationSoftwareSuite"]["name"] = inputs.simulation_software_name
        request["simulationSoftwareSuite"][
            "version"
        ] = inputs.simulation_software_version
        request["robotSoftwareSuite"]["name"] = inputs.robot_software_name
        request["robotSoftwareSuite"]["version"] = inputs.robot_software_version

        if inputs.rendering_engine_name:
            request["renderingEngine"]["name"] = inputs.rendering_engine_name
            request["renderingEngine"]["version"] = inputs.rendering_engine_version
        else:
            request.pop("renderingEngine")

        return request

    def _submit_job_request(self, request: Dict) -> Dict:
        return self._rm_client.create_simulation_application(**request)

    def _after_submit_job_request(
        self,
        job: Dict,
        request: Dict,
        inputs: RoboMakerCreateSimulationAppInputs,
        outputs: RoboMakerCreateSimulationAppOutputs,
    ):
        outputs.arn = self._arn = job["arn"]
        logging.info(
            f"Created Robomaker Simulation Application with name: {self._app_name}"
        )

    def _print_logs_for_job(self):
        pass


if __name__ == "__main__":
    import sys

    spec = RoboMakerCreateSimulationAppSpec(sys.argv[1:])

    component = RoboMakerCreateSimulationAppComponent()
    component.Do(spec)

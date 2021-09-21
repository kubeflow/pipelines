"""RoboMaker component for deleting a simulation application."""
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
from delete_simulation_app.src.robomaker_delete_simulation_app_spec import (
    RoboMakerDeleteSimulationAppSpec,
    RoboMakerDeleteSimulationAppInputs,
    RoboMakerDeleteSimulationAppOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)
from common.boto3_manager import Boto3Manager
from common.common_inputs import SageMakerComponentCommonInputs


@ComponentMetadata(
    name="RoboMaker - Delete Simulation Application",
    description="Delete a simulation application.",
    spec=RoboMakerDeleteSimulationAppSpec,
)
class RoboMakerDeleteSimulationAppComponent(SageMakerComponent):
    """RoboMaker component for deleting a simulation application."""

    def Do(self, spec: RoboMakerDeleteSimulationAppSpec):
        self._arn = spec.inputs.arn
        self._version = spec.inputs.version
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        try:
            response = self._rm_client.describe_simulation_application(
                application=self._arn
            )
            status = response["arn"]

            if status is not None:
                return SageMakerJobStatus(is_completed=False, raw_status=status,)
            else:
                return SageMakerJobStatus(is_completed=True, raw_status="Item deleted")
        except Exception as ex:
            return SageMakerJobStatus(is_completed=True, raw_status=str(ex))

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
        inputs: RoboMakerDeleteSimulationAppInputs,
        outputs: RoboMakerDeleteSimulationAppOutputs,
    ):
        outputs.arn = self._arn
        logging.info("Simulation Application {} has been deleted".format(outputs.arn))

    def _on_job_terminated(self):
        logging.info("Simulation Application {} failed to delete".format(self._arn))

    def _create_job_request(
        self,
        inputs: RoboMakerDeleteSimulationAppInputs,
        outputs: RoboMakerDeleteSimulationAppOutputs,
    ) -> Dict:
        """
        Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/robomaker.html#RoboMaker.Client.delete_simulation_application
        """
        request = self._get_request_template("robomaker.delete.simulation.app")
        request["application"] = self._arn

        # If we have a version then use it, else remove it from request object
        if inputs.version:
            request["applicationVersion"] = inputs.version
        else:
            request.pop("applicationVersion")

        return request

    def _submit_job_request(self, request: Dict) -> Dict:
        return self._rm_client.delete_simulation_application(**request)

    def _after_submit_job_request(
        self,
        job: Dict,
        request: Dict,
        inputs: RoboMakerDeleteSimulationAppInputs,
        outputs: RoboMakerDeleteSimulationAppOutputs,
    ):
        logging.info(f"Deleted Robomaker Simulation Application with arn: {self._arn}")

    def _print_logs_for_job(self):
        pass


if __name__ == "__main__":
    import sys

    spec = RoboMakerDeleteSimulationAppSpec(sys.argv[1:])

    component = RoboMakerDeleteSimulationAppComponent()
    component.Do(spec)

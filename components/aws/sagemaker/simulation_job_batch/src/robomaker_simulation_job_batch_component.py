"""RoboMaker component for creating a simulation job batch."""
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
from simulation_job_batch.src.robomaker_simulation_job_batch_spec import (
    RoboMakerSimulationJobBatchSpec,
    RoboMakerSimulationJobBatchInputs,
    RoboMakerSimulationJobBatchOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)
from common.boto3_manager import Boto3Manager
from common.common_inputs import SageMakerComponentCommonInputs


@ComponentMetadata(
    name="RoboMaker - Create Simulation Job Batch",
    description="Creates a simulation job batch.",
    spec=RoboMakerSimulationJobBatchSpec,
)
class RoboMakerSimulationJobBatchComponent(SageMakerComponent):
    """RoboMaker component for creating a simulation job."""

    def Do(self, spec: RoboMakerSimulationJobBatchSpec):
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        batch_response = self._rm_client.describe_simulation_job_batch(batch=self._arn)
        batch_status = batch_response["status"]

        if batch_status in ["Completed"]:
            return SageMakerJobStatus(
                is_completed=True, has_error=False, raw_status=batch_status
            )

        if batch_status in ["TimedOut", "Canceled"]:
            simulation_message = "Simulation jobs are completed\n"
            has_error = False
            for completed_request in batch_response["createdRequests"]:
                self._sim_request_ids.add(completed_request["arn"].split("/")[-1])
                simulation_response = self._rm_client.describe_simulation_job(
                    job=completed_request["arn"]
                )
                if "failureCode" in simulation_response:
                    simulation_message += f"Simulation job: {simulation_response['arn']} failed with errorCode:{simulation_response['failureCode']}\n"
                    has_error = True
            return SageMakerJobStatus(
                is_completed=True,
                has_error=has_error,
                error_message=simulation_message,
                raw_status=batch_status,
            )

        if batch_status in ["Failed"]:
            failure_message = f"Simulation batch job is in status:{batch_status}\n"
            if "failureReason" in batch_response:
                failure_message += (
                    f"Simulation failed with reason:{batch_response['failureReason']}"
                )
            if "failureCode" in batch_response:
                failure_message += (
                    f"Simulation failed with errorCode:{batch_response['failureCode']}"
                )
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=failure_message,
                raw_status=batch_status,
            )

        return SageMakerJobStatus(is_completed=False, raw_status=batch_status)

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
        inputs: RoboMakerSimulationJobBatchInputs,
        outputs: RoboMakerSimulationJobBatchOutputs,
    ):
        for sim_request_id in self._sim_request_ids:
            logging.info(
                "Simulation Job in RoboMaker: https://{}.console.aws.amazon.com/robomaker/home?region={}#/simulationJobBatches/{}".format(
                    inputs.region, inputs.region, sim_request_id
                )
            )

    def _on_job_terminated(self):
        self._rm_client.cancel_simulation_job_batch(batch=self._arn)

    def _create_job_request(
        self,
        inputs: RoboMakerSimulationJobBatchInputs,
        outputs: RoboMakerSimulationJobBatchOutputs,
    ) -> Dict:
        """
        Documentation:https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/robomaker.html#RoboMaker.Client.start_simulation_job_batch
        """
        request = self._get_request_template("robomaker.simulation.job.batch")

        # Set batch policy inputs
        if inputs.timeout_in_secs:
            request["batchPolicy"]["timeoutInSeconds"] = inputs.timeout_in_secs
        if inputs.max_concurrency:
            request["batchPolicy"]["maxConcurrency"] = inputs.max_concurrency
        if not inputs.timeout_in_secs and not inputs.max_concurrency:
            request.pop("batchPolicy")

        # Set the simulation job inputs
        request["createSimulationJobRequests"] = inputs.simulation_job_requests

        # Override with ARN of sim application from input. Can be used to pass ARN from create sim app component.
        if inputs.sim_app_arn:
            for sim_job_request in request["createSimulationJobRequests"]:
                for sim_jobs in sim_job_request["simulationApplications"]:
                    sim_jobs["application"] = inputs.sim_app_arn

        return request

    def _submit_job_request(self, request: Dict) -> Dict:
        return self._rm_client.start_simulation_job_batch(**request)

    def _after_submit_job_request(
        self,
        job: Dict,
        request: Dict,
        inputs: RoboMakerSimulationJobBatchInputs,
        outputs: RoboMakerSimulationJobBatchOutputs,
    ):
        outputs.arn = self._arn = job["arn"]
        outputs.batch_job_id = self._batch_job_id = job["arn"].split("/")[-1]
        logging.info(
            f"Started Robomaker Simulation Job Batch with ID: {self._batch_job_id}"
        )
        logging.info(
            "Simulation Job Batch in RoboMaker: https://{}.console.aws.amazon.com/robomaker/home?region={}#/simulationJobBatches/{}".format(
                inputs.region, inputs.region, self._batch_job_id
            )
        )
        self._sim_request_ids = set()
        for created_request in job["createdRequests"]:
            self._sim_request_ids.add(created_request["arn"].split("/")[-1])
            logging.info(
                f"Started Robomaker Simulation Job with ID: {created_request['arn'].split('/')[-1]}"
            )

        # Inform if we have any pending or failed requests
        if job["pendingRequests"]:
            logging.info("Some Simulation Requests are in state Pending")

        if job["failedRequests"]:
            logging.info("Some Simulation Requests are in state Failed")

    def _print_logs_for_job(self):
        for sim_request_id in self._sim_request_ids:
            self._print_cloudwatch_logs("/aws/robomaker/SimulationJobs", sim_request_id)


if __name__ == "__main__":
    import sys

    spec = RoboMakerSimulationJobBatchSpec(sys.argv[1:])

    component = RoboMakerSimulationJobBatchComponent()
    component.Do(spec)

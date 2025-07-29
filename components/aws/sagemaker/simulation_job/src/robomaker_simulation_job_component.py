"""RoboMaker component for creating a simulation job."""
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
from simulation_job.src.robomaker_simulation_job_spec import (
    RoboMakerSimulationJobSpec,
    RoboMakerSimulationJobInputs,
    RoboMakerSimulationJobOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)
from common.boto3_manager import Boto3Manager
from common.common_inputs import SageMakerComponentCommonInputs


@ComponentMetadata(
    name="RoboMaker - Create Simulation Job",
    description="Creates a simulation job.",
    spec=RoboMakerSimulationJobSpec,
)
class RoboMakerSimulationJobComponent(SageMakerComponent):
    """RoboMaker component for creating a simulation job."""

    def Do(self, spec: RoboMakerSimulationJobSpec):
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        response = self._rm_client.describe_simulation_job(job=self._arn)
        status = response["status"]

        if status in ["Completed"]:
            return SageMakerJobStatus(
                is_completed=True, has_error=False, raw_status=status
            )

        if status in ["Terminating", "Terminated", "Canceled"]:
            if "failureCode" in response:
                simulation_message = (
                    f"Simulation failed with code:{response['failureCode']}"
                )
                return SageMakerJobStatus(
                    is_completed=True,
                    has_error=True,
                    error_message=simulation_message,
                    raw_status=status,
                )
            else:
                simulation_message = "Exited without error code.\n"
                if "failureReason" in response:
                    simulation_message += (
                        f"Simulation exited with reason:{response['failureReason']}\n"
                    )
                return SageMakerJobStatus(
                    is_completed=True,
                    has_error=False,
                    error_message=simulation_message,
                    raw_status=status,
                )

        if status in ["Failed", "RunningFailed"]:
            failure_message = f"Simulation job is in status:{status}\n"
            if "failureReason" in response:
                failure_message += (
                    f"Simulation failed with reason:{response['failureReason']}"
                )
            if "failureCode" in response:
                failure_message += (
                    f"Simulation failed with errorCode:{response['failureCode']}"
                )
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=failure_message,
                raw_status=status,
            )

        return SageMakerJobStatus(is_completed=False, raw_status=status)

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
        inputs: RoboMakerSimulationJobInputs,
        outputs: RoboMakerSimulationJobOutputs,
    ):
        outputs.output_artifacts = self._get_job_outputs()
        logging.info(
            "Simulation Job in RoboMaker: https://{}.console.aws.amazon.com/robomaker/home?region={}#/simulationJobs/{}".format(
                inputs.region, inputs.region, self._job_id
            )
        )

    def _on_job_terminated(self):
        self._rm_client.cancel_simulation_job(application=self._arn)

    def _create_job_request(
        self,
        inputs: RoboMakerSimulationJobInputs,
        outputs: RoboMakerSimulationJobOutputs,
    ) -> Dict:
        """
        Documentation:https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/robomaker.html#RoboMaker.Client.create_simulation_job
        """

        # Need one of sim_app_arn or robot_app_arn to be provided
        if not inputs.sim_app_arn and not inputs.robot_app_arn:
            logging.error("Must specify a Simulation App ARN or a Robot App ARN.")
            raise Exception("Could not create simulation job request")

        request = self._get_request_template("robomaker.simulation.job")

        # Set the required inputs
        request["outputLocation"]["s3Bucket"] = inputs.output_bucket
        request["outputLocation"]["s3Prefix"] = inputs.output_path
        request["maxJobDurationInSeconds"] = inputs.max_run
        request["iamRole"] = inputs.role

        # Set networking inputs
        if inputs.vpc_subnets:
            request["vpcConfig"]["subnets"] = inputs.vpc_subnets
            if inputs.vpc_security_group_ids:
                request["vpcConfig"]["securityGroups"] = inputs.vpc_security_group_ids
            if inputs.use_public_ip:
                request["vpcConfig"]["assignPublicIp"] = inputs.use_public_ip
        else:
            request.pop("vpcConfig")

        # Set simulation application inputs
        if inputs.sim_app_arn:
            if not inputs.sim_app_launch_config:
                logging.error("Must specify a Launch Config for your Simulation App")
                raise Exception("Could not create simulation job request")
            sim_app = {
                "application": inputs.sim_app_arn,
                "launchConfig": inputs.sim_app_launch_config,
            }
            if inputs.sim_app_version:
                sim_app["version"]: inputs.sim_app_version
            if inputs.sim_app_world_config:
                sim_app["worldConfigs"]: inputs.sim_app_world_config
            request["simulationApplications"].append(sim_app)
        else:
            request.pop("simulationApplications")

        # Set robot application inputs
        if inputs.robot_app_arn:
            if not inputs.robot_app_launch_config:
                logging.error("Must specify a Launch Config for your Robot App")
                raise Exception("Could not create simulation job request")
            robot_app = {
                "application": inputs.robot_app_arn,
                "launchConfig": inputs.robot_app_launch_config,
            }
            if inputs.robot_app_version:
                robot_app["version"]: inputs.robot_app_version
            request["robotApplications"].append(robot_app)
        else:
            request.pop("robotApplications")

        # Set optional inputs
        if inputs.record_ros_topics:
            request["loggingConfig"]["recordAllRosTopics"] = inputs.record_ros_topics
        else:
            request.pop("loggingConfig")

        if inputs.failure_behavior:
            request["failureBehavior"] = inputs.failure_behavior
        else:
            request.pop("failureBehavior")

        if inputs.data_sources:
            request["dataSources"] = inputs.data_sources
        else:
            request.pop("dataSources")

        if inputs.sim_unit_limit:
            request["compute"]["simulationUnitLimit"] = inputs.sim_unit_limit

        self._enable_tag_support(request, inputs)

        return request

    def _submit_job_request(self, request: Dict) -> Dict:
        return self._rm_client.create_simulation_job(**request)

    def _after_submit_job_request(
        self,
        job: Dict,
        request: Dict,
        inputs: RoboMakerSimulationJobInputs,
        outputs: RoboMakerSimulationJobOutputs,
    ):
        outputs.arn = self._arn = job["arn"]
        outputs.job_id = self._job_id = job["arn"].split("/")[-1]
        logging.info(f"Started Robomaker Simulation Job with ID: {self._job_id}")
        logging.info(
            "Simulation Job in RoboMaker: https://{}.console.aws.amazon.com/robomaker/home?region={}#/simulationJobs/{}".format(
                inputs.region, inputs.region, self._job_id
            )
        )

    def _print_logs_for_job(self):
        self._print_cloudwatch_logs("/aws/robomaker/SimulationJobs", self._job_id)

    def _get_job_outputs(self):
        """Map the S3 outputs of a simulation job to a dictionary object.

        Returns:
            dict: A dictionary of output S3 URIs.
        """
        response = self._rm_client.describe_simulation_job(job=self._arn)
        artifact_uri = f"s3://{response['outputLocation']['s3Bucket']}/{response['outputLocation']['s3Prefix']}"
        return artifact_uri


if __name__ == "__main__":
    import sys

    spec = RoboMakerSimulationJobSpec(sys.argv[1:])

    component = RoboMakerSimulationJobComponent()
    component.Do(spec)

"""SageMaker component for process."""
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

from process.src.sagemaker_process_spec import (
    SageMakerProcessSpec,
    SageMakerProcessInputs,
    SageMakerProcessOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)


@ComponentMetadata(
    name="SageMaker - Processing Job",
    description="Perform data pre-processing, post-processing, feature engineering, data validation, and model evaluation, and interpretation on using SageMaker",
    spec=SageMakerProcessSpec,
)
class SageMakerProcessComponent(SageMakerComponent):
    """SageMaker component for process."""

    def Do(self, spec: SageMakerProcessSpec):
        self._processing_job_name = (
            spec.inputs.job_name
            if spec.inputs.job_name
            else self._generate_unique_timestamped_id(prefix="ProcessingJob")
        )
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        response = self._sm_client.describe_processing_job(
            ProcessingJobName=self._processing_job_name
        )
        status = response["ProcessingJobStatus"]

        if status == "Completed":
            return SageMakerJobStatus(
                is_completed=True, has_error=False, raw_status=status
            )
        if status == "Failed":
            message = response["FailureReason"]
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=message,
                raw_status=status,
            )

        return SageMakerJobStatus(is_completed=False, raw_status=status)

    def _after_job_complete(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerProcessInputs,
        outputs: SageMakerProcessOutputs,
    ):
        outputs.job_name = self._processing_job_name
        outputs.output_artifacts = self._get_job_outputs()

    def _on_job_terminated(self):
        self._sm_client.stop_processing_job(ProcessingJobName=self._processing_job_name)

    def _create_job_request(
        self, inputs: SageMakerProcessInputs, outputs: SageMakerProcessOutputs,
    ) -> Dict:
        ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_processing_job
        request = self._get_request_template("process")

        request["ProcessingJobName"] = self._processing_job_name
        request["RoleArn"] = inputs.role

        ### Update processing container settings
        request["AppSpecification"]["ImageUri"] = inputs.image

        if inputs.container_entrypoint:
            request["AppSpecification"][
                "ContainerEntrypoint"
            ] = inputs.container_entrypoint
        else:
            request["AppSpecification"].pop("ContainerEntrypoint")
        if inputs.container_arguments:
            request["AppSpecification"][
                "ContainerArguments"
            ] = inputs.container_arguments
        else:
            request["AppSpecification"].pop("ContainerArguments")

        ### Update or pop VPC configs
        if inputs.vpc_security_group_ids and inputs.vpc_subnets:
            request["NetworkConfig"]["VpcConfig"][
                "SecurityGroupIds"
            ] = inputs.vpc_security_group_ids.split(",")
            request["NetworkConfig"]["VpcConfig"]["Subnets"] = inputs.vpc_subnets.split(
                ","
            )
        else:
            request["NetworkConfig"].pop("VpcConfig")
        request["NetworkConfig"]["EnableNetworkIsolation"] = inputs.network_isolation
        request["NetworkConfig"][
            "EnableInterContainerTrafficEncryption"
        ] = inputs.traffic_encryption

        ### Update input channels, not a required field
        if inputs.input_config:
            request["ProcessingInputs"] = inputs.input_config
        else:
            request.pop("ProcessingInputs")

        ### Update output channels, must have at least one specified
        if len(inputs.output_config) > 0:
            request["ProcessingOutputConfig"]["Outputs"] = inputs.output_config
        else:
            logging.error("Must specify at least one output channel.")
            raise Exception("Could not create job request")

        if inputs.output_encryption_key:
            request["ProcessingOutputConfig"]["KmsKeyId"] = inputs.output_encryption_key
        else:
            request["ProcessingOutputConfig"].pop("KmsKeyId")

        ### Update cluster config resources
        request["ProcessingResources"]["ClusterConfig"][
            "InstanceType"
        ] = inputs.instance_type
        request["ProcessingResources"]["ClusterConfig"][
            "InstanceCount"
        ] = inputs.instance_count
        request["ProcessingResources"]["ClusterConfig"][
            "VolumeSizeInGB"
        ] = inputs.volume_size

        if inputs.resource_encryption_key:
            request["ProcessingResources"]["ClusterConfig"][
                "VolumeKmsKeyId"
            ] = inputs.resource_encryption_key
        else:
            request["ProcessingResources"]["ClusterConfig"].pop("VolumeKmsKeyId")

        if inputs.max_run_time:
            request["StoppingCondition"]["MaxRuntimeInSeconds"] = inputs.max_run_time
        else:
            request["StoppingCondition"]["MaxRuntimeInSeconds"].pop("max_run_time")

        request["Environment"] = inputs.environment

        self._enable_tag_support(request, inputs)

        return request

    def _submit_job_request(self, request: Dict) -> object:
        return self._sm_client.create_processing_job(**request)

    def _after_submit_job_request(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerProcessInputs,
        outputs: SageMakerProcessOutputs,
    ):
        logging.info("Created Processing Job with name: " + self._processing_job_name)
        logging.info(
            "CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/ProcessingJobs;prefix={};streamFilter=typeLogStreamPrefix".format(
                inputs.region, inputs.region, self._processing_job_name
            )
        )

    def _get_job_outputs(self):
        """Map the S3 outputs of a processing job to a dictionary object.

        Returns:
            dict: A dictionary of output S3 URIs.
        """
        response = self._sm_client.describe_processing_job(
            ProcessingJobName=self._processing_job_name
        )
        outputs = {}
        for output in response["ProcessingOutputConfig"]["Outputs"]:
            outputs[output["OutputName"]] = output["S3Output"]["S3Uri"]

        return outputs


if __name__ == "__main__":
    import sys

    spec = SageMakerProcessSpec(sys.argv[1:])

    component = SageMakerProcessComponent()
    component.Do(spec)

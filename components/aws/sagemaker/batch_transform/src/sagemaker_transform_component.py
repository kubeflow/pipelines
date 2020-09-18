"""SageMaker component for transform."""
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

from batch_transform.src.sagemaker_transform_spec import (
    SageMakerTransformSpec,
    SageMakerTransformInputs,
    SageMakerTransformOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)


@ComponentMetadata(
    name="SageMaker - Batch Transformation",
    description="Batch Transformation Jobs in SageMaker",
    spec=SageMakerTransformSpec,
)
class SageMakerTransformComponent(SageMakerComponent):
    """SageMaker component for transform."""

    def Do(self, spec: SageMakerTransformSpec):
        self._transform_job_name = (
            spec.inputs.job_name
            if spec.inputs.job_name
            else SageMakerComponent._generate_unique_timestamped_id(
                prefix="BatchTransform"
            )
        )
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        response = self._sm_client.describe_transform_job(
            TransformJobName=self._transform_job_name
        )
        status = response["TransformJobStatus"]

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
        inputs: SageMakerTransformInputs,
        outputs: SageMakerTransformOutputs,
    ):
        outputs.output_location = inputs.output_location

    def _on_job_terminated(self):
        self._sm_client.stop_transform_job(TransformJobName=self._transform_job_name)

    def _create_job_request(
        self, inputs: SageMakerTransformInputs, outputs: SageMakerTransformOutputs,
    ) -> Dict:
        ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_transform_job
        request = self._get_request_template("transform")

        request["TransformJobName"] = self._transform_job_name
        request["ModelName"] = inputs.model_name

        if inputs.max_concurrent:
            request["MaxConcurrentTransforms"] = inputs.max_concurrent

        if inputs.max_payload or inputs.max_payload == 0:
            request["MaxPayloadInMB"] = inputs.max_payload

        if inputs.batch_strategy:
            request["BatchStrategy"] = inputs.batch_strategy
        else:
            request.pop("BatchStrategy")

        if inputs.environment:
            request["Environment"] = inputs.environment

        if inputs.data_type:
            request["TransformInput"]["DataSource"]["S3DataSource"][
                "S3DataType"
            ] = inputs.data_type

        request["TransformInput"]["DataSource"]["S3DataSource"][
            "S3Uri"
        ] = inputs.input_location
        request["TransformInput"]["ContentType"] = inputs.content_type

        if inputs.compression_type:
            request["TransformInput"]["CompressionType"] = inputs.compression_type

        if inputs.split_type:
            request["TransformInput"]["SplitType"] = inputs.split_type

        request["TransformOutput"]["S3OutputPath"] = inputs.output_location
        request["TransformOutput"]["Accept"] = inputs.accept
        request["TransformOutput"]["KmsKeyId"] = inputs.output_encryption_key

        if inputs.assemble_with:
            request["TransformOutput"]["AssembleWith"] = inputs.assemble_with
        else:
            request["TransformOutput"].pop("AssembleWith")

        request["TransformResources"]["InstanceType"] = inputs.instance_type
        request["TransformResources"]["InstanceCount"] = inputs.instance_count
        request["TransformResources"]["VolumeKmsKeyId"] = inputs.resource_encryption_key
        request["DataProcessing"]["InputFilter"] = inputs.input_filter
        request["DataProcessing"]["OutputFilter"] = inputs.output_filter

        if inputs.join_source:
            request["DataProcessing"]["JoinSource"] = inputs.join_source

        self._enable_tag_support(request, inputs)

        return request

    def _submit_job_request(self, request: Dict) -> object:
        return self._sm_client.create_transform_job(**request)

    def _after_submit_job_request(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerTransformInputs,
        outputs: SageMakerTransformOutputs,
    ):
        logging.info(f"Created Transform Job with name: {self._transform_job_name}")
        logging.info(
            "Transform job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/jobs/{}".format(
                inputs.region, inputs.region, self._transform_job_name,
            )
        )
        logging.info(
            "CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/TransformJobs;prefix={};streamFilter=typeLogStreamPrefix".format(
                inputs.region, inputs.region, self._transform_job_name,
            )
        )


if __name__ == "__main__":
    import sys

    spec = SageMakerTransformSpec(sys.argv[1:])

    component = SageMakerTransformComponent()
    component.Do(spec)

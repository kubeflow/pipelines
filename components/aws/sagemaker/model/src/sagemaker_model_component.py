"""SageMaker component for create model."""
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

from model.src.sagemaker_model_spec import (
    SageMakerCreateModelSpec,
    SageMakerCreateModelInputs,
    SageMakerCreateModelOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)


@ComponentMetadata(
    name="SageMaker - Create Model",
    description="Create Models in SageMaker",
    spec=SageMakerCreateModelSpec,
)
class SageMakerCreateModelComponent(SageMakerComponent):
    """SageMaker component for create model."""

    def Do(self, spec: SageMakerCreateModelSpec):
        self._model_name = spec.inputs.model_name
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        return SageMakerJobStatus(is_completed=True, raw_status="")

    def _after_job_complete(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerCreateModelInputs,
        outputs: SageMakerCreateModelOutputs,
    ):
        outputs.model_name = self._model_name

    def _create_job_request(
        self, inputs: SageMakerCreateModelInputs, outputs: SageMakerCreateModelOutputs,
    ) -> Dict:
        ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_model
        request = self._get_request_template("model")

        request["ModelName"] = self._model_name
        request["PrimaryContainer"]["Environment"] = inputs.environment

        if inputs.secondary_containers:
            request["Containers"] = inputs.secondary_containers
            request.pop("PrimaryContainer")
        else:
            request.pop("Containers")
            ### Update primary container and handle input errors
            if inputs.container_host_name:
                request["PrimaryContainer"][
                    "ContainerHostname"
                ] = inputs.container_host_name
            else:
                request["PrimaryContainer"].pop("ContainerHostname")

            if inputs.model_package:
                request["PrimaryContainer"]["ModelPackageName"] = inputs.model_package
                request["PrimaryContainer"].pop("Image")
                request["PrimaryContainer"].pop("ModelDataUrl")
            elif inputs.image and inputs.model_artifact_url:
                request["PrimaryContainer"]["Image"] = inputs.image
                request["PrimaryContainer"]["ModelDataUrl"] = inputs.model_artifact_url
                request["PrimaryContainer"].pop("ModelPackageName")
            else:
                logging.error(
                    "Please specify an image AND model artifact url, OR a model package name."
                )
                raise Exception("Could not make create model request.")

        request["ExecutionRoleArn"] = inputs.role
        request["EnableNetworkIsolation"] = inputs.network_isolation

        ### Update or pop VPC configs
        if inputs.vpc_security_group_ids and inputs.vpc_subnets:
            request["VpcConfig"][
                "SecurityGroupIds"
            ] = inputs.vpc_security_group_ids.split(",")
            request["VpcConfig"]["Subnets"] = inputs.vpc_subnets.split(",")
        else:
            request.pop("VpcConfig")

        self._enable_tag_support(request, inputs)

        return request

    def _submit_job_request(self, request: Dict) -> object:
        return self._sm_client.create_model(**request)

    def _after_submit_job_request(
        self,
        job: Dict,
        request: Dict,
        inputs: SageMakerCreateModelInputs,
        outputs: SageMakerCreateModelOutputs,
    ):
        logging.info(f"Model Config Arn: {job['ModelArn']}")


if __name__ == "__main__":
    import sys

    spec = SageMakerCreateModelSpec(sys.argv[1:])

    component = SageMakerCreateModelComponent()
    component.Do(spec)

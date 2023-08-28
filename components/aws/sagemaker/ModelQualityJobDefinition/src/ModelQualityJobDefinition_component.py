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
import json

from ModelQualityJobDefinition.src.ModelQualityJobDefinition_spec import (
    SageMakerModelQualityJobDefinitionInputs,
    SageMakerModelQualityJobDefinitionOutputs,
    SageMakerModelQualityJobDefinitionSpec,
)
from commonv2.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)
from commonv2 import snake_to_camel


@ComponentMetadata(
    name="SageMaker - ModelQualityJobDefinition",
    description="",
    spec=SageMakerModelQualityJobDefinitionSpec,
)
class SageMakerModelQualityJobDefinitionComponent(SageMakerComponent):

    """SageMaker component for ModelQualityJobDefinition."""

    def Do(self, spec: SageMakerModelQualityJobDefinitionSpec):

        self.namespace = self._get_current_namespace()
        logging.info("Current namespace: " + self.namespace)

        ############GENERATED SECTION BELOW############

        self.job_name = spec.inputs.job_definition_name = (
            spec.inputs.job_definition_name
            if spec.inputs.job_definition_name
            else SageMakerComponent._generate_unique_timestamped_id(
                prefix="model-quality-job-definition"
            )
        )

        self.group = "sagemaker.services.k8s.aws"
        self.version = "v1alpha1"
        self.plural = "modelqualityjobdefinitions"
        self.spaced_out_resource_name = "Model Quality Job Definition"

        self.job_request_outline_location = (
            "ModelQualityJobDefinition/src/ModelQualityJobDefinition_request.yaml.tpl"
        )
        self.job_request_location = (
            "ModelQualityJobDefinition/src/ModelQualityJobDefinition_request.yaml"
        )
        self.update_supported = False
        ############GENERATED SECTION ABOVE############

        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _create_job_request(
        self,
        inputs: SageMakerModelQualityJobDefinitionInputs,
        outputs: SageMakerModelQualityJobDefinitionOutputs,
    ) -> Dict:

        return super()._create_job_yaml(inputs, outputs)

    def _submit_job_request(self, request: Dict) -> object:

        return super()._create_resource(request, 12, 15)

    def _on_job_terminated(self):
        super()._delete_custom_resource()

    def _after_submit_job_request(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerModelQualityJobDefinitionInputs,
        outputs: SageMakerModelQualityJobDefinitionOutputs,
    ):
        pass

    def _get_job_status(self):
        return SageMakerJobStatus(is_completed=True, raw_status="Completed")

    def _get_upgrade_status(self):

        return self._get_job_status()

    def _after_job_complete(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerModelQualityJobDefinitionInputs,
        outputs: SageMakerModelQualityJobDefinitionOutputs,
    ):
        # prepare component outputs (defined in the spec)

        ack_statuses = super()._get_resource()["status"]

        ############GENERATED SECTION BELOW############

        outputs.ack_resource_metadata = str(
            ack_statuses["ackResourceMetadata"]
            if "ackResourceMetadata" in ack_statuses
            else None
        )
        outputs.conditions = str(
            ack_statuses["conditions"] if "conditions" in ack_statuses else None
        )
        outputs.sagemaker_resource_name = self.job_name

        ############GENERATED SECTION ABOVE############


if __name__ == "__main__":
    import sys

    spec = SageMakerModelQualityJobDefinitionSpec(sys.argv[1:])

    component = SageMakerModelQualityJobDefinitionComponent()
    component.Do(spec)

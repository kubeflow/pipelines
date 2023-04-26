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

from ModelBiasJobDefinition.src.ModelBiasJobDefinition_spec import (
    SageMakerModelBiasJobDefinitionInputs,
    SageMakerModelBiasJobDefinitionOutputs,
    SageMakerModelBiasJobDefinitionSpec,
)
from commonv2.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)
from commonv2 import snake_to_camel


@ComponentMetadata(
    name="SageMaker - ModelBiasJobDefinition",
    description="",
    spec=SageMakerModelBiasJobDefinitionSpec,
)
class SageMakerModelBiasJobDefinitionComponent(SageMakerComponent):

    """SageMaker component for ModelBiasJobDefinition."""

    def Do(self, spec: SageMakerModelBiasJobDefinitionSpec):

        self.namespace = self._get_current_namespace()
        logging.info("Current namespace: " + self.namespace)

        ############GENERATED SECTION BELOW############

        self.job_name = spec.inputs.model_bias_job_definition_name = (
            spec.inputs.job_definition_name
            if spec.inputs.job_definition_name
            else SageMakerComponent._generate_unique_timestamped_id(
                prefix="model-bias-job-definition"
            )
        )

        self.group = "sagemaker.services.k8s.aws"
        self.version = "v1alpha1"
        self.plural = "modelbiasjobdefinitions"

        self.job_request_outline_location = (
            "ModelBiasJobDefinition/src/ModelBiasJobDefinition_request.yaml.tpl"
        )
        self.job_request_location = (
            "ModelBiasJobDefinition/src/ModelBiasJobDefinition_request.yaml"
        )
        ############GENERATED SECTION ABOVE############

        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _create_job_request(
        self,
        inputs: SageMakerModelBiasJobDefinitionInputs,
        outputs: SageMakerModelBiasJobDefinitionOutputs,
    ) -> Dict:

        return super()._create_job_yaml(inputs, outputs)

    def _submit_job_request(self, request: Dict) -> object:

        if self.resource_upgrade:
            self.initial_resouce_condition_times = self._get_condition_times()
            return super()._patch_custom_resource(request)
        else:
            return super()._create_resource(request, 6, 10)

    def _on_job_terminated(self):
        super()._delete_custom_resource()

    def _after_submit_job_request(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerModelBiasJobDefinitionInputs,
        outputs: SageMakerModelBiasJobDefinitionOutputs,
    ):
        pass

    def _get_job_status(self):
        return SageMakerJobStatus(is_completed=True, raw_status="Completed")

    def _get_upgrade_status(self):

        job_status = self._get_job_status()
        # Needed because Requeue errors are not counted in _check_resource_conditions.
        recoverable_conditions = self._get_conditions_of_type("ACK.Recoverable")
        if len(recoverable_conditions) == 0:
            return job_status
        else:
            sm_job_status = job_status.raw_status
        return SageMakerJobStatus(
            is_completed=False, has_error=False, raw_status=sm_job_status
        )

    def _after_job_complete(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerModelBiasJobDefinitionInputs,
        outputs: SageMakerModelBiasJobDefinitionOutputs,
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
        ############GENERATED SECTION ABOVE############


if __name__ == "__main__":
    import sys

    spec = SageMakerModelBiasJobDefinitionSpec(sys.argv[1:])

    component = SageMakerModelBiasJobDefinitionComponent()
    component.Do(spec)

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

from TrainingJob.src.TrainingJob_spec import (
    SageMakerTrainingJobInputs,
    SageMakerTrainingJobOutputs,
    SageMakerTrainingJobSpec,
)
from commonv2.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)
from commonv2 import snake_to_camel


@ComponentMetadata(
    name="SageMaker - TrainingJob",
    description="",
    spec=SageMakerTrainingJobSpec,
)
class SageMakerTrainingJobComponent(SageMakerComponent):

    """SageMaker component for TrainingJob."""

    def Do(self, spec: SageMakerTrainingJobSpec):

        self.namespace = self._get_current_namespace()
        logging.info("Current namespace: " + self.namespace)

        ############GENERATED SECTION BELOW############

        self.job_name = spec.inputs.training_job_name = (
            spec.inputs.training_job_name
            if spec.inputs.training_job_name
            else SageMakerComponent._generate_unique_timestamped_id(
                prefix="training-job"
            )
        )

        self.group = "sagemaker.services.k8s.aws"
        self.version = "v1alpha1"
        self.plural = "trainingjobs"
        self.spaced_out_resource_name = "Training Job"

        self.job_request_outline_location = (
            "TrainingJob/src/TrainingJob_request.yaml.tpl"
        )
        self.job_request_location = "TrainingJob/src/TrainingJob_request.yaml"
        self.update_supported = False
        ############GENERATED SECTION ABOVE############

        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _check_debugger_rule_error(self, debugger_statuses: list):
        for debug_rule in debugger_statuses:
            if debug_rule["ruleEvaluationStatus"] in ["Error", "IssuesFound"]:
                return True
        return False

    def _print_rule_statuses(self, debugger_statuses: list, rule_type: str):
        logging.info(
            f"Status of {rule_type}Rules:\n {json.dumps(debugger_statuses, indent=2)}"
        )

    def _create_job_request(
        self,
        inputs: SageMakerTrainingJobInputs,
        outputs: SageMakerTrainingJobOutputs,
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
        inputs: SageMakerTrainingJobInputs,
        outputs: SageMakerTrainingJobOutputs,
    ):
        logging.info(
            f"Created Sagemaker Training Job with name: %s",
            request["spec"]["trainingJobName"],
        )
        logging.info(
            "Training job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/jobs/{}".format(
                inputs.region,
                inputs.region,
                self.job_name,
            )
        )
        logging.info(
            "CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/TrainingJobs;prefix={};streamFilter=typeLogStreamPrefix".format(
                inputs.region,
                inputs.region,
                self.job_name,
            )
        )

    def _get_job_status(self):

        ack_statuses = super()._get_resource()["status"]
        sm_job_status = ack_statuses["trainingJobStatus"]  # todo: developer customize

        if "debugRuleEvaluationStatuses" in ack_statuses:
            self._print_rule_statuses(
                ack_statuses["debugRuleEvaluationStatuses"], "Debug"
            )
        if "profilerRuleEvaluationStatuses" in ack_statuses:
            self._print_rule_statuses(
                ack_statuses["profilerRuleEvaluationStatuses"], "Profile"
            )

        if sm_job_status == "Completed":
            resourceSynced = self._get_resource_synced_status(ack_statuses)
            if not resourceSynced:
                return SageMakerJobStatus(
                    is_completed=False,
                    raw_status=sm_job_status,
                )
            # Debugger/Profiler is either Complete or has Failed
            if (
                "debugRuleEvaluationStatuses" not in ack_statuses
                and "profilerRuleEvaluationStatuses" not in ack_statuses
            ):
                return SageMakerJobStatus(
                    is_completed=True, has_error=False, raw_status="Completed"
                )
            else:
                if "debugRuleEvaluationStatuses" in ack_statuses:
                    debugger_statuses = ack_statuses["debugRuleEvaluationStatuses"]
                    if self._check_debugger_rule_error(debugger_statuses):
                        return SageMakerJobStatus(
                            is_completed=True, has_error=True, raw_status=sm_job_status
                        )
                if "profilerRuleEvaluationStatuses" in ack_statuses:
                    profiler_statuses = ack_statuses["profilerRuleEvaluationStatuses"]
                    if self._check_debugger_rule_error(profiler_statuses):
                        return SageMakerJobStatus(
                            is_completed=True, has_error=True, raw_status=sm_job_status
                        )
                # Profiler/Debugger cannot be in InProgress state if the resource synced condition is set to true
                return SageMakerJobStatus(
                    is_completed=True, has_error=False, raw_status=sm_job_status
                )
        if sm_job_status == "Failed":
            message = ack_statuses["failureReason"]
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=message,
                raw_status=sm_job_status,
            )
        if sm_job_status == "Stopped":
            message = "Sagemaker job was stopped"
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=message,
                raw_status=sm_job_status,
            )
        return SageMakerJobStatus(is_completed=False, raw_status=sm_job_status)

    def _get_upgrade_status(self):

        return self._get_job_status()

    def _after_job_complete(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerTrainingJobInputs,
        outputs: SageMakerTrainingJobOutputs,
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
        outputs.creation_time = str(
            ack_statuses["creationTime"] if "creationTime" in ack_statuses else None
        )
        outputs.debug_rule_evaluation_statuses = str(
            ack_statuses["debugRuleEvaluationStatuses"]
            if "debugRuleEvaluationStatuses" in ack_statuses
            else None
        )
        outputs.failure_reason = str(
            ack_statuses["failureReason"] if "failureReason" in ack_statuses else None
        )
        outputs.last_modified_time = str(
            ack_statuses["lastModifiedTime"]
            if "lastModifiedTime" in ack_statuses
            else None
        )
        outputs.model_artifacts = str(
            ack_statuses["modelArtifacts"] if "modelArtifacts" in ack_statuses else None
        )
        outputs.profiler_rule_evaluation_statuses = str(
            ack_statuses["profilerRuleEvaluationStatuses"]
            if "profilerRuleEvaluationStatuses" in ack_statuses
            else None
        )
        outputs.profiling_status = str(
            ack_statuses["profilingStatus"]
            if "profilingStatus" in ack_statuses
            else None
        )
        outputs.secondary_status = str(
            ack_statuses["secondaryStatus"]
            if "secondaryStatus" in ack_statuses
            else None
        )
        outputs.training_job_status = str(
            ack_statuses["trainingJobStatus"]
            if "trainingJobStatus" in ack_statuses
            else None
        )
        outputs.warm_pool_status = str(
            ack_statuses["warmPoolStatus"] if "warmPoolStatus" in ack_statuses else None
        )
        ############GENERATED SECTION ABOVE############


if __name__ == "__main__":
    import sys

    spec = SageMakerTrainingJobSpec(sys.argv[1:])

    component = SageMakerTrainingJobComponent()
    component.Do(spec)

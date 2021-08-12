"""SageMaker component for RLEstimator."""
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
import os
from sagemaker.rl import RLEstimator, RLToolkit, RLFramework
from rlestimator.src.sagemaker_rlestimator_spec import (
    SageMakerRLEstimatorSpec,
    SageMakerRLEstimatorInputs,
    SageMakerRLEstimatorOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
    DebugRulesStatus,
)
from common.boto3_manager import Boto3Manager
from common.common_inputs import SageMakerComponentCommonInputs
from common.spec_input_parsers import SpecInputParsers


@ComponentMetadata(
    name="SageMaker - RLEstimator Training Job",
    description="Handle end-to-end training and deployment of custom RLEstimator code.",
    spec=SageMakerRLEstimatorSpec,
)
class SageMakerRLEstimatorComponent(SageMakerComponent):
    """SageMaker component for RLEstimator."""

    def Do(self, spec: SageMakerRLEstimatorSpec):
        self._rlestimator_job_name = (
            spec.inputs.job_name
            if spec.inputs.job_name
            else SageMakerComponent._generate_unique_timestamped_id(
                prefix="RLEstimatorJob"
            )
        )
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _configure_aws_clients(self, inputs: SageMakerComponentCommonInputs):
        """Configures the internal AWS clients for the component.

        Args:
            inputs: A populated list of user inputs.
        """
        self._sm_client = Boto3Manager.get_sagemaker_client(
            self._get_component_version(),
            inputs.region,
            endpoint_url=inputs.endpoint_url,
            assume_role_arn=inputs.assume_role,
        )
        self._cw_client = Boto3Manager.get_cloudwatch_client(
            inputs.region, assume_role_arn=inputs.assume_role
        )
        self._sagemaker_session = Boto3Manager.get_sagemaker_session(
            self._get_component_version(),
            inputs.region,
            assume_role_arn=inputs.assume_role,
        )

    def _get_job_status(self) -> SageMakerJobStatus:
        response = self._sm_client.describe_training_job(
            TrainingJobName=self._rlestimator_job_name
        )
        status = response["TrainingJobStatus"]

        if status == "Completed" or status == "Stopped":
            return self._get_debug_rule_status()
        if status == "Failed":
            message = response["FailureReason"]
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=message,
                raw_status=status,
            )

        return SageMakerJobStatus(is_completed=False, raw_status=status)

    def _get_debug_rule_status(self) -> SageMakerJobStatus:
        """Get the job status of the training debugging rules.

        Returns:
            SageMakerJobStatus: A status object.
        """
        response = self._sm_client.describe_training_job(
            TrainingJobName=self._rlestimator_job_name
        )

        # No debugging configured
        if "DebugRuleEvaluationStatuses" not in response:
            return SageMakerJobStatus(is_completed=True, has_error=False, raw_status="")

        raw_status = DebugRulesStatus.from_describe(response)
        if raw_status != DebugRulesStatus.INPROGRESS:
            logging.info("Rules have ended with status:\n")
            self._print_debug_rule_status(response, True)
            return SageMakerJobStatus(
                is_completed=True,
                has_error=(raw_status == DebugRulesStatus.ERRORED),
                raw_status=raw_status,
            )

        self._print_debug_rule_status(response)
        return SageMakerJobStatus(is_completed=False, raw_status=raw_status)

    def _print_debug_rule_status(self, response, last_print=False):
        """Prints the status of each debug rule.

            Example of DebugRuleEvaluationStatuses:
            response['DebugRuleEvaluationStatuses'] =
                [{
                    "RuleConfigurationName": "VanishingGradient",
                    "RuleEvaluationStatus": "IssuesFound",
                    "StatusDetails": "There was an issue."
                }]
            If last_print is False:
            INFO:root: - LossNotDecreasing: InProgress
            INFO:root: - Overtraining: NoIssuesFound
            ERROR:root:- CustomGradientRule: Error
            If last_print is True:
            INFO:root: - LossNotDecreasing: IssuesFound
            INFO:root:   - RuleEvaluationConditionMet: Evaluation of the rule LossNotDecreasing at step 10 resulted in the condition being met

        Args:
            response: A describe training job response.
            last_print: If true, prints each of the debug rule issues if found.
        """
        for debug_rule in response["DebugRuleEvaluationStatuses"]:
            line_ending = "\n" if last_print else ""
            if "StatusDetails" in debug_rule:
                status_details = (
                    f"- {debug_rule['StatusDetails'].rstrip()}{line_ending}"
                )
                line_ending = ""
            else:
                status_details = ""
            rule_status = f"- {debug_rule['RuleConfigurationName']}: {debug_rule['RuleEvaluationStatus']}{line_ending}"
            if debug_rule["RuleEvaluationStatus"] == "Error":
                log_fn = logging.error
                status_padding = 1
            else:
                log_fn = logging.info
                status_padding = 2

            log_fn(f"{status_padding * ' '}{rule_status}")
            if last_print and status_details:
                log_fn(f"{(status_padding + 2) * ' '}{status_details}")
        self._print_log_header(50)

    def _after_job_complete(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerRLEstimatorInputs,
        outputs: SageMakerRLEstimatorOutputs,
    ):
        outputs.job_name = self._rlestimator_job_name
        outputs.model_artifact_url = self._get_model_artifacts_from_job(
            self._rlestimator_job_name
        )
        outputs.training_image = self._get_image_from_job(self._rlestimator_job_name)

    def _on_job_terminated(self):
        self._sm_client.stop_training_job(TrainingJobName=self._rlestimator_job_name)

    def _print_logs_for_job(self):
        self._print_cloudwatch_logs(
            "/aws/sagemaker/TrainingJobs", self._rlestimator_job_name
        )

    def _create_job_request(
        self, inputs: SageMakerRLEstimatorInputs, outputs: SageMakerRLEstimatorOutputs,
    ) -> RLEstimator:
        # Documentation: https://sagemaker.readthedocs.io/en/stable/frameworks/rl/sagemaker.rl.html
        # We need to configure region and it is not something we can do via the RLEstimator class.

        # Only use max wait time default value if electing to use spot instances
        if not inputs.spot_instance:
            max_wait_time = None
        else:
            max_wait_time = inputs.max_wait_time

        estimator = RLEstimator(
            entry_point=inputs.entry_point,
            source_dir=inputs.source_dir,
            image_uri=inputs.image,
            toolkit=self._get_toolkit(inputs.toolkit),
            toolkit_version=inputs.toolkit_version,
            framework=self._get_framework(inputs.framework),
            role=inputs.role,
            debugger_hook_config=self._nullable(inputs.debug_hook_config),
            rules=self._nullable(inputs.debug_rule_config),
            instance_type=inputs.instance_type,
            instance_count=inputs.instance_count,
            output_path=inputs.model_artifact_path,
            metric_definitions=inputs.metric_definitions,
            input_mode=inputs.training_input_mode,
            max_run=inputs.max_run,
            hyperparameters=self._validate_hyperparameters(inputs.hyperparameters),
            subnets=self._nullable(inputs.vpc_subnets),
            security_group_ids=self._nullable(inputs.vpc_security_group_ids),
            use_spot_instances=inputs.spot_instance,
            enable_network_isolation=inputs.network_isolation,
            encrypt_inter_container_traffic=inputs.traffic_encryption,
            max_wait=max_wait_time,
            sagemaker_session=self._sagemaker_session,
        )

        return estimator

    def _submit_job_request(self, estimator: RLEstimator) -> object:
        # By setting wait to false we don't block the current thread.
        estimator.fit(job_name=self._rlestimator_job_name, wait=False)
        job_name = estimator.latest_training_job.job_name
        self._rlestimator_job_name = job_name
        response = self._sm_client.describe_training_job(TrainingJobName=job_name)
        return response

    def _after_submit_job_request(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerRLEstimatorInputs,
        outputs: SageMakerRLEstimatorOutputs,
    ):
        logging.info(f"Created Training Job with name: {self._rlestimator_job_name}")
        logging.info(
            "Training job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/jobs/{}".format(
                inputs.region, inputs.region, self._rlestimator_job_name,
            )
        )
        logging.info(
            "CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/TrainingJobs;prefix={};streamFilter=typeLogStreamPrefix".format(
                inputs.region, inputs.region, self._rlestimator_job_name,
            )
        )

    @staticmethod
    def _get_toolkit(toolkit_type: str) -> RLToolkit:
        if toolkit_type == "":
            return None
        return RLToolkit[toolkit_type.upper()]

    @staticmethod
    def _get_framework(framework_type: str) -> RLFramework:
        if framework_type == "":
            return None
        return RLFramework[framework_type.upper()]

    @staticmethod
    def _nullable(value: str):
        if value:
            return value
        else:
            return None


if __name__ == "__main__":
    import sys

    spec = SageMakerRLEstimatorSpec(sys.argv[1:])

    component = SageMakerRLEstimatorComponent()
    component.Do(spec)

"""SageMaker component for training."""
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
from enum import Enum, auto
from sagemaker.image_uris import retrieve

from train.src.built_in_algos import BUILT_IN_ALGOS
from train.src.sagemaker_training_spec import (
    SageMakerTrainingSpec,
    SageMakerTrainingInputs,
    SageMakerTrainingOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
    DebugRulesStatus,
)


@ComponentMetadata(
    name="SageMaker - Training Job",
    description="Train Machine Learning and Deep Learning Models using SageMaker",
    spec=SageMakerTrainingSpec,
)
class SageMakerTrainingComponent(SageMakerComponent):
    """SageMaker component for training."""

    def Do(self, spec: SageMakerTrainingSpec):
        self._training_job_name = (
            spec.inputs.job_name
            if spec.inputs.job_name
            else SageMakerComponent._generate_unique_timestamped_id(
                prefix="TrainingJob"
            )
        )
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        response = self._sm_client.describe_training_job(
            TrainingJobName=self._training_job_name
        )
        status = response["TrainingJobStatus"]

        if status == "Completed":
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
            TrainingJobName=self._training_job_name
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
        inputs: SageMakerTrainingInputs,
        outputs: SageMakerTrainingOutputs,
    ):
        outputs.job_name = self._training_job_name
        outputs.model_artifact_url = self._get_model_artifacts_from_job(
            self._training_job_name
        )
        outputs.training_image = self._get_image_from_job(self._training_job_name)

    def _on_job_terminated(self):
        self._sm_client.stop_training_job(TrainingJobName=self._training_job_name)

    def _print_logs_for_job(self):
        self._print_cloudwatch_logs(
            "/aws/sagemaker/TrainingJobs", self._training_job_name
        )

    def _create_job_request(
        self, inputs: SageMakerTrainingInputs, outputs: SageMakerTrainingOutputs,
    ) -> Dict:
        ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_training_job
        request = self._get_request_template("train")

        request["TrainingJobName"] = self._training_job_name
        request["RoleArn"] = inputs.role
        request["HyperParameters"] = self._validate_hyperparameters(
            inputs.hyperparameters
        )
        request["AlgorithmSpecification"][
            "TrainingInputMode"
        ] = inputs.training_input_mode

        ### Update training image (for BYOC and built-in algorithms) or algorithm resource name
        if not inputs.image and not inputs.algorithm_name:
            logging.error("Please specify training image or algorithm name.")
            raise Exception("Could not create job request")
        if inputs.image and inputs.algorithm_name:
            logging.error(
                "Both image and algorithm name inputted, only one should be specified. Proceeding with image."
            )

        if inputs.image:
            request["AlgorithmSpecification"]["TrainingImage"] = inputs.image
            request["AlgorithmSpecification"].pop("AlgorithmName")
        else:
            # TODO: Adjust this implementation to account for custom algorithm resources names that are the same as built-in algorithm names
            algo_name = inputs.algorithm_name.lower().strip()
            if algo_name in BUILT_IN_ALGOS.keys():
                request["AlgorithmSpecification"]["TrainingImage"] = retrieve(
                    BUILT_IN_ALGOS[algo_name], inputs.region
                )
                request["AlgorithmSpecification"].pop("AlgorithmName")
                logging.warning(
                    "Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm."
                )
            # Just to give the user more leeway for built-in algorithm name inputs
            elif algo_name in BUILT_IN_ALGOS.values():
                request["AlgorithmSpecification"]["TrainingImage"] = retrieve(
                    algo_name, inputs.region
                )
                request["AlgorithmSpecification"].pop("AlgorithmName")
                logging.warning(
                    "Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm."
                )
            else:
                request["AlgorithmSpecification"][
                    "AlgorithmName"
                ] = inputs.algorithm_name
                request["AlgorithmSpecification"].pop("TrainingImage")

        ### Update metric definitions
        if inputs.metric_definitions:
            for key, val in inputs.metric_definitions.items():
                request["AlgorithmSpecification"]["MetricDefinitions"].append(
                    {"Name": key, "Regex": val}
                )
        else:
            request["AlgorithmSpecification"].pop("MetricDefinitions")

        ### Update or pop VPC configs
        if inputs.vpc_security_group_ids and inputs.vpc_subnets:
            request["VpcConfig"][
                "SecurityGroupIds"
            ] = inputs.vpc_security_group_ids.split(",")
            request["VpcConfig"]["Subnets"] = inputs.vpc_subnets.split(",")
        else:
            request.pop("VpcConfig")

        ### Update input channels, must have at least one specified
        if len(inputs.channels) > 0:
            request["InputDataConfig"] = inputs.channels
        else:
            logging.error("Must specify at least one input channel.")
            raise Exception("Could not create job request")

        request["OutputDataConfig"]["S3OutputPath"] = inputs.model_artifact_path
        request["OutputDataConfig"]["KmsKeyId"] = inputs.output_encryption_key
        request["ResourceConfig"]["InstanceType"] = inputs.instance_type
        request["ResourceConfig"]["VolumeKmsKeyId"] = inputs.resource_encryption_key
        request["EnableNetworkIsolation"] = inputs.network_isolation
        request["EnableInterContainerTrafficEncryption"] = inputs.traffic_encryption

        ### Update InstanceCount, VolumeSizeInGB, and MaxRuntimeInSeconds if input is non-empty and > 0, otherwise use default values
        if inputs.instance_count:
            request["ResourceConfig"]["InstanceCount"] = inputs.instance_count

        if inputs.volume_size:
            request["ResourceConfig"]["VolumeSizeInGB"] = inputs.volume_size

        ### Update DebugHookConfig and DebugRuleConfigurations
        if inputs.debug_hook_config:
            request["DebugHookConfig"] = inputs.debug_hook_config
        else:
            request.pop("DebugHookConfig")

        if inputs.debug_rule_config:
            request["DebugRuleConfigurations"] = inputs.debug_rule_config
        else:
            request.pop("DebugRuleConfigurations")

        self._enable_spot_instance_support(request, inputs)
        self._enable_tag_support(request, inputs)

        return request

    def _submit_job_request(self, request: Dict) -> object:
        return self._sm_client.create_training_job(**request)

    def _after_submit_job_request(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerTrainingInputs,
        outputs: SageMakerTrainingOutputs,
    ):
        logging.info(f"Created Training Job with name: {self._training_job_name}")
        logging.info(
            "Training job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/jobs/{}".format(
                inputs.region, inputs.region, self._training_job_name,
            )
        )
        logging.info(
            "CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/TrainingJobs;prefix={};streamFilter=typeLogStreamPrefix".format(
                inputs.region, inputs.region, self._training_job_name,
            )
        )


if __name__ == "__main__":
    import sys

    spec = SageMakerTrainingSpec(sys.argv[1:])

    component = SageMakerTrainingComponent()
    component.Do(spec)

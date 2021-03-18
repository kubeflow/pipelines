"""SageMaker component for ground truth."""
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

from ground_truth.src.sagemaker_ground_truth_spec import (
    SageMakerGroundTruthSpec,
    SageMakerGroundTruthInputs,
    SageMakerGroundTruthOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)


@ComponentMetadata(
    name="SageMaker - Ground Truth",
    description="Ground Truth Jobs in SageMaker",
    spec=SageMakerGroundTruthSpec,
)
class SageMakerGroundTruthComponent(SageMakerComponent):
    """SageMaker component for ground truth."""

    def Do(self, spec: SageMakerGroundTruthSpec):
        self._labeling_job_name = (
            spec.inputs.job_name
            if spec.inputs.job_name
            else self._generate_unique_timestamped_id(prefix="LabelingJob")
        )
        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        response = self._sm_client.describe_labeling_job(
            LabelingJobName=self._labeling_job_name
        )
        status = response["LabelingJobStatus"]

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

    def _get_labeling_job_outputs(self, auto_labeling):
        """Get labeling job output values.

        Args:
            auto_labeling: Determines whether to return a active learning model ARN.

        Returns:
            tuple: (Output manifest S3 URI, active learning model ARN)
        """
        ### Get and return labeling job outputs
        info = self._sm_client.describe_labeling_job(
            LabelingJobName=self._labeling_job_name
        )
        output_manifest = info["LabelingJobOutput"]["OutputDatasetS3Uri"]
        if auto_labeling:
            active_learning_model_arn = info["LabelingJobOutput"][
                "FinalActiveLearningModelArn"
            ]
        else:
            active_learning_model_arn = " "
        return output_manifest, active_learning_model_arn

    def _after_job_complete(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerGroundTruthInputs,
        outputs: SageMakerGroundTruthOutputs,
    ):
        (
            outputs.output_manifest_location,
            outputs.active_learning_model_arn,
        ) = self._get_labeling_job_outputs(inputs.enable_auto_labeling)

    def _on_job_terminated(self):
        self._sm_client.stop_labeling_job(LabelingJobName=self._labeling_job_name)

    def _create_job_request(
        self, inputs: SageMakerGroundTruthInputs, outputs: SageMakerGroundTruthOutputs,
    ) -> Dict:
        ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_labeling_job
        request = self._get_request_template("gt")

        # Mapping are extracted from ARNs listed in https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_labeling_job
        algorithm_arn_map = {
            "us-west-2": "081040173940",
            "us-east-1": "432418664414",
            "us-east-2": "266458841044",
            "ca-central-1": "918755190332",
            "eu-west-1": "568282634449",
            "eu-west-2": "487402164563",
            "eu-central-1": "203001061592",
            "ap-northeast-1": "477331159723",
            "ap-northeast-2": "845288260483",
            "ap-south-1": "565803892007",
            "ap-southeast-1": "377565633583",
            "ap-southeast-2": "454466003867",
        }

        task_map = {
            "bounding box": "BoundingBox",
            "image classification": "ImageMultiClass",
            "semantic segmentation": "SemanticSegmentation",
            "text classification": "TextMultiClass",
        }

        auto_labeling_map = {
            "bounding box": "object-detection",
            "image classification": "image-classification",
            "text classification": "text-classification",
        }

        task = inputs.task_type.lower()

        request["LabelingJobName"] = self._labeling_job_name

        if inputs.label_attribute_name:
            name_check = inputs.label_attribute_name.split("-")[-1]
            if (
                task == "semantic segmentation"
                and name_check == "ref"
                or task != "semantic segmentation"
                and name_check != "metadata"
                and name_check != "ref"
            ):
                request["LabelAttributeName"] = inputs.label_attribute_name
            else:
                logging.error(
                    'Invalid label attribute name. If task type is semantic segmentation, name must end in "-ref". Else, name must not end in "-ref" or "-metadata".'
                )
        else:
            request["LabelAttributeName"] = inputs.job_name

        request["InputConfig"]["DataSource"]["S3DataSource"][
            "ManifestS3Uri"
        ] = inputs.manifest_location
        request["OutputConfig"]["S3OutputPath"] = inputs.output_location
        request["OutputConfig"]["KmsKeyId"] = inputs.output_encryption_key
        request["RoleArn"] = inputs.role

        ### Update or pop label category config s3 uri
        if not inputs.label_category_config:
            request.pop("LabelCategoryConfigS3Uri")
        else:
            request["LabelCategoryConfigS3Uri"] = inputs.label_category_config

        ### Update or pop stopping conditions
        if not inputs.max_human_labeled_objects and not inputs.max_percent_objects:
            request.pop("StoppingConditions")
        else:
            if inputs.max_human_labeled_objects:
                request["StoppingConditions"][
                    "MaxHumanLabeledObjectCount"
                ] = inputs.max_human_labeled_objects
            else:
                request["StoppingConditions"].pop("MaxHumanLabeledObjectCount")
            if inputs.max_percent_objects:
                request["StoppingConditions"][
                    "MaxPercentageOfInputDatasetLabeled"
                ] = inputs.max_percent_objects
            else:
                request["StoppingConditions"].pop("MaxPercentageOfInputDatasetLabeled")

        ### Update or pop automatic labeling configs
        if inputs.enable_auto_labeling:
            if (
                task == "image classification"
                or task == "bounding box"
                or task == "text classification"
            ):
                labeling_algorithm_arn = "arn:aws:sagemaker:{}:027400017018:labeling-job-algorithm-specification/image-classification".format(
                    inputs.region, auto_labeling_map[task]
                )
                request["LabelingJobAlgorithmsConfig"][
                    "LabelingJobAlgorithmSpecificationArn"
                ] = labeling_algorithm_arn
                if inputs.initial_model_arn:
                    request["LabelingJobAlgorithmsConfig"][
                        "InitialActiveLearningModelArn"
                    ] = inputs.initial_model_arn
                else:
                    request["LabelingJobAlgorithmsConfig"].pop(
                        "InitialActiveLearningModelArn"
                    )
                request["LabelingJobAlgorithmsConfig"]["LabelingJobResourceConfig"][
                    "VolumeKmsKeyId"
                ] = inputs.resource_encryption_key
            else:
                logging.error(
                    "Automated data labeling not available for semantic segmentation or custom algorithms. Proceeding without automated data labeling."
                )
        else:
            request.pop("LabelingJobAlgorithmsConfig")

        ### Update pre-human and annotation consolidation task lambda functions
        if (
            task == "image classification"
            or task == "bounding box"
            or task == "text classification"
            or task == "semantic segmentation"
        ):
            prehuman_arn = "arn:aws:lambda:{}:{}:function:PRE-{}".format(
                inputs.region, algorithm_arn_map[inputs.region], task_map[task]
            )
            acs_arn = "arn:aws:lambda:{}:{}:function:ACS-{}".format(
                inputs.region, algorithm_arn_map[inputs.region], task_map[task]
            )
            request["HumanTaskConfig"]["PreHumanTaskLambdaArn"] = prehuman_arn
            request["HumanTaskConfig"]["AnnotationConsolidationConfig"][
                "AnnotationConsolidationLambdaArn"
            ] = acs_arn
        elif task == "custom" or task == "":
            if inputs.pre_human_task_function and inputs.post_human_task_function:
                request["HumanTaskConfig"][
                    "PreHumanTaskLambdaArn"
                ] = inputs.pre_human_task_function
                request["HumanTaskConfig"]["AnnotationConsolidationConfig"][
                    "AnnotationConsolidationLambdaArn"
                ] = inputs.post_human_task_function
            else:
                logging.error(
                    "Must specify pre-human task lambda arn and annotation consolidation post-human task lambda arn."
                )
        else:
            logging.error(
                "Task type must be Bounding Box, Image Classification, Semantic Segmentation, Text Classification, or Custom."
            )

        request["HumanTaskConfig"]["UiConfig"]["UiTemplateS3Uri"] = inputs.ui_template
        request["HumanTaskConfig"]["TaskTitle"] = inputs.title
        request["HumanTaskConfig"]["TaskDescription"] = inputs.description
        request["HumanTaskConfig"][
            "NumberOfHumanWorkersPerDataObject"
        ] = inputs.num_workers_per_object
        request["HumanTaskConfig"]["TaskTimeLimitInSeconds"] = inputs.time_limit

        if inputs.task_availibility:
            request["HumanTaskConfig"][
                "TaskAvailabilityLifetimeInSeconds"
            ] = inputs.task_availibility
        else:
            request["HumanTaskConfig"].pop("TaskAvailabilityLifetimeInSeconds")

        if inputs.max_concurrent_tasks:
            request["HumanTaskConfig"][
                "MaxConcurrentTaskCount"
            ] = inputs.max_concurrent_tasks
        else:
            request["HumanTaskConfig"].pop("MaxConcurrentTaskCount")

        if inputs.task_keywords:
            for word in [n.strip() for n in inputs.task_keywords.split(",")]:
                request["HumanTaskConfig"]["TaskKeywords"].append(word)
        else:
            request["HumanTaskConfig"].pop("TaskKeywords")

        ### Update worker configurations
        if inputs.worker_type.lower() == "public":
            if inputs.no_adult_content:
                request["InputConfig"]["DataAttributes"]["ContentClassifiers"].append(
                    "FreeOfAdultContent"
                )
            if inputs.no_ppi:
                request["InputConfig"]["DataAttributes"]["ContentClassifiers"].append(
                    "FreeOfPersonallyIdentifiableInformation"
                )

            request["HumanTaskConfig"][
                "WorkteamArn"
            ] = "arn:aws:sagemaker:{}:394669845002:workteam/public-crowd/default".format(
                inputs.region
            )

            dollars = int(inputs.workforce_task_price)
            cents = int(100 * (inputs.workforce_task_price - dollars))
            tenth_of_cents = int(
                (inputs.workforce_task_price * 1000) - (dollars * 1000) - (cents * 10)
            )
            request["HumanTaskConfig"]["PublicWorkforceTaskPrice"]["AmountInUsd"][
                "Dollars"
            ] = dollars
            request["HumanTaskConfig"]["PublicWorkforceTaskPrice"]["AmountInUsd"][
                "Cents"
            ] = cents
            request["HumanTaskConfig"]["PublicWorkforceTaskPrice"]["AmountInUsd"][
                "TenthFractionsOfACent"
            ] = tenth_of_cents
        else:
            request["InputConfig"].pop("DataAttributes")
            request["HumanTaskConfig"]["WorkteamArn"] = inputs.workteam_arn
            request["HumanTaskConfig"].pop("PublicWorkforceTaskPrice")

        self._enable_tag_support(request, inputs)

        return request

    def _submit_job_request(self, request: Dict) -> object:
        return self._sm_client.create_labeling_job(**request)

    def _after_submit_job_request(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerGroundTruthInputs,
        outputs: SageMakerGroundTruthOutputs,
    ):
        logging.info(
            f"Created Ground Truth Labeling Job with name: {self._labeling_job_name}"
        )
        logging.info(
            "Ground Truth job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/groundtruth?region={}#/labeling-jobs/details/{}".format(
                inputs.region, inputs.region, self._labeling_job_name
            )
        )


if __name__ == "__main__":
    import sys

    spec = SageMakerGroundTruthSpec(sys.argv[1:])

    component = SageMakerGroundTruthComponent()
    component.Do(spec)

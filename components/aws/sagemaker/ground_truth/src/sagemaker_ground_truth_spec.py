"""Specification for the SageMaker ground truth component."""
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

from dataclasses import dataclass

from typing import List
from common.sagemaker_component_spec import (
    SageMakerComponentSpec,
    SageMakerComponentBaseOutputs,
)
from common.spec_input_parsers import SpecInputParsers
from common.common_inputs import (
    COMMON_INPUTS,
    SageMakerComponentCommonInputs,
    SageMakerComponentInput as Input,
    SageMakerComponentOutput as Output,
    SageMakerComponentInputValidator as InputValidator,
    SageMakerComponentOutputValidator as OutputValidator,
)


@dataclass(frozen=True)
class SageMakerGroundTruthInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the ground truth component."""

    role: Input
    job_name: Input
    label_attribute_name: Input
    manifest_location: Input
    output_location: Input
    output_encryption_key: Input
    task_type: Input
    worker_type: Input
    workteam_arn: Input
    no_adult_content: Input
    no_ppi: Input
    label_category_config: Input
    max_human_labeled_objects: Input
    max_percent_objects: Input
    enable_auto_labeling: Input
    initial_model_arn: Input
    resource_encryption_key: Input
    ui_template: Input
    pre_human_task_function: Input
    post_human_task_function: Input
    task_keywords: Input
    title: Input
    description: Input
    num_workers_per_object: Input
    time_limit: Input
    task_availibility: Input
    max_concurrent_tasks: Input
    workforce_task_price: Input


@dataclass
class SageMakerGroundTruthOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the groundtruth component."""

    output_manifest_location: Output
    active_learning_model_arn: Output


class SageMakerGroundTruthSpec(
    SageMakerComponentSpec[SageMakerGroundTruthInputs, SageMakerGroundTruthOutputs]
):
    INPUTS: SageMakerGroundTruthInputs = SageMakerGroundTruthInputs(
        role=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.",
        ),
        job_name=InputValidator(
            input_type=str, description="The name of the labeling job."
        ),
        label_attribute_name=InputValidator(
            input_type=str,
            required=False,
            description="The attribute name to use for the label in the output manifest file. Default is the job name.",
            default="",
        ),
        manifest_location=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon S3 location of the manifest file that describes the input data objects.",
        ),
        output_location=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon S3 location to write output data.",
        ),
        output_encryption_key=InputValidator(
            input_type=str,
            required=False,
            description="The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.",
            default="",
        ),
        task_type=InputValidator(
            input_type=str,
            required=True,
            description="Built in image classification, bounding box, text classification, or semantic segmentation, or custom. If custom, please provide pre- and post-labeling task lambda functions.",
        ),
        worker_type=InputValidator(
            input_type=str,
            required=True,
            description="The workteam for data labeling, either public, private, or vendor.",
        ),
        workteam_arn=InputValidator(
            input_type=str,
            required=False,
            description="The ARN of the work team assigned to complete the tasks.",
        ),
        no_adult_content=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            required=False,
            description="If true, your data is free of adult content.",
            default="False",
        ),
        no_ppi=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            required=False,
            description="If true, your data is free of personally identifiable information.",
            default="False",
        ),
        label_category_config=InputValidator(
            input_type=str,
            required=False,
            description="The S3 URL of the JSON structured file that defines the categories used to label the data objects.",
            default="",
        ),
        max_human_labeled_objects=InputValidator(
            input_type=int,
            required=False,
            description="The maximum number of objects that can be labeled by human workers.",
            default=0,
        ),
        max_percent_objects=InputValidator(
            input_type=int,
            required=False,
            description="The maximum percentatge of input data objects that should be labeled.",
            default=0,
        ),
        enable_auto_labeling=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            required=False,
            description="Enables auto-labeling, only for bounding box, text classification, and image classification.",
            default=False,
        ),
        initial_model_arn=InputValidator(
            input_type=str,
            required=False,
            description="The ARN of the final model used for a previous auto-labeling job.",
            default="",
        ),
        resource_encryption_key=InputValidator(
            input_type=str,
            required=False,
            description="The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).",
            default="",
        ),
        ui_template=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon S3 bucket location of the UI template.",
        ),
        pre_human_task_function=InputValidator(
            input_type=str,
            required=False,
            description="The ARN of a Lambda function that is run before a data object is sent to a human worker.",
            default="",
        ),
        post_human_task_function=InputValidator(
            input_type=str,
            required=False,
            description="The ARN of a Lambda function implements the logic for annotation consolidation.",
            default="",
        ),
        task_keywords=InputValidator(
            input_type=str,
            required=False,
            description="Keywords used to describe the task so that workers on Amazon Mechanical Turk can discover the task.",
            default="",
        ),
        title=InputValidator(
            input_type=str,
            required=True,
            description="A title for the task for your human workers.",
        ),
        description=InputValidator(
            input_type=str,
            required=True,
            description="A description of the task for your human workers.",
        ),
        num_workers_per_object=InputValidator(
            input_type=int,
            required=True,
            description="The number of human workers that will label an object.",
        ),
        time_limit=InputValidator(
            input_type=int,
            required=True,
            description="The amount of time that a worker has to complete a task in seconds",
        ),
        task_availibility=InputValidator(
            input_type=int,
            required=False,
            description="The length of time that a task remains available for labelling by human workers.",
            default="0",
        ),
        max_concurrent_tasks=InputValidator(
            input_type=int,
            required=False,
            description="The maximum number of data objects that can be labeled by human workers at the same time.",
            default="0",
        ),
        workforce_task_price=InputValidator(
            input_type=float,
            required=False,
            description="The price that you pay for each task performed by a public worker in USD. Specify to the tenth fractions of a cent. Format as '0.000'.",
            default="0.000",
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = SageMakerGroundTruthOutputs(
        output_manifest_location=OutputValidator(
            description="The Amazon S3 bucket location of the manifest file for labeled data."
        ),
        active_learning_model_arn=OutputValidator(
            description="The ARN for the most recent Amazon SageMaker model trained as part of automated data labeling."
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(
            arguments, SageMakerGroundTruthInputs, SageMakerGroundTruthOutputs
        )

    @property
    def inputs(self) -> SageMakerGroundTruthInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerGroundTruthOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerGroundTruthOutputs:
        return self._output_paths

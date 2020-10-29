"""Specification for the SageMaker transform component."""
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
class SageMakerTransformInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the transform component."""

    job_name: Input
    model_name: Input
    max_concurrent: Input
    max_payload: Input
    batch_strategy: Input
    environment: Input
    input_location: Input
    data_type: Input
    content_type: Input
    split_type: Input
    compression_type: Input
    output_location: Input
    accept: Input
    assemble_with: Input
    output_encryption_key: Input
    input_filter: Input
    output_filter: Input
    join_source: Input
    instance_type: Input
    instance_count: Input
    resource_encryption_key: Input


@dataclass
class SageMakerTransformOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the transform component."""

    output_location: Output


class SageMakerTransformSpec(
    SageMakerComponentSpec[SageMakerTransformInputs, SageMakerTransformOutputs]
):
    INPUTS: SageMakerTransformInputs = SageMakerTransformInputs(
        job_name=InputValidator(
            input_type=str,
            required=False,
            description="The name of the transform job.",
            default="",
        ),
        model_name=InputValidator(
            input_type=str,
            required=True,
            description="The name of the model that you want to use for the transform job.",
        ),
        max_concurrent=InputValidator(
            input_type=int,
            required=False,
            description="The maximum number of parallel requests that can be sent to each instance in a transform job.",
            default="0",
        ),
        max_payload=InputValidator(
            input_type=int,
            required=False,
            description="The maximum allowed size of the payload, in MB.",
            default="6",
        ),
        batch_strategy=InputValidator(
            input_type=str,
            choices=["MultiRecord", "SingleRecord", ""],
            required=False,
            description="The number of records to include in a mini-batch for an HTTP inference request.",
            default="",
        ),
        environment=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            required=False,
            description="The dictionary of the environment variables to set in the Docker container. Up to 16 key-value entries in the map.",
            default={},
        ),
        input_location=InputValidator(
            input_type=str,
            required=True,
            description="The S3 location of the data source that is associated with a channel.",
        ),
        data_type=InputValidator(
            input_type=str,
            choices=["ManifestFile", "S3Prefix", "AugmentedManifestFile", ""],
            required=False,
            description="Data type of the input. Can be ManifestFile, S3Prefix, or AugmentedManifestFile.",
            default="S3Prefix",
        ),
        content_type=InputValidator(
            input_type=str,
            required=False,
            description="The multipurpose internet mail extension (MIME) type of the data.",
            default="",
        ),
        split_type=InputValidator(
            input_type=str,
            choices=["None", "Line", "RecordIO", "TFRecord", ""],
            required=False,
            description="The method to use to split the transform job data files into smaller batches.",
            default="None",
        ),
        compression_type=InputValidator(
            input_type=str,
            choices=["None", "Gzip", ""],
            required=False,
            description="If the transform data is compressed, the specification of the compression type.",
            default="None",
        ),
        output_location=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon S3 path where you want Amazon SageMaker to store the results of the transform job.",
        ),
        accept=InputValidator(
            input_type=str,
            required=False,
            description="The MIME type used to specify the output data.",
        ),
        assemble_with=InputValidator(
            input_type=str,
            choices=["None", "Line", ""],
            required=False,
            description="Defines how to assemble the results of the transform job as a single S3 object. Either None or Line.",
        ),
        output_encryption_key=InputValidator(
            input_type=str,
            required=False,
            description="The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.",
            default="",
        ),
        input_filter=InputValidator(
            input_type=str,
            required=False,
            description="A JSONPath expression used to select a portion of the input data to pass to the algorithm.",
            default="",
        ),
        output_filter=InputValidator(
            input_type=str,
            required=False,
            description="A JSONPath expression used to select a portion of the joined dataset to save in the output file for a batch transform job.",
            default="",
        ),
        join_source=InputValidator(
            input_type=str,
            choices=["None", "Input", ""],
            required=False,
            description="Specifies the source of the data to join with the transformed data.",
            default="None",
        ),
        instance_type=InputValidator(
            input_type=str,
            required=False,
            description="The ML compute instance type for the transform job.",
            default="ml.m4.xlarge",
        ),
        instance_count=InputValidator(
            input_type=int,
            required=False,
            description="The number of ML compute instances to use in the transform job.",
            default="1",
        ),
        resource_encryption_key=InputValidator(
            input_type=str,
            required=False,
            description="The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).",
            default="",
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS: SageMakerTransformOutputs = SageMakerTransformOutputs(
        output_location=OutputValidator(
            description="S3 URI of the transform job results."
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(arguments, SageMakerTransformInputs, SageMakerTransformOutputs)

    @property
    def inputs(self) -> SageMakerTransformInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerTransformOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerTransformOutputs:
        return self._output_paths

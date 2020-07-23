"""Specification for the SageMaker training component"""
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

from common.sagemaker_component_spec import SageMakerComponentSpec
from common.spec_validators import SpecValidators


class SageMakerTrainingSpec(SageMakerComponentSpec):
    INPUTS = {
        **SageMakerComponentSpec.INPUTS,
        **{
            "job_name": dict(
                type=str,
                required=False,
                help="The name of the training job.",
                default="",
            ),
            "role": dict(
                type=str,
                required=True,
                help="The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.",
            ),
            "image": dict(
                type=str,
                required=False,
                help="The registry path of the Docker image that contains the training algorithm.",
                default="",
            ),
            "algorithm_name": dict(
                type=str,
                required=False,
                help="The name of the resource algorithm to use for the training job. Do not specify a value for this if using training image.",
                default="",
            ),
            "metric_definitions": dict(
                type=SpecValidators.yaml_or_json_dict,
                required=False,
                help="The dictionary of name-regex pairs specify the metrics that the algorithm emits.",
                default={},
            ),
            "training_input_mode": dict(
                choices=["File", "Pipe"],
                type=str,
                help="The input mode that the algorithm supports. File or Pipe.",
                default="File",
            ),
            "hyperparameters": dict(
                type=SpecValidators.yaml_or_json_dict,
                help="Dictionary of hyperparameters for the the algorithm.",
                default={},
            ),
            "channels": dict(
                type=SpecValidators.yaml_or_json_list,
                required=True,
                help="A list of dicts specifying the input channels. Must have at least one.",
            ),
            "instance_type": dict(
                required=False,
                type=str,
                help="The ML compute instance type.",
                default="ml.m4.xlarge",
            ),
            "instance_count": dict(
                required=True,
                type=int,
                help="The registry path of the Docker image that contains the training algorithm.",
                default=1,
            ),
            "volume_size": dict(
                type=int,
                required=True,
                help="The size of the ML storage volume that you want to provision.",
                default=30,
            ),
            "resource_encryption_key": dict(
                type=str,
                required=False,
                help="The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).",
                default="",
            ),
            "max_run_time": dict(
                type=int,
                required=True,
                help="The maximum run time in seconds for the training job.",
                default=86400,
            ),
            "model_artifact_path": dict(
                type=str,
                required=True,
                help="Identifies the S3 path where you want Amazon SageMaker to store the model artifacts.",
            ),
            "output_encryption_key": dict(
                type=str,
                required=False,
                help="The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.",
                default="",
            ),
            "vpc_security_group_ids": dict(
                type=str,
                required=False,
                help="The VPC security group IDs, in the form sg-xxxxxxxx.",
            ),
            "vpc_subnets": dict(
                type=str,
                required=False,
                help="The ID of the subnets in the VPC to which you want to connect your hpo job.",
            ),
            "network_isolation": dict(
                type=SpecValidators.str_to_bool,
                required=False,
                help="Isolates the training container.",
                default=True,
            ),
            "traffic_encryption": dict(
                type=SpecValidators.str_to_bool,
                required=False,
                help="Encrypts all communications between ML compute instances in distributed training.",
                default=False,
            ),
            "spot_instance": dict(
                type=SpecValidators.str_to_bool,
                required=False,
                help="Use managed spot training.",
                default=False,
            ),
            "max_wait_time": dict(
                type=int,
                required=False,
                help="The maximum time in seconds you are willing to wait for a managed spot training job to complete.",
                default=86400,
            ),
            "checkpoint_config": dict(
                type=SpecValidators.yaml_or_json_dict,
                required=False,
                help="Dictionary of information about the output location for managed spot training checkpoint data.",
                default={},
            ),
        },
    }

    OUTPUTS = {
        **SageMakerComponentSpec.OUTPUTS,
        **{
            "model_artifact_url": dict(help="The model artifacts URL.",),
            "job_name": dict(help="The training job name.",),
            "training_image": dict(
                help="The registry path of the Docker image that contains the training algorithm.",
            ),
        },
    }

# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""KFP Container component that performs bulk inference."""

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
import kfp


@kfp.dsl.container_component
def bulk_inferrer(
    project: str,
    location: str,
    inputs_sequence_length: int,
    targets_sequence_length: int,
    accelerator_type: str,
    accelerator_count: int,
    machine_type: str,
    image_uri: str,
    dataset_split: str,
    large_model_reference: str,
    input_model: str,
    input_dataset_path: str,
    output_prediction: kfp.dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    output_prediction_gcs_path: kfp.dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    gcp_resources: kfp.dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    sampling_strategy: str = 'greedy',
    encryption_spec_key_name: str = '',
) -> kfp.dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Performs bulk inference.

  Args:
    project: Project used to run the job.
    location: Location used to run the job.
    inputs_sequence_length: Maximum encoder/prefix length. Inputs will be padded
      or truncated to this length.
    targets_sequence_length: Maximum decoder steps. Outputs will be at most this
      length.
    accelerator_type: Type of accelerator.
    accelerator_count: Number of accelerators.
    machine_type: Type of machine.
    image: Location of reward model Docker image.
    input_model: Model to use for inference.
    large_model_reference: Predefined model used to create the ``input_model``.
    input_dataset_path: Path to dataset to use for inference.
    sampling_strategy: The sampling strategy for inference.
    dataset_split: Perform inference on this split of the input dataset.
    encryption_spec_key_name: Customer-managed encryption key. If this is set,
      then all resources created by the CustomJob will be encrypted with the
      provided encryption key. Note that this is not supported for TPU at the
      moment.

  Returns:
    output_prediction: Where to save the output prediction.
    gcp_resources: GCP resources that can be used to track the custom finetuning
      job.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_payload(
          display_name='BulkInferrer',
          accelerator_type=accelerator_type,
          accelerator_count=accelerator_count,
          machine_type=machine_type,
          image_uri=image_uri,
          args=[
              '--app_name=bulk_inferrer',
              f'--input_model={input_model}',
              f'--input_dataset={input_dataset_path}',
              f'--dataset_split={dataset_split}',
              f'--large_model_reference={large_model_reference}',
              f'--inputs_sequence_length={inputs_sequence_length}',
              f'--targets_sequence_length={targets_sequence_length}',
              f'--sampling_strategy={sampling_strategy}',
              f'--output_prediction={output_prediction}',
              f'--output_prediction_gcs_path={output_prediction_gcs_path}',
          ],
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )

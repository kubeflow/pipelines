# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
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
"""Data Labeling Evaluation component."""


from google_cloud_pipeline_components import _image
from kfp import dsl


@dsl.container_component
def evaluation_data_labeling(
    project: str,
    location: str,
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    job_display_name: str,
    dataset_name: str,
    instruction_uri: str,
    inputs_schema_uri: str,
    annotation_spec: str,
    labeler_count: int,
    annotation_label: str,
):
  """Builds a container spec that launches a data labeling job.

  Args:
    project: Project to run the job in.
    location: Location to run the job in.
    gcp_resources: GCP resources that can be used to track the job.
    job_display_name: Display name of the data labeling job.
    dataset_name: Name of the dataset to use for the data labeling job.
    instruction_uri: URI of the instruction for the data labeling job.
    inputs_schema_uri: URI of the inputs schema for the data labeling job.
    annotation_spec: Annotation spec for the data labeling job.
    labeler_count: Number of labelers to use for the data labeling job.
    annotation_label: Label of the data labeling job.

  Returns:
    Container spec that launches a data labeling job with the specified payload.
  """
  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container._implementation.model_evaluation.data_labeling_job.launcher',
      ],
      args=[
          '--type',
          'DataLabelingJob',
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
          '--job_display_name',
          job_display_name,
          '--dataset_name',
          dataset_name,
          '--instruction_uri',
          instruction_uri,
          '--inputs_schema_uri',
          inputs_schema_uri,
          '--annotation_spec',
          annotation_spec,
          '--labeler_count',
          labeler_count,
          '--annotation_label',
          annotation_label,
      ],
  )

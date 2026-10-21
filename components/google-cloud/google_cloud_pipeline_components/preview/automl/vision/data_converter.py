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
"""AutoML Vision training data converter."""

from typing import Optional

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components import _placeholders
# pylint: disable-next=g-importing-member, g-multiple-import
from google_cloud_pipeline_components.preview.automl.vision.json_utils import Concat, IfPresent, Json
from kfp import dsl


# pylint: disable=singleton-comparison
# pylint: disable=g-doc-args
@dsl.container_component
def data_converter(
    display_name: str,
    input_file_path: str,
    input_file_type: str,
    objective: str,
    output_dir: dsl.Output[dsl.Artifact],
    gcp_resources: dsl.OutputPath(str),
    enable_input_validation: bool = True,
    location: str = 'us-central1',
    timeout: str = '604800s',
    service_account: Optional[str] = None,
    machine_type: str = 'n1-highmem-4',
    output_shape: Optional[str] = None,
    split_ratio: Optional[str] = None,
    num_shard: Optional[str] = None,
    encryption_spec_key_name: str = '',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
):
  # fmt: off
  """Runs AutoML Vision data conversion. It will be launched as a Vertex AI [custom training job](https://cloud.google.com/vertex-ai/docs/training/create-custom-job) using the [CustomJob](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.customJobs) API.

  Args:
    display_name: The name of the CustomJob.
    input_file_path: Input file path. Please refer to different input formats in Vertex AI Documentation. For example, [image classification prepare data](https://cloud.google.com/vertex-ai/docs/image-data/classification/prepare-data) page.
    input_file_type: 'csv', 'jsonl', or 'coco_json'. Must be one of the input file types supported by the objective.
    objective: One of 'icn', 'iod', 'isg', 'vcn', or 'var'.
    location: Location for creating the custom training job. If not set, default to us-central1.
    timeout: The maximum job running time. The default is 7 days. A duration in seconds with up to nine fractional digits, terminated by 's', for example: "3.5s".
    service_account: Sets the default service account for workload run-as account. The [service account](https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account) running the pipeline submitting jobs must have act-as permission on this run-as account. If unspecified, the Vertex AI Custom Code [Service Agent ](https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents) for the CustomJob's project.
    machine_type: [Machine type](https://cloud.google.com/vertex-ai/docs/training/configure-compute#machine-types) for the CustomJob. If conversion failed, consider using a machine type with more RAM or splitting dataset into smaller pieces.
    output_shape: Output shape (height,width) for images.
    split_ratio: Proportion of data to split into train/validation/test, separated by comma.
    num_shard: Number of train/validation/test shards, separated by comma.
    encryption_spec_key_name: Customer-managed encryption key options for the CustomJob. If this is set, then all resources created by the CustomJob will be encrypted with the provided encryption key.
    project: Project to create the custom training job in. Defaults to the project in which the PipelineJob is run.
  Returns:
    output_dir: Cloud Storage directory storing converted data.
    gcp_resources: Serialized JSON of `gcp_resources` [proto](https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/google_cloud_pipeline_components/proto) which tracks the CustomJob.
  """
  # fmt: on
  payload = {
      'display_name': display_name,
      'job_spec': {
          'worker_pool_specs': [{
              'machine_spec': {
                  'machine_type': machine_type,
              },
              'replica_count': 1,
              'container_spec': {
                  'image_uri': 'us-docker.pkg.dev/vertex-ai/vertex-vision-model-garden-dockers/data-converter',
                  'args': [
                      '--enable_input_validation',
                      str(enable_input_validation),
                      '--input_file_path',
                      input_file_path,
                      '--input_file_type',
                      input_file_type,
                      '--objective',
                      objective,
                      '--output_dir',
                      output_dir.path,
                      IfPresent(
                          input_name='output_shape',
                          then=Concat('--output_shape', output_shape),
                      ),
                      IfPresent(
                          input_name='split_ratio',
                          then=Concat('--split_ratio', split_ratio),
                      ),
                      IfPresent(
                          input_name='num_shard',
                          then=Concat('--num_shard', num_shard),
                      ),
                  ],
              },
          }],
          'scheduling': {'timeout': timeout},
          'service_account': IfPresent(
              input_name='service_account',
              then=service_account,
          ),
          'enable_web_access': False,
      },
      'encryption_spec': {'kms_key_name': encryption_spec_key_name},
  }
  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.custom_job.launcher',
      ],
      args=[
          '--type',
          'CustomJob',
          '--payload',
          Json(payload),
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
      ],
  )

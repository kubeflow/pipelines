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
from kfp import dsl


# pylint: disable=g-doc-args
@dsl.container_component
def data_converter(
    display_name: str,
    input_file_path: str,
    input_file_type: str,
    objective: str,
    output_dir: str,
    gcp_resources: dsl.OutputPath(str),
    location: str = 'us-central1',
    timeout: str = '604800s',
    service_account: Optional[str] = None,
    machine_type: str = 'n1-highmem-4',
    output_shape: Optional[str] = None,
    split_ratio: Optional[str] = None,
    num_shard: Optional[str] = None,
    output_fps: Optional[int] = None,
    num_frames: Optional[int] = None,
    min_duration_sec: Optional[float] = None,
    pos_neg_ratio: Optional[float] = None,
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
    output_dir: Cloud Storage directory for storing converted data and pipeline information.
    location: Location for creating the custom training job. If not set, default to us-central1.
    timeout: The maximum job running time. The default is 7 days. A duration in seconds with up to nine fractional digits, terminated by 's', for example: "3.5s".
    service_account: Sets the default service account for workload run-as account. The [service account](https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account) running the pipeline submitting jobs must have act-as permission on this run-as account. If unspecified, the Vertex AI Custom Code [Service Agent ](https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents) for the CustomJob's project.
    machine_type: [Machine type](https://cloud.google.com/vertex-ai/docs/training/configure-compute#machine-types) for the CustomJob. If conversion failed, consider using a machine type with more RAM or splitting dataset into smaller pieces.
    output_shape: Video only. Output shape (height,width) for video frames.
    split_ratio: Proportion of data to split into train/validation/test, separated by comma.
    num_shard: Number of train/validation/test shards, separated by comma.
    output_fps: Video only. Output frames per second.
    num_frames: VAR only. Number of frames inside a single video clip window.
    min_duration_sec: VAR only. Minimum duration of a video clip annotation in seconds.
    pos_neg_ratio: VAR only. Sampling ratio between positive and negative segments.
    encryption_spec_key_name: Customer-managed encryption key options for the CustomJob. If this is set, then all resources created by the CustomJob will be encrypted with the provided encryption key.
    project: Project to create the custom training job in. Defaults to the project in which the PipelineJob is run.
  Returns:
    gcp_resources: Serialized JSON of `gcp_resources` [proto](https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/google_cloud_pipeline_components/proto) which tracks the CustomJob.
  """
  # fmt: on
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
          dsl.ConcatPlaceholder([
              '{',
              '"display_name": "',
              display_name,
              '",',
              '"job_spec": {',
              '"worker_pool_specs": [{',
              '"machine_spec": {',
              '"machine_type": "',
              machine_type,
              '"},',
              '"replica_count": 1,',
              '"container_spec": {',
              (
                  '"image_uri":'
                  ' "us-docker.pkg.dev/vertex-ai/vertex-vision-model-garden-dockers/data-converter",'
              ),
              '"args": [',
              '"--input_file_path", "',
              input_file_path,
              '",',
              '"--input_file_type", "',
              input_file_type,
              '",',
              '"--objective", "',
              objective,
              '",',
              '"--output_dir", "',
              output_dir,
              '"',
              dsl.IfPresentPlaceholder(
                  input_name='output_shape',
                  then=dsl.ConcatPlaceholder(
                      [',"--output_shape","', output_shape, '"']
                  ),
              ),
              dsl.IfPresentPlaceholder(
                  input_name='split_ratio',
                  then=dsl.ConcatPlaceholder(
                      [',"--split_ratio","', split_ratio, '"']
                  ),
              ),
              dsl.IfPresentPlaceholder(
                  input_name='num_shard',
                  then=dsl.ConcatPlaceholder(
                      [',"--num_shard","', num_shard, '"']
                  ),
              ),
              dsl.IfPresentPlaceholder(
                  input_name='output_fps',
                  then=dsl.ConcatPlaceholder(
                      [',"--output_fps","', output_fps, '"']
                  ),
              ),
              dsl.IfPresentPlaceholder(
                  input_name='num_frames',
                  then=dsl.ConcatPlaceholder(
                      [',"--num_frames","', num_frames, '"']
                  ),
              ),
              dsl.IfPresentPlaceholder(
                  input_name='min_duration_sec',
                  then=dsl.ConcatPlaceholder(
                      [',"--min_duration_sec","', min_duration_sec, '"']
                  ),
              ),
              dsl.IfPresentPlaceholder(
                  input_name='pos_neg_ratio',
                  then=dsl.ConcatPlaceholder(
                      [',"--pos_neg_ratio","', pos_neg_ratio, '"']
                  ),
              ),
              ']}}],',
              '"scheduling": {',
              '"timeout": "',
              timeout,
              '"',
              '},',
              dsl.IfPresentPlaceholder(
                  input_name='service_account',
                  then=dsl.ConcatPlaceholder(
                      ['"service_account": "', service_account, '",']
                  ),
              ),
              '"enable_web_access": false,',
              '"base_output_directory": {',
              '"output_uri_prefix": "',
              output_dir,
              '"',
              '}},',
              '"encryption_spec": {',
              '"kms_key_name": "',
              encryption_spec_key_name,
              '"',
              '}}',
          ]),
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
      ],
  )

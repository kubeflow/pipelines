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

from google_cloud_pipeline_components import _image
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_JOB_ID_PLACEHOLDER
from kfp.dsl import PIPELINE_TASK_ID_PLACEHOLDER


@container_component
def error_analysis_annotation(
    gcp_resources: OutputPath(str),
    error_analysis_output_uri: OutputPath(str),
    project: str,
    embeddings_dir: str,
    root_dir: str,
    location: str = 'us-central1',
    machine_type: str = 'n1-standard-8',
    num_neighbors: int = 5,
    encryption_spec_key_name: str = '',
):
  # fmt: off
  """Computes error analysis annotations from image embeddings.

  Args:
      project: GCP Project ID.
      location: GCP Region. If not set, defaulted to
        `us-central1`.
      embeddings_dir: The GCS directory containing embeddings
        output from the embeddings component.
      root_dir: The GCS directory for storing error analysis
        annotation output.
      machine_type: The machine type computing error analysis
        annotations. If not set, defaulted to
        `n1-standard-8`. More details:
        https://cloud.google.com/compute/docs/machine-resource
      num_neighbors: Number of nearest neighbors to look for. If
        not set, defaulted to `5`.
      encryption_spec_key_name: Customer-managed encryption key
        options for the CustomJob. If this is set, then all resources created by
        the CustomJob will be encrypted with the provided encryption key.

  Returns:
      error_analysis_output_uri: Google Cloud Storage URI to the computed error analysis annotations.
      gcp_resources: Serialized gcp_resources proto tracking the custom job.

        For more details, see
        https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return ContainerSpec(
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
          '--project',
          project,
          '--location',
          location,
          '--gcp_resource',
          gcp_resources,
          '--payload',
          ConcatPlaceholder([
              '{',
              '"display_name":',
              f' "error-analysis-annotation-{PIPELINE_JOB_ID_PLACEHOLDER}',
              f'-{PIPELINE_TASK_ID_PLACEHOLDER}", ',
              '"job_spec": {"worker_pool_specs": [{"replica_count":"1',
              '", "machine_spec": {"machine_type": "',
              machine_type,
              '"},',
              ' "container_spec": {"image_uri":"',
              'us-docker.pkg.dev/vertex-ai-restricted/vision-error-analysis/error-analysis:v0.2',
              '", "args": ["--embeddings_dir=',
              embeddings_dir,
              '", "--root_dir=',
              f'{root_dir}/{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
              '", "--num_neighbors=',
              num_neighbors,
              '", "--error_analysis_output_uri=',
              error_analysis_output_uri,
              '", "--executor_input={{$.json_escape[1]}}"]}}]}',
              ', "encryption_spec": {"kms_key_name":"',
              encryption_spec_key_name,
              '"}',
              '}',
          ]),
      ],
  )

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
    num_neighbors: int = 5,
):
  """Computes error analysis annotations from image embeddings.

  Args:
      project (str): Required. GCP Project ID.
      location (Optional[str]): GCP Region. If not set, defaulted to
        `us-central1`.
      embeddings_dir (str): Required. The GCS directory containing embeddings
        output from the embeddings component.
      root_dir (str): Required. The GCS directory for storing error analysis
        annotation output.
      num_neighbors (Optional[int]): Number of nearest neighbors to look for. If
        not set, defaulted to `5`.

  Returns:
      error_analysis_output_uri (str):
        Google Cloud Storage URI to the computed error analysis annotations.
      gcp_resources (str):
        Serialized gcp_resources proto tracking the custom job.

        For more details, see
        https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:latest',
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
              '", "machine_spec": {"machine_type": "n1-standard-8"},',
              ' "container_spec": {"image_uri":"',
              'us-docker.pkg.dev/vertex-ai-restricted/vision-error-analysis/error-analysis:v0.2',
              '", "args": ["--embeddings_dir=', embeddings_dir,
              '", "--root_dir=',
              f'{root_dir}/{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
              '", "--num_neighbors=', num_neighbors,
              '", "--error_analysis_output_uri=', error_analysis_output_uri,
              '", "--executor_input={{$.json_escape[1]}}"]}}]}}',
          ]),
      ],
  )

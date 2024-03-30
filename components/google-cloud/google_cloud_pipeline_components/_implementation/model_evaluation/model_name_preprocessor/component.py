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
"""Model name preprocessor component used in KFP pipelines."""

from google_cloud_pipeline_components._implementation.model_evaluation import version
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER


@container_component
def model_name_preprocessor(
    gcp_resources: OutputPath(str),
    processed_model_name: OutputPath(str),
    project: str,
    location: str,
    model_name: str,
    service_account: str = '',
):
  """Preprocess inputs for text2sql evaluation pipeline.

  Args:
      project: Required. The GCP project that runs the pipeline component.
      location: Required. The GCP region that runs the pipeline component.
      model_name: The Model name used to run evaluation. Must be a publisher
        Model or a managed Model sharing the same ancestor location. Starting
        this job has no impact on any existing deployments of the Model and
        their resources.
      service_account: Sets the default service account for workload run-as
        account. The service account running the pipeline
        (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)

  Returns:
      gcp_resources (str):
        Serialized gcp_resources proto tracking the custom job.
      processed_model_name (str):
        Preprocessed model name.
  """

  return ContainerSpec(
      image=version.LLM_EVAL_IMAGE_TAG,
      args=[
          '--model_name_preprocessor',
          'true',
          '--project',
          project,
          '--location',
          location,
          '--root_dir',
          f'{PIPELINE_ROOT_PLACEHOLDER}',
          '--model_name',
          model_name,
          '--processed_model_name',
          processed_model_name,
          '--service_account',
          service_account,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )

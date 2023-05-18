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
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec


@container_component
def vertex_pipelines_prompt_validation(
    prompt_dataset: str,
    model_id: str,
    rai_validation_enabled: bool,
    fail_on_warning: bool,
):
  # fmt: off
  """Validates the large language models prompt tuning dataset.

  It expects a JSONL file and validates it for correct format, tokenize limits
  of input/output prompts, harmful content etc.

  This component works only on Vertex Pipelines. This component raises an
  exception when run on Kubeflow Pipelines.

  Args:
    prompt_dataset: GCS path to the file containing the prompt tuning data.
    model_id: Large Language Model to tune.
    rai_validation_enabled: If enabled, validates prompt data for harmful content.
    fail_on_warning: If enabled, fails the component on warnings.
  """
  # fmt: on
  return ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.vertex_prompt_validation.executor',
      ],
      args=[
          '--type',
          'VertexPromptValidation',
          '--payload',
          '',
      ],
  )

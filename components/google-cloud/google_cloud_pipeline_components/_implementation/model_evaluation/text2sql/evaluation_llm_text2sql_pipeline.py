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
"""Text2SQL evaluation pipeline."""

from google_cloud_pipeline_components.types import artifact_types
import kfp


_PIPELINE_NAME = 'evaluation_llm_text2sql_pipeline'


@kfp.dsl.pipeline(name=_PIPELINE_NAME)
def evaluation_llm_text2sql_pipeline(
    location: str,
    model_name: str,
):
  """The LLM Evaluation Text2SQL Pipeline.

  Args:
    location: Required. The GCP region that runs the pipeline components.
    model_name: The path for model to generate embeddings.
  """

  get_vertex_model_task = kfp.dsl.importer(
      artifact_uri=(
          f'https://{location}-aiplatform.googleapis.com/v1/{model_name}'
      ),
      artifact_class=artifact_types.VertexModel,
      metadata={'resourceName': model_name},
  )
  get_vertex_model_task.set_display_name('get-vertex-model')

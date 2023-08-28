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


from typing import Optional

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp import dsl
from kfp.dsl import Input


@dsl.container_component
def evaluated_annotation_import(
    model: Input[VertexModel],
    evaluated_annotation_output_uri: str,
    evaluation_importer_gcp_resources: str,
    gcp_resources: dsl.OutputPath(str),
    error_analysis_output_uri: Optional[str] = None,
):
  # fmt: off
  """Imports evaluated annotations to an existing Vertex model with
  ModelService.BatchImportEvaluatedAnnotations.

  Evaluated Annotation inputs must be provided. ErrorAnalysisAnnotation,
  EvaluatedAnnotationExplanation are optional.

  Args:
    model: Vertex model resource that will be the parent resource of the
      uploaded evaluation.
    evaluated_annotation_output_uri: Path of evaluated annotations generated
      from EvaluatedAnnotation component.
    evaluation_importer_gcp_resources: GCP resource created by
      ModelEvaluationImporter comopnent. Specify this parameter in KFP pipeline
      by passing output from the ModelEvaluationImporter:
      `evaluation_importer_gcp_resources=eval_importer_task.outputs['gcp_resources']`
    error_analysis_output_uri: Path of error analysis annotations computed from
      image embeddings by finding nearest neighbors in the training set for each
      item in the test set.
  """
  # fmt: on
  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container._implementation.model_evaluation.import_evaluated_annotation',
      ],
      args=[
          '--evaluated_annotation_output_uri',
          evaluated_annotation_output_uri,
          '--evaluation_importer_gcp_resources',
          evaluation_importer_gcp_resources,
          dsl.IfPresentPlaceholder(
              input_name='error_analysis_output_uri',
              then=[
                  '--error_analysis_output_uri',
                  error_analysis_output_uri,
              ],
          ),
          '--pipeline_job_id',
          dsl.PIPELINE_JOB_ID_PLACEHOLDER,
          '--pipeline_job_resource_name',
          dsl.PIPELINE_JOB_RESOURCE_NAME_PLACEHOLDER,
          '--model_name',
          model.metadata['resourceName'],
          '--gcp_resources',
          gcp_resources,
      ],
  )

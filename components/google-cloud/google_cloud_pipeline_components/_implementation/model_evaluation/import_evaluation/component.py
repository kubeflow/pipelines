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
from google_cloud_pipeline_components.types.artifact_types import ClassificationMetrics
from google_cloud_pipeline_components.types.artifact_types import ForecastingMetrics
from google_cloud_pipeline_components.types.artifact_types import RegressionMetrics
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp import dsl
from kfp.dsl import Input
from kfp.dsl import Metrics


@dsl.container_component
def model_evaluation_import(
    model: Input[VertexModel],
    gcp_resources: dsl.OutputPath(str),
    metrics: Optional[Input[Metrics]] = None,
    problem_type: Optional[str] = None,
    classification_metrics: Optional[Input[ClassificationMetrics]] = None,
    forecasting_metrics: Optional[Input[ForecastingMetrics]] = None,
    regression_metrics: Optional[Input[RegressionMetrics]] = None,
    text_generation_metrics: Optional[Input[Metrics]] = None,
    question_answering_metrics: Optional[Input[Metrics]] = None,
    summarization_metrics: Optional[Input[Metrics]] = None,
    explanation: Optional[Input[Metrics]] = None,
    feature_attributions: Optional[Input[Metrics]] = None,
    display_name: Optional[str] = "",
    dataset_path: Optional[str] = "",
    dataset_paths: Optional[list] = [],
    dataset_type: Optional[str] = "",
):
  # fmt: off
  """Imports a model evaluation artifact to an existing Vertex model with
  ModelService.ImportModelEvaluation.

  For more details, see
  https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models.evaluations
  One of the four metrics inputs must be provided, metrics & problem_type,
  classification_metrics, regression_metrics, or forecasting_metrics.

  Args:
    model: Vertex model resource that will be the parent resource of the
      uploaded evaluation.
    metrics: Path of metrics generated from an evaluation component.
    problem_type: The problem type of the metrics being imported to the
      VertexModel. `classification`, `regression`, and `forecasting` are the
      currently supported problem types. Must be provided when `metrics` is
      provided.
    classification_metrics: Path of classification metrics generated from the
      classification evaluation component.
    forecasting_metrics: Path of forecasting metrics generated from the
      forecasting evaluation component.
    regression_metrics: Path of regression metrics generated from the regression
      evaluation component.
    explanation: Path for model explanation metrics generated from an evaluation
      component.
    feature_attributions: The feature attributions metrics artifact generated
      from the feature attribution component.
    display_name: The display name for the uploaded model evaluation resource.
  """
  # fmt: on
  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          "python3",
          "-u",
          "-m",
          "google_cloud_pipeline_components.container._implementation.model_evaluation.import_model_evaluation",
      ],
      args=[
          dsl.IfPresentPlaceholder(
              input_name="metrics",
              then=[
                  "--metrics",
                  metrics.uri,
                  "--metrics_explanation",
                  metrics.metadata["explanation_gcs_path"],
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name="explanation",
              then=[
                  "--explanation",
                  explanation.metadata["explanation_gcs_path"],
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name="classification_metrics",
              then=[
                  "--classification_metrics",
                  classification_metrics.uri,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name="forecasting_metrics",
              then=[
                  "--forecasting_metrics",
                  forecasting_metrics.uri,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name="regression_metrics",
              then=[
                  "--regression_metrics",
                  regression_metrics.uri,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name="text_generation_metrics",
              then=[
                  "--text_generation_metrics",
                  text_generation_metrics.uri,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name="question_answering_metrics",
              then=[
                  "--question_answering_metrics",
                  question_answering_metrics.uri,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name="summarization_metrics",
              then=[
                  "--summarization_metrics",
                  "{{$.inputs.artifacts['summarization_metrics'].uri}}",
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name="feature_attributions",
              then=[
                  "--feature_attributions",
                  feature_attributions.uri,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name="problem_type",
              then=[
                  "--problem_type",
                  problem_type,
              ],
          ),
          "--display_name",
          display_name,
          "--dataset_path",
          dataset_path,
          "--dataset_paths",
          dataset_paths,
          "--dataset_type",
          dataset_type,
          "--pipeline_job_id",
          dsl.PIPELINE_JOB_ID_PLACEHOLDER,
          "--pipeline_job_resource_name",
          dsl.PIPELINE_JOB_RESOURCE_NAME_PLACEHOLDER,
          "--model_name",
          model.metadata["resourceName"],
          "--gcp_resources",
          gcp_resources,
      ],
  )

# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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
"""Model evaluation components."""

import os

from google_cloud_pipeline_components.experimental.evaluation.classification.component import model_evaluation_classification as ModelEvaluationClassificationOp
from google_cloud_pipeline_components.experimental.evaluation.data_sampler.component import evaluation_data_sampler as EvaluationDataSamplerOp
from google_cloud_pipeline_components.experimental.evaluation.feature_attribution.component import feature_attribution as ModelEvaluationFeatureAttributionOp
from google_cloud_pipeline_components.experimental.evaluation.forecasting.component import model_evaluation_forecasting as ModelEvaluationForecastingOp
from google_cloud_pipeline_components.experimental.evaluation.regression.component import model_evaluation_regression as ModelEvaluationRegressionOp
from google_cloud_pipeline_components.experimental.evaluation.target_field_data_remover.component import target_field_data_remover as TargetFieldDataRemoverOp
from kfp import components

__all__ = [
    'ModelImportEvaluationOp',
    'EvaluationDataSamplerOp',
    'TargetFieldDataRemoverOp',
    'ModelEvaluationClassificationOp',
    'ModelEvaluationRegressionOp',
    'ModelEvaluationForecastingOp',
    'ModelEvaluationFeatureAttributionOp',
]


ModelImportEvaluationOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'import_evaluation/component.yaml')
)
"""
Imports a model evaluation artifact to an existing Vertex model with ModelService.ImportModelEvaluation
For more details, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models.evaluations
One of the four metrics inputs must be provided, metrics & problem_type, classification_metrics,
regression_metrics, or forecasting_metrics.

Args:
  model (google.VertexModel):
    Vertex model resource that will be the parent resource of the uploaded evaluation.
  metrics (system.Metrics):
    Path of metrics generated from an evaluation component.
  problem_type (Optional[str]):
    The problem type of the metrics being imported to the VertexModel.
      `classification`, `regression`, and `forecasting` are the currently supported problem types.
      Must be provided when `metrics` is provided.
  classification_metrics (Optional[google.ClassificationMetrics]):
    Path of classification metrics generated from the classification evaluation component.
  forecasting_metrics (Optional[google.ForecastingMetrics]):
    Path of forecasting metrics generated from the forecasting evaluation component.
  regression_metrics (Optional[google.RegressionMetrics]):
    Path of regression metrics generated from the regression evaluation component.
  explanation (Optional[system.Metrics]):
    Path for model explanation metrics generated from an evaluation component.
  feature_attributions (Optional[system.Metrics]):
    The feature attributions metrics artifact generated from the feature attribution component.
  display_name (str):
    The display name for the uploaded model evaluation resource.
"""

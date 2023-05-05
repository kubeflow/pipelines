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
"""Google Cloud Pipeline Model Evaluation components."""

import os

from kfp.components import load_component_from_file

from .classification import component as classification_component
from .data_sampler import component as data_sampler_component
from .feature_attribution import component as feature_attribution_component
from .forecasting import component as forecasting_component
from .regression import component as regression_component
from .target_field_data_remover import component as target_field_data_remover_component

__all__ = [
    'ModelImportEvaluationOp',
    'EvaluationDataSamplerOp',
    'TargetFieldDataRemoverOp',
    'ModelEvaluationClassificationOp',
    'ModelEvaluationRegressionOp',
    'ModelEvaluationForecastingOp',
    'ModelEvaluationFeatureAttributionOp',
]


ModelImportEvaluationOp = load_component_from_file(
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

EvaluationDataSamplerOp = data_sampler_component.evaluation_data_sampler

TargetFieldDataRemoverOp = (
    target_field_data_remover_component.target_field_data_remover
)

ModelEvaluationClassificationOp = (
    classification_component.model_evaluation_classification
)

ModelEvaluationRegressionOp = regression_component.model_evaluation_regression

ModelEvaluationForecastingOp = (
    forecasting_component.model_evaluation_forecasting
)

ModelEvaluationFeatureAttributionOp = (
    feature_attribution_component.feature_attribution
)

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

"""Model evaluation pipelines."""

from google_cloud_pipeline_components.v1.model_evaluation.classification_component import model_evaluation_classification as ModelEvaluationClassificationOp
from google_cloud_pipeline_components.v1.model_evaluation.error_analysis_pipeline import vision_model_error_analysis_pipeline
from google_cloud_pipeline_components.v1.model_evaluation.evaluated_annotation_pipeline import evaluated_annotation_pipeline
from google_cloud_pipeline_components.v1.model_evaluation.evaluation_automl_tabular_feature_attribution_pipeline import evaluation_automl_tabular_feature_attribution_pipeline
from google_cloud_pipeline_components.v1.model_evaluation.evaluation_automl_tabular_pipeline import evaluation_automl_tabular_pipeline
from google_cloud_pipeline_components.v1.model_evaluation.evaluation_automl_unstructure_data_pipeline import evaluation_automl_unstructure_data_pipeline
from google_cloud_pipeline_components.v1.model_evaluation.evaluation_feature_attribution_pipeline import evaluation_feature_attribution_pipeline
from google_cloud_pipeline_components.v1.model_evaluation.forecasting_component import model_evaluation_forecasting as ModelEvaluationForecastingOp
from google_cloud_pipeline_components.v1.model_evaluation.regression_component import model_evaluation_regression as ModelEvaluationRegressionOp

__all__ = [
    'vision_model_error_analysis_pipeline',
    'evaluated_annotation_pipeline',
    'evaluation_automl_tabular_feature_attribution_pipeline',
    'evaluation_automl_tabular_pipeline',
    'evaluation_automl_unstructure_data_pipeline',
    'evaluation_feature_attribution_pipeline',
    'ModelEvaluationClassificationOp',
    'ModelEvaluationRegressionOp',
    'ModelEvaluationForecastingOp',
]

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
"""Google Cloud Pipeline BigQuery components."""

import os

from .create_model import component as create_model_component
from .detect_anomalies_model import component as detect_anomalies_model_component
from .drop_model import component as drop_model_component
from .evaluate_model import component as evaluate_model_component
from .explain_forecast_model import component as explain_forecast_model_component
from .explain_predict_model import component as explain_predict_model
from .export_model import component as export_model_component
from .feature_importance import component as feature_importance_component
from .forecast_model import component as forecast_model_component
from .global_explain import component as global_explain_component
from .ml_advanced_weights import component as ml_advanced_weights_component
from .ml_arima_coefficients import component as ml_arima_coefficients_component
from .ml_arima_evaluate import component as ml_arima_evaluate_component
from .ml_centroids import component as ml_centroids_component
from .ml_confusion_matrix import component as ml_confusion_matrix_component
from .ml_feature_info import component as ml_feature_info_component
from .ml_principal_component_info import component as ml_principal_component_info_component
from .ml_principal_components import component as ml_principal_components_component
from .ml_recommend import component as ml_recommend_component
from .ml_reconstruction_loss import component as ml_reconstruction_loss_component
from .ml_roc_curve import component as ml_roc_curve_component
from .ml_training_info import component as ml_training_info_component
from .ml_trial_info import component as ml_trial_info_component
from .ml_weights import component as ml_weights_component
from .predict_model import component as predict_model_component
from .query_job import component as query_job_component

__all__ = [
    'BigqueryCreateModelJobOp',
    'BigqueryDetectAnomaliesModelJobOp',
    'BigqueryDropModelJobOp',
    'BigqueryEvaluateModelJobOp',
    'BigqueryExplainForecastModelJobOp',
    'BigqueryExplainPredictModelJobOp',
    'BigqueryExportModelJobOp',
    'BigqueryForecastModelJobOp',
    'BigqueryMLAdvancedWeightsJobOp',
    'BigqueryMLArimaCoefficientsJobOp',
    'BigqueryMLArimaEvaluateJobOp',
    'BigqueryMLCentroidsJobOp',
    'BigqueryMLConfusionMatrixJobOp',
    'BigqueryMLFeatureImportanceJobOp',
    'BigqueryMLFeatureInfoJobOp',
    'BigqueryMLGlobalExplainJobOp',
    'BigqueryMLPrincipalComponentInfoJobOp',
    'BigqueryMLPrincipalComponentsJobOp',
    'BigqueryMLRecommendJobOp',
    'BigqueryMLReconstructionLossJobOp',
    'BigqueryMLRocCurveJobOp',
    'BigqueryMLTrainingInfoJobOp',
    'BigqueryMLTrialInfoJobOp',
    'BigqueryMLWeightsJobOp',
    'BigqueryPredictModelJobOp',
    'BigqueryQueryJobOp',
]

BigqueryCreateModelJobOp = create_model_component.bigquery_create_model_job
BigqueryDetectAnomaliesModelJobOp = (
    detect_anomalies_model_component.bigquery_detect_anomalies_job
)
BigqueryDropModelJobOp = drop_model_component.bigquery_drop_model_job
BigqueryEvaluateModelJobOp = (
    evaluate_model_component.bigquery_evaluate_model_job
)
BigqueryExplainForecastModelJobOp = (
    explain_forecast_model_component.bigquery_explain_forecast_model_job
)
BigqueryExplainPredictModelJobOp = (
    explain_predict_model.bigquery_explain_predict_model_job
)
BigqueryExportModelJobOp = export_model_component.bigquery_export_model_job
BigqueryForecastModelJobOp = (
    forecast_model_component.bigquery_forecast_model_job
)
BigqueryMLAdvancedWeightsJobOp = (
    ml_advanced_weights_component.bigquery_ml_advanced_weights_job
)
BigqueryMLArimaCoefficientsJobOp = (
    ml_arima_coefficients_component.bigquery_ml_arima_coefficients
)
BigqueryMLArimaEvaluateJobOp = (
    ml_arima_evaluate_component.bigquery_ml_arima_evaluate_job
)
BigqueryMLCentroidsJobOp = ml_centroids_component.bigquery_ml_centroids_job
BigqueryMLConfusionMatrixJobOp = (
    ml_confusion_matrix_component.bigquery_ml_confusion_matrix_job
)
BigqueryMLFeatureImportanceJobOp = (
    feature_importance_component.bigquery_ml_feature_importance_job
)
BigqueryMLFeatureInfoJobOp = (
    ml_feature_info_component.bigquery_ml_feature_info_job
)
BigqueryMLGlobalExplainJobOp = (
    global_explain_component.bigquery_ml_global_explain_job
)
BigqueryMLPrincipalComponentInfoJobOp = (
    ml_principal_component_info_component.bigquery_ml_principal_component_info_job
)
BigqueryMLPrincipalComponentsJobOp = (
    ml_principal_components_component.bigquery_ml_principal_components_job
)
BigqueryMLRecommendJobOp = ml_recommend_component.bigquery_ml_recommend_job
BigqueryMLReconstructionLossJobOp = (
    ml_reconstruction_loss_component.bigquery_ml_reconstruction_loss_job
)
BigqueryMLRocCurveJobOp = ml_roc_curve_component.bigquery_ml_roc_curve_job
BigqueryMLTrainingInfoJobOp = (
    ml_training_info_component.bigquery_ml_training_info_job
)
BigqueryMLTrialInfoJobOp = ml_trial_info_component.bigquery_ml_trial_info_job
BigqueryMLWeightsJobOp = ml_weights_component.bigquery_ml_weights_job
BigqueryPredictModelJobOp = predict_model_component.bigquery_predict_model_job
BigqueryQueryJobOp = query_job_component.bigquery_query_job

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
"""Google Cloud Pipeline Bigquery components."""

import os

try:
  from kfp.v2.components import load_component_from_file
except ImportError:
  from kfp.components import load_component_from_file

__all__ = [
    'BigqueryQueryJobOp',
    'BigqueryCreateModelJobOp',
    'BigqueryExportModelJobOp',
    'BigqueryMLArimaEvaluateJobOp',
    'BigqueryPredictModelJobOp',
    'BigqueryEvaluateModelJobOp',
    'BigqueryMLArimaCoefficientsJobOp',
    'BigqueryMLWeightsJobOp',
    'BigqueryMLReconstructionLossJobOp',
    'BigqueryMLTrialInfoJobOp',
    'BigqueryMLTrainingInfoJobOp',
    'BigqueryExplainPredictModelJobOp',
    'BigqueryExplainForecastModelJobOp',
    'BigqueryMLAdvancedWeightsJobOp',
    'BigqueryDropModelJobOp',
    'BigqueryMLCentroidsJobOp',
    'BigqueryMLConfusionMatrixJobOp',
    'BigqueryMLFeatureInfoJobOp',
    'BigqueryMLRocCurveJobOp',
    'BigqueryMLPrincipalComponentsJobOp',
    'BigqueryMLPrincipalComponentInfoJobOp',
    'BigqueryMLFeatureImportanceJobOp',
    'BigqueryMLRecommendJobOp',
    'BigqueryForecastModelJobOp',
    'BigqueryForecastModelJobOp',
    'BigqueryMLGlobalExplainJobOp',
    'BigqueryDetectAnomaliesModelJobOp',
]

BigqueryQueryJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'query_job/component.yaml'))

BigqueryCreateModelJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'create_model/component.yaml'))

BigqueryExportModelJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'export_model/component.yaml'))

BigqueryPredictModelJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'predict_model/component.yaml'))

BigqueryExplainPredictModelJobOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'explain_predict_model/component.yaml'))

BigqueryExplainForecastModelJobOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'explain_forecast_model/component.yaml'))

BigqueryEvaluateModelJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'evaluate_model/component.yaml'))

BigqueryMLArimaCoefficientsJobOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'ml_arima_coefficients/component.yaml'))

BigqueryMLArimaEvaluateJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'ml_arima_evaluate/component.yaml'))

BigqueryMLReconstructionLossJobOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'ml_reconstruction_loss/component.yaml'))

BigqueryMLTrialInfoJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'ml_trial_info/component.yaml'))

BigqueryMLWeightsJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'ml_weights/component.yaml'))

BigqueryMLTrainingInfoJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'ml_training_info/component.yaml'))

BigqueryMLAdvancedWeightsJobOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'ml_advanced_weights/component.yaml'))

BigqueryDropModelJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'drop_model/component.yaml'))

BigqueryMLCentroidsJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'ml_centroids/component.yaml'))

BigqueryMLConfusionMatrixJobOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'ml_confusion_matrix/component.yaml'))

BigqueryMLFeatureInfoJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'ml_feature_info/component.yaml'))

BigqueryMLRocCurveJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'ml_roc_curve/component.yaml'))

BigqueryMLPrincipalComponentsJobOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'ml_principal_components/component.yaml'))

BigqueryMLPrincipalComponentInfoJobOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'ml_principal_component_info/component.yaml'))

BigqueryMLFeatureImportanceJobOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'feature_importance/component.yaml'))

BigqueryMLRecommendJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'ml_recommend/component.yaml'))

BigqueryForecastModelJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'forecast_model/component.yaml'))

BigqueryMLGlobalExplainJobOp = load_component_from_file(
        os.path.join(os.path.dirname(__file__), 'global_explain/component.yaml'))

BigqueryDetectAnomaliesModelJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'detect_anomalies_model/component.yaml'))

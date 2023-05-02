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

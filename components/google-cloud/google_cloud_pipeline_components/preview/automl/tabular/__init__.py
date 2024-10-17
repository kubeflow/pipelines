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

"""Preview AutoML tabular components."""

import os

from google_cloud_pipeline_components.preview.automl.tabular.auto_feature_engineering import automated_feature_engineering as AutoFeatureEngineeringOp
from google_cloud_pipeline_components.preview.automl.tabular.distillation_stage_feature_transform_engine import distillation_stage_feature_transform_engine as DistillationStageFeatureTransformEngineOp
from google_cloud_pipeline_components.preview.automl.tabular.feature_selection import tabular_feature_ranking_and_selection as FeatureSelectionOp
from google_cloud_pipeline_components.preview.automl.tabular.feature_transform_engine import feature_transform_engine as FeatureTransformEngineOp
from google_cloud_pipeline_components.preview.automl.tabular.tabnet_hyperparameter_tuning_job import tabnet_hyperparameter_tuning_job as TabNetHyperparameterTuningJobOp
from google_cloud_pipeline_components.preview.automl.tabular.tabnet_trainer import tabnet_trainer as TabNetTrainerOp
from google_cloud_pipeline_components.preview.automl.tabular.utils import get_tabnet_hyperparameter_tuning_job_pipeline_and_parameters
from google_cloud_pipeline_components.preview.automl.tabular.utils import get_tabnet_trainer_pipeline_and_parameters
from google_cloud_pipeline_components.preview.automl.tabular.utils import get_wide_and_deep_hyperparameter_tuning_job_pipeline_and_parameters
from google_cloud_pipeline_components.preview.automl.tabular.utils import get_wide_and_deep_trainer_pipeline_and_parameters
from google_cloud_pipeline_components.preview.automl.tabular.utils import get_xgboost_hyperparameter_tuning_job_pipeline_and_parameters
from google_cloud_pipeline_components.preview.automl.tabular.utils import get_xgboost_trainer_pipeline_and_parameters
from google_cloud_pipeline_components.preview.automl.tabular.wide_and_deep_hyperparameter_tuning_job import wide_and_deep_hyperparameter_tuning_job as WideAndDeepHyperparameterTuningJobOp
from google_cloud_pipeline_components.preview.automl.tabular.wide_and_deep_trainer import wide_and_deep_trainer as WideAndDeepTrainerOp
from google_cloud_pipeline_components.preview.automl.tabular.xgboost_hyperparameter_tuning_job import xgboost_hyperparameter_tuning_job as XGBoostHyperparameterTuningJobOp
from google_cloud_pipeline_components.preview.automl.tabular.xgboost_trainer import xgboost_trainer as XGBoostTrainerOp
from kfp import components

__all__ = [
    'AutoFeatureEngineeringOp',
    'DistillationStageFeatureTransformEngineOp',
    'FeatureSelectionOp',
    'FeatureTransformEngineOp',
    'TabNetHyperparameterTuningJobOp',
    'TabNetTrainerOp',
    'WideAndDeepHyperparameterTuningJobOp',
    'WideAndDeepTrainerOp',
    'XGBoostHyperparameterTuningJobOp',
    'XGBoostTrainerOp',
    'get_tabnet_hyperparameter_tuning_job_pipeline_and_parameters',
    'get_tabnet_trainer_pipeline_and_parameters',
    'get_wide_and_deep_hyperparameter_tuning_job_pipeline_and_parameters',
    'get_wide_and_deep_trainer_pipeline_and_parameters',
    'get_xgboost_hyperparameter_tuning_job_pipeline_and_parameters',
    'get_xgboost_trainer_pipeline_and_parameters',
]

tabnet_trainer_pipeline = components.load_component_from_file(
    # Note, please don't name it as `component.yaml` which will conflict with
    # the generated file.
    os.path.join(os.path.dirname(__file__), 'tabnet_trainer_pipeline.yaml')
)

wide_and_deep_trainer_pipeline = components.load_component_from_file(
    # Note, please don't name it as `component.yaml` which will conflict with
    # the generated file.
    os.path.join(
        os.path.dirname(__file__), 'wide_and_deep_trainer_pipeline.yaml'
    )
)

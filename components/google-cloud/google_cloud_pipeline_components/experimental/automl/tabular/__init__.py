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
"""Experimental AutoML tabular components."""

import os

from kfp import components

__all__ = [
    'CvTrainerOp',
    'InfraValidatorOp',
    'Stage1TunerOp',
    'EnsembleOp',
    'StatsAndExampleGenOp',
    'FeatureSelectionOp',
    'TransformOp',
    'FinalizerOp',
    'WideAndDeepHyperparameterTuningJobOp',
    'WideAndDeepTrainerOp',
    'TabNetHyperparameterTuningJobOp',
    'TabNetTrainerOp',
    'FeatureTransformEngineOp',
    'SplitMaterializedDataOp',
    'TrainingConfiguratorAndValidatorOp',
    'TrainingConfiguratorAndValidatorOp',
    'XGBoostHyperparameterTuningJobOp',
    'XGBoostTrainerOp',
]

CvTrainerOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'cv_trainer.yaml')
)
InfraValidatorOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'infra_validator.yaml')
)
Stage1TunerOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'stage_1_tuner.yaml')
)
EnsembleOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'ensemble.yaml')
)
StatsAndExampleGenOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'stats_and_example_gen.yaml')
)
FeatureSelectionOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'feature_selection.yaml')
)
TransformOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'transform.yaml')
)
FeatureTransformEngineOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'feature_transform_engine.yaml')
)
SplitMaterializedDataOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'split_materialized_data.yaml')
)
FinalizerOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'finalizer.yaml')
)
WideAndDeepTrainerOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'wide_and_deep_trainer.yaml')
)
WideAndDeepHyperparameterTuningJobOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'wide_and_deep_hyperparameter_tuning_job.yaml',
    )
)
TabNetHyperparameterTuningJobOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'tabnet_hyperparameter_tuning_job.yaml'
    )
)
TabNetTrainerOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'tabnet_trainer.yaml')
)
TrainingConfiguratorAndValidatorOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'training_configurator_and_validator.yaml'
    )
)
XGBoostTrainerOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'xgboost_trainer.yaml')
)
XGBoostHyperparameterTuningJobOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'xgboost_hyperparameter_tuning_job.yaml'
    )
)

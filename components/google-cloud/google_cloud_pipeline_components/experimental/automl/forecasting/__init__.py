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
"""Module for AutoML Tabular Forecasting KFP components."""

import os

from kfp.components import load_component_from_file  # pylint: enable=g-import-not-at-top

__all__ = [
    'ForecastingStage1TunerOp',
    'ForecastingEnsembleOp',
    'ForecastingStage2TunerOp',
    'ProphetTrainerOp',
    'ModelEvaluationForecastingOp',
]

ProphetTrainerOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'prophet_trainer.yaml')
)
ForecastingStage1TunerOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'forecasting_stage_1_tuner.yaml')
)
ForecastingEnsembleOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'forecasting_ensemble.yaml')
)
ForecastingStage2TunerOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'forecasting_stage_2_tuner.yaml')
)
ModelEvaluationForecastingOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'model_evaluation_forecasting.yaml')
)

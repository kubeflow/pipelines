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

"""Preview AutoML forecasting components."""

import os

from google_cloud_pipeline_components.preview.automl.forecasting.forecasting_ensemble import automl_forecasting_ensemble as ForecastingEnsembleOp
from google_cloud_pipeline_components.preview.automl.forecasting.forecasting_stage_1_tuner import automl_forecasting_stage_1_tuner as ForecastingStage1TunerOp
from google_cloud_pipeline_components.preview.automl.forecasting.forecasting_stage_2_tuner import automl_forecasting_stage_2_tuner as ForecastingStage2TunerOp
from google_cloud_pipeline_components.v1.automl.forecasting import learn_to_learn_forecasting_pipeline
from google_cloud_pipeline_components.v1.automl.forecasting import sequence_to_sequence_forecasting_pipeline
from google_cloud_pipeline_components.v1.automl.forecasting import temporal_fusion_transformer_forecasting_pipeline
from google_cloud_pipeline_components.v1.automl.forecasting import time_series_dense_encoder_forecasting_pipeline
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_learn_to_learn_forecasting_pipeline_and_parameters
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_sequence_to_sequence_forecasting_pipeline_and_parameters
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_temporal_fusion_transformer_forecasting_pipeline_and_parameters
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_time_series_dense_encoder_forecasting_pipeline_and_parameters
from kfp import components


__all__ = [
    'ForecastingEnsembleOp',
    'ForecastingStage1TunerOp',
    'ForecastingStage2TunerOp',
    'get_learn_to_learn_forecasting_pipeline_and_parameters',
    'get_sequence_to_sequence_forecasting_pipeline_and_parameters',
    'get_temporal_fusion_transformer_forecasting_pipeline_and_parameters',
    'get_time_series_dense_encoder_forecasting_pipeline_and_parameters',
    'learn_to_learn_forecasting_pipeline',
    'sequence_to_sequence_forecasting_pipeline',
    'temporal_fusion_transformer_forecasting_pipeline',
    'time_series_dense_encoder_forecasting_pipeline',
]

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

"""GA AutoML forecasting components."""
import os

from google_cloud_pipeline_components.v1.automl.forecasting.prophet_trainer import prophet_trainer as ProphetTrainerOp
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_bqml_arima_predict_pipeline_and_parameters
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_bqml_arima_train_pipeline_and_parameters
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_learn_to_learn_forecasting_pipeline_and_parameters
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_prophet_prediction_pipeline_and_parameters
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_prophet_train_pipeline_and_parameters
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_sequence_to_sequence_forecasting_pipeline_and_parameters
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_temporal_fusion_transformer_forecasting_pipeline_and_parameters
from google_cloud_pipeline_components.v1.automl.forecasting.utils import get_time_series_dense_encoder_forecasting_pipeline_and_parameters
from kfp import components

__all__ = [
    'ProphetTrainerOp',
    'get_bqml_arima_predict_pipeline_and_parameters',
    'get_bqml_arima_train_pipeline_and_parameters',
    'get_prophet_prediction_pipeline_and_parameters',
    'get_prophet_train_pipeline_and_parameters',
    'get_learn_to_learn_forecasting_pipeline_and_parameters',
    'get_sequence_to_sequence_forecasting_pipeline_and_parameters',
    'get_temporal_fusion_transformer_forecasting_pipeline_and_parameters',
    'get_time_series_dense_encoder_forecasting_pipeline_and_parameters',
    'learn_to_learn_forecasting_pipeline',
    'sequence_to_sequence_forecasting_pipeline',
    'temporal_fusion_transformer_forecasting_pipeline',
    'time_series_dense_encoder_forecasting_pipeline',
]

learn_to_learn_forecasting_pipeline = components.load_component_from_file(
    # Note, please don't name it as `component.yaml` which will conflict with
    # the generated file.
    os.path.join(
        os.path.dirname(__file__), 'learn_to_learn_forecasting_pipeline.yaml'
    )
)

sequence_to_sequence_forecasting_pipeline = components.load_component_from_file(
    # Note, please don't name it as `component.yaml` which will conflict with
    # the generated file.
    os.path.join(
        os.path.dirname(__file__),
        'sequence_to_sequence_forecasting_pipeline.yaml',
    )
)

temporal_fusion_transformer_forecasting_pipeline = components.load_component_from_file(
    # Note, please don't name it as `component.yaml` which will conflict with
    # the generated file.
    os.path.join(
        os.path.dirname(__file__),
        'temporal_fusion_transformer_forecasting_pipeline.yaml',
    )
)

time_series_dense_encoder_forecasting_pipeline = components.load_component_from_file(
    # Note, please don't name it as `component.yaml` which will conflict with
    # the generated file.
    os.path.join(
        os.path.dirname(__file__),
        'time_series_dense_encoder_forecasting_pipeline.yaml',
    )
)

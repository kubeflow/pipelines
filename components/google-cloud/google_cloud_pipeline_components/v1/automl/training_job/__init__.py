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
"""Google Cloud Pipeline AutoML Training Job components."""
import os

from kfp import components

__all__ = [
    'AutoMLImageTrainingJobRunOp',
    'AutoMLTextTrainingJobRunOp',
    'AutoMLTabularTrainingJobRunOp',
    'AutoMLForecastingTrainingJobRunOp',
    'AutoMLVideoTrainingJobRunOp',
]

AutoMLImageTrainingJobRunOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'automl_image_training_job/component.yaml',
    )
)

AutoMLTextTrainingJobRunOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'automl_text_training_job/component.yaml',
    )
)

AutoMLTabularTrainingJobRunOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'automl_tabular_training_job/component.yaml',
    )
)

AutoMLForecastingTrainingJobRunOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'automl_forecasting_training_job/component.yaml',
    )
)

AutoMLVideoTrainingJobRunOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'automl_video_training_job/component.yaml',
    )
)

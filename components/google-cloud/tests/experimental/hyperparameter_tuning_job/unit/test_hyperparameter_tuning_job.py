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
"""Test Hyperparameter Tuning Job module."""

from google.cloud.aiplatform import hyperparameter_tuning as hpt
from google_cloud_pipeline_components.experimental.hyperparameter_tuning_job import serialize_parameters, serialize_metrics
import unittest


class HyperparameterTuningJobTest(unittest.TestCase):

  def test_serialize_parameters(self):
    parameters = {
        'lr':
            hpt.DoubleParameterSpec(min=0.001, max=0.1, scale='log'),
        'units':
            hpt.IntegerParameterSpec(min=4, max=128, scale='linear'),
        'activation':
            hpt.CategoricalParameterSpec(values=['relu', 'selu']),
        'batch_size':
            hpt.DiscreteParameterSpec(values=[128, 256], scale='linear')
    }
    expected_outputs = [
        {
            'parameter_id': 'lr',
            'double_value_spec': {
                'min_value': 0.001,
                'max_value': 0.1
            },
            'scale_type': 2,
            'conditional_parameter_specs': []
        },
        {
            'parameter_id': 'units',
            'integer_value_spec': {
                'min_value': '4',
                'max_value': '128'
            },
            'scale_type': 1,
            'conditional_parameter_specs': []
        },
        {
            'parameter_id': 'activation',
            'categorical_value_spec': {
                'values': ['relu', 'selu']
            },
            'scale_type': 0,
            'conditional_parameter_specs': []
        },
        {
            'parameter_id': 'batch_size',
            'discrete_value_spec': {
                'values': [128.0, 256.0]
            },
            'scale_type': 1,
            'conditional_parameter_specs': []
        },
    ]

    outputs = serialize_parameters(parameters)
    self.assertEqual(outputs, expected_outputs)

  def test_serialize_metrics(self):
    metrics = {
        'loss': 'minimize',
        'accuracy': 'maximize',
    }
    expected_outputs = [
        {'metric_id': 'loss', 'goal': 2},
        {'metric_id': 'accuracy', 'goal': 1},
    ]

    outputs = serialize_metrics(metrics)
    self.assertEqual(outputs, expected_outputs)

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
from google_cloud_pipeline_components.experimental.hyperparameter_tuning_job import serialize_parameters
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
          '{\n  "parameterId": "lr",\n  "doubleValueSpec": {\n'
          '    "minValue": 0.001,\n    "maxValue": 0.1\n  },\n'
          '  "scaleType": 2,\n  "conditionalParameterSpecs": []\n}',
          '{\n  "parameterId": "units",\n  "integerValueSpec": {\n'
          '    "minValue": "4",\n    "maxValue": "128"\n  },\n'
          '  "scaleType": 1,\n  "conditionalParameterSpecs": []\n}',
          '{\n  "parameterId": "activation",\n  "categoricalValueSpec": {\n'
          '    "values": [\n      "relu",\n      "selu"\n    ]\n  },\n'
          '  "scaleType": 0,\n  "conditionalParameterSpecs": []\n}',
          '{\n  "parameterId": "batch_size",\n  "discreteValueSpec": {\n'
          '    "values": [\n      128.0,\n      256.0\n    ]\n  },\n'
          '  "scaleType": 1,\n  "conditionalParameterSpecs": []\n}',
        ]

        outputs = serialize_parameters(parameters)
        self.assertEqual(outputs, expected_outputs)

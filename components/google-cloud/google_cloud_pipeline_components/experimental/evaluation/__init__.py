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

try:
  from kfp.v2.components import load_component_from_file
except ImportError:
  from kfp.components import load_component_from_file

__all__ = [
    'ModelEvaluationOp', 'ModelImportEvaluationOp', 'EvaluationDataSamplerOp',
    'EvaluationDataSplitterOp', 'ModelEvaluationFeatureAttributionOp'
]

ModelEvaluationOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'component.yaml'))

ModelImportEvaluationOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'import_model_evaluation.yaml'))

EvaluationDataSamplerOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'evaluation_data_sampler.yaml'))

EvaluationDataSplitterOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'evaluation_data_splitter.yaml'))

ModelEvaluationFeatureAttributionOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'feature_attribution.yaml'))

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
"""Module for AutoML Tables KFP components."""

import os

try:
  from kfp.v2.components import load_component_from_file
except ImportError:
  from kfp.components import load_component_from_file

__all__ = [
    'CvTrainerOp',
    'InfraValidatorOp',
    'Stage1TunerOp',
    'EnsembleOp',
    'StatsAndExampleGenOp',
    'FeatureSelectionOp',
    'TransformOp',
    'FinalizerOp',
]

CvTrainerOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'cv_trainer.yaml'))
InfraValidatorOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'infra_validator.yaml'))
Stage1TunerOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'stage_1_tuner.yaml'))
EnsembleOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'ensemble.yaml'))
StatsAndExampleGenOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'stats_and_example_gen.yaml'))
FeatureSelectionOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'feature_selection.yaml'))
TransformOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'transform.yaml'))
FinalizerOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'finalizer.yaml'))

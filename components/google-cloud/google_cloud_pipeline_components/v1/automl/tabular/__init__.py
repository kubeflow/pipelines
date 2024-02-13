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

"""GA AutoML tabular components."""

import os

from google_cloud_pipeline_components.v1.automl.tabular.cv_trainer import automl_tabular_cv_trainer as CvTrainerOp
from google_cloud_pipeline_components.v1.automl.tabular.ensemble import automl_tabular_ensemble as EnsembleOp
from google_cloud_pipeline_components.v1.automl.tabular.finalizer import automl_tabular_finalizer as FinalizerOp
from google_cloud_pipeline_components.v1.automl.tabular.infra_validator import automl_tabular_infra_validator as InfraValidatorOp
from google_cloud_pipeline_components.v1.automl.tabular.split_materialized_data import split_materialized_data as SplitMaterializedDataOp
from google_cloud_pipeline_components.v1.automl.tabular.stage_1_tuner import automl_tabular_stage_1_tuner as Stage1TunerOp
from google_cloud_pipeline_components.v1.automl.tabular.stats_and_example_gen import tabular_stats_and_example_gen as StatsAndExampleGenOp
from google_cloud_pipeline_components.v1.automl.tabular.training_configurator_and_validator import training_configurator_and_validator as TrainingConfiguratorAndValidatorOp
from google_cloud_pipeline_components.v1.automl.tabular.transform import automl_tabular_transform as TransformOp
from google_cloud_pipeline_components.v1.automl.tabular.utils import get_automl_tabular_pipeline_and_parameters
from kfp import components

__all__ = [
    'CvTrainerOp',
    'EnsembleOp',
    'FinalizerOp',
    'InfraValidatorOp',
    'SplitMaterializedDataOp',
    'Stage1TunerOp',
    'StatsAndExampleGenOp',
    'TrainingConfiguratorAndValidatorOp',
    'TransformOp',
    'get_automl_tabular_pipeline_and_parameters',
]

automl_tabular_pipeline = components.load_component_from_file(
    # Note, please don't name it as `component.yaml` which will conflict with
    # the generated file.
    os.path.join(os.path.dirname(__file__), 'automl_tabular_pipeline.yaml')
)

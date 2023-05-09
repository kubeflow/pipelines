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

import unittest


class TestAutomlTabular(unittest.TestCase):

  def test_experimental_import_path(self):
    from google_cloud_pipeline_components.v1.automl.tabular import (
        CvTrainerOp,
        InfraValidatorOp,
        Stage1TunerOp,
        EnsembleOp,
        StatsAndExampleGenOp,
        FeatureSelectionOp,
        TransformOp,
        FinalizerOp,
        WideAndDeepHyperparameterTuningJobOp,
        WideAndDeepTrainerOp,
        TabNetHyperparameterTuningJobOp,
        TabNetTrainerOp,
        FeatureTransformEngineOp,
        SplitMaterializedDataOp,
        TrainingConfiguratorAndValidatorOp,
        XGBoostHyperparameterTuningJobOp,
        XGBoostTrainerOp,
    )

    print(CvTrainerOp)
    print(InfraValidatorOp)
    print(Stage1TunerOp)
    print(EnsembleOp)
    print(StatsAndExampleGenOp)
    print(FeatureSelectionOp)
    print(TransformOp)
    print(FinalizerOp)
    print(WideAndDeepHyperparameterTuningJobOp)
    print(WideAndDeepTrainerOp)
    print(TabNetHyperparameterTuningJobOp)
    print(TabNetTrainerOp)
    print(FeatureTransformEngineOp)
    print(SplitMaterializedDataOp)
    print(TrainingConfiguratorAndValidatorOp)
    print(XGBoostHyperparameterTuningJobOp)
    print(XGBoostTrainerOp)

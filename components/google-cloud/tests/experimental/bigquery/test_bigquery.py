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


class TestBigquery(unittest.TestCase):

  def test_experimental_import_path(self):
    from google_cloud_pipeline_components.experimental.bigquery import (
        BigqueryQueryJobOp,
        BigqueryCreateModelJobOp,
        BigqueryExportModelJobOp,
        BigqueryMLArimaEvaluateJobOp,
        BigqueryPredictModelJobOp,
        BigqueryEvaluateModelJobOp,
        BigqueryMLArimaCoefficientsJobOp,
        BigqueryMLWeightsJobOp,
        BigqueryMLReconstructionLossJobOp,
        BigqueryMLTrialInfoJobOp,
        BigqueryMLTrainingInfoJobOp,
        BigqueryExplainPredictModelJobOp,
        BigqueryExplainForecastModelJobOp,
        BigqueryMLAdvancedWeightsJobOp,
        BigqueryDropModelJobOp,
        BigqueryMLCentroidsJobOp,
        BigqueryMLConfusionMatrixJobOp,
        BigqueryMLFeatureInfoJobOp,
        BigqueryMLRocCurveJobOp,
        BigqueryMLPrincipalComponentsJobOp,
        BigqueryMLPrincipalComponentInfoJobOp,
        BigqueryMLFeatureImportanceJobOp,
        BigqueryMLRecommendJobOp,
        BigqueryForecastModelJobOp,
        BigqueryForecastModelJobOp,
        BigqueryMLGlobalExplainJobOp,
        BigqueryDetectAnomaliesModelJobOp,
    )
    # use variable to avoid removal of unused imports by linting tools
    print(BigqueryQueryJobOp)
    print(BigqueryCreateModelJobOp)
    print(BigqueryExportModelJobOp)
    print(BigqueryMLArimaEvaluateJobOp)
    print(BigqueryPredictModelJobOp)
    print(BigqueryEvaluateModelJobOp)
    print(BigqueryMLArimaCoefficientsJobOp)
    print(BigqueryMLWeightsJobOp)
    print(BigqueryMLReconstructionLossJobOp)
    print(BigqueryMLTrialInfoJobOp)
    print(BigqueryMLTrainingInfoJobOp)
    print(BigqueryExplainPredictModelJobOp)
    print(BigqueryExplainForecastModelJobOp)
    print(BigqueryMLAdvancedWeightsJobOp)
    print(BigqueryDropModelJobOp)
    print(BigqueryMLCentroidsJobOp)
    print(BigqueryMLConfusionMatrixJobOp)
    print(BigqueryMLFeatureInfoJobOp)
    print(BigqueryMLRocCurveJobOp)
    print(BigqueryMLPrincipalComponentsJobOp)
    print(BigqueryMLPrincipalComponentInfoJobOp)
    print(BigqueryMLFeatureImportanceJobOp)
    print(BigqueryMLRecommendJobOp)
    print(BigqueryForecastModelJobOp)
    print(BigqueryForecastModelJobOp)
    print(BigqueryMLGlobalExplainJobOp)
    print(BigqueryDetectAnomaliesModelJobOp)

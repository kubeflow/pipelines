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


class TestEvaluation(unittest.TestCase):

  def test_experimental_import_path(self):
    from google_cloud_pipeline_components.experimental.evaluation import (
        ModelImportEvaluationOp,
        EvaluationDataSamplerOp,
        TargetFieldDataRemoverOp,
        ModelEvaluationClassificationOp,
        ModelEvaluationRegressionOp,
        ModelEvaluationForecastingOp,
        ModelEvaluationFeatureAttributionOp,
    )
    # use variable to avoid removal of unused imports by linting tools
    print(ModelImportEvaluationOp)
    print(EvaluationDataSamplerOp)
    print(TargetFieldDataRemoverOp)
    print(ModelEvaluationClassificationOp)
    print(ModelEvaluationRegressionOp)
    print(ModelEvaluationForecastingOp)
    print(ModelEvaluationFeatureAttributionOp)

    from google_cloud_pipeline_components.experimental.evaluation.error_analysis import (
        EvaluationDatasetPreprocessorOp,
        EvaluatedAnnotationOp,
        FeatureExtractorOp,
        ErrorAnalysisAnnotationOp,
        ModelImportEvaluatedAnnotationOp,
    )

    # use variable to avoid removal of unused imports by linting tools
    print(EvaluationDatasetPreprocessorOp)
    print(EvaluatedAnnotationOp)
    print(FeatureExtractorOp)
    print(ErrorAnalysisAnnotationOp)
    print(ModelImportEvaluatedAnnotationOp)

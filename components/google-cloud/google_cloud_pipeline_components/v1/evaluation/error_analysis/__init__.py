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
"""Google Cloud Pipeline Model Evaluation Error Analysis components."""

import os

from .feature_extractor import component as feature_extractor
from .dataset_preprocessor import component as dataset_preprocessor
from .error_analysis_annotation import component as error_analysis_annotation

from kfp.components import load_component_from_file

__all__ = [
    'EvaluationDatasetPreprocessorOp',
    'EvaluatedAnnotationOp',
    'FeatureExtractorOp',
    'ErrorAnalysisAnnotationOp',
    'ModelImportEvaluatedAnnotationOp',
]

EvaluationDatasetPreprocessorOp = (
    dataset_preprocessor.dataset_preprocessor_error_analysis
)

EvaluatedAnnotationOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'evaluated_annotation/component.yaml',
    )
)

FeatureExtractorOp = feature_extractor.feature_extractor_error_analysis

ErrorAnalysisAnnotationOp = error_analysis_annotation.error_analysis_annotation

ModelImportEvaluatedAnnotationOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_evaluated_annotation/component.yaml'
    )
)

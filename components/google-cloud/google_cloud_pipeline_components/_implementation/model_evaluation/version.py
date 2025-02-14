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
"""Version constants for model evaluation components."""

_EVAL_VERSION = 'v0.9.4'
_LLM_EVAL_VERSION = 'v0.7'

_EVAL_IMAGE_NAME = 'gcr.io/ml-pipeline/model-evaluation'
_LLM_EVAL_IMAGE_NAME = 'gcr.io/ml-pipeline/llm-model-evaluation'


EVAL_IMAGE_TAG = f'{_EVAL_IMAGE_NAME}:{_EVAL_VERSION}'
LLM_EVAL_IMAGE_TAG = f'{_LLM_EVAL_IMAGE_NAME}:{_LLM_EVAL_VERSION}'

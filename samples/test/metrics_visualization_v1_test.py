# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from pprint import pprint
from .metrics_visualization_v1 import metrics_visualization_v1_pipeline
from kfp.samples.test.utils import run_pipeline_func, TestCase

import kfp

run_pipeline_func([
    TestCase(
        pipeline_func=metrics_visualization_v1_pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    )
])

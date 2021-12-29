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

import kfp
from .placeholder_if import pipeline_both, pipeline_none
# from .placeholder_if_v2 import pipeline_both as pipeline_both_v2, pipeline_none as pipeline_none_v2
from kfp.samples.test.utils import run_pipeline_func, TestCase

run_pipeline_func([
    # TODO(chesu): fix compile failure, https://github.com/kubeflow/pipelines/issues/6966
    # TestCase(
    #     pipeline_func=pipeline_none_v2,
    #     mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE),
    # TestCase(
    #     pipeline_func=pipeline_both_v2,
    #     mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE),
    TestCase(
        pipeline_func=pipeline_none,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY),
    TestCase(
        pipeline_func=pipeline_both,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY),
])

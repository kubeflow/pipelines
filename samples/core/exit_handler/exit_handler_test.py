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
from .exit_handler import pipeline_exit_handler
from ...test.util import run_pipeline_func, TestCase

run_pipeline_func([
    TestCase(
        pipeline_func=pipeline_exit_handler,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    ),
    # TODO(v2-compatible): The following test case fails with:
    #   File "/Users/gongyuan/kfp/pipelines/sdk/python/kfp/compiler/compiler.py", line 678, in _create_dag_templates
    #     launcher_image=self._launcher_image)
    #   File "/Users/gongyuan/kfp/pipelines/sdk/python/kfp/compiler/v2_compat.py", line 108, in update_op
    #     artifact_info = {"fileInputPath": op.input_artifact_paths[artifact_name]}
    # KeyError: 'GCS path'
    #
    # TestCase(
    #     pipeline_func=pipeline_exit_handler,
    #     mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
    # ),
])

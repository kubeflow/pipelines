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
from .xgboost_sample import xgboost_pipeline
from ...test.util import run_pipeline_func, TestCase, NEEDS_A_FIX

run_pipeline_func([
    # TODO(v2-compatible): fix this sample for v2 mode.
    # V2_COMPATIBLE mode fails with:
    # File "/Users/gongyuan/kfp/pipelines/sdk/python/kfp/compiler/v2_compat.py", line 108, in update_op
    # artifact_info = {"fileInputPath": op.input_artifact_paths[artifact_name]}
    # KeyError: 'text'
    #
    # And another error:
    # This step is in Error state with this message: withParam value could not
    # be parsed as a JSON list:
    # {
    #   "id":"733",
    #   "typeId":"114",
    #   "uri":"gs://gongyuan-test/kfp/output/pipeline-with-loop-parameter/pipeline-with-loop-parameter-pn8nj/pipeline-with-loop-parameter-pn8nj-1197421549/data",
    #   "createTimeSinceEpoch":"1617689193656",
    #   "lastUpdateTimeSinceEpoch":"1617689193656"
    # }
    TestCase(
        pipeline_func=xgboost_pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
        verify_func=NEEDS_A_FIX,
    ),
    TestCase(
        pipeline_func=xgboost_pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    ),
])

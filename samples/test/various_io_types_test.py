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
"""Two step v2-compatible pipeline."""

import kfp
from .various_io_types import my_pipeline
from .util import run_pipeline_func, TestCase


def verify(run, run_id: str, **kwargs):
    assert run.status == 'Succeeded'
    # TODO(Bobgy): verify MLMD status


run_pipeline_func([
    # Currently fails with:
    # sing_rewriter.py", line 520, in _refactor_inputs_if_uri_placeholder
    #     container_template['container']['args'])
    #   File "/Users/gongyuan/kfp/pipelines/sdk/python/kfp/compiler/_data_passing_rewriter.py", line 510, in reconcile_filename
    #     'supported.' % artifact_input['name'])
    # RuntimeError: Cannot find input3 in output to file name mapping.Please note currently connecting URI placeholder with path placeholder is not supported.
    # TestCase(
    #     pipeline_func=my_pipeline,
    #     verify_func=verify,
    #     mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY
    # ),

    # Currently fails with:
    # RuntimeError: Internal compiler error: Compiler has produced Argo-incompatible workflow.
    # Please create a new issue at https://github.com/kubeflow/pipelines/issues attaching the pipeline code and the pipeline package.
    # Error: time="2021-04-06T16:50:06.048Z" level=error msg="Error in file /dev/stdin: templates.pipeline-with-various-types inputs.parameters.input_3-producer-pod-id- was not supplied"
    # time="2021-04-06T16:50:06.048Z" level=fatal msg="Errors encountered in validation"
    # TestCase(pipeline_func=my_pipeline, verify_func=verify),
])

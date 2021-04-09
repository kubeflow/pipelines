# Copyright 2021 Google LLC
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
from .simple_pipeline_without_importer import my_pipeline
from .util import run_pipeline_func, TestCase


def verify(run, run_id: str):
    assert run.status == 'Succeeded'
    # TODO(Bobgy): verify MLMD status


run_pipeline_func([
    # Both cases fail with:
    # RuntimeError: Internal compiler error: Compiler has produced Argo-incompatible workflow.
    # Please create a new issue at https://github.com/kubeflow/pipelines/issues attaching the pipeline code and the pipeline package.
    # Error: time="2021-04-06T16:57:32.165Z" level=error msg="Error in file /dev/stdin: templates.simple-two-step-pipeline inputs.parameters.pipeline-output-directory was not supplied"
    # time="2021-04-06T16:57:32.165Z" level=fatal msg="Errors encountered in validation"
    # TestCase(
    #     pipeline_func=my_pipeline,
    #     verify_func=verify,
    #     mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    # ),
    # TestCase(pipeline_func=my_pipeline),
])

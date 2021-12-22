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
from .parameter_with_format import my_pipeline
from kfp.samples.test.utils import run_pipeline_func, TestCase


def verify(run, run_id: str, **kwargs):
    assert run.status == 'Succeeded'
    # TODO(Bobgy): verify output


run_pipeline_func([
    # Cannot test V2_ENGINE and V1_LEGACY using the same code.
    # V2_ENGINE requires importing everything from v2 namespace.
    # TestCase(
    #     pipeline_func=my_pipeline,
    #     verify_func=verify,
    #     mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
    # ),
    TestCase(
        pipeline_func=my_pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    ),
])

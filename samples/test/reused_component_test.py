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
from .reused_component import my_pipeline
from .util import run_pipeline_func, TestCase, NEEDS_A_FIX


def verify(run, run_id: str, **kwargs):
    assert run.status == 'Succeeded'
    # TODO(Bobgy): verify MLMD status


run_pipeline_func([
    TestCase(
        pipeline_func=my_pipeline,
        verify_func=verify,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    ),
    # TODO(v2-compatibility): fix this
    # Current failure:
    # main.go:56] Failed to successfuly execute component: %vstrconv.ParseInt: parsing "5\n": invalid syntax
    TestCase(pipeline_func=my_pipeline, verify_func=NEEDS_A_FIX),
])

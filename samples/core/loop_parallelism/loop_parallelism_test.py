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
from .loop_parallelism import pipeline
from ...test.util import run_pipeline_func, TestCase

run_pipeline_func([
    TestCase(
        pipeline_func=pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    ),

    # TODO(v2-compatible): one pod fails randomly with this error
    # F0414 13:49:21.024015       1 main.go:56] Failed to successfuly execute component: %vrpc error: code = AlreadyExists desc = Given node already exists: type_id: 57
    # name: "my-pipeline"
    # Internal: mysql_query failed: errno: 1062, error: Duplicate entry '57-my-pipeline' for key 'type_id'
    # goroutine 1 [running]:

    # TODO(v2-compatible): two pods fail stably at the following op:
    # print_op(item)
    #
    # Error message:
    # F0414 13:52:28.364169       1 main.go:52] Failed to create component launcher: %vinvalid character 'A' after object key:value pair
    # goroutine 1 [running]:
    #
    # The following was the runtime info, the problem was that one parameter
    # value contains a JSON string, but we didn't escape it.
    #
    # {"inputParameters": {"s": {"parameterType": "STRING",
    # "parameterValue": "{"A_a":1,"B_b":2}"}}, "inputArtifacts": {},
    # "outputParameters": {}, "outputArtifacts": {}}
    # TestCase(
    #     pipeline_func=pipeline,
    #     mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
    # ),
])

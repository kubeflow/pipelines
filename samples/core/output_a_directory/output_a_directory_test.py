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
from .output_a_directory import dir_pipeline
from ...test.util import run_pipeline_func, TestCase

run_pipeline_func([
    # TestCase(
    #     pipeline_func=dir_pipeline,
    #     mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    # ),

    # TODO(v2-compatible): error when the pipeline does not have a @pipeline.
    # Failed to create component launcher: %vMust specify PipelineName
    #
    # We should document that @pipeline decorator and pipeline name is required
    # for v2 compatible pipelines.
    #
    # TODO(v2-compatible): error after adding @pipeline decorator
    #
    # 1 main.go:56] Failed to successfuly execute component: %vread /tmp/kfp_launcher_outputs/output_dir/data: is a directory
    #
    TestCase(
        pipeline_func=dir_pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
    ),
])

#!/usr/bin/env python3
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
from .legacy_exit_handler import download_and_print
from .util import run_pipeline_func, TestCase


def verify(run, run_id: str, **kwargs):
    assert run.status == 'Succeeded'
    # TODO(Bobgy): verify exit handler pod ran


run_pipeline_func([
    TestCase(
        pipeline_func=download_and_print,
        verify_func=verify,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY
    ),
    # TODO(v2-compatible): fails with the following
    #   File "/Users/gongyuan/kfp/pipelines/sdk/python/kfp/compiler/compiler.py", line 678, in _create_dag_templates
    #     launcher_image=self._launcher_image)
    #   File "/Users/gongyuan/kfp/pipelines/sdk/python/kfp/compiler/v2_compat.py", line 86, in update_op
    #     op.arguments = list(op.container_spec.command) + list(op.container_spec.args)
    # AttributeError: 'NoneType' object has no attribute 'command'
    # TestCase(
    #     pipeline_func=download_and_print,
    #     verify_func=verify,
    #     mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE
    # ),
])

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
"""Test pipeline for bug report https://github.com/kubeflow/pipelines/issues/5830"""

import kfp
from kfp import dsl
from kfp.components import create_component_from_func


@create_component_from_func
def test(json_param: str):
    print(json_param)


@dsl.pipeline(name='parameter-values')
def param_values_pipeline(
    json_param: str = '{"a":"b"}',
    value: str = 'c',
):
    test(json_param).set_display_name('test1')
    test('{"a":"' + f'{value}' + '"}').set_display_name('test2')
    test("").set_display_name('test3')


if __name__ == '__main__':
    import os
    kfp.Client().create_run_from_pipeline_func(
        param_values_pipeline,
        arguments={},
        mode=dsl.PipelineExecutionMode.V2_COMPATIBLE,
        launcher_image=os.environ.get("KFP_LAUNCHER_IMAGE"),
    )

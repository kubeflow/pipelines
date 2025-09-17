# Copyright 2020 The Kubeflow Authors
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
import os
import pathlib

from kfp import compiler
from kfp import components
from kfp import dsl
from sdk.python.test.test_utils.file_utils import FileUtils

test_data_dir = FileUtils.TEST_DATA
add_op = components.load_component_from_file(
    str(os.path.join(test_data_dir, "pipeline_files", "valid", "critical",'add_numbers.yaml')))


@dsl.pipeline(name='add-pipeline')
def my_pipeline(
    a: int = 2,
    b: int = 5,
):
    first_add_task = add_op(a=a, b=3)
    second_add_task = add_op(a=first_add_task.outputs['Output'], b=b)
    third_add_task = add_op(a=second_add_task.outputs['Output'], b=7)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))

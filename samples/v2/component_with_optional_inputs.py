# Copyright 2023 The Kubeflow Authors
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
from typing import Optional, Dict, List

from kfp import compiler
from kfp import dsl
from kfp.dsl import component


@component
def component_op(
    input_str1: Optional[str] = 'string default value',
    input_str2: Optional[str] = None,
    input_str3: Optional[str] = None,
    input_bool1: Optional[bool] = True,
    input_bool2: Optional[bool] = None,
    input_dict: Optional[Dict[str, int]] = {"a": 1},
    input_list: Optional[List[str]] = ["123"],
    input_int: Optional[int] = 100,
):
    print(f'input_str1: {input_str1}, type: {type(input_str1)}')
    print(f'input_str2: {input_str2}, type: {type(input_str2)}')
    print(f'input_str3: {input_str3}, type: {type(input_str3)}')
    print(f'input_bool1: {input_bool1}, type: {type(input_bool1)}')
    print(f'input_bool2: {input_bool2}, type: {type(input_bool2)}')
    print(f'input_bool: {input_dict}, type: {type(input_dict)}')
    print(f'input_bool: {input_list}, type: {type(input_list)}')
    print(f'input_bool: {input_int}, type: {type(input_int)}')


@dsl.pipeline(name='v2-component-optional-input')
def pipeline():
    component_op(
        input_str1='Hello',
        input_str2='World',
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace('.py', '.yaml'))

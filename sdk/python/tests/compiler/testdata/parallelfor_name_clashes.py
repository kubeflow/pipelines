#!/usr/bin/env python3
# Copyright 2020 Google LLC
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

from typing import NamedTuple

import kfp
from kfp.components import func_to_container_op

@func_to_container_op
def produce_str() -> str:
    return "Hello"

@func_to_container_op
def produce_list() -> list:
    return ["1", "2"]

@func_to_container_op
def consume(param1):
    print(param1)

@kfp.dsl.pipeline()
def parallelfor_name_clashes_pipeline():
    produce_str_task = produce_str()
    produce_list_task = produce_list()
    with kfp.dsl.ParallelFor(produce_list_task.output) as loop_item:
        consume(produce_list_task.output)
        consume(produce_str_task.output)
        consume(loop_item)
        consume(loop_item.aaa)


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(parallelfor_name_clashes_pipeline, __file__ + '.yaml')


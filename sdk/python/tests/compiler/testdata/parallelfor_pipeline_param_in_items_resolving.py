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

import kfp
from kfp.components import func_to_container_op


@func_to_container_op
def produce_message(fname1: str) -> str:
    return "My name is %s" % fname1


@func_to_container_op
def consume(param1):
    print(param1)


@kfp.dsl.pipeline()
def parallelfor_pipeline_param_in_items_resolving(fname1: str, fname2: str):

    simple_list = ["My name is %s" % fname1,
                   "My name is %s" % fname2]

    list_of_dict = [{"first_name": fname1, "message": "My name is %s" % fname1},
                    {"first_name": fname2, "message": "My name is %s" % fname2}]

    list_of_complex_dict = [
        {"first_name": fname1, "message": produce_message(fname1).output},
        {"first_name": fname2, "message": produce_message(fname2).output}]

    with kfp.dsl.ParallelFor(simple_list) as loop_item:
        consume(loop_item)

    with kfp.dsl.ParallelFor(list_of_dict) as loop_item2:
        consume(loop_item2.first_name)
        consume(loop_item2.message)

    with kfp.dsl.ParallelFor(list_of_complex_dict) as loop_item:
        consume(loop_item.first_name)
        consume(loop_item.message)


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(parallelfor_pipeline_param_in_items_resolving, __file__ + '.yaml')

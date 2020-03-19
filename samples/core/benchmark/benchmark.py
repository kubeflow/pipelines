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


import random
import kfp
from kfp.components import create_component_from_func


@create_component_from_func
def add_op(a: float, b: float) -> float:
    return a + b


def generated_pipeline():
    """Generates a pipeline with randomly connected component graph."""
    tasks = [add_op(3, 5)]
    for _ in range(20):
        a = random.choice(tasks).output
        b = random.choice(tasks).output
        task = add_op(a, b)
        tasks.append(task)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(generated_pipeline, __file__ + '.yaml')

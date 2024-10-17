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

from kfp import compiler, dsl
from typing import List


@dsl.component
def args_generator_op() -> List[str]:
    return ['1.1', '1.2', '1.3']


# TODO(Bobgy): how can we make this component with type float?
# got error: kfp.dsl.types.type_utils.InconsistentTypeException:
# Incompatible argument passed to the input "s" of component "Print op": Argument
# type "STRING" is incompatible with the input type "NUMBER_DOUBLE"
@dsl.component
def print_op(s: str):
    print(s)


@dsl.pipeline(name='pipeline-with-loop-output-v2')
def my_pipeline():
    args_generator = args_generator_op()
    with dsl.ParallelFor(args_generator.output) as item:
        print_op(s=item)

if __name__ == '__main__':
    compiler.Compiler().compile(my_pipeline, __file__ + '.yaml')

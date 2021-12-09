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

import os
from kfp.v2 import dsl

# In tests, we install a KFP package from the PR under test. Users should not
# normally need to specify `kfp_package_path` in their component definitions.
_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')


@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def args_generator_op() -> str:
    return '[1.1, 1.2, 1.3]'


# TODO(Bobgy): how can we make this component with type float?
# got error: kfp.v2.components.types.type_utils.InconsistentTypeException:
# Incompatible argument passed to the input "s" of component "Print op": Argument
# type "STRING" is incompatible with the input type "NUMBER_DOUBLE"
@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def print_op(s: str):
    print(s)


@dsl.pipeline(name='pipeline-with-loop-output-v2')
def my_pipeline():
    args_generator = args_generator_op()
    with dsl.ParallelFor(args_generator.output) as item:
        print_op(s=item)

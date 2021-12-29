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
def flip_coin(force_flip_result: str = '') -> str:
    """Flip a coin and output heads or tails randomly."""
    if force_flip_result:
        return force_flip_result
    import random
    result = 'heads' if random.randint(0, 1) == 0 else 'tails'
    return result


@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def print_msg(msg: str):
    """Print a message."""
    print(msg)


@dsl.pipeline(name='condition-v2')
def condition(text: str = 'condition test', force_flip_result: str = ''):
    flip1 = flip_coin(force_flip_result=force_flip_result)
    print_msg(msg=flip1.output)

    with dsl.Condition(flip1.output == 'heads'):
        flip2 = flip_coin()
        print_msg(msg=flip2.output)
        print_msg(msg=text)

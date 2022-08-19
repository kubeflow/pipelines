# Copyright 2022 The Kubeflow Authors
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
import typing
from typing import NamedTuple

from kfp import compiler
from kfp import dsl

# In tests, we install a KFP package from the PR under test. Users should not
# normally need to specify `kfp_package_path` in their component definitions.
_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')


@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def create_named_tuple_from_module(
) -> typing.NamedTuple('Output', [('name', str), ('id', int)]):
    import typing
    nt = typing.NamedTuple('Output', [('name', str), ('id', int)])
    return nt(name='name', id=1)


@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def create_named_tuple_from_class_factory(
) -> NamedTuple('Output', [('name', str), ('id', int)]):
    from typing import NamedTuple
    nt = NamedTuple('Output', [('name', str), ('id', int)])
    return nt(name='name', id=1)


@dsl.pipeline()
def pipeline_with_named_tuples():
    create_named_tuple_from_module()
    create_named_tuple_from_class_factory()


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_named_tuples,
        package_path='pipeline_with_named_tuples.json')

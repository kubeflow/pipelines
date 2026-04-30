# Copyright 2026 The Kubeflow Authors
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
"""Pipeline exercising `dsl.OneOf` for local-execution regression tests.

`dsl.OneOf` picks the output from whichever conditional branch actually
executed. The pipeline returns that value so the test can assert which
branch ran.
"""
from kfp import compiler
from kfp import dsl


@dsl.component
def branch_a() -> str:
    return 'A'


@dsl.component
def branch_b() -> str:
    return 'B'


@dsl.component
def tag_selected(value: str) -> str:
    return f'selected:{value}'


@dsl.pipeline
def pipeline_with_oneof(pick_a: bool = True) -> str:
    with dsl.If(pick_a == True):
        a = branch_a()
    with dsl.Else():
        b = branch_b()
    tagged = tag_selected(value=dsl.OneOf(a.output, b.output))
    return tagged.output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_oneof,
        package_path=__file__.replace('.py', '.yaml'),
    )

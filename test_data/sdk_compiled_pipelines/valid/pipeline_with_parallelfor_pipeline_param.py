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
"""ParallelFor pipelines that propagate pipeline parameters through sub-DAGs.

These pipelines verify that pipeline-level parameters are correctly surfaced
through ParallelFor sub-DAG boundaries and accessible to components inside
the loop, alongside the loop item variable.
"""

from typing import List

from kfp import compiler
from kfp import dsl
from kfp.dsl import component


@component
def print_with_prefix(prefix: str, item: str) -> str:
    result = f'{prefix}: {item}'
    print(result)
    return result


@component
def make_prefix() -> str:
    return 'dynamic-prefix'


@dsl.pipeline(name='pipeline-parallelfor-pipeline-param')
def my_pipeline(prefix: str, loop_parameter: List[str]):
    """Pipeline parameter used alongside loop item inside ParallelFor."""

    # Pipeline parameter propagated into ParallelFor sub-DAG
    with dsl.ParallelFor(items=loop_parameter, parallelism=1) as item:
        print_with_prefix(prefix=prefix, item=item)

    # Outer task output propagated into ParallelFor sub-DAG
    prefix_task = make_prefix()
    with dsl.ParallelFor(items=loop_parameter) as item:
        print_with_prefix(prefix=prefix_task.output, item=item)

    # Nested ParallelFor with pipeline parameter
    static_items = ['x', 'y']
    with dsl.ParallelFor(items=static_items, parallelism=2) as outer_item:
        with dsl.ParallelFor(items=loop_parameter) as inner_item:
            print_with_prefix(prefix=prefix, item=inner_item)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))

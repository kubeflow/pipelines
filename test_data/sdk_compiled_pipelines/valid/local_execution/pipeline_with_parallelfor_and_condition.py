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
"""Pipeline exercising dsl.ParallelFor + dsl.If together for the local-
execution regression suite.

A ParallelFor fans out a small fixed list, each iteration doubles its
value, and dsl.Collected aggregates the iteration outputs. The
aggregate then flows into a dsl.If / dsl.Else pair whose chosen branch
is selected via dsl.OneOf and returned as the pipeline output, so the
test asserts both that the ParallelFor iterations all ran and that the
correct conditional branch executed.

With the default threshold=5 the expected output is ``'high:12'``
(1+2+3 doubled and summed = 12, which is > 5).
"""
from typing import List

from kfp import compiler
from kfp import dsl


@dsl.component
def double(num: float) -> float:
    return num * 2.0


@dsl.component
def sum_values(nums: List[float]) -> float:
    return float(sum(nums))


@dsl.component
def label(prefix: str, value: float) -> str:
    # Format as int when the value is whole so the test's exact-match
    # assertion stays stable across runtime numeric round-trips.
    rendered = int(value) if float(value).is_integer() else value
    return f'{prefix}:{rendered}'


@dsl.component
def passthrough(value: str) -> str:
    return value


@dsl.pipeline
def pipeline_with_parallelfor_and_condition(threshold: float = 5.0) -> str:
    with dsl.ParallelFor([1.0, 2.0, 3.0]) as item:
        doubled = double(num=item)
    summed = sum_values(nums=dsl.Collected(doubled.output))

    with dsl.If(summed.output > threshold):
        high = label(prefix='high', value=summed.output)
    with dsl.Else():
        low = label(prefix='low', value=summed.output)

    chosen = passthrough(value=dsl.OneOf(high.output, low.output))
    return chosen.output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_parallelfor_and_condition,
        package_path=__file__.replace('.py', '.yaml'),
    )

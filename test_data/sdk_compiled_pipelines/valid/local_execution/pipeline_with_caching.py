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
"""Pipeline exercising `.set_caching_options()` for local-execution regression
tests.

Run twice against the same cache_root with identical inputs: the second
run should short-circuit via the local cache. The test observes this via
captured logs.
"""
from kfp import compiler
from kfp import dsl


@dsl.component
def add_one(x: int) -> int:
    return x + 1


@dsl.pipeline
def pipeline_with_caching(x: int = 1) -> int:
    task = add_one(x=x)
    task.set_caching_options(enable_caching=True)
    return task.output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_caching,
        package_path=__file__.replace('.py', '.yaml'),
    )

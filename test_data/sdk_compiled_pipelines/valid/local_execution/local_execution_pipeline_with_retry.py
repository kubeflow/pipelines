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
"""Pipeline exercising `.set_retry()` for local-execution regression tests.

The flaky component fails on its first invocation and succeeds on
subsequent ones, using a counter file passed in as an arg so the caller
controls uniqueness across runs.
"""
from kfp import compiler
from kfp import dsl


@dsl.component
def flaky(counter_path: str) -> str:
    import os
    count = 0
    if os.path.exists(counter_path):
        with open(counter_path) as f:
            count = int(f.read().strip() or '0')
    count += 1
    with open(counter_path, 'w') as f:
        f.write(str(count))
    if count <= 1:
        raise RuntimeError(f'flaky attempt {count}')
    return f'attempt={count}'


@dsl.pipeline
def pipeline_with_retry(counter_path: str) -> str:
    task = flaky(counter_path=counter_path)
    task.set_retry(num_retries=3, backoff_duration='0s', backoff_factor=1.0)
    return task.output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_retry,
        package_path=__file__.replace('.py', '.yaml'),
    )

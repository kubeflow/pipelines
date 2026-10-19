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
"""A pipeline whose component fails on its first attempt and succeeds on retry.

The backend compiler injects ``KFP_RETRY_INDEX`` (Argo's ``{{retries}}``) into
every executor that has a retry policy, so the failure is deterministic: the
first attempt (index 0) always fails and the retry (index 1) always succeeds.
Caching is disabled so a rerun exercises the retry again instead of hitting
the cache. The consumer checks the retried task's output, so the run only
succeeds when the retry attempt's output flowed downstream.

Used by backend/test/end2end/driver_plugin_e2e_test.go.
"""

from kfp import compiler
from kfp import dsl


@dsl.component
def flaky_add(a: float, b: float) -> float:
    import os
    attempt = int(os.environ.get('KFP_RETRY_INDEX') or '0')
    print(f'attempt index {attempt}')
    if attempt == 0:
        raise RuntimeError('failing on purpose on the first attempt')
    return a + b


@dsl.component
def check_sum(total: float, expected: float):
    if total != expected:
        raise ValueError(f'expected {expected}, got {total}')
    print(f'sum {total} matches {expected}')


@dsl.pipeline(name='pipeline-with-flaky-retry')
def my_pipeline(a: float = 1, b: float = 7):
    add_task = flaky_add(a=a, b=b)
    add_task.set_retry(num_retries=2)
    add_task.set_caching_options(False)
    check_task = check_sum(total=add_task.output, expected=8)
    check_task.set_caching_options(False)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))

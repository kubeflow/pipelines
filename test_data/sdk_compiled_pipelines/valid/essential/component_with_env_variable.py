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
"""Pipeline exercising `.set_env_variable()` for local-execution regression
tests.

Confirms that env vars configured on a task at compile time are
propagated to the underlying container at runtime (subprocess or
docker).
"""
from kfp import compiler
from kfp import dsl


@dsl.component
def read_env_var(var_name: str) -> str:
    import os
    return os.environ.get(var_name, 'UNSET')


@dsl.pipeline
def pipeline_with_env_variable(var_name: str = 'KFP_TEST_ENV_VAR') -> str:
    task = read_env_var(var_name=var_name)
    task.set_env_variable(name='KFP_TEST_ENV_VAR', value='env_var_value')
    return task.output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_env_variable,
        package_path=__file__.replace('.py', '.yaml'))

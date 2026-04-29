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
"""Pipeline exercising `@warn_if_final` for local-execution regression tests.

Kubernetes-only methods (set_cpu_limit, set_memory_limit,
set_accelerator_type, ...) are called on the task during local execution.
Locally they should emit a warning but still let the task run; the test
asserts both the warning and the expected output.
"""
from kfp import compiler
from kfp import dsl


@dsl.component
def echo(message: str) -> str:
    return message


@dsl.pipeline
def pipeline_with_k8s_only_methods(message: str = 'hello') -> str:
    task = echo(message=message)
    task.set_cpu_limit('1000m')
    task.set_memory_limit('256Mi')
    task.set_accelerator_type('nvidia.com/gpu')
    return task.output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_k8s_only_methods,
        package_path=__file__.replace('.py', '.yaml'),
    )

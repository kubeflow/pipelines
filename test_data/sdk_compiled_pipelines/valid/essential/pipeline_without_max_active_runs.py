# Copyright 2025 The Kubeflow Authors
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
"""A minimal pipeline that explicitly tests the "no parallelism" scenario where max_active_runs is not set."""

from kfp import compiler, dsl


@dsl.component(base_image='python:3.9')
def produce_message(msg: str) -> str:
    print(f'Processing {msg}...')
    return msg


@dsl.pipeline(
    name='pipeline-without-max-active-runs',
    # pipeline_config is not set, which means max_active_runs is None/unset
    # This explicitly tests the "no max_active_runs" scenario
)
def pipeline_without_max_active_runs():
    produce_message(msg='test')


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_without_max_active_runs,
        package_path=__file__.replace('.py', '.yaml'))


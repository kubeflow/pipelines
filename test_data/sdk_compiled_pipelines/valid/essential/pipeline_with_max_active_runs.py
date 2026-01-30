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
"""A minimal pipeline that customizes max_active_runs."""

from kfp import compiler, dsl


@dsl.component(base_image='python:3.9')
def produce_message(msg: str, sleep_seconds: int = 20) -> str:
    import time

    print(f'Processing {msg}...')
    time.sleep(sleep_seconds)
    return msg


@dsl.pipeline(
    name='pipeline-with-max-active-runs',
    pipeline_config=dsl.PipelineConfig(max_active_runs=2),
)
def pipeline_with_max_active_runs():
    loop_args = ['one', 'two', 'three', 'four', 'five']
    with dsl.ParallelFor(items=loop_args) as item:
        produce_message(msg=item)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_max_active_runs,
        package_path=__file__.replace('.py', '.yaml'))

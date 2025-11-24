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
"""A minimal pipeline that customizes pipeline_run_parallelism."""

from kfp import compiler, dsl


@dsl.component
def produce_message(msg: str) -> str:
    print(msg)
    return msg


@dsl.pipeline(
    name='pipeline-with-run-parallelism',
    pipeline_config=dsl.PipelineConfig(pipeline_run_parallelism=2),
)
def pipeline_with_run_parallelism():
    with dsl.ParallelFor(items=['one', 'two', 'three']) as item:
        produce_message(msg=item)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_run_parallelism,
        package_path=__file__.replace('.py', '.yaml'))

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
"""A minimal pipeline that explicitly tests the "no parallelism" scenario where pipeline_version_concurrency_limit is not set."""

from kfp import compiler, dsl


@dsl.component(base_image='python:3.9')
def produce_message(msg: str) -> str:
    print(f'Processing {msg}...')
    return msg


@dsl.pipeline(
    name='pipeline-with-no-concurrency-limit',
    # pipeline_config is not set, which means pipeline_version_concurrency_limit is None/unset
    # This explicitly tests the "no concurrency limit" scenario
)
def pipeline_with_no_concurrency_limit():
    produce_message(msg='test')


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_no_concurrency_limit,
        package_path=__file__.replace('.py', '.yaml'))


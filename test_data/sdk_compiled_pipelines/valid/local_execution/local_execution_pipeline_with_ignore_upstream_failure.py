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
"""Pipeline exercising `.ignore_upstream_failure()` for local-execution
regression tests.

The upstream task fails intentionally. The downstream task uses
`.ignore_upstream_failure()` so it still runs; it writes a marker file
the test checks to confirm execution. Callers must use
`raise_on_error=False` since the overall pipeline DAG still reports
failure.
"""
from kfp import compiler
from kfp import dsl


@dsl.component
def fail_always() -> str:
    raise RuntimeError('upstream failed intentionally')


@dsl.component
def write_marker(marker_path: str) -> str:
    with open(marker_path, 'w') as f:
        f.write('cleanup_ran')
    return 'cleanup_ran'


@dsl.pipeline
def pipeline_with_ignore_upstream_failure(marker_path: str):
    upstream = fail_always()
    cleanup = write_marker(marker_path=marker_path)
    cleanup.after(upstream).ignore_upstream_failure()


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_ignore_upstream_failure,
        package_path=__file__.replace('.py', '.yaml'),
    )

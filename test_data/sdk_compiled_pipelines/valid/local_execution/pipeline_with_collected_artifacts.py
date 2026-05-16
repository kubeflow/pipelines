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
"""Pipeline exercising `dsl.Collected` over artifacts (not just parameters) for
local-execution regression tests.

A ParallelFor fans out over integers, each iteration writes an artifact
whose body is the integer. A downstream task consumes the collected list
of artifacts, reads each, and returns the concatenation so the test has
a deterministic output to assert on.
"""
from typing import List

from kfp import compiler
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Output


@dsl.component
def write_artifact(index: int, out: Output[Artifact]):
    with open(out.path, 'w') as f:
        f.write(str(index))


@dsl.component
def read_and_join(artifacts: List[Artifact]) -> str:
    parts = []
    for artifact in artifacts:
        with open(artifact.path) as f:
            parts.append(f.read().strip())
    return ','.join(sorted(parts))


@dsl.pipeline
def pipeline_with_collected_artifacts() -> str:
    with dsl.ParallelFor([0, 1, 2]) as i:
        producer = write_artifact(index=i)
    reader = read_and_join(artifacts=dsl.Collected(producer.outputs['out']))
    return reader.output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_collected_artifacts,
        package_path=__file__.replace('.py', '.yaml'),
    )

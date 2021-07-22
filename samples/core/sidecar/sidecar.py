#!/usr/bin/env python3
# Copyright 2019, 2021 The Kubeflow Authors
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

import kfp
import kfp.dsl as dsl


@dsl.pipeline(
    name="pipeline-with-sidecar",
    description=
    "A pipeline that demonstrates how to add a sidecar to an operation."
)
def pipeline_with_sidecar():
    # sidecar with sevice that reply "hello world" to any GET request
    echo = dsl.Sidecar(
        name="echo",
        image="nginx:1.13",
        command=["nginx", "-g", "daemon off;"],
    )

    # container op with sidecar
    op1 = dsl.ContainerOp(
        name="download",
        image="busybox:latest",
        command=["sh", "-c"],
        arguments=[
            "until wget http://localhost:80 -O /tmp/results.txt; do sleep 5; done && cat /tmp/results.txt"
        ],
        sidecars=[echo],
        file_outputs={"downloaded": "/tmp/results.txt"},
    )


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(pipeline_with_sidecar, __file__ + '.yaml')

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
"""A failing run whose workflow node status is over Argo's size limit.

The test passes the same 30 KB parameter to every task. The driver puts
it into the task's pod-spec-patch, and Argo records that patch on four
nodes per task. Twenty tasks give about 2.5 MB of node status. The
controller stores it compressed, and retrying the run has to write it
back that way.
"""

from kfp import compiler
from kfp import dsl

TASK_COUNT = 20


@dsl.container_component
def carry(payload: str):
    return dsl.ContainerSpec(
        image='alpine:3.23',
        command=['sh', '-c', 'true'],
    )


@dsl.container_component
def fail():
    return dsl.ContainerSpec(
        image='alpine:3.23',
        command=['sh', '-c', 'exit 1'],
    )


@dsl.pipeline(name='argo-compatibility-large-status-retry')
def large_status_retry(payload: str):
    tasks = []
    for _ in range(TASK_COUNT):
        task = carry(payload=payload)
        task.set_caching_options(False)
        tasks.append(task)
    fail().set_caching_options(False).after(*tasks)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=large_status_retry,
        package_path=__file__.replace('.py', '.yaml'),
    )

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
"""A static 120-task DAG with bounded width and negligible task work."""

from kfp import compiler
from kfp import dsl


@dsl.container_component
def scale_node(node_id: int):
    return dsl.ContainerSpec(
        image='docker.io/alpine:3.23',
        command=['echo'],
        args=[node_id],
    )


@dsl.pipeline(name='dag-120')
def dag_120():
    previous_layer = []
    for layer in range(6):
        current_layer = []
        for column in range(20):
            node_id = layer * 20 + column
            task = scale_node(node_id=node_id).set_display_name(
                f'scale-node-{node_id:03d}').set_caching_options(False)
            if previous_layer:
                # Overlapping dependencies exercise both fan-in and fan-out.
                task.after(previous_layer[column],
                           previous_layer[(column + 1) % 20])
            current_layer.append(task)
        previous_layer = current_layer


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=dag_120, package_path=__file__.replace('.py', '.yaml'))

# Copyright 2024 The Kubeflow Authors
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
"""Example pipeline demonstrating PVC mount with subPath."""

from kfp import dsl
from kfp import kubernetes


@dsl.component
def write_to_models() -> None:
    """Writes data to the models subdirectory."""
    import os
    os.makedirs('/data', exist_ok=True)
    with open('/data/model.txt', 'w') as f:
        f.write('model data')
    print('Written model data')


@dsl.component
def write_to_logs() -> None:
    """Writes data to the logs subdirectory."""
    import os
    os.makedirs('/data', exist_ok=True)
    with open('/data/log.txt', 'w') as f:
        f.write('log data')
    print('Written log data')


@dsl.component
def read_from_models() -> None:
    """Reads data from the models subdirectory."""
    with open('/data/model.txt', 'r') as f:
        content = f.read()
    print(f'Read from models: {content}')


@dsl.component
def read_from_logs() -> None:
    """Reads data from the logs subdirectory."""
    with open('/data/log.txt', 'r') as f:
        content = f.read()
    print(f'Read from logs: {content}')


@dsl.pipeline(name='pvc-mount-subpath-pipeline')
def pvc_mount_subpath_pipeline(pvc_name: str):
    write_models_task = write_to_models()
    kubernetes.mount_pvc(
        write_models_task,
        pvc_name=pvc_name,
        mount_path='/data',
        sub_path='models',
    )

    write_logs_task = write_to_logs()
    kubernetes.mount_pvc(
        write_logs_task,
        pvc_name=pvc_name,
        mount_path='/data',
        sub_path='logs',
    )

    read_models_task = read_from_models().after(write_models_task)
    kubernetes.mount_pvc(
        read_models_task,
        pvc_name=pvc_name,
        mount_path='/data',
        sub_path='models',
    )

    read_logs_task = read_from_logs().after(write_logs_task)
    kubernetes.mount_pvc(
        read_logs_task,
        pvc_name=pvc_name,
        mount_path='/data',
        sub_path='logs',
    )


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pvc_mount_subpath_pipeline,
        'pvc_mount_subpath.yaml'
    )

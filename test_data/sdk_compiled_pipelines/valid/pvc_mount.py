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
"""Small pipeline that mounts an existing PVC (by name) and exercises caching."""

from kfp import dsl
from kfp import kubernetes


@dsl.component
def producer() -> str:
    with open('/data/file.txt', 'w') as f:
        f.write('hello')
    with open('/data/file.txt', 'r') as f:
        return f.read()


@dsl.component
def consumer() -> None:
    with open('/data/file.txt', 'r') as f:
        print(f.read())


@dsl.pipeline(name='pvc-mount-pipeline')
def pvc_mount_pipeline(pvc_name: str):
    p = producer()
    c = consumer().after(p)

    # Mount the provided PVC name to both tasks at /data
    kubernetes.mount_pvc(p, pvc_name=pvc_name, mount_path='/data')
    kubernetes.mount_pvc(c, pvc_name=pvc_name, mount_path='/data')


if __name__ == '__main__':
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=pvc_mount_pipeline,
        package_path=__file__.replace('.py', '.yaml'),
    )



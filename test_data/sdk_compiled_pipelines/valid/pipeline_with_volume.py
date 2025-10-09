# Copyright 2023 The Kubeflow Authors
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
"""Pipeline with volume creation, mount and deletion in v2 engine pipeline."""
from kfp import dsl
from kfp import kubernetes


@dsl.component
def producer() -> str:
    with open('/data/file.txt', 'w') as file:
        file.write('Hello world')
    with open('/data/file.txt', 'r') as file:
        content = file.read()
    print(content)
    return content


@dsl.component
def consumer() -> str:
    with open('/data/file.txt', 'r') as file:
        content = file.read()
    print(content)
    return content


@dsl.pipeline
def pipeline_with_volume():
    pvc1 = kubernetes.CreatePVC(
        pvc_name_suffix='-my-pvc',
        access_modes=['ReadWriteOnce'],
        size='5Mi',
        storage_class_name='standard',
    )

    task1 = producer()
    task2 = consumer().after(task1)

    kubernetes.mount_pvc(
        task1,
        pvc_name=pvc1.outputs['name'],
        mount_path='/data',
    )
    kubernetes.mount_pvc(
        task2,
        pvc_name=pvc1.outputs['name'],
        mount_path='/data',
    )

    delete_pvc1 = kubernetes.DeletePVC(
        pvc_name=pvc1.outputs['name']).after(task2)

if __name__ == '__main__':
    # execute only if run as a script
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_volume,
        package_path='pipeline_with_volume.json')
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

from kfp import dsl, compiler
from kfp import kubernetes

@dsl.component
def make_data():
    with open('/data/file.txt', 'w') as f:
        f.write('my data')

@dsl.component
def read_data():
    with open('/reused_data/file.txt') as f:
        print(f.read())

@dsl.pipeline(
    name="kubernetes-pvc-basic",
    description="A Basic Example on Kubernetes PVC Usage."
)
def my_pipeline():
    pvc1 = kubernetes.CreatePVC(
        # can also use pvc_name instead of pvc_name_suffix to use a pre-existing PVC
        pvc_name_suffix='-my-pvc',
        access_modes=['ReadWriteOnce'],
        size='5Gi',
        storage_class_name='standard',
    )

    task1 = make_data()
    # normally task sequencing is handled by data exchange via component inputs/outputs
    # but since data is exchanged via volume, we need to call .after explicitly to sequence tasks
    task2 = read_data().after(task1)

    kubernetes.mount_pvc(
        task1,
        pvc_name=pvc1.outputs['name'],
        mount_path='/data',
    )
    kubernetes.mount_pvc(
        task2,
        pvc_name=pvc1.outputs['name'],
        mount_path='/reused_data',
    )

    # wait to delete the PVC until after task2 completes
    delete_pvc1 = kubernetes.DeletePVC(
        pvc_name=pvc1.outputs['name']).after(task2)

if __name__ == '__main__':
    compiler.Compiler().compile(my_pipeline, __file__ + '.yaml')

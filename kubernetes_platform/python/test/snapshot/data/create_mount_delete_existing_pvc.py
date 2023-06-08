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

from kfp import dsl
from kfp import kubernetes


@dsl.component
def comp():
    pass


@dsl.pipeline
def my_pipeline():
    pvc1 = kubernetes.CreatePVC(
        pvc_name='static-pvc-name',
        access_modes=['ReadWriteOnce'],
        size='5Gi',
        storage_class_name='standard',
    )
    task1 = comp()

    kubernetes.mount_pvc(
        task1,
        pvc_name=pvc1.outputs['name'],
        mount_path='/data',
    )

    delete_pvc1 = kubernetes.DeletePVC(
        pvc_name=pvc1.outputs['name']).after(task1)


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(my_pipeline, __file__.replace('.py', '.yaml'))

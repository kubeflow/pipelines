# Copyright 2019 The Kubeflow Authors
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
from kfp import components

import kfp.dsl as dsl


@dsl.pipeline(name='volume')
def volume_pipeline():
    # create PVC and dynamically provision volume
    create_pvc_task = dsl.VolumeOp(
        name='create_volume',
        generate_unique_name=True,
        resource_name='pvc1',
        size='1Gi',
        storage_class='standard',
        modes=dsl.VOLUME_MODE_RWM,
    )

    # use PVC; write to file
    task_a = dsl.ContainerOp(
        name='step1_ingest',
        image='alpine',
        command=['sh', '-c'],
        arguments=[
            'mkdir /data/step1 && '
            'echo hello > /data/step1/file1.txt'
        ],
        pvolumes={'/data': create_pvc_task.volume},
    )

    # create a PVC from a pre-existing volume
    create_pvc_from_existing_vol = dsl.VolumeOp(
        name='pre_existing_volume',
        generate_unique_name=True,
        resource_name='pvc2',
        size='5Gi',
        storage_class='standard',
        volume_name='my-pre-existing-volume',
        modes=dsl.VOLUME_MODE_RWM,
    )

    # use the previous task's PVC and the newly created PVC
    task_b = dsl.ContainerOp(
        name='step2_gunzip',
        image='library/bash:4.4.23',
        command=['sh', '-c'],
        arguments=['cat /data/step1/file.txt'],
        pvolumes={
            '/data': task_a.pvolume,
            '/other_data': create_pvc_from_existing_vol.volume
        },
    )


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(volumeop_basic, __file__ + '.yaml')

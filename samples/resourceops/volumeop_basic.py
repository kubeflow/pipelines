# Copyright 2019 Google LLC
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


import kfp.dsl as dsl
from kubernetes import client as k8s_client


@dsl.pipeline(
    name="VolumeOp Basic",
    description="A Basic Example on VolumeOp Usage."
)
def volumeop_basic(size):
    vop = dsl.VolumeOp(
        name="create_pvc",
        resource_name="my-pvc",
        modes=dsl.VOLUME_MODE_RWM,
        size=size
    )

    pvc_source = k8s_client.V1PersistentVolumeClaimVolumeSource(
        claim_name=vop.outputs["name"]
    )
    volume = k8s_client.V1Volume(
        name="create-pvc",
        persistent_volume_claim=pvc_source
    )

    cop = dsl.ContainerOp(
        name="cop",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["echo foo > /mnt/file1"],
        pvolumes={"/mnt": volume}
    )


if __name__ == "__main__":
    import kfp.compiler as compiler
    compiler.Compiler().compile(volumeop_basic, __file__ + ".tar.gz")

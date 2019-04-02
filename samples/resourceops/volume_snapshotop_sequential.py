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


@dsl.pipeline(
    name="VolumeSnapshotOp Sequential",
    description="The fourth example of the design doc."
)
def volume_snapshotop_sequential(url):
    vop = dsl.VolumeOp(
        name="create_volume",
        resource_name="vol1",
        size="1Gi",
        modes=dsl.VOLUME_MODE_RWM
    )

    step1 = dsl.ContainerOp(
        name="step1_ingest",
        image="google/cloud-sdk:216.0.0",
        command=["sh", "-c"],
        arguments=["gsutil cat %s | tee /data/file1.gz" % url],
        volumes={"/data": vop.volume}
    )

    step1_snap = step1.volume.snapshot()

    step2 = dsl.ContainerOp(
        name="step2_gunzip",
        image="library/bash:4.4.23",
        command=["gunzip", "/data/file1.gz"],
        volumes={"/data": step1.volume}
    )

    step2_snap = step2.volume.snapshot()

    step3 = dsl.ContainerOp(
        name="step3_output",
        image="library/bash:4.4.23",
        command=["cat", "/data/file1"],
        volumes={"/data": step2.volume}
    )


if __name__ == "__main__":
    import kfp.compiler as compiler
    compiler.Compiler().compile(volume_snapshotop_sequential,
                                __file__ + ".tar.gz")

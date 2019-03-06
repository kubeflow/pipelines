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
    name="Example 5",
    description="The fifth example of the design doc."
)
def example5(rok_url):
    vol1 = dsl.PipelineVolume(
        name="vol1",
        size="1Gi",
        annotations={"rok/origin": rok_url},
        mode=dsl.VOLUME_MODE_RWM
    )

    step1 = dsl.ContainerOp(
        name="step1_concat",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["cat /data/file*| gzip -c >/data/full.gz"],
        volumes={"/data": vol1}
    )

    step1_snap = dsl.PipelineVolumeSnapshot(
        step1.volume,
        name="snap1"
    )

    vol2 = dsl.PipelineVolume(
        data_source=step1_snap,
        name="vol2"
    )

    step2 = dsl.ContainerOp(
        name="step2_gunzip",
        image="library/bash:4.4.23",
        command=["gunzip", "-k", "/data/full.gz"],
        volumes={"/data": vol2}
    )

    step2_snap = dsl.PipelineVolumeSnapshot(
        step2.volume,
        name="snap2"
    )

    vol3 = dsl.PipelineVolume(
        data_source=step2_snap,
        name="vol3"
    )

    step3 = dsl.ContainerOp(
        name="step3_output",
        image="library/bash:4.4.23",
        command=["cat", "/data/full"],
        volumes={"/data": vol3}
    )


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(example5, __file__ + '.tar.gz')

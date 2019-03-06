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
    name="Example 2",
    description="The second example of the design doc."
)
def example2():
    vol_common = dsl.PipelineVolume(
        name="mypvc",
        size="10Gi",
        mode=dsl.VOLUME_MODE_RWM
    )

    step1 = dsl.ContainerOp(
        name="step1",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["echo 1 | tee /mnt/file1"],
        volumes={"/mnt": vol_common}
    )

    step2 = dsl.ContainerOp(
        name="step2",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["echo 2 | tee /mnt2/file2"],
        volumes={"/mnt2": vol_common}
    )

    step3 = dsl.ContainerOp(
        name="step3",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["cat /mnt/file1 /mnt/file2"],
        volumes={"/mnt": vol_common.after(step1, step2)}
    )


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(example2, __file__ + '.tar.gz')

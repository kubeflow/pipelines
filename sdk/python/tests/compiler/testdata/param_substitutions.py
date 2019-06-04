#!/usr/bin/env python3
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
    name="Param Substitutions",
    description="Test the same PipelineParam getting substituted in multiple "
                "places"
)
def param_substitutions():
    vop = dsl.VolumeOp(
        name="create_volume",
        resource_name="data",
        size="1Gi"
    )

    op = dsl.ContainerOp(
        name="cop",
        image="image",
        arguments=["--param", vop.output],
        pvolumes={"/mnt": vop.volume}
    )


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(param_substitutions, __file__ + '.tar.gz')

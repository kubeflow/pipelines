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
from kfp.components import create_component_from_func

def write_to_volume():
    with open("/mnt/file.txt", "w") as file:
        file.write("Hello world")

write_to_volume_op = create_component_from_func(write_to_volume)

@dsl.pipeline(
    name="volumeop-basic",
    description="A Basic Example on VolumeOp Usage."
)
def volumeop_basic(size):
    vop = dsl.VolumeOp(
        name="create-pvc",
        resource_name="my-pvc",
        modes=dsl.VOLUME_MODE_RWO,
        size=size
    )

    write_to_volume_op.add_pvolume({"/mnt'": vop.volume})

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(volumeop_basic, __file__ + '.yaml')

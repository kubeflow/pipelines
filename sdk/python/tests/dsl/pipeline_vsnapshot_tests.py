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


from kfp.dsl import Pipeline, PipelineParam, ContainerOp, PipelineVolume, \
                    PipelineVolumeSnapshot
import unittest


class TestPipelineVolumeSnapshot(unittest.TestCase):

    def test_basic(self):
        """Test basic usage."""
        with Pipeline("somename") as p:
            param = PipelineParam("param")
            vol = PipelineVolume(
                name="vol",
                size=param
            )
            op1 = ContainerOp(
                name="op1",
                image="image",
                volumes={"/out": vol}
            )
            snap1 = PipelineVolumeSnapshot(op1.volume, name="snap")
            op2 = ContainerOp(
                name="op2",
                image="image",
                volumes={"/mnt": snap1}
            )
            snap2 = PipelineVolumeSnapshot(op2.volume)

        self.assertCountEqual(
            [x.op_name for x in snap1.inputs], ["vol"]
        )
        self.assertEqual(snap1.deps, {"op1"})
        self.assertEqual(op2.volume.name, "snap-clone")
        self.assertEqual(op2.deps, ["snap-clone"])
        self.assertCountEqual(
            [x.op_name for x in snap2.inputs], ["snap-clone"]
        )
        self.assertEqual(snap2.deps, {"op2"})

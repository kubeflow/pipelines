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


from kfp.dsl import Pipeline, VolumeOp, ContainerOp, PipelineVolume
import unittest


class TestPipelineVolume(unittest.TestCase):

    def test_basic(self):
        """Test basic usage."""
        with Pipeline("somename") as p:
            vol = VolumeOp(
                name="myvol_creation",
                resource_name="myvol",
                size="1Gi"
            )
            op1 = ContainerOp(
                name="op1",
                image="image",
                volumes={"/mnt": vol.volume}
            )
            op2 = ContainerOp(
                name="op2",
                image="image",
                volumes={"/data": op1.volume}
            )

        self.assertEqual(vol.volume.deps, [])
        self.assertEqual(op1.volume.deps, [op1.name])
        self.assertEqual(op2.deps, [op1.name])

    def test_after_method(self):
        """Test the after method."""
        with Pipeline("somename") as p:
            op1 = ContainerOp(name="op1", image="image")
            op2 = ContainerOp(name="op2", image="image").after(op1)
            op3 = ContainerOp(name="op3", image="image")
            vol1 = PipelineVolume(name="pipeline-volume")
            vol2 = vol1.after(op1)
            vol3 = vol2.after(op2)
            vol4 = vol3.after(op1, op2)
            vol5 = vol4.after(op3)

        self.assertEqual(vol1.deps, [])
        self.assertEqual(vol2.deps, [op1.name])
        self.assertEqual(vol3.deps, [op2.name])
        self.assertEqual(sorted(vol4.deps), [op1.name, op2.name])
        self.assertEqual(sorted(vol5.deps), [op1.name, op2.name, op3.name])

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


from kfp.dsl import Pipeline, PipelineParam, ContainerOp, PipelineVolume
import unittest


class TestPipelineVolume(unittest.TestCase):

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
            op2 = ContainerOp(
                name="op2",
                image="image",
                volumes={"/mnt": op1.volume}
            )

        self.assertCountEqual(
            [x.name for x in vol.inputs], ["param"]
        )
        self.assertCountEqual(list(op1.volumes.keys()), ["/out"])
        self.assertEqual(op1.volume.name, "vol")
        self.assertEqual(op1.volume.deps, {"op1"})
        self.assertEqual(op2.deps, ["op1"])

    def test_after_method(self):
        with Pipeline('somename') as p:
            vol1 = PipelineVolume(name="vol1", data_source="snap")
            vol2 = PipelineVolume(name="vol2", size="10Gi")
            vol3 = PipelineVolume(pvc="existing-pvc")
            op1 = ContainerOp(
                name="op1",
                image="image",
                volumes={
                    "/out": vol1,
                    "/mnt": vol3
                }
            )
            op2 = ContainerOp(
                name="op2",
                image="image",
                volumes={
                    "/mnt": vol1.after(op1),
                    "/out": vol3.after(op1)
                }
            )
            op3 = ContainerOp(
                name="op3",
                image="image",
                volumes={"/mnt": vol2.after(op1, op2)}
            )
        self.assertCountEqual(op1.deps, [vol1.name])
        self.assertCountEqual(op2.deps, [op1.name, op1.name])
        self.assertCountEqual(op3.deps, [vol2.name, op1.name, op2.name])

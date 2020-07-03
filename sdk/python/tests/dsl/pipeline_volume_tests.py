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

import kfp
from kfp.dsl import VolumeOp, ContainerOp, PipelineVolume
import unittest


class TestPipelineVolume(unittest.TestCase):

    def test_basic(self):
        """Test basic usage."""
        def my_pipeline():
            vol = VolumeOp(
                name="myvol_creation",
                resource_name="myvol",
                size="1Gi"
            )
            op1 = ContainerOp(
                name="op1",
                image="image",
                pvolumes={"/mnt": vol.volume}
            )
            op2 = ContainerOp(
                name="op2",
                image="image",
                pvolumes={"/data": op1.pvolume}
            )

            self.assertEqual(vol.volume.dependent_names, [])
            self.assertEqual(op1.pvolume.dependent_names, [op1.name])
            self.assertEqual(op2.dependent_names, [op1.name])
        
        kfp.compiler.Compiler()._compile(my_pipeline)

    def test_after_method(self):
        """Test the after method."""
        def my_pipeline():
            op1 = ContainerOp(name="op1", image="image")
            op2 = ContainerOp(name="op2", image="image").after(op1)
            op3 = ContainerOp(name="op3", image="image")
            vol1 = PipelineVolume(name="pipeline-volume")
            vol2 = vol1.after(op1)
            vol3 = vol2.after(op2)
            vol4 = vol3.after(op1, op2)
            vol5 = vol4.after(op3)

            self.assertEqual(vol1.dependent_names, [])
            self.assertEqual(vol2.dependent_names, [op1.name])
            self.assertEqual(vol3.dependent_names, [op2.name])
            self.assertEqual(sorted(vol4.dependent_names), [op1.name, op2.name])
            self.assertEqual(sorted(vol5.dependent_names),
                            [op1.name, op2.name, op3.name])

        kfp.compiler.Compiler()._compile(my_pipeline)

    def test_omitting_name(self):
        """Test PipelineVolume creation when omitting "name"."""
        def my_pipeline(param='foo'):
            vol1 = PipelineVolume(pvc="foo")
            vol2 = PipelineVolume(name="provided", pvc="foo")
            name1 = ("pvolume-4cf668b8c7be134cfcbd7758d1eef9643d1bd7ed9925a98e707635b")
            name2 = "provided"
            self.assertEqual(vol1.name, name1)
            self.assertEqual(vol2.name, name2)

            # Testing json.dumps() when pvc is a PipelineParam to avoid
            # `TypeError: Object of type PipelineParam is not JSON serializable`
            vol3 = PipelineVolume(pvc=param)
        kfp.compiler.Compiler()._compile(my_pipeline)

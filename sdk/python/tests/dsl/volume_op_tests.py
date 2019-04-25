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


from kubernetes.client.models import (
    V1Volume, V1PersistentVolumeClaimVolumeSource
)

from kfp.dsl import Pipeline, PipelineParam, VolumeOp, PipelineVolume
import unittest


class TestVolumeOp(unittest.TestCase):

    def test_basic(self):
        """Test basic usage."""
        with Pipeline("somename") as p:
            param1 = PipelineParam("param1")
            param2 = PipelineParam("param2")
            vol = VolumeOp(
                name="myvol_creation",
                resource_name=param1,
                size=param2,
                annotations={"test": "annotation"}
            )

        self.assertCountEqual(
            [x.name for x in vol.inputs], ["param1", "param2"]
        )
        self.assertEqual(
            vol.k8s_resource.metadata.name,
            "{{workflow.name}}-%s" % PipelineParam("param1")
        )
        expected_attribute_outputs = {
            "manifest": "{}",
            "name": "{.metadata.name}",
            "size": "{.status.capacity.storage}"
        }
        self.assertEqual(vol.attribute_outputs, expected_attribute_outputs)
        expected_outputs = {
            "manifest": PipelineParam(name="manifest", op_name=vol.name),
            "name": PipelineParam(name="name", op_name=vol.name),
            "size": PipelineParam(name="size", op_name=vol.name)
        }
        self.assertEqual(vol.outputs, expected_outputs)
        self.assertEqual(
            vol.output,
            PipelineParam(name="name", op_name=vol.name)
        )
        self.assertEqual(vol.dependent_names, [])
        expected_volume = PipelineVolume(
            name="myvol-creation",
            persistent_volume_claim=V1PersistentVolumeClaimVolumeSource(
                claim_name=PipelineParam(name="name", op_name=vol.name)
            )
        )
        self.assertEqual(vol.volume, expected_volume)

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

from kubernetes import client as k8s_client
from kfp.dsl import (
        Pipeline, PipelineParam, VolumeOp, VolumeSnapshotOp
)
import unittest


class TestVolumeSnapshotOp(unittest.TestCase):

    def test_basic(self):
        """Test basic usage."""
        with Pipeline("somename") as p:
            param1 = PipelineParam("param1")
            param2 = PipelineParam("param2")
            vol = VolumeOp(
                name="myvol_creation",
                resource_name="myvol",
                size="1Gi",
            )
            snap1 = VolumeSnapshotOp(
                name="mysnap_creation",
                resource_name=param1,
                volume=vol.volume,
            )
            snap2 = VolumeSnapshotOp(
                name="mysnap_creation",
                resource_name="mysnap",
                pvc=param2,
                attribute_outputs={"size": "test"}
            )

        self.assertEqual(
            sorted([x.name for x in snap1.inputs]), ["name", "param1"]
        )
        self.assertEqual(
            sorted([x.name for x in snap2.inputs]), ["param2"]
        )
        expected_attribute_outputs_1 = {
            "manifest": "{}",
            "name": "{.metadata.name}",
            "size": "{.status.restoreSize}"
        }
        self.assertEqual(snap1.attribute_outputs, expected_attribute_outputs_1)
        expected_attribute_outputs_2 = {
            "manifest": "{}",
            "name": "{.metadata.name}",
            "size": "test"
        }
        self.assertEqual(snap2.attribute_outputs, expected_attribute_outputs_2)
        expected_outputs_1 = {
            "manifest": PipelineParam(name="manifest", op_name=snap1.name),
            "name": PipelineParam(name="name", op_name=snap1.name),
            "size": PipelineParam(name="name", op_name=snap1.name),
        }
        self.assertEqual(snap1.outputs, expected_outputs_1)
        expected_outputs_2 = {
            "manifest": PipelineParam(name="manifest", op_name=snap2.name),
            "name": PipelineParam(name="name", op_name=snap2.name),
            "size": PipelineParam(name="name", op_name=snap2.name),
        }
        self.assertEqual(snap2.outputs, expected_outputs_2)
        self.assertEqual(
            snap1.output,
            PipelineParam(name="name", op_name=snap1.name)
        )
        self.assertEqual(
            snap2.output,
            PipelineParam(name="size", op_name=snap2.name)
        )
        self.assertEqual(snap1.dependent_names, [])
        self.assertEqual(snap2.dependent_names, [])
        expected_snapshot_1 = k8s_client.V1TypedLocalObjectReference(
            api_group="snapshot.storage.k8s.io",
            kind="VolumeSnapshot",
            name=PipelineParam(name="name", op_name=vol.name)
        )
        self.assertEqual(snap1.snapshot, expected_snapshot_1)
        expected_snapshot_2 = k8s_client.V1TypedLocalObjectReference(
            api_group="snapshot.storage.k8s.io",
            kind="VolumeSnapshot",
            name=PipelineParam(name="param1")
        )
        self.assertEqual(snap2.snapshot, expected_snapshot_2)

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


from kfp.dsl import Pipeline, PipelineParam, ResourceOp
from kubernetes import client as k8s_client
import unittest


class TestResourceOp(unittest.TestCase):

    def test_basic(self):
        """Test basic usage."""
        with Pipeline("somename") as p:
            param = PipelineParam("param")
            resource_metadata = k8s_client.V1ObjectMeta(
                name="my-resource"
            )
            k8s_resource = k8s_client.V1PersistentVolumeClaim(
                api_version="v1",
                kind="PersistentVolumeClaim",
                metadata=resource_metadata
            )
            res = ResourceOp(
                name="resource",
                k8s_resource=k8s_resource,
                success_condition=param,
                attribute_outputs={"test": "attr"}
            )

        self.assertCountEqual(
            [x.name for x in res.inputs], ["param"]
        )
        self.assertEqual(res.name, "resource")
        self.assertEqual(
            res.resource.success_condition,
            PipelineParam("param")
        )
        self.assertEqual(res.resource.action, "create")
        self.assertEqual(res.resource.failure_condition, None)
        self.assertEqual(res.resource.manifest, None)
        expected_attribute_outputs = {
            "manifest": "{}",
            "name": "{.metadata.name}",
            "test": "attr"
        }
        self.assertEqual(res.attribute_outputs, expected_attribute_outputs)
        expected_outputs = {
            "manifest": PipelineParam(name="manifest", op_name=res.name),
            "name": PipelineParam(name="name", op_name=res.name),
            "test": PipelineParam(name="test", op_name=res.name),
        }
        self.assertEqual(res.outputs, expected_outputs)
        self.assertEqual(
            res.output,
            PipelineParam(name="test", op_name=res.name)
        )
        self.assertEqual(res.dependent_names, [])

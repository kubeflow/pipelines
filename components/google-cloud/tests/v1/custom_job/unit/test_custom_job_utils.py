# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Test Vertex AI Custom Job Client module."""

import json
import os
from google_cloud_pipeline_components.v1.custom_job import utils
from kfp import components

import unittest


class VertexAICustomJobUtilsTests(unittest.TestCase):

  def _create_a_container_based_component(self) -> callable:
    """Creates a test container based component factory."""

    return components.load_component_from_text("""
name: ContainerComponent
inputs:
- {name: input_text, type: String, description: "Represents an input parameter."}
outputs:
- {name: output_value, type: String, description: "Represents an output parameter."}
implementation:
  container:
    image: google/cloud-sdk:latest
    command:
    - sh
    - -c
    - |
      set -e -x
      echo "$0, this is an output parameter"
    - {inputValue: input_text}
    - {outputPath: output_value}
""")

  def test_run_as_vertex_ai_custom_job_on_container_spec_with_default_values_converts_correctly(
      self):
    with open(
        os.path.join(
            os.path.dirname(os.path.dirname(__file__)),
            'testdata/custom_training_job_spec.json')) as ef:
      expected_output_json = json.load(ef, strict=False)
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function)
    self.assertDictEqual(custom_job_spec.component_spec.to_dict(),
                         expected_output_json)

  def test_run_as_vertex_ai_custom_with_accelerator_type_and_count_converts_correctly(
      self):
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function,
        accelerator_type='test_accelerator_type',
        accelerator_count=2)
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='worker_pool_specs',
            type='JsonArray',
            description=None,
            default='[{"machine_spec": {"machine_type": "n1-standard-4", "accelerator_type": "test_accelerator_type", "accelerator_count": 2}, "replica_count": 1, "container_spec": {"image_uri": "google/cloud-sdk:latest", "command": ["sh", "-c", "set -e -x\\necho \\"$0, this is an output parameter\\"\\n", "{{$.inputs.parameters[\'input_text\']}}", "{{$.outputs.parameters[\'output_value\'].output_file}}"]}, "disk_spec": {"boot_disk_type": "pd-ssd", "boot_disk_size_gb": 100}}]',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_boot_disk_type_and_size_converts_correctly(
      self):
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function,
        boot_disk_type='test_type',
        boot_disk_size_gb=200)
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='worker_pool_specs',
            type='JsonArray',
            description=None,
            default='[{"machine_spec": {"machine_type": "n1-standard-4"}, "replica_count": 1, "container_spec": {"image_uri": "google/cloud-sdk:latest", "command": ["sh", "-c", "set -e -x\\necho \\"$0, this is an output parameter\\"\\n", "{{$.inputs.parameters[\'input_text\']}}", "{{$.outputs.parameters[\'output_value\'].output_file}}"]}, "disk_spec": {"boot_disk_type": "test_type", "boot_disk_size_gb": 200}}]',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_replica_count_greater_than_1_converts_correctly(
      self):
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, replica_count=5)
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='worker_pool_specs',
            type='JsonArray',
            description=None,
            default='[{"machine_spec": {"machine_type": "n1-standard-4"}, "replica_count": 1, "container_spec": {"image_uri": "google/cloud-sdk:latest", "command": ["sh", "-c", "set -e -x\\necho \\"$0, this is an output parameter\\"\\n", "{{$.inputs.parameters[\'input_text\']}}", "{{$.outputs.parameters[\'output_value\'].output_file}}"]}, "disk_spec": {"boot_disk_type": "pd-ssd", "boot_disk_size_gb": 100}}, {"machine_spec": {"machine_type": "n1-standard-4"}, "replica_count": "4", "container_spec": {"image_uri": "google/cloud-sdk:latest", "command": ["sh", "-c", "set -e -x\\necho \\"$0, this is an output parameter\\"\\n", "{{$.inputs.parameters[\'input_text\']}}", "{{$.outputs.parameters[\'output_value\'].output_file}}"]}, "disk_spec": {"boot_disk_type": "pd-ssd", "boot_disk_size_gb": 100}}]',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_time_out_converts_correctly(self):
    component_factory_function = self._create_a_container_based_component()

    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, timeout='2s')
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='timeout',
            type='String',
            description=None,
            default='2s',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_restart_job_on_worker_restart_converts_correctly(
      self):
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, restart_job_on_worker_restart=True)
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='restart_job_on_worker_restart',
            type='Boolean',
            description=None,
            default='true',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_custom_service_account_converts_correctly(
      self):
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, service_account='test_service_account')
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='service_account',
            type='String',
            description=None,
            default='test_service_account',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_display_name_converts_correctly(self):
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, display_name='test_display_name')
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='display_name',
            type='String',
            description=None,
            default='test_display_name',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_network_converts_correctly(self):
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, network='test_network')
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='network',
            type='String',
            description=None,
            default='test_network',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_labels_converts_correctly(self):
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, labels={'test_key': 'test_value'})
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='labels',
            type='JsonObject',
            description=None,
            default='{"test_key": "test_value"}',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_reserved_ip_ranges(self):
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function,
        reserved_ip_ranges=['test_ip_range_network'])
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='reserved_ip_ranges',
            type='JsonArray',
            description=None,
            default='["test_ip_range_network"]',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_nfs_mount(self):
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, nfs_mounts=[{
            'server': 's1',
            'path': 'p1'
        }])
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='worker_pool_specs',
            type='JsonArray',
            description=None,
            default='[{"machine_spec": {"machine_type": "n1-standard-4"}, "replica_count": 1, "container_spec": {"image_uri": "google/cloud-sdk:latest", "command": ["sh", "-c", "set -e -x\\necho \\"$0, this is an output parameter\\"\\n", "{{$.inputs.parameters[\'input_text\']}}", "{{$.outputs.parameters[\'output_value\'].output_file}}"]}, "disk_spec": {"boot_disk_type": "pd-ssd", "boot_disk_size_gb": 100}, "nfs_mounts": [{"server": "s1", "path": "p1"}]}]',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_environment_variable(self):
    component_factory_function = self._create_a_container_based_component()
    component_factory_function.component_spec.implementation.container.env = [{
        'name': 'FOO',
        'value': 'BAR'
    }]
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function)
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='worker_pool_specs',
            type='JsonArray',
            description=None,
            default='[{"machine_spec": {"machine_type": "n1-standard-4"}, "replica_count": 1, "container_spec": {"image_uri": "google/cloud-sdk:latest", "command": ["sh", "-c", "set -e -x\\necho \\"$0, this is an output parameter\\"\\n", "{{$.inputs.parameters[\'input_text\']}}", "{{$.outputs.parameters[\'output_value\'].output_file}}"], "env": [{"name": "FOO", "value": "BAR"}]}, "disk_spec": {"boot_disk_type": "pd-ssd", "boot_disk_size_gb": 100}}]',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_executor_input_in_command(self):
    component_factory_function = components.load_component_from_text("""
name: ContainerComponent
outputs:
- {name: output_artifact, type: system.Artifact, description: "Represents an output artifact."}
implementation:
  container:
    image: gcr.io/repo/image:latest
    command: [python3, -u, -m, launcher, --executor_input, "{{{{$}}}}"]
""")
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function)
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='worker_pool_specs',
            type='JsonArray',
            description=None,
            default='[{"machine_spec": {"machine_type": "n1-standard-4"}, "replica_count": 1, "container_spec": {"image_uri": "gcr.io/repo/image:latest", "command": ["python3", "-u", "-m", "launcher", "--executor_input", "{{$.json_escape[1]}}"]}, "disk_spec": {"boot_disk_type": "pd-ssd", "boot_disk_size_gb": 100}}]',
            optional=True,
            annotations=None)
    ])

  def test_run_as_vertex_ai_custom_with_executor_input_in_args(self):
    component_factory_function = components.load_component_from_text("""
name: ContainerComponent
inputs:
- {name: input_text, type: String, description: "Represents an input parameter."}
outputs:
- {name: output_artifact, type: system.Artifact, description: "Represents an output artifact."}
implementation:
  container:
    image: gcr.io/repo/image:latest
    command: [python3, -u, -m, launcher]
    args: [
        --input_text, {inputValue: input_text},
        --executor_input, "{{{{$}}}}"
    ]
""")
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function)
    self.assertContainsSubsequence(custom_job_spec.component_spec.inputs, [
        components.structures.InputSpec(
            name='worker_pool_specs',
            type='JsonArray',
            description=None,
            default='[{"machine_spec": {"machine_type": "n1-standard-4"}, "replica_count": 1, "container_spec": {"image_uri": "gcr.io/repo/image:latest", "command": ["python3", "-u", "-m", "launcher"], "args": ["--input_text", "{{$.inputs.parameters[\'input_text\']}}", "--executor_input", "{{$.json_escape[1]}}"]}, "disk_spec": {"boot_disk_type": "pd-ssd", "boot_disk_size_gb": 100}}]',
            optional=True,
            annotations=None)
    ])

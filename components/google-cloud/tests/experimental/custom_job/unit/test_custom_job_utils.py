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

from google_cloud_pipeline_components.experimental.custom_job import utils
from kfp import components

import unittest


class VertexAICustomJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(VertexAICustomJobUtilsTests, self).setUp()
    utils._DEFAULT_CUSTOM_JOB_CONTAINER_IMAGE = 'test_launcher_image'

  def _create_a_container_based_component(self) -> callable:
    """Creates a test container based component factory."""

    return components.load_component_from_text("""
name: ContainerComponent
inputs:
- {name: input_text, type: String, description: "Represents an input parameter."}
outputs:
- {name: output_value, type: String, description: "Represents an output paramter."}
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

  def test_run_as_vertex_ai_custom_job_on_container_spec_with_defualts_values_converts_correctly(
      self):
    expected_results = {
        'name': 'ContainerComponent',
        'inputs': [{
            'name': 'input_text',
            'type': 'String',
            'description': 'Represents an input parameter.'
        }, {
            'name': 'base_output_directory',
            'type': 'String',
            'default': '',
            'optional': True
        }, {
            'name': 'tensorboard',
            'type': 'String',
            'default': '',
            'optional': True
        }, {
            'name': 'network',
            'type': 'String',
            'default': '',
            'optional': True
        }, {
            'name': 'service_account',
            'type': 'String',
            'default': '',
            'optional': True
        }, {
            'name': 'project',
            'type': 'String'
        }, {
            'name': 'location',
            'type': 'String'
        }],
        'outputs': [{
            'name': 'output_value',
            'type': 'String',
            'description': 'Represents an output paramter.'
        }, {
            'name': 'gcp_resources',
            'type': 'String'
        }],
        'implementation': {
            'container': {
                'image':
                    'test_launcher_image',
                'command': [
                    'python3', '-u', '-m',
                    'google_cloud_pipeline_components.container.experimental.gcp_launcher.launcher'
                ],
                'args': [
                    '--type', 'CustomJob', '--payload',
                    '{"display_name": "ContainerComponent", "job_spec": '
                    '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
                    '"n1-standard-4"}, "replica_count": 1, "container_spec": '
                    '{"image_uri": "google/cloud-sdk:latest", "command": '
                    '["sh", "-c", "set -e -x\\necho \\"$0, this is an output '
                    'parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}], "service_account": '
                    '"{{$.inputs.parameters[\'service_account\']}}", '
                    '"network": "{{$.inputs.parameters[\'network\']}}", '
                    '"tensorboard": '
                    '"{{$.inputs.parameters[\'tensorboard\']}}", '
                    '"base_output_directory": {"output_uri_prefix": '
                    '"{{$.inputs.parameters[\'base_output_directory\']}}"}}}',
                    '--project', {
                        'inputValue': 'project'
                    }, '--location', {
                        'inputValue': 'location'
                    }, '--gcp_resources', {
                        'outputPath': 'gcp_resources'
                    }
                ]
            }
        }
    }
    component_factory_function = self._create_a_container_based_component()
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function)
    self.assertDictEqual(custom_job_spec.component_spec.to_dict(),
                         expected_results)

  def test_run_as_vertex_ai_custom_with_accelerator_type_and_count_converts_correctly(
      self):
    component_factory_function = self._create_a_container_based_component()

    expected_sub_results = {
        'implementation': {
            'container': {
                'image':
                    'test_launcher_image',
                'command': [
                    'python3', '-u', '-m',
                    'google_cloud_pipeline_components.container.experimental.gcp_launcher.launcher'
                ],
                'args': [
                    '--type', 'CustomJob', '--payload',
                    '{"display_name": "ContainerComponent", "job_spec": '
                    '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
                    '"n1-standard-4", "accelerator_type": '
                    '"test_accelerator_type", "accelerator_count": 2}, '
                    '"replica_count": 1, "container_spec": {"image_uri": '
                    '"google/cloud-sdk:latest", "command": ["sh", "-c", "set '
                    '-e -x\\necho \\"$0, this is an output parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}], "service_account": '
                    '"{{$.inputs.parameters[\'service_account\']}}", '
                    '"network": "{{$.inputs.parameters[\'network\']}}", '
                    '"tensorboard": '
                    '"{{$.inputs.parameters[\'tensorboard\']}}", '
                    '"base_output_directory": {"output_uri_prefix": '
                    '"{{$.inputs.parameters[\'base_output_directory\']}}"}}}',
                    '--project', {
                        'inputValue': 'project'
                    }, '--location', {
                        'inputValue': 'location'
                    }, '--gcp_resources', {
                        'outputPath': 'gcp_resources'
                    }
                ]
            }
        }
    }
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function,
        accelerator_type='test_accelerator_type',
        accelerator_count=2)

    self.assertDictContainsSubset(
        subset=expected_sub_results,
        dictionary=custom_job_spec.component_spec.to_dict())

  def test_run_as_vertex_ai_custom_with_boot_disk_type_and_size_converts_correctly(
      self):
    component_factory_function = self._create_a_container_based_component()

    expected_sub_results = {
        'implementation': {
            'container': {
                'image':
                    'test_launcher_image',
                'command': [
                    'python3', '-u', '-m',
                    'google_cloud_pipeline_components.container.experimental.gcp_launcher.launcher'
                ],
                'args': [
                    '--type', 'CustomJob', '--payload',
                    '{"display_name": "ContainerComponent", "job_spec": '
                    '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
                    '"n1-standard-4"}, "replica_count": 1, "container_spec": '
                    '{"image_uri": "google/cloud-sdk:latest", "command": '
                    '["sh", "-c", "set -e -x\\necho \\"$0, this is an output '
                    'parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}, {"machine_spec": '
                    '{"machine_type": "n1-standard-4"}, "replica_count": "1", '
                    '"container_spec": {"image_uri": '
                    '"google/cloud-sdk:latest", "command": ["sh", "-c", "set '
                    '-e -x\\necho \\"$0, this is an output parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}], "service_account": '
                    '"{{$.inputs.parameters[\'service_account\']}}", '
                    '"network": "{{$.inputs.parameters[\'network\']}}", '
                    '"tensorboard": '
                    '"{{$.inputs.parameters[\'tensorboard\']}}", '
                    '"base_output_directory": {"output_uri_prefix": '
                    '"{{$.inputs.parameters[\'base_output_directory\']}}"}}}',
                    '--project', {
                        'inputValue': 'project'
                    }, '--location', {
                        'inputValue': 'location'
                    }, '--gcp_resources', {
                        'outputPath': 'gcp_resources'
                    }
                ]
            }
        }
    }
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, replica_count=2)

    self.assertDictContainsSubset(
        subset=expected_sub_results,
        dictionary=custom_job_spec.component_spec.to_dict())

  def test_run_as_vertex_ai_custom_with_replica_count_greater_than_1_converts_correctly(
      self):
    component_factory_function = self._create_a_container_based_component()

    expected_sub_results = {
        'implementation': {
            'container': {
                'image':
                    'test_launcher_image',
                'command': [
                    'python3', '-u', '-m',
                    'google_cloud_pipeline_components.container.experimental.gcp_launcher.launcher'
                ],
                'args': [
                    '--type', 'CustomJob', '--payload',
                    '{"display_name": "ContainerComponent", "job_spec": '
                    '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
                    '"n1-standard-4"}, "replica_count": 1, "container_spec": '
                    '{"image_uri": "google/cloud-sdk:latest", "command": '
                    '["sh", "-c", "set -e -x\\necho \\"$0, this is an output '
                    'parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}, {"machine_spec": '
                    '{"machine_type": "n1-standard-4"}, "replica_count": "1", '
                    '"container_spec": {"image_uri": '
                    '"google/cloud-sdk:latest", "command": ["sh", "-c", "set '
                    '-e -x\\necho \\"$0, this is an output parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}], "service_account": '
                    '"{{$.inputs.parameters[\'service_account\']}}", '
                    '"network": "{{$.inputs.parameters[\'network\']}}", '
                    '"tensorboard": '
                    '"{{$.inputs.parameters[\'tensorboard\']}}", '
                    '"base_output_directory": {"output_uri_prefix": '
                    '"{{$.inputs.parameters[\'base_output_directory\']}}"}}}',
                    '--project', {
                        'inputValue': 'project'
                    }, '--location', {
                        'inputValue': 'location'
                    }, '--gcp_resources', {
                        'outputPath': 'gcp_resources'
                    }
                ]
            }
        }
    }
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, replica_count=2)

    self.assertDictContainsSubset(
        subset=expected_sub_results,
        dictionary=custom_job_spec.component_spec.to_dict())

  def test_run_as_vertex_ai_custom_with_time_out_converts_correctly(self):
    component_factory_function = self._create_a_container_based_component()

    expected_sub_results = {
        'implementation': {
            'container': {
                'image':
                    'test_launcher_image',
                'command': [
                    'python3', '-u', '-m',
                    'google_cloud_pipeline_components.container.experimental.gcp_launcher.launcher'
                ],
                'args': [
                    '--type', 'CustomJob', '--payload',
                    '{"display_name": "ContainerComponent", "job_spec": '
                    '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
                    '"n1-standard-4"}, "replica_count": 1, "container_spec": '
                    '{"image_uri": "google/cloud-sdk:latest", "command": '
                    '["sh", "-c", "set -e -x\\necho \\"$0, this is an output '
                    'parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}], "scheduling": {"timeout": '
                    '2}, "service_account": '
                    '"{{$.inputs.parameters[\'service_account\']}}", '
                    '"network": "{{$.inputs.parameters[\'network\']}}", '
                    '"tensorboard": '
                    '"{{$.inputs.parameters[\'tensorboard\']}}", '
                    '"base_output_directory": {"output_uri_prefix": '
                    '"{{$.inputs.parameters[\'base_output_directory\']}}"}}}',
                    '--project', {
                        'inputValue': 'project'
                    }, '--location', {
                        'inputValue': 'location'
                    }, '--gcp_resources', {
                        'outputPath': 'gcp_resources'
                    }
                ]
            }
        }
    }
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, timeout=2)

    self.assertDictContainsSubset(
        subset=expected_sub_results,
        dictionary=custom_job_spec.component_spec.to_dict())

  def test_run_as_vertex_ai_custom_with_restart_job_on_worker_restart_converts_correctly(
      self):
    component_factory_function = self._create_a_container_based_component()

    expected_sub_results = {
        'implementation': {
            'container': {
                'image':
                    'test_launcher_image',
                'command': [
                    'python3', '-u', '-m',
                    'google_cloud_pipeline_components.container.experimental.gcp_launcher.launcher'
                ],
                'args': [
                    '--type', 'CustomJob', '--payload',
                    '{"display_name": "ContainerComponent", "job_spec": '
                    '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
                    '"n1-standard-4"}, "replica_count": 1, "container_spec": '
                    '{"image_uri": "google/cloud-sdk:latest", "command": '
                    '["sh", "-c", "set -e -x\\necho \\"$0, this is an output '
                    'parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}], "scheduling": '
                    '{"restart_job_on_worker_restart": true}, '
                    '"service_account": '
                    '"{{$.inputs.parameters[\'service_account\']}}", '
                    '"network": "{{$.inputs.parameters[\'network\']}}", '
                    '"tensorboard": '
                    '"{{$.inputs.parameters[\'tensorboard\']}}", '
                    '"base_output_directory": {"output_uri_prefix": '
                    '"{{$.inputs.parameters[\'base_output_directory\']}}"}}}',
                    '--project', {
                        'inputValue': 'project'
                    }, '--location', {
                        'inputValue': 'location'
                    }, '--gcp_resources', {
                        'outputPath': 'gcp_resources'
                    }
                ]
            }
        }
    }
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, restart_job_on_worker_restart=True)

    self.assertDictContainsSubset(
        subset=expected_sub_results,
        dictionary=custom_job_spec.component_spec.to_dict())

  def test_run_as_vertex_ai_custom_with_custom_service_account_converts_correctly(
      self):
    component_factory_function = self._create_a_container_based_component()

    expected_sub_results = {
        'implementation': {
            'container': {
                'image':
                    'test_launcher_image',
                'command': [
                    'python3', '-u', '-m',
                    'google_cloud_pipeline_components.container.experimental.gcp_launcher.launcher'
                ],
                'args': [
                    '--type', 'CustomJob', '--payload',
                    '{"display_name": "ContainerComponent", "job_spec": '
                    '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
                    '"n1-standard-4"}, "replica_count": 1, "container_spec": '
                    '{"image_uri": "google/cloud-sdk:latest", "command": '
                    '["sh", "-c", "set -e -x\\necho \\"$0, this is an output '
                    'parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}], "service_account": '
                    '"{{$.inputs.parameters[\'service_account\']}}", '
                    '"network": "{{$.inputs.parameters[\'network\']}}", '
                    '"tensorboard": '
                    '"{{$.inputs.parameters[\'tensorboard\']}}", '
                    '"base_output_directory": {"output_uri_prefix": '
                    '"{{$.inputs.parameters[\'base_output_directory\']}}"}}}',
                    '--project', {
                        'inputValue': 'project'
                    }, '--location', {
                        'inputValue': 'location'
                    }, '--gcp_resources', {
                        'outputPath': 'gcp_resources'
                    }
                ]
            }
        }
    }
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, service_account='test_service_account')

    self.assertDictContainsSubset(
        subset=expected_sub_results,
        dictionary=custom_job_spec.component_spec.to_dict())

  def test_run_as_vertex_ai_custom_with_display_name_converts_correctly(self):
    component_factory_function = self._create_a_container_based_component()

    expected_sub_results = {
        'implementation': {
            'container': {
                'image':
                    'test_launcher_image',
                'command': [
                    'python3', '-u', '-m',
                    'google_cloud_pipeline_components.container.experimental.gcp_launcher.launcher'
                ],
                'args': [
                    '--type', 'CustomJob', '--payload',
                    '{"display_name": "test_display_name", "job_spec": '
                    '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
                    '"n1-standard-4"}, "replica_count": 1, "container_spec": '
                    '{"image_uri": "google/cloud-sdk:latest", "command": '
                    '["sh", "-c", "set -e -x\\necho \\"$0, this is an output '
                    'parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}], "service_account": '
                    '"{{$.inputs.parameters[\'service_account\']}}", '
                    '"network": "{{$.inputs.parameters[\'network\']}}", '
                    '"tensorboard": '
                    '"{{$.inputs.parameters[\'tensorboard\']}}", '
                    '"base_output_directory": {"output_uri_prefix": '
                    '"{{$.inputs.parameters[\'base_output_directory\']}}"}}}',
                    '--project', {
                        'inputValue': 'project'
                    }, '--location', {
                        'inputValue': 'location'
                    }, '--gcp_resources', {
                        'outputPath': 'gcp_resources'
                    }
                ]
            }
        }
    }
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, display_name='test_display_name')

    self.assertDictContainsSubset(
        subset=expected_sub_results,
        dictionary=custom_job_spec.component_spec.to_dict())

  def test_run_as_vertex_ai_custom_with_network_converts_correctly(self):
    component_factory_function = self._create_a_container_based_component()

    expected_sub_results = {
        'implementation': {
            'container': {
                'image':
                    'test_launcher_image',
                'command': [
                    'python3', '-u', '-m',
                    'google_cloud_pipeline_components.container.experimental.gcp_launcher.launcher'
                ],
                'args': [
                    '--type', 'CustomJob', '--payload',
                    '{"display_name": "ContainerComponent", "job_spec": '
                    '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
                    '"n1-standard-4"}, "replica_count": 1, "container_spec": '
                    '{"image_uri": "google/cloud-sdk:latest", "command": '
                    '["sh", "-c", "set -e -x\\necho \\"$0, this is an output '
                    'parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}], "service_account": '
                    '"{{$.inputs.parameters[\'service_account\']}}", '
                    '"network": "{{$.inputs.parameters[\'network\']}}", '
                    '"tensorboard": '
                    '"{{$.inputs.parameters[\'tensorboard\']}}", '
                    '"base_output_directory": {"output_uri_prefix": '
                    '"{{$.inputs.parameters[\'base_output_directory\']}}"}}}',
                    '--project', {
                        'inputValue': 'project'
                    }, '--location', {
                        'inputValue': 'location'
                    }, '--gcp_resources', {
                        'outputPath': 'gcp_resources'
                    }
                ]
            }
        }
    }
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, network='test_network')

    self.assertDictContainsSubset(
        subset=expected_sub_results,
        dictionary=custom_job_spec.component_spec.to_dict())

  def test_run_as_vertex_ai_custom_with_labels_converts_correctly(self):
    component_factory_function = self._create_a_container_based_component()

    expected_sub_results = {
        'implementation': {
            'container': {
                'image':
                    'test_launcher_image',
                'command': [
                    'python3', '-u', '-m',
                    'google_cloud_pipeline_components.container.experimental.gcp_launcher.launcher'
                ],
                'args': [
                    '--type', 'CustomJob', '--payload',
                    '{"display_name": "ContainerComponent", "job_spec": '
                    '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
                    '"n1-standard-4"}, "replica_count": 1, "container_spec": '
                    '{"image_uri": "google/cloud-sdk:latest", "command": '
                    '["sh", "-c", "set -e -x\\necho \\"$0, this is an output '
                    'parameter\\"\\n", '
                    '"{{$.inputs.parameters[\'input_text\']}}", '
                    '"{{$.outputs.parameters[\'output_value\'].output_file}}"]},'
                    ' "disk_spec": {"boot_disk_type": "pd-ssd", '
                    '"boot_disk_size_gb": 100}}], "labels": {"test_key": '
                    '"test_value"}, "service_account": '
                    '"{{$.inputs.parameters[\'service_account\']}}", '
                    '"network": "{{$.inputs.parameters[\'network\']}}", '
                    '"tensorboard": '
                    '"{{$.inputs.parameters[\'tensorboard\']}}", '
                    '"base_output_directory": {"output_uri_prefix": '
                    '"{{$.inputs.parameters[\'base_output_directory\']}}"}}}',
                    '--project', {
                        'inputValue': 'project'
                    }, '--location', {
                        'inputValue': 'location'
                    }, '--gcp_resources', {
                        'outputPath': 'gcp_resources'
                    }
                ]
            }
        }
    }
    custom_job_spec = utils.create_custom_training_job_op_from_component(
        component_factory_function, labels={'test_key': 'test_value'})

    self.assertDictContainsSubset(
        subset=expected_sub_results,
        dictionary=custom_job_spec.component_spec.to_dict())

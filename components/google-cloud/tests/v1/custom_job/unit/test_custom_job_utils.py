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

from google_cloud_pipeline_components.v1.custom_job import utils
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
        'name':
            'ContainerComponent',
        'description':
            'Launch a Custom training job using Vertex CustomJob API.\n\n  '
            'Args:\n      project (str):\n          Required. Project to '
            'create the custom training job in.\n      location '
            '(Optional[str]):\n          Location for creating the custom '
            'training job. If not set,\n          default to us-central1.\n'
            '      display_name (str): The name of the custom training job.\n'
            '      worker_pool_specs (Optional[Sequence[str]]): Serialized '
            'json spec of the worker pools\n          including machine type '
            'and Docker image. All worker pools except the first one are\n'
            '          optional and can be skipped by providing an empty '
            'value.\n\n          For more details about the WorkerPoolSpec, '
            'see\n          '
            'https://cloud.google.com/vertex-ai/docs/reference/rest/v1/CustomJobSpec#WorkerPoolSpec\n'
            '      timeout (Optional[str]): The maximum job running time. The '
            'default is 7\n          days. A duration in seconds with up to '
            'nine fractional digits, terminated\n          by \'s\', for '
            'example: "3.5s".\n      restart_job_on_worker_restart '
            '(Optional[bool]): Restarts the entire\n          CustomJob if a '
            'worker gets restarted. This feature can be used by\n          '
            'distributed training jobs that are not resilient to workers '
            'leaving and\n          joining a job.\n      service_account '
            '(Optional[str]): Sets the default service account for\n          '
            'workload run-as account. The service account running the '
            'pipeline\n            '
            '(https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)\n'
            '              submitting jobs must have act-as permission on this'
            ' run-as account. If\n              unspecified, the Vertex AI '
            'Custom Code Service\n          '
            'Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)\n'
            '              for the CustomJob\'s project.\n      tensorboard '
            '(Optional[str]): The name of a Vertex AI Tensorboard resource '
            'to\n          which this CustomJob will upload Tensorboard '
            'logs.\n      enable_web_access (Optional[bool]): Whether you want'
            ' Vertex AI to enable\n          [interactive shell '
            'access](https://cloud.google.com/vertex-ai/docs/training/monitor-debug-interactive-shell)\n'
            '          to training containers.\n          If set to `true`, '
            'you can access interactive shells at the URIs given\n          by'
            ' [CustomJob.web_access_uris][].\n      network (Optional[str]): '
            'The full name of the Compute Engine network to\n          which '
            'the job should be peered. For example,\n          '
            'projects/12345/global/networks/myVPC. Format is of the form\n'
            '          projects/{project}/global/networks/{network}. Where '
            '{project} is a project\n          number, as in 12345, and '
            '{network} is a network name. Private services\n          access '
            'must already be configured for the network. If left '
            'unspecified,\n          the job is not peered with any network.\n'
            '      reserved_ip_ranges (Optional[Sequence[str]]): A list of '
            'names for the reserved ip ranges\n          under the VPC network'
            ' that can be used for this job.\n          If set, we will deploy'
            ' the job within the provided ip ranges. Otherwise,\n          the'
            ' job will be deployed to any ip ranges under the provided VPC '
            'network.\n      nfs_mounts (Optional[Sequence[Dict[str, str]]]):A'
            ' list of NFS mount specs in Json\n          dict format. For API '
            'spec, see\n          '
            'https://cloud.devsite.corp.google.com/vertex-ai/docs/reference/rest/v1/CustomJobSpec#NfsMount\n'
            '            For more details about mounting NFS for CustomJob, '
            'see\n          '
            'https://cloud.devsite.corp.google.com/vertex-ai/docs/training/train-nfs-share\n'
            '      base_output_directory (Optional[str]): The Cloud Storage '
            'location to store\n          the output of this CustomJob or '
            'HyperparameterTuningJob. see below for more details:\n          '
            'https://cloud.google.com/vertex-ai/docs/reference/rest/v1/GcsDestination\n'
            '      labels (Optional[Dict[str, str]]): The labels with '
            'user-defined metadata to organize CustomJobs.\n          See '
            'https://goo.gl/xmQnxf for more information.\n      '
            'encryption_spec_key_name (Optional[str]): Customer-managed '
            'encryption key\n          options for the CustomJob. If this is '
            'set, then all resources created by\n          the CustomJob will '
            'be encrypted with the provided encryption key.\n\n  Returns:\n'
            '      gcp_resources (str):\n          Serialized gcp_resources '
            'proto tracking the custom training job.\n          For more '
            'details, see '
            'https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.\n',
        'inputs': [{
            'name': 'project',
            'type': 'String'
        }, {
            'name': 'location',
            'type': 'String',
            'default': 'us-central1'
        }, {
            'name': 'display_name',
            'type': 'String',
            'default': 'ContainerComponent',
            'optional': True
        }, {
            'name':
                'worker_pool_specs',
            'type':
                'JsonArray',
            'default':
                '[{"machine_spec": {"machine_type": "n1-standard-4"}, '
                '"replica_count": 1, "container_spec": {"image_uri": '
                '"google/cloud-sdk:latest", "command": ["sh", "-c", "set -e '
                '-x\\necho \\"$0, this is an output parameter\\"\\n", '
                '"{{$.inputs.parameters[\'input_text\']}}", '
                '"{{$.outputs.parameters[\'output_value\'].output_file}}"]}, '
                '"disk_spec": {"boot_disk_type": "pd-ssd", '
                '"boot_disk_size_gb": 100}}]',
            'optional':
                True
        }, {
            'name': 'timeout',
            'type': 'String',
            'default': '604800s',
            'optional': True
        }, {
            'name': 'restart_job_on_worker_restart',
            'type': 'Boolean',
            'default': 'false',
            'optional': True
        }, {
            'name': 'service_account',
            'type': 'String',
            'default': '',
            'optional': True
        }, {
            'name': 'tensorboard',
            'type': 'String',
            'default': '',
            'optional': True
        }, {
            'name': 'enable_web_access',
            'type': 'Boolean',
            'default': 'false',
            'optional': True
        }, {
            'name': 'network',
            'type': 'String',
            'default': '',
            'optional': True
        }, {
            'name': 'reserved_ip_ranges',
            'type': 'JsonArray',
            'default': '[]',
            'optional': True
        }, {
            'name': 'nfs_mounts',
            'type': 'JsonArray',
            'default': '{}',
            'optional': True
        }, {
            'name': 'base_output_directory',
            'type': 'String',
            'default': '',
            'optional': True
        }, {
            'name': 'labels',
            'type': 'JsonObject',
            'default': '{}',
            'optional': True
        }, {
            'name': 'encryption_spec_key_name',
            'type': 'String',
            'default': '',
            'optional': True
        }, {
            'name': 'input_text',
            'type': 'String',
            'description': 'Represents an input parameter.'
        }],
        'outputs': [{
            'name': 'gcp_resources',
            'type': 'String'
        }, {
            'name': 'output_value',
            'type': 'String',
            'description': 'Represents an output paramter.'
        }],
        'implementation': {
            'container': {
                'image':
                    'gcr.io/ml-pipeline/google-cloud-pipeline-components:latest',
                'command': [
                    'python3', '-u', '-m',
                    'google_cloud_pipeline_components.container.v1.gcp_launcher.launcher'
                ],
                'args': [
                    '--type', 'CustomJob', '--payload', {
                        'concat': [
                            '{', '"display_name": "', {
                                'inputValue': 'display_name'
                            }, '"', ', "job_spec": {', '"worker_pool_specs": ',
                            {
                                'inputValue': 'worker_pool_specs'
                            }, ', "scheduling": {', '"timeout": "', {
                                'inputValue': 'timeout'
                            }, '"', ', "restart_job_on_worker_restart": "', {
                                'inputValue': 'restart_job_on_worker_restart'
                            }, '"', '}', ', "service_account": "', {
                                'inputValue': 'service_account'
                            }, '"', ', "tensorboard": "', {
                                'inputValue': 'tensorboard'
                            }, '"', ', "enable_web_access": "', {
                                'inputValue': 'enable_web_access'
                            }, '"', ', "network": "', {
                                'inputValue': 'network'
                            }, '"', ', "reserved_ip_ranges": ', {
                                'inputValue': 'reserved_ip_ranges'
                            }, ', "nfs_mounts": ', {
                                'inputValue': 'nfs_mounts'
                            }, ', "base_output_directory": {',
                            '"output_uri_prefix": "', {
                                'inputValue': 'base_output_directory'
                            }, '"', '}', '}', ', "labels": ', {
                                'inputValue': 'labels'
                            }, ', "encryption_spec": {"kms_key_name":"', {
                                'inputValue': 'encryption_spec_key_name'
                            }, '"}', '}'
                        ]
                    }, '--project', {
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
            name='nfs_mounts',
            type='JsonArray',
            description=None,
            default='[{"server": "s1", "path": "p1"}]',
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

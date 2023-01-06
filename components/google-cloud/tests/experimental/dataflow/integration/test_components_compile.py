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
"""Test google-cloud-pipeline-Components to ensure they compile correctly."""

import json
import os
from google_cloud_pipeline_components.experimental.dataflow import DataflowFlexTemplateJobOp
from google_cloud_pipeline_components.experimental.dataflow import DataflowPythonJobOp
import kfp
from kfp.v2 import compiler
import unittest


class ComponentsCompileTest(unittest.TestCase):

  def setUp(self):
    super(ComponentsCompileTest, self).setUp()
    self._project = 'test_project'
    self._location = 'us-central1'
    self._python_module_path = 'gs://python_module_path'
    self._requirements_file_path = 'gs://requirements_file_path'
    self._args = ['test_arg']
    self._gcs_source = 'gs://test_gcs_source'
    self._temp_location = 'gs://temp_location'
    self._pipeline_root = 'gs://test_pipeline_root'
    self._package_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'pipeline.json')

    self._job_name = 'test_job_name'
    self._container_spec_gcs_path = 'gs://test_container_spec_gcs_path'
    self._parameters = {
        'key1': 'value1',
        'key2': 'value2',
    }
    self._launch_options = {
        'key1': 'value1',
        'key2': 'value2',
    }
    self._num_workers = 1
    self._max_workers = 1000
    self._service_account_email = 'test_service_account_email'
    self._machine_type = 'e2-standard-1'
    self._network = 'test_network'
    self._subnetwork = 'test_subnetwork'
    self._additional_experiments = ['experiment1', 'experiment2']
    self._additional_user_labels = {
        'key1': 'value1',
        'key2': 'value2',
    }
    self._kms_key_name = 'test_kms_key_name'
    self._ip_configuration = 'test_ip_configuration'
    self._worker_region = 'us-central1'
    self._worker_zone = 'us-central1-b'
    self._staging_location = 'test_staging_location'
    self._autoscaling_algorithm = 'THROUGHPUT_BASED'
    self._disk_size_gb = 10
    self._dump_heap_on_oom = False
    self._enable_launcher_vm_serial_port_logging = False
    self._enable_streaming_engine = False
    self._flexrs_goal = 'COST_OPTIMIZED'
    self._sdk_container_image = 'gcr.io/test-project/test-image:latest'
    self._save_heap_dumps_to_gcs_path = 'gs://test_save_heap_dumps_to_gcs_path'
    self._launcher_machine_type = 'n1-standard-1'
    self._update = False
    self._transform_name_mappings = {
        'oldTransformName1': 'newTransformName1',
        'oldTransformName2': 'newTransformName2',
    }
    self._validate_only = False

  def tearDown(self):
    super(ComponentsCompileTest, self).tearDown()
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def test_dataflow_python_op_compile(self):

    @kfp.dsl.pipeline(name='dataflow-python-test')
    def pipeline():
      DataflowPythonJobOp(
          project=self._project,
          location=self._location,
          python_module_path=self._python_module_path,
          temp_location=self._temp_location,
          requirements_file_path=self._requirements_file_path,
          args=self._args)

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)

    with open('testdata/dataflow_python_job_component_pipeline.json') as ef:
      expected_executor_output_json = json.load(ef, strict=False)

    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json['sdkVersion']
    del executor_output_json['schemaVersion']
    self.assertDictEqual(executor_output_json, expected_executor_output_json)

  def test_dataflow_flex_template_op_compile(self):

    @kfp.dsl.pipeline(name='dataflow-flex-template-test')
    def pipeline():
      DataflowFlexTemplateJobOp(
          project=self._project,
          location=self._location,
          job_name=self._job_name,
          container_spec_gcs_path=self._container_spec_gcs_path,
          parameters=self._parameters,
          launch_options=self._launch_options,
          num_workers=self._num_workers,
          max_workers=self._max_workers,
          service_account_email=self._service_account_email,
          temp_location=self._temp_location,
          machine_type=self._machine_type,
          additional_experiments=self._additional_experiments,
          network=self._network,
          subnetwork=self._subnetwork,
          additional_user_labels=self._additional_user_labels,
          kms_key_name=self._kms_key_name,
          ip_configuration=self._ip_configuration,
          worker_region=self._worker_region,
          worker_zone=self._worker_zone,
          enable_streaming_engine=self._enable_streaming_engine,
          flexrs_goal=self._flexrs_goal,
          staging_location=self._staging_location,
          sdk_container_image=self._sdk_container_image,
          disk_size_gb=self._disk_size_gb,
          autoscaling_algorithm=self._autoscaling_algorithm,
          dump_heap_on_oom=self._dump_heap_on_oom,
          save_heap_dumps_to_gcs_path=self._save_heap_dumps_to_gcs_path,
          enable_launcher_vm_serial_port_logging=self._enable_launcher_vm_serial_port_logging,
          launcher_machine_type=self._launcher_machine_type,
          update=self._update,
          transform_name_mappings=self._transform_name_mappings,
          validate_only=self._validate_only,
      )

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)

    with open('testdata/dataflow_flex_template_component_pipeline.json') as ef:
      expected_executor_output_json = json.load(ef, strict=False)

    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json['sdkVersion']
    del executor_output_json['schemaVersion']

    self.assertDictEqual(executor_output_json, expected_executor_output_json)

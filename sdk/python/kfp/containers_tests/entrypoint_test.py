# Copyright 2021 Google LLC
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
"""Tests for kfp.containers.entrypoint module."""
import mock
import unittest

from kfp.containers import entrypoint
from kfp.containers import entrypoint_utils
# Import testdata to mock entrypoint_utils.import_func_from_source function.
from kfp.containers_tests.testdata import main

_OUTPUT_METADATA_JSON_LOCATION = 'executor_output_metadata.json'

_PRODUCER_EXECUTOR_OUTPUT = """{
  "parameters": {
    "param_output": {
      "stringValue": "hello from producer"
    }
  },
  "artifacts": {
    "artifact_output": {
      "artifacts": [
        {
          "type": {
            "instanceSchema": "properties:\\ntitle: kfp.Dataset\\ntype: object\\n"
          },
          "uri": "gs://root/producer/artifact_output"
        }
      ]
    }
  }
}"""

_EXPECTED_EXECUTOR_OUTPUT_1 = """{
  "parameters": {
    "test_output2": {
      "stringValue": "bye world"
    }
  },
  "artifacts": {
    "test_output1": {
      "artifacts": [
        {
          "type": {
            "instanceSchema": "properties:\\ntitle: kfp.Model\\ntype: object\\n"
          },
          "uri": "gs://root/consumer/output1"
        }
      ]
    }
  }
}"""


class EntrypointTest(unittest.TestCase):

  def setUp(self):
    # Prepare mock
    self._import_func = mock.patch.object(
        entrypoint_utils,
        'import_func_from_source').start()
    self._mock_gcs_read = mock.patch(
        'kfp.containers._gcs_helper.GCSHelper.read_from_gcs_path',
    ).start()
    self._mock_gcs_write = mock.patch(
        'kfp.containers._gcs_helper.GCSHelper.write_to_gcs_path',
    ).start()

    self.addCleanup(mock.patch.stopall)


  def testMainWithV1Producer(self):
    """Tests the entrypoint with data passing with conventional KFP components.

    This test case emulates the following scenario:
    - User provides a function, namely `test_func`.
    - In test function, there are an input parameter (`test_param`) and an input
      artifact (`test_artifact`). And the user code generates an output
      artifact (`test_output1`) and an output parameter (`test_output2`).
    - The specified metadata JSON file location is at
      'executor_output_metadata.json'
    - The inputs of this step are all provided by conventional KFP components.
    """
    # Set mocked user function.
    self._import_func.return_value = main.test_func

    entrypoint.main(
        executor_metadata_json_file=_OUTPUT_METADATA_JSON_LOCATION,
        function_name='test_func',
        test_param_input_argo_param='hello from producer',
        test_artifact_input_path='gs://root/producer/output',
        test_output1_artifact_output_path='gs://root/consumer/output1',
        test_output2_parameter_output_path='gs://root/consumer/output2'
    )

    self._mock_gcs_write.assert_called_with(
        path=_OUTPUT_METADATA_JSON_LOCATION,
        content=_EXPECTED_EXECUTOR_OUTPUT_1)

  def testMainWithV2Producer(self):
    """Tests the entrypoint with data passing with new-styled KFP components.

    This test case emulates a similar scenario as testMainWithV1Producer, except
    for that the inputs of this step are all provided by a new-styled KFP
    component.
    """
    # Set mocked user function.
    self._import_func.return_value = main.test_func2
    # Set GFile read function
    self._mock_gcs_read.return_value = _PRODUCER_EXECUTOR_OUTPUT

    entrypoint.main(
        executor_metadata_json_file=_OUTPUT_METADATA_JSON_LOCATION,
        function_name='test_func2',
        test_param_input_param_metadata_file='gs://root/producer/executor_output_metadata.json',
        test_param_input_field_name='param_output',
        test_artifact_input_artifact_metadata_file='gs://root/producer/executor_output_metadata.json',
        test_artifact_input_output_name='artifact_output',
        test_output1_artifact_output_path='gs://root/consumer/output1',
        test_output2_parameter_output_path='gs://root/consumer/output2'
    )

    self._mock_gcs_write.assert_called_with(
        path=_OUTPUT_METADATA_JSON_LOCATION,
        content=_EXPECTED_EXECUTOR_OUTPUT_1)

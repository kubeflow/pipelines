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
import os
import shutil
import tempfile
import unittest

from kfp.containers import entrypoint
from kfp.containers import entrypoint_utils
# Import testdata to mock entrypoint_utils.import_func_from_source function.
from kfp.containers_tests.testdata import main

_OUTPUT_METADATA_JSON_LOCATION = 'executor_output_metadata.json'

_TEST_EXECUTOR_INPUT_V1_PRODUCER = """
{
  "inputs": {
    "artifacts": {
      "test_artifact": {
        "artifacts": [
          {
            "uri": "gs://root/test_artifact/",
            "name": "test_artifact",
            "type": { 
              "instanceSchema": "title: kfp.Artifact\\ntype: object\\n"
            }
          }
        ]
      }
    },
    "parameters": {
      "test_param": {
        "stringValue": "hello from producer"
      }
    }
  },
  "outputs": {
    "artifacts": {
      "test_output1": {
        "artifacts": [
          {
            "uri": "gs://root/test_output1/",
            "name": "test_output1",
            "type": { 
              "instanceSchema": "title: kfp.Model\\ntype: object\\nproperties:\\n  framework:\\n    type: string\\n  framework_version:\\n    type: string\\n"
            }
          }
        ]
      }
    }
  }
}
"""

_TEST_EXECUTOR_INPUT_V2_PRODUCER = """
{
  "inputs": {
    "artifacts": {
      "test_artifact": {
        "artifacts": [
          {
            "uri": "gs://root/test_artifact/",
            "name": "test_artifact",
            "type": { 
              "instanceSchema": "title: kfp.Dataset\\ntype: object\\nproperties:\\n  payload_format:\\n    type: string\\n  container_format:\\n    type: string"
            }
          }
        ]
      }
    },
    "parameters": {
      "test_param": {
        "stringValue": "hello from producer"
      }
    }
  },
  "outputs": {
    "artifacts": {
      "test_output1": {
        "artifacts": [
          {
            "uri": "gs://root/test_output1/",
            "name": "test_output1",
            "type": { 
              "instanceSchema": "title: kfp.Model\\ntype: object\\nproperties:\\n  framework:\\n    type: string\\n  framework_version:\\n    type: string\\n"
            }
          }
        ]
      }
    }
  }
}
"""

_EXPECTED_EXECUTOR_OUTPUT = """{
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
            "instanceSchema": "title: kfp.Model\\ntype: object\\nproperties:\\n  framework:\\n    type: string\\n  framework_version:\\n    type: string\\n"
          },
          "uri": "gs://root/test_output1/"
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

    # Create a temporary directory
    self._test_dir = tempfile.mkdtemp()
    self.addCleanup(shutil.rmtree, self._test_dir)
    self._old_dir = os.getcwd()
    os.chdir(self._test_dir)
    self.addCleanup(os.chdir, self._old_dir)
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
        executor_input_str=_TEST_EXECUTOR_INPUT_V1_PRODUCER,
        function_name='test_func',
        output_metadata_path=_OUTPUT_METADATA_JSON_LOCATION
    )

    # Check the actual executor output.
    with open(_OUTPUT_METADATA_JSON_LOCATION, 'r') as f:
      self.assertEqual(f.read(), _EXPECTED_EXECUTOR_OUTPUT)

  def testMainWithV2Producer(self):
    """Tests the entrypoint with data passing with new-styled KFP components.

    This test case emulates a similar scenario as testMainWithV1Producer, except
    for that the inputs of this step are all provided by a new-styled KFP
    component.
    """
    # Set mocked user function.
    self._import_func.return_value = main.test_func2

    entrypoint.main(
        executor_input_str=_TEST_EXECUTOR_INPUT_V2_PRODUCER,
        function_name='test_func2',
        output_metadata_path=_OUTPUT_METADATA_JSON_LOCATION
    )

    # Check the actual executor output.
    with open(_OUTPUT_METADATA_JSON_LOCATION, 'r') as f:
      self.assertEqual(f.read(), _EXPECTED_EXECUTOR_OUTPUT)

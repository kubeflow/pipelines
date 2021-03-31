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
"""Tests for kfp.components.executor"""

import os
import tempfile
from typing import Callable, NamedTuple, Optional
import unittest
import json

from kfp.components import executor, InputPath, OutputPath
from kfp.dsl import io_types
from kfp.dsl.io_types import Artifact, Dataset, InputArtifact, Metrics, Model, OutputArtifact

_EXECUTOR_INPUT = """\
{
  "inputs": {
    "parameters": {
      "input_parameter": {
        "stringValue": "Hello, KFP"
      }
    },
    "artifacts": {
      "input_artifact_one": {
        "artifacts": [
          {
            "metadata": {},
            "name": "input_artifact_one",
            "type": {
              "schemaTitle": "system.Dataset"
            },
            "uri": "gs://some-bucket/input_artifact_one"
          }
        ]
      }
    }
  },
  "outputs": {
    "artifacts": {
      "output_artifact_one": {
        "artifacts": [
          {
            "metadata": {},
            "name": "output_artifact_one",
            "type": {
              "schemaTitle": "system.Model"
            },
            "uri": "gs://some-bucket/output_artifact_one"
          }
        ]
      },
      "output_artifact_two": {
        "artifacts": [
          {
            "metadata": {},
            "name": "output_artifact_two",
            "type": {
              "schemaTitle": "system.Metrics"
            },
            "uri": "gs://some-bucket/output_artifact_two"
          }
        ]
      }
    },
    "parameters": {
      "output_parameter": {
        "outputFile": "gs://some-bucket/output_parameter"
      }
    },
    "outputFile": "%s/output_metadata.json"
  }
}
"""


class ExecutorTest(unittest.TestCase):

  def setUp(self):
    self.maxDiff = None
    self._test_dir = tempfile.mkdtemp()
    io_types._GCS_LOCAL_MOUNT_PREFIX = self._test_dir + '/'
    return super().setUp()

  def _get_executor(self,
                    func: Callable,
                    executor_input: Optional[str] = None) -> executor.Executor:
    if executor_input is None:
      executor_input = _EXECUTOR_INPUT

    executor_input_dict = json.loads(executor_input % self._test_dir)

    return executor.Executor(executor_input=executor_input_dict,
                             function_to_execute=func)

  def test_input_parameter(self):

    def test_func(input_parameter: str):
      self.assertEqual(input_parameter, "Hello, KFP")

    self._get_executor(test_func).execute()

  def test_input_artifact(self):

    def test_func(input_artifact_one: InputArtifact(Dataset)):
      self.assertEqual(input_artifact_one.uri,
                       'gs://some-bucket/input_artifact_one')
      self.assertEqual(
          input_artifact_one.path,
          os.path.join(self._test_dir, 'some-bucket/input_artifact_one'))
      self.assertEqual(input_artifact_one.get().name, 'input_artifact_one')

    self._get_executor(test_func).execute()

  def test_output_artifact(self):

    def test_func(output_artifact_one: OutputArtifact(Model)):
      # Test that output artifacts always have filename 'data' added.
      self.assertEqual(output_artifact_one.uri,
                       'gs://some-bucket/output_artifact_one/data')

      self.assertEqual(
          output_artifact_one.path,
          os.path.join(self._test_dir, 'some-bucket/output_artifact_one',
                       'data'))
      self.assertEqual(output_artifact_one.get().name, 'output_artifact_one')

    self._get_executor(test_func).execute()

  def test_output_parameter(self):

    def test_func(output_parameter_path: OutputPath(str)):
      # Test that output parameters just use the passed in filename.
      self.assertEqual(output_parameter_path,
                       'gs://some-bucket/output_parameter')

    self._get_executor(test_func).execute()

  def test_input_path_artifact(self):

    def test_func(input_artifact_one_path: InputPath('Dataset')):
      self.assertEqual(
          input_artifact_one_path,
          os.path.join(self._test_dir, 'some-bucket/input_artifact_one'))

    self._get_executor(test_func).execute()

  def test_output_path_artifact(self):

    def test_func(output_artifact_one_path: OutputPath('Model')):
      # Test that output path also get 'data' appended.
      self.assertEqual(
          output_artifact_one_path,
          os.path.join(self._test_dir, 'some-bucket/output_artifact_one/data'))

    self._get_executor(test_func).execute()

  def test_output_metadata(self):

    def test_func(output_artifact_two: OutputArtifact(Metrics)):
      output_artifact_two.get().metadata['key_1'] = 'value_1'
      output_artifact_two.get().metadata['key_2'] = 2
      output_artifact_two.uri = 'new-uri'

      # log_metric works here since the schema is specified as Metrics.
      output_artifact_two.get().log_metric('metric', 0.9)

    self._get_executor(test_func).execute()
    with open(os.path.join(self._test_dir, 'output_metadata.json'), 'r') as f:
      output_metadata = json.loads(f.read())
    self.assertDictEqual(
        output_metadata, {
            'artifacts': {
                'output_artifact_one': {
                    'artifacts': [{
                        'name': 'output_artifact_one',
                        'uri': 'gs://some-bucket/output_artifact_one/data',
                        'metadata': {}
                    }]
                },
                'output_artifact_two': {
                    'artifacts': [{
                        'name': 'output_artifact_two',
                        'uri': 'new-uri',
                        'metadata': {
                            'key_1': 'value_1',
                            'key_2': 2,
                            'metric': 0.9
                        }
                    }]
                }
            }
        })

  def test_function_string_output(self):
    executor_input = """\
    {
      "inputs": {
        "parameters": {
          "first_message": {
            "stringValue": "Hello"
          },
          "second_message": {
            "stringValue": "World"
          }
        }
      },
      "outputs": {
        "parameters": {
          "output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%s/output_metadata.json"
      }
    }
    """

    def test_func(first_message: str, second_message: str) -> str:
      return first_message + ", " + second_message

    self._get_executor(test_func, executor_input).execute()
    with open(os.path.join(self._test_dir, 'output_metadata.json'), 'r') as f:
      output_metadata = json.loads(f.read())
    self.assertDictEqual(output_metadata, {
        "parameters": {
            "Output": {
                "stringValue": "Hello, World"
            }
        },
    })

  def test_function_with_int_output(self):
    executor_input = """\
    {
      "inputs": {
        "parameters": {
          "first": {
            "intValue": 40
          },
          "second": {
            "intValue": 2
          }
        }
      },
      "outputs": {
        "parameters": {
          "output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%s/output_metadata.json"
      }
    }
    """

    def test_func(first: int, second: int) -> int:
      return first + second

    self._get_executor(test_func, executor_input).execute()
    with open(os.path.join(self._test_dir, 'output_metadata.json'), 'r') as f:
      output_metadata = json.loads(f.read())
    self.assertDictEqual(output_metadata, {
        "parameters": {
            "Output": {
                "intValue": 42
            }
        },
    })

  def test_function_with_int_output(self):
    executor_input = """\
    {
      "inputs": {
        "parameters": {
          "first_message": {
            "stringValue": "Hello"
          },
          "second_message": {
            "stringValue": "World"
          }
        }
      },
      "outputs": {
        "artifacts": {
          "Output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%s/output_metadata.json"
      }
    }
    """

    def test_func(first_message: str, second_message: str) -> str:
      return first_message + ", " + second_message

    self._get_executor(test_func, executor_input).execute()
    with open(os.path.join(self._test_dir, 'output_metadata.json'), 'r') as f:
      output_metadata = json.loads(f.read())
    self.assertDictEqual(output_metadata, {
        "parameters": {
            "Output": {
                "stringValue": "Hello, World"
            }
        },
    })

  def test_artifact_output(self):
    executor_input = """\
    {
      "inputs": {
        "parameters": {
          "first": {
            "stringValue": "Hello"
          },
          "second": {
            "stringValue": "World"
          }
        }
      },
      "outputs": {
        "artifacts": {
          "Output": {
            "artifacts": [
              {
                "name": "output",
                "type": {
                  "schemaTitle": "system.Artifact"
                },
                "uri": "gs://some-bucket/output"
              }
            ]
          }
        },
        "outputFile": "%s/output_metadata.json"
      }
    }
    """

    def test_func(first: str, second: str) -> Artifact:
      return first + ", " + second

    self._get_executor(test_func, executor_input).execute()
    with open(os.path.join(self._test_dir, 'output_metadata.json'), 'r') as f:
      output_metadata = json.loads(f.read())
    self.assertDictEqual(
        output_metadata, {
            'artifacts': {
                'Output': {
                    'artifacts': [{
                        'metadata': {},
                        'name': 'output',
                        'uri': 'gs://some-bucket/output/data'
                    }]
                }
            }
        })

    with open(os.path.join(self._test_dir, 'some-bucket/output/data'),
              'r') as f:
      artifact_payload = f.read()
    self.assertEqual(artifact_payload, "Hello, World")

  def test_named_tuple_output(self):
    executor_input = """\
    {
      "outputs": {
        "artifacts": {
          "output_dataset": {
            "artifacts": [
              {
                "name": "output_dataset",
                "type": {
                  "schemaTitle": "system.Dataset"
                },
                "uri": "gs://some-bucket/output_dataset"
              }
            ]
          }
        },
        "parameters": {
          "output_int": {
            "outputFile": "gs://some-bucket/output_int"
          },
          "output_string": {
            "outputFile": "gs://some-bucket/output_string"
          }
        },
        "outputFile": "%s/output_metadata.json"
      }
    }
    """

    def test_func() -> NamedTuple('Outputs', [
        ("output_dataset", Dataset),
        ("output_int", int),
        ("output_string", str),
    ]):
      from collections import namedtuple
      output = namedtuple('Outputs',
                          ['output_dataset', 'output_int', 'output_string'])
      return output("Dataset contents", 101, "Some output string")

    self._get_executor(test_func, executor_input).execute()
    with open(os.path.join(self._test_dir, 'output_metadata.json'), 'r') as f:
      output_metadata = json.loads(f.read())
    self.assertDictEqual(
        output_metadata, {
            'artifacts': {
                'output_dataset': {
                    'artifacts': [{
                        'metadata': {},
                        'name': 'output_dataset',
                        'uri': 'gs://some-bucket/output_dataset/data'
                    }]
                }
            },
            "parameters": {
                "output_string": {
                    "stringValue": "Some output string"
                },
                "output_int": {
                    "intValue": 101
                }
            },
        })

    with open(os.path.join(self._test_dir, 'some-bucket/output_dataset/data'),
              'r') as f:
      artifact_payload = f.read()
    self.assertEqual(artifact_payload, "Dataset contents")

# Copyright 2021 The Kubeflow Authors
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
"""Tests for kfp.components.executor."""

import json
import os
import tempfile
import unittest
from typing import Callable, Dict, List, NamedTuple, Optional

from kfp.components import executor
from kfp.components.task_final_status import PipelineTaskFinalStatus
from kfp.components.types import artifact_types
from kfp.components.types.artifact_types import Artifact
from kfp.components.types.artifact_types import Dataset
from kfp.components.types.artifact_types import Metrics
from kfp.components.types.artifact_types import Model
from kfp.components.types.type_annotations import Input
from kfp.components.types.type_annotations import InputPath
from kfp.components.types.type_annotations import Output
from kfp.components.types.type_annotations import OutputPath

_EXECUTOR_INPUT = """\
{
  "inputs": {
    "parameterValues": {
      "input_parameter": "Hello, KFP"
    },
    "artifacts": {
      "input_artifact_one_path": {
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
      "output_artifact_one_path": {
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
      "output_parameter_path": {
        "outputFile": "%(test_dir)s/gcs/some-bucket/some_task/nested/output_parameter"
      }
    },
    "outputFile": "%(test_dir)s/output_metadata.json"
  }
}
"""


class ExecutorTest(unittest.TestCase):

    @classmethod
    def setUp(cls):
        cls.maxDiff = None
        cls._test_dir = tempfile.mkdtemp()
        artifact_types._GCS_LOCAL_MOUNT_PREFIX = cls._test_dir + '/'
        artifact_types._MINIO_LOCAL_MOUNT_PREFIX = cls._test_dir + '/minio/'
        artifact_types._S3_LOCAL_MOUNT_PREFIX = cls._test_dir + '/s3/'

    def _get_executor(
            self,
            func: Callable,
            executor_input: Optional[str] = None) -> executor.Executor:
        if executor_input is None:
            executor_input = _EXECUTOR_INPUT
        executor_input_dict = json.loads(executor_input %
                                         {"test_dir": self._test_dir})

        return executor.Executor(
            executor_input=executor_input_dict, function_to_execute=func)

    def test_input_parameter(self):

        def test_func(input_parameter: str):
            self.assertEqual(input_parameter, "Hello, KFP")

        self._get_executor(test_func).execute()

    def test_input_artifact(self):

        def test_func(input_artifact_one_path: Input[Dataset]):
            self.assertEqual(input_artifact_one_path.uri,
                             'gs://some-bucket/input_artifact_one')
            self.assertEqual(
                input_artifact_one_path.path,
                os.path.join(self._test_dir, 'some-bucket/input_artifact_one'))
            self.assertEqual(input_artifact_one_path.name, 'input_artifact_one')

        self._get_executor(test_func).execute()

    def test_output_artifact(self):

        def test_func(output_artifact_one_path: Output[Model]):
            self.assertEqual(output_artifact_one_path.uri,
                             'gs://some-bucket/output_artifact_one')

            self.assertEqual(
                output_artifact_one_path.path,
                os.path.join(self._test_dir, 'some-bucket/output_artifact_one'))
            self.assertEqual(output_artifact_one_path.name,
                             'output_artifact_one')

        self._get_executor(test_func).execute()

    def test_output_parameter(self):

        def test_func(output_parameter_path: OutputPath(str)):
            # Test that output parameters just use the passed in filename.
            self.assertEqual(
                output_parameter_path, self._test_dir +
                '/gcs/some-bucket/some_task/nested/output_parameter')
            with open(output_parameter_path, 'w') as f:
                f.write('Hello, World!')

        self._get_executor(test_func).execute()

    def test_input_path_artifact(self):

        def test_func(input_artifact_one_path: InputPath('Dataset')):
            self.assertEqual(
                input_artifact_one_path,
                os.path.join(self._test_dir, 'some-bucket/input_artifact_one'))

        self._get_executor(test_func).execute()

    def test_output_path_artifact(self):

        def test_func(output_artifact_one_path: OutputPath('Model')):
            self.assertEqual(
                output_artifact_one_path,
                os.path.join(self._test_dir, 'some-bucket/output_artifact_one'))

        self._get_executor(test_func).execute()

    def test_output_metadata(self):

        def test_func(output_artifact_two: Output[Metrics]):
            output_artifact_two.metadata['key_1'] = 'value_1'
            output_artifact_two.metadata['key_2'] = 2
            output_artifact_two.uri = 'new-uri'

            # log_metric works here since the schema is specified as Metrics.
            output_artifact_two.log_metric('metric', 0.9)

        self._get_executor(test_func).execute()
        with open(os.path.join(self._test_dir, 'output_metadata.json'),
                  'r') as f:
            output_metadata = json.loads(f.read())
        self.assertDictEqual(
            output_metadata, {
                'artifacts': {
                    'output_artifact_one_path': {
                        'artifacts': [{
                            'name': 'output_artifact_one',
                            'uri': 'gs://some-bucket/output_artifact_one',
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
        "parameterValues": {
          "first_message": "Hello",
          "second_message": "",
          "third_message": "World"
        }
      },
      "outputs": {
        "parameters": {
          "output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%(test_dir)s/output_metadata.json"
      }
    }
    """

        def test_func(
            first_message: str,
            second_message: str,
            third_message: str,
        ) -> str:
            return first_message + ", " + second_message + ", " + third_message

        self._get_executor(test_func, executor_input).execute()
        with open(os.path.join(self._test_dir, 'output_metadata.json'),
                  'r') as f:
            output_metadata = json.loads(f.read())
        self.assertDictEqual(output_metadata, {
            "parameterValues": {
                "Output": "Hello, , World"
            },
        })

    def test_function_with_int_output(self):
        executor_input = """\
    {
      "inputs": {
        "parameterValues": {
          "first": 40,
          "second": 2
        }
      },
      "outputs": {
        "parameters": {
          "output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%(test_dir)s/output_metadata.json"
      }
    }
    """

        def test_func(first: int, second: int) -> int:
            return first + second

        self._get_executor(test_func, executor_input).execute()
        with open(os.path.join(self._test_dir, 'output_metadata.json'),
                  'r') as f:
            output_metadata = json.loads(f.read())
        self.assertDictEqual(output_metadata, {
            "parameterValues": {
                "Output": 42
            },
        })

    def test_function_with_float_output(self):
        executor_input = """\
    {
      "inputs": {
        "parameterValues": {
          "first": 0.0,
          "second": 1.2
        }
      },
      "outputs": {
        "parameters": {
          "output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%(test_dir)s/output_metadata.json"
      }
    }
    """

        def test_func(first: float, second: float) -> float:
            return first + second

        self._get_executor(test_func, executor_input).execute()
        with open(os.path.join(self._test_dir, 'output_metadata.json'),
                  'r') as f:
            output_metadata = json.loads(f.read())
        self.assertDictEqual(output_metadata, {
            "parameterValues": {
                "Output": 1.2
            },
        })

    def test_function_with_list_output(self):
        executor_input = """\
    {
      "inputs": {
        "parameterValues": {
          "first": 40,
          "second": 2
        }
      },
      "outputs": {
        "parameters": {
          "output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%(test_dir)s/output_metadata.json"
      }
    }
    """

        def test_func(first: int, second: int) -> List:
            return [first, second]

        self._get_executor(test_func, executor_input).execute()
        with open(os.path.join(self._test_dir, 'output_metadata.json'),
                  'r') as f:
            output_metadata = json.loads(f.read())
        self.assertDictEqual(output_metadata, {
            "parameterValues": {
                "Output": [40, 2]
            },
        })

    def test_function_with_dict_output(self):
        executor_input = """\
    {
      "inputs": {
        "parameterValues": {
          "first": 40,
          "second": 2
        }
      },
      "outputs": {
        "parameters": {
          "output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%(test_dir)s/output_metadata.json"
      }
    }
    """

        def test_func(first: int, second: int) -> Dict:
            return {"first": first, "second": second}

        self._get_executor(test_func, executor_input).execute()
        with open(os.path.join(self._test_dir, 'output_metadata.json'),
                  'r') as f:
            output_metadata = json.loads(f.read())
        self.assertDictEqual(output_metadata, {
            'parameterValues': {
                'Output': {
                    'first': 40,
                    'second': 2
                }
            },
        })

    def test_function_with_typed_list_output(self):
        executor_input = """\
    {
      "inputs": {
        "parameterValues": {
          "first": 40,
          "second": 2
        }
      },
      "outputs": {
        "parameters": {
          "output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%(test_dir)s/output_metadata.json"
      }
    }
    """

        def test_func(first: int, second: int) -> List[int]:
            return [first, second]

        self._get_executor(test_func, executor_input).execute()
        with open(os.path.join(self._test_dir, 'output_metadata.json'),
                  'r') as f:
            output_metadata = json.loads(f.read())
        self.assertDictEqual(output_metadata, {
            "parameterValues": {
                "Output": [40, 2]
            },
        })

    def test_function_with_typed_dict_output(self):
        executor_input = """\
    {
      "inputs": {
        "parameterValues": {
          "first": 40,
          "second": 2
        }
      },
      "outputs": {
        "parameters": {
          "output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%(test_dir)s/output_metadata.json"
      }
    }
    """

        def test_func(first: int, second: int) -> Dict[str, int]:
            return {"first": first, "second": second}

        self._get_executor(test_func, executor_input).execute()
        with open(os.path.join(self._test_dir, 'output_metadata.json'),
                  'r') as f:
            output_metadata = json.loads(f.read())
        self.assertDictEqual(output_metadata, {
            "parameterValues": {
                "Output": {
                    "first": 40,
                    "second": 2
                }
            },
        })

    def test_artifact_output(self):
        executor_input = """\
    {
      "inputs": {
        "parameterValues": {
          "first":  "Hello",
          "second": "World"
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
        "outputFile": "%(test_dir)s/output_metadata.json"
      }
    }
    """

        def test_func(first: str, second: str) -> Artifact:
            return first + ", " + second

        self._get_executor(test_func, executor_input).execute()
        with open(os.path.join(self._test_dir, 'output_metadata.json'),
                  'r') as f:
            output_metadata = json.loads(f.read())
        self.assertDictEqual(
            output_metadata, {
                'artifacts': {
                    'Output': {
                        'artifacts': [{
                            'metadata': {},
                            'name': 'output',
                            'uri': 'gs://some-bucket/output'
                        }]
                    }
                }
            })

        with open(os.path.join(self._test_dir, 'some-bucket/output'), 'r') as f:
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
        "outputFile": "%(test_dir)s/output_metadata.json"
      }
    }
    """

        # Functions returning named tuples should work.
        def func_returning_named_tuple() -> NamedTuple('Outputs', [
            ("output_dataset", Dataset),
            ("output_int", int),
            ("output_string", str),
        ]):
            from collections import namedtuple
            output = namedtuple(
                'Outputs', ['output_dataset', 'output_int', 'output_string'])
            return output("Dataset contents", 101, "Some output string")

        # Functions returning plain tuples should work too.
        def func_returning_plain_tuple() -> NamedTuple('Outputs', [
            ("output_dataset", Dataset),
            ("output_int", int),
            ("output_string", str),
        ]):
            return ("Dataset contents", 101, "Some output string")

        for test_func in [
                func_returning_named_tuple, func_returning_plain_tuple
        ]:
            self._get_executor(test_func, executor_input).execute()
            with open(
                    os.path.join(self._test_dir, 'output_metadata.json'),
                    'r') as f:
                output_metadata = json.loads(f.read())
            self.assertDictEqual(
                output_metadata, {
                    'artifacts': {
                        'output_dataset': {
                            'artifacts': [{
                                'metadata': {},
                                'name': 'output_dataset',
                                'uri': 'gs://some-bucket/output_dataset'
                            }]
                        }
                    },
                    "parameterValues": {
                        "output_int": 101,
                        "output_string": "Some output string"
                    },
                })

            with open(
                    os.path.join(self._test_dir, 'some-bucket/output_dataset'),
                    'r') as f:
                artifact_payload = f.read()
            self.assertEqual(artifact_payload, "Dataset contents")

    def test_function_with_optional_inputs(self):
        executor_input = """\
    {
      "inputs": {
        "parameterValues": {
          "first_message": "Hello",
          "second_message": "World"
        }
      },
      "outputs": {
        "parameters": {
          "output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%(test_dir)s/output_metadata.json"
      }
    }
    """

        def test_func(
            first_message: str = 'default value',
            second_message: Optional[str] = None,
            third_message: Optional[str] = None,
            forth_argument: str = 'abc',
            fifth_argument: int = 100,
            sixth_argument: float = 1.23,
            seventh_argument: bool = True,
            eighth_argument: list = [1, 2],
            ninth_argument: dict = {'a': 1},
        ) -> str:
            return (f'{first_message} ({type(first_message)}), '
                    f'{second_message} ({type(second_message)}), '
                    f'{third_message} ({type(third_message)}), '
                    f'{forth_argument} ({type(forth_argument)}), '
                    f'{fifth_argument} ({type(fifth_argument)}), '
                    f'{sixth_argument} ({type(sixth_argument)}), '
                    f'{seventh_argument} ({type(seventh_argument)}), '
                    f'{eighth_argument} ({type(eighth_argument)}), '
                    f'{ninth_argument} ({type(ninth_argument)}).')

        self._get_executor(test_func, executor_input).execute()
        with open(os.path.join(self._test_dir, 'output_metadata.json'),
                  'r') as f:
            output_metadata = json.loads(f.read())
        self.assertDictEqual(
            output_metadata, {
                "parameterValues": {
                    "Output": "Hello (<class 'str'>), "
                              "World (<class 'str'>), "
                              "None (<class 'NoneType'>), "
                              "abc (<class 'str'>), "
                              "100 (<class 'int'>), "
                              "1.23 (<class 'float'>), "
                              "True (<class 'bool'>), "
                              "[1, 2] (<class 'list'>), "
                              "{'a': 1} (<class 'dict'>)."
                },
            })

    def test_function_with_pipeline_task_final_status(self):
        executor_input = """\
    {
      "inputs": {
        "parameterValues": {
          "status": {"error":{"code":9,"message":"The DAG failed because some tasks failed. The failed tasks are: [fail-op]."},"pipelineJobResourceName":"projects/123/locations/us-central1/pipelineJobs/pipeline-456", "pipelineTaskName": "upstream-task", "state":"FAILED"}
        }
      },
      "outputs": {
        "parameters": {
          "output": {
            "outputFile": "gs://some-bucket/output"
          }
        },
        "outputFile": "%(test_dir)s/output_metadata.json"
      }
    }
    """

        def test_func(status: PipelineTaskFinalStatus) -> str:
            return (f'Pipeline status: {status.state}\n'
                    f'Job resource name: {status.pipeline_job_resource_name}\n'
                    f'Pipeline task name: {status.pipeline_task_name}\n'
                    f'Error code: {status.error_code}\n'
                    f'Error message: {status.error_message}')

        self._get_executor(test_func, executor_input).execute()
        with open(os.path.join(self._test_dir, 'output_metadata.json'),
                  'r') as f:
            output_metadata = json.loads(f.read())
        self.assertDictEqual(
            output_metadata, {
                'parameterValues': {
                    'Output':
                        'Pipeline status: FAILED\n'
                        'Job resource name: projects/123/locations/us-central1/pipelineJobs/pipeline-456\n'
                        'Pipeline task name: upstream-task\n'
                        'Error code: 9\n'
                        'Error message: The DAG failed because some tasks failed. The failed tasks are: [fail-op].'
                },
            })


if __name__ == '__main__':
    unittest.main()

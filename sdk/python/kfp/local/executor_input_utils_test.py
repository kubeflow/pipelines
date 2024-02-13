# Copyright 2023 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License")
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
"""Tests for executor_input_utils.py."""

import unittest

from google.protobuf import json_format
from kfp import dsl
from kfp.local import executor_input_utils
from kfp.local import testing_utilities
from kfp.pipeline_spec import pipeline_spec_pb2


class GetLocalPipelineResourceName(testing_utilities.MockedDatetimeTestCase):

    def test(self):
        actual = executor_input_utils.get_local_pipeline_resource_name(
            'my-pipeline')
        expected = 'my-pipeline-2023-10-10-13-32-59-420710'
        self.assertEqual(actual, expected)


class GetLocalTaskResourceName(unittest.TestCase):

    def test(self):
        actual = executor_input_utils.get_local_task_resource_name(
            'comp-my-comp')
        expected = 'my-comp'
        self.assertEqual(actual, expected)


class TestConstructLocalTaskRoot(testing_utilities.MockedDatetimeTestCase):

    def test(self):

        task_root = executor_input_utils.construct_local_task_root(
            pipeline_root='/foo/bar',
            pipeline_resource_name='my-pipeline-2023-10-10-13-32-59-420710',
            task_resource_name='my-comp',
        )
        self.assertEqual(
            task_root,
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/my-comp',
        )


class TestConstructExecutorInput(unittest.TestCase):

    def test_no_inputs(self):
        component_spec = pipeline_spec_pb2.ComponentSpec()
        json_format.ParseDict(
            {
                'outputDefinitions': {
                    'parameters': {
                        'Output': {
                            'parameterType': 'STRING'
                        }
                    }
                },
                'executorLabel': 'exec-comp'
            }, component_spec)
        arguments = {}
        task_root = '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp'

        actual = executor_input_utils.construct_executor_input(
            component_spec=component_spec,
            arguments=arguments,
            task_root=task_root,
            block_input_artifact=True,
        )
        expected = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(
            {
                'inputs': {},
                'outputs': {
                    'parameters': {
                        'Output': {
                            'outputFile':
                                '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/Output'
                        }
                    },
                    'outputFile':
                        '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/executor_output.json'
                }
            }, expected)
        self.assertEqual(actual, expected)

    def test_various_io_types(self):
        component_spec = pipeline_spec_pb2.ComponentSpec()
        json_format.ParseDict(
            {
                'inputDefinitions': {
                    'parameters': {
                        'boolean': {
                            'parameterType': 'BOOLEAN'
                        }
                    }
                },
                'outputDefinitions': {
                    'artifacts': {
                        'out_a': {
                            'artifactType': {
                                'schemaTitle': 'system.Dataset',
                                'schemaVersion': '0.0.1'
                            }
                        }
                    },
                    'parameters': {
                        'Output': {
                            'parameterType': 'NUMBER_INTEGER'
                        }
                    }
                },
                'executorLabel': 'exec-comp'
            }, component_spec)
        arguments = {'boolean': False}
        task_root = '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp'

        actual = executor_input_utils.construct_executor_input(
            component_spec=component_spec,
            arguments=arguments,
            task_root=task_root,
            block_input_artifact=True,
        )
        expected = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(
            {
                'inputs': {
                    'parameterValues': {
                        'boolean': False
                    }
                },
                'outputs': {
                    'parameters': {
                        'Output': {
                            'outputFile':
                                '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/Output'
                        }
                    },
                    'artifacts': {
                        'out_a': {
                            'artifacts': [{
                                'name':
                                    'out_a',
                                'type': {
                                    'schemaTitle': 'system.Dataset',
                                    'schemaVersion': '0.0.1'
                                },
                                'uri':
                                    '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/out_a',
                                'metadata': {}
                            }]
                        }
                    },
                    'outputFile':
                        '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/executor_output.json'
                }
            }, expected)
        self.assertEqual(actual, expected)

    def test_block_input_artifact(self):
        component_spec = pipeline_spec_pb2.ComponentSpec()
        json_format.ParseDict(
            {
                'inputDefinitions': {
                    'artifacts': {
                        'in_artifact': {
                            'artifactType': {
                                'schemaTitle': 'system.Artifact',
                                'schemaVersion': '0.0.1'
                            }
                        }
                    }
                },
                'executorLabel': 'exec-comp'
            }, component_spec)
        arguments = {}
        task_root = '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp'
        with self.assertRaisesRegex(
                ValueError,
                'Input artifacts are not yet supported for local execution.'):
            executor_input_utils.construct_executor_input(
                component_spec=component_spec,
                arguments=arguments,
                task_root=task_root,
                block_input_artifact=True,
            )

    def test_allow_input_artifact(self):
        component_spec = pipeline_spec_pb2.ComponentSpec()
        json_format.ParseDict(
            {
                'inputDefinitions': {
                    'artifacts': {
                        'in_artifact': {
                            'artifactType': {
                                'schemaTitle': 'system.Artifact',
                                'schemaVersion': '0.0.1'
                            }
                        }
                    }
                },
                'executorLabel': 'exec-comp'
            }, component_spec)
        task_root = '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp'
        arguments = {
            'in_artifact':
                dsl.Artifact(
                    name='artifact',
                    uri='/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/prev-comp/artifact',
                    metadata={'foo': 'bar'})
        }
        actual = executor_input_utils.construct_executor_input(
            component_spec=component_spec,
            arguments=arguments,
            task_root=task_root,
            # this param says input artifacts should be permitted
            block_input_artifact=False,
        )
        expected = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(
            {
                'inputs': {
                    'artifacts': {
                        'in_artifact': {
                            'artifacts': [{
                                'name':
                                    'artifact',
                                'type': {
                                    'schemaTitle': 'system.Artifact',
                                    'schemaVersion': '0.0.1'
                                },
                                'uri':
                                    '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/prev-comp/artifact',
                                'metadata': {
                                    'foo': 'bar'
                                }
                            }]
                        }
                    }
                },
                'outputs': {
                    'outputFile':
                        '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/executor_output.json'
                }
            }, expected)
        self.assertEqual(actual, expected)

    def test_fstring_case(self):
        component_spec = pipeline_spec_pb2.ComponentSpec()
        json_format.ParseDict(
            {
                'inputDefinitions': {
                    'parameters': {
                        'string': {
                            'parameterType': 'STRING'
                        }
                    }
                },
                'outputDefinitions': {
                    'parameters': {
                        'Output': {
                            'parameterType': 'STRING'
                        }
                    }
                },
                'executorLabel': 'exec-identity'
            }, component_spec)
        expected_executor_input = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(
            {
                'inputs': {
                    'parameterValues': {
                        'pipelinechannel--string':
                            'baz',
                        'string':
                            "bar-{{$.inputs.parameters['pipelinechannel--string']}}"
                    }
                },
                'outputs': {
                    'parameters': {
                        'Output': {
                            'outputFile':
                                '/foo/bar/local_outputs/my-pipeline-2024-01-26-11-10-57XX-530768/identity/Output'
                        }
                    },
                    'outputFile':
                        '/foo/bar/local_outputs/my-pipeline-2024-01-26-11-10-57XX-530768/identity/executor_output.json'
                }
            }, expected_executor_input)
        actual_executor_input = executor_input_utils.construct_executor_input(
            component_spec=component_spec,
            arguments={
                'pipelinechannel--string':
                    'baz',
                # covers the case of an f-string, where the value of
                # string includes an interpolation of
                # pipelinechannel--string
                'string':
                    "bar-{{$.inputs.parameters['pipelinechannel--string']}}"
            },
            task_root='/foo/bar/local_outputs/my-pipeline-2024-01-26-11-10-57XX-530768/identity',
            block_input_artifact=True,
        )
        self.assertEqual(
            expected_executor_input,
            actual_executor_input,
        )


class TestExecutorInputToDict(unittest.TestCase):

    def test_with_ints_and_floats(self):
        component_spec = pipeline_spec_pb2.ComponentSpec()
        json_format.ParseDict(
            {
                'inputDefinitions': {
                    'parameters': {
                        'x': {
                            'parameterType': 'NUMBER_INTEGER'
                        },
                        'y': {
                            'parameterType': 'NUMBER_DOUBLE'
                        }
                    }
                },
                'outputDefinitions': {
                    'parameters': {
                        'Output': {
                            'parameterType': 'STRING'
                        }
                    }
                },
                'executorLabel': 'exec-comp'
            }, component_spec)

        executor_input = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(
            {
                'inputs': {
                    'parameterValues': {
                        'x': 1.0,
                        'y': 2.0
                    }
                },
                'outputs': {
                    'parameters': {
                        'Output': {
                            'outputFile':
                                '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/Output'
                        }
                    },
                    'outputFile':
                        '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/executor_output.json'
                }
            }, executor_input)

        executor_input_dict = executor_input_utils.executor_input_to_dict(
            executor_input=executor_input,
            component_spec=component_spec,
        )
        expected = {
            'inputs': {
                'parameterValues': {
                    'x': 1,
                    'y': 2.0
                }
            },
            'outputs': {
                'parameters': {
                    'Output': {
                        'outputFile':
                            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/Output'
                    }
                },
                'outputFile':
                    '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/executor_output.json'
            }
        }
        # assert types since 1.0 == 1
        self.assertIsInstance(
            executor_input_dict['inputs']['parameterValues']['x'], int)
        self.assertIsInstance(
            executor_input_dict['inputs']['parameterValues']['y'], float)
        self.assertEqual(executor_input_dict, expected)


if __name__ == '__main__':
    unittest.main()

# Copyright 2023 The Kubeflow Authors
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
"""Tests for executor_output_utils.py."""

import json
import os
import tempfile
from typing import List
import unittest

from absl.testing import parameterized
from google.protobuf import json_format
from google.protobuf import struct_pb2
from kfp import dsl
from kfp.local import executor_output_utils
from kfp.local import testing_utilities
from kfp.pipeline_spec import pipeline_spec_pb2


class TestGetOutputsFromMessages(
        testing_utilities.LocalRunnerEnvironmentTestCase):

    def test(self):
        executor_input = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(
            {
                'inputs': {
                    'parameterValues': {
                        'string_in': 'foo'
                    }
                },
                'outputs': {
                    'parameters': {
                        'int_out': {
                            'outputFile':
                                'foo/multiple-io-2023-11-09-12-12-05-528112/multiple-io/int_out'
                        },
                        'str_out': {
                            'outputFile':
                                'foo/multiple-io-2023-11-09-12-12-05-528112/multiple-io/str_out'
                        }
                    },
                    'artifacts': {
                        'dataset_out': {
                            'artifacts': [{
                                'name':
                                    'dataset_out',
                                'type': {
                                    'schemaTitle': 'system.Dataset',
                                    'schemaVersion': '0.0.1'
                                },
                                'uri':
                                    'foo/multiple-io-2023-11-09-12-12-05-528112/multiple-io/dataset_out',
                                'metadata': {}
                            }]
                        }
                    },
                    'outputFile':
                        'foo/multiple-io-2023-11-09-12-12-05-528112/multiple-io/executor_output.json'
                }
            }, executor_input)
        component_spec = pipeline_spec_pb2.ComponentSpec()
        json_format.ParseDict(
            {
                'inputDefinitions': {
                    'parameters': {
                        'string_in': {
                            'parameterType': 'STRING'
                        }
                    }
                },
                'outputDefinitions': {
                    'artifacts': {
                        'dataset_out': {
                            'artifactType': {
                                'schemaTitle': 'system.Dataset',
                                'schemaVersion': '0.0.1'
                            }
                        }
                    },
                    'parameters': {
                        'int_out': {
                            'parameterType': 'NUMBER_INTEGER'
                        },
                        'str_out': {
                            'parameterType': 'STRING'
                        }
                    }
                },
                'executorLabel': 'exec-multiple-io'
            }, component_spec)
        executor_output = pipeline_spec_pb2.ExecutorOutput()
        json_format.ParseDict(
            {
                'parameterValues': {
                    'int_out': 1,
                    'str_out': 'foo'
                },
                'artifacts': {
                    'dataset_out': {
                        'artifacts': [{
                            'name':
                                'dataset_out',
                            'uri':
                                'foo/multiple-io-2023-11-09-12-12-05-528112/multiple-io/dataset_out',
                            'metadata': {
                                'foo': 'bar'
                            }
                        }]
                    }
                }
            }, executor_output)

        os.makedirs(os.path.dirname(executor_input.outputs.output_file))
        with open(executor_input.outputs.output_file, 'w') as f:
            f.write(json_format.MessageToJson(executor_output))
        outputs = executor_output_utils.get_outputs_for_task(
            executor_input=executor_input,
            component_spec=component_spec,
        )

        self.assertEqual(outputs['int_out'], 1)
        self.assertEqual(outputs['str_out'], 'foo')
        assert_artifacts_equal(
            self,
            outputs['dataset_out'],
            dsl.Dataset(
                name='dataset_out',
                uri='foo/multiple-io-2023-11-09-12-12-05-528112/multiple-io/dataset_out',
                metadata={'foo': 'bar'}),
        )


class TestLoadExecutorOutput(unittest.TestCase):

    def test_exists(self):
        with tempfile.TemporaryDirectory() as tempdir:
            executor_output = pipeline_spec_pb2.ExecutorOutput(
                parameter_values={
                    'foo': struct_pb2.Value(string_value='foo_value')
                })
            path = os.path.join(tempdir, 'executor_output.json')
            testing_utilities.write_proto_to_json_file(executor_output, path)

            actual = executor_output_utils.load_executor_output(path)
            expected = pipeline_spec_pb2.ExecutorOutput()
            expected.parameter_values['foo'].CopyFrom(
                struct_pb2.Value(string_value='foo_value'))
            self.assertEqual(
                actual.SerializeToString(deterministic=True),
                expected.SerializeToString(deterministic=True),
            )

    def test_not_exists(self):
        non_existent_path = 'non_existent_path.json'
        actual = executor_output_utils.load_executor_output(non_existent_path)
        expected = pipeline_spec_pb2.ExecutorOutput()
        self.assertEqual(
            actual.SerializeToString(deterministic=True),
            expected.SerializeToString(deterministic=True),
        )


class TestGetOutputsFromExecutorOutput(unittest.TestCase):

    def test_param_and_artifact_outputs(self):
        # include the special case of an output int for more complete testing of behavior
        executor_output = pipeline_spec_pb2.ExecutorOutput()
        json_format.ParseDict(
            {
                'parameterValues': {
                    'int_out': 1,
                    'str_out': 'foo'
                },
                'artifacts': {
                    'dataset_out': {
                        'artifacts': [{
                            'name':
                                'dataset_out',
                            'uri':
                                'foo/multiple-io-2023-11-09-11-31-31-064429/multiple-io/dataset_out',
                            'metadata': {
                                'foo': 'bar'
                            }
                        }]
                    }
                }
            }, executor_output)
        executor_input = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(
            {
                'inputs': {
                    'parameterValues': {
                        'string_in': 'foo'
                    }
                },
                'outputs': {
                    'parameters': {
                        'int_out': {
                            'outputFile':
                                'foo/temp_root/multiple-io-2023-11-09-11-31-31-064429/multiple-io/int_out'
                        },
                        'str_out': {
                            'outputFile':
                                'foo/multiple-io-2023-11-09-11-31-31-064429/multiple-io/str_out'
                        }
                    },
                    'artifacts': {
                        'dataset_out': {
                            'artifacts': [{
                                'name':
                                    'dataset_out',
                                'type': {
                                    'schemaTitle': 'system.Dataset',
                                    'schemaVersion': '0.0.1'
                                },
                                'uri':
                                    'foo/multiple-io-2023-11-09-11-31-31-064429/multiple-io/dataset_out',
                                'metadata': {}
                            }]
                        }
                    },
                    'outputFile':
                        'foo/multiple-io-2023-11-09-11-31-31-064429/multiple-io/executor_output.json'
                }
            }, executor_input)
        component_spec = pipeline_spec_pb2.ComponentSpec()
        json_format.ParseDict(
            {
                'inputDefinitions': {
                    'parameters': {
                        'string_in': {
                            'parameterType': 'STRING'
                        }
                    }
                },
                'outputDefinitions': {
                    'artifacts': {
                        'dataset_out': {
                            'artifactType': {
                                'schemaTitle': 'system.Dataset',
                                'schemaVersion': '0.0.1'
                            }
                        }
                    },
                    'parameters': {
                        'int_out': {
                            'parameterType': 'NUMBER_INTEGER'
                        },
                        'str_out': {
                            'parameterType': 'STRING'
                        }
                    }
                },
                'executorLabel': 'exec-multiple-io'
            }, component_spec)

        outputs = executor_output_utils.get_outputs_from_executor_output(
            executor_output=executor_output,
            executor_input=executor_input,
            component_spec=component_spec,
        )
        self.assertIsInstance(outputs, dict)
        self.assertIsInstance(outputs['dataset_out'], dsl.Dataset)
        self.assertEqual(outputs['dataset_out'].name, 'dataset_out')
        self.assertEqual(
            outputs['dataset_out'].uri,
            'foo/multiple-io-2023-11-09-11-31-31-064429/multiple-io/dataset_out'
        )
        self.assertEqual(outputs['dataset_out'].metadata, {'foo': 'bar'})
        self.assertEqual(outputs['int_out'], 1)
        self.assertEqual(outputs['str_out'], 'foo')


class TestPb2ValueToPython(unittest.TestCase):

    def test_null(self):
        inp = struct_pb2.Value(null_value=struct_pb2.NullValue.NULL_VALUE)
        actual = executor_output_utils.pb2_value_to_python(inp)
        expected = None
        self.assertEqual(actual, expected)

    def test_string(self):
        inp = struct_pb2.Value(string_value='foo_value')
        actual = executor_output_utils.pb2_value_to_python(inp)
        expected = 'foo_value'
        self.assertEqual(actual, expected)

    def test_number_int(self):
        inp = struct_pb2.Value(number_value=1)
        actual = executor_output_utils.pb2_value_to_python(inp)
        expected = 1.0
        self.assertEqual(actual, expected)

    def test_number_float(self):
        inp = struct_pb2.Value(number_value=1.0)
        actual = executor_output_utils.pb2_value_to_python(inp)
        expected = 1.0
        self.assertEqual(actual, expected)

    def test_bool(self):
        inp = struct_pb2.Value(bool_value=True)
        actual = executor_output_utils.pb2_value_to_python(inp)
        expected = True
        self.assertIs(actual, expected)

    def test_dict(self):
        struct_value = struct_pb2.Struct()
        struct_value.fields['my_key'].string_value = 'my_value'
        struct_value.fields['other_key'].bool_value = True
        inp = struct_pb2.Value(struct_value=struct_value)
        actual = executor_output_utils.pb2_value_to_python(inp)
        expected = {'my_key': 'my_value', 'other_key': True}
        self.assertEqual(actual, expected)


class TestRuntimeArtifactToDslArtifact(unittest.TestCase):

    def test_artifact(self):
        metadata = struct_pb2.Struct()
        metadata.fields['foo'].string_value = 'bar'
        type_ = pipeline_spec_pb2.ArtifactTypeSchema(
            schema_title='system.Artifact',
            schema_version='0.0.1',
        )
        runtime_artifact = pipeline_spec_pb2.RuntimeArtifact(
            name='a',
            uri='gs://bucket/foo',
            metadata=metadata,
            type=type_,
        )
        actual = executor_output_utils.runtime_artifact_to_dsl_artifact(
            runtime_artifact)
        expected = dsl.Artifact(
            name='a',
            uri='gs://bucket/foo',
            metadata={'foo': 'bar'},
        )
        assert_artifacts_equal(self, actual, expected)

    def test_dataset(self):
        metadata = struct_pb2.Struct()
        metadata.fields['baz'].string_value = 'bat'
        type_ = pipeline_spec_pb2.ArtifactTypeSchema(
            schema_title='system.Dataset',
            schema_version='0.0.1',
        )
        runtime_artifact = pipeline_spec_pb2.RuntimeArtifact(
            name='d',
            uri='gs://bucket/foo',
            metadata=metadata,
            type=type_,
        )
        actual = executor_output_utils.runtime_artifact_to_dsl_artifact(
            runtime_artifact)
        expected = dsl.Dataset(
            name='d',
            uri='gs://bucket/foo',
            metadata={'baz': 'bat'},
        )
        assert_artifacts_equal(self, actual, expected)


class TestArtifactListToDslArtifact(unittest.TestCase):

    def test_not_list(self):
        metadata = struct_pb2.Struct()
        metadata.fields['foo'].string_value = 'bar'
        type_ = pipeline_spec_pb2.ArtifactTypeSchema(
            schema_title='system.Artifact',
            schema_version='0.0.1',
        )
        runtime_artifact = pipeline_spec_pb2.RuntimeArtifact(
            name='a',
            uri='gs://bucket/foo',
            metadata=metadata,
            type=type_,
        )
        artifact_list = pipeline_spec_pb2.ArtifactList(
            artifacts=[runtime_artifact])

        actual = executor_output_utils.artifact_list_to_dsl_artifact(
            artifact_list,
            is_artifact_list=False,
        )
        expected = dsl.Artifact(
            name='a',
            uri='gs://bucket/foo',
            metadata={'foo': 'bar'},
        )
        assert_artifacts_equal(self, actual, expected)

    def test_single_entry_list(self):
        metadata = struct_pb2.Struct()
        metadata.fields['foo'].string_value = 'bar'
        type_ = pipeline_spec_pb2.ArtifactTypeSchema(
            schema_title='system.Dataset',
            schema_version='0.0.1',
        )
        runtime_artifact = pipeline_spec_pb2.RuntimeArtifact(
            name='a',
            uri='gs://bucket/foo',
            metadata=metadata,
            type=type_,
        )
        artifact_list = pipeline_spec_pb2.ArtifactList(
            artifacts=[runtime_artifact])

        actual = executor_output_utils.artifact_list_to_dsl_artifact(
            artifact_list,
            is_artifact_list=True,
        )
        expected = [
            dsl.Dataset(
                name='a',
                uri='gs://bucket/foo',
                metadata={'foo': 'bar'},
            )
        ]
        assert_artifact_lists_equal(self, actual, expected)

    def test_multi_entry_list(self):
        metadata = struct_pb2.Struct()
        metadata.fields['foo'].string_value = 'bar'
        type_ = pipeline_spec_pb2.ArtifactTypeSchema(
            schema_title='system.Dataset',
            schema_version='0.0.1',
        )
        runtime_artifact1 = pipeline_spec_pb2.RuntimeArtifact(
            name='a',
            uri='gs://bucket/foo/a',
            metadata=metadata,
            type=type_,
        )
        runtime_artifact2 = pipeline_spec_pb2.RuntimeArtifact(
            name='b',
            uri='gs://bucket/foo/b',
            type=type_,
        )
        artifact_list = pipeline_spec_pb2.ArtifactList(
            artifacts=[runtime_artifact1, runtime_artifact2])

        actual = executor_output_utils.artifact_list_to_dsl_artifact(
            artifact_list,
            is_artifact_list=True,
        )
        expected = [
            dsl.Dataset(
                name='a',
                uri='gs://bucket/foo/a',
                metadata={'foo': 'bar'},
            ),
            dsl.Dataset(
                name='b',
                uri='gs://bucket/foo/b',
            )
        ]

        assert_artifact_lists_equal(self, actual, expected)


class AddTypeToExecutorOutput(unittest.TestCase):

    def test(self):
        executor_input = pipeline_spec_pb2.ExecutorInput()
        json_format.ParseDict(
            {
                'inputs': {},
                'outputs': {
                    'artifacts': {
                        'dataset_out': {
                            'artifacts': [{
                                'name':
                                    'dataset_out',
                                'type': {
                                    'schemaTitle': 'system.Dataset',
                                    'schemaVersion': '0.0.1'
                                },
                                'uri':
                                    'foo/multiple-io-2023-11-09-12-04-18-616263/multiple-io/dataset_out',
                                'metadata': {}
                            }]
                        },
                        'model_out': {
                            'artifacts': [{
                                'name':
                                    'model_out',
                                'type': {
                                    'schemaTitle': 'system.Model',
                                    'schemaVersion': '0.0.1'
                                },
                                'uri':
                                    'foo/multiple-io-2023-11-09-12-04-18-616263/multiple-io/model_out',
                                'metadata': {}
                            }]
                        }
                    },
                    'outputFile':
                        'foo/multiple-io-2023-11-09-12-04-18-616263/multiple-io/executor_output.json'
                }
            }, executor_input)
        executor_output = pipeline_spec_pb2.ExecutorOutput()
        json_format.ParseDict(
            {
                'artifacts': {
                    'dataset_out': {
                        'artifacts': [{
                            'name':
                                'dataset_out',
                            'uri':
                                'foo/multiple-io-2023-11-09-12-04-18-616263/multiple-io/dataset_out',
                            'metadata': {
                                'foo': 'bar'
                            }
                        }]
                    },
                    'model_out': {
                        'artifacts': [{
                            'name':
                                'model_out',
                            'uri':
                                'foo/multiple-io-2023-11-09-12-04-18-616263/multiple-io/model_out',
                            'metadata': {
                                'baz': 'bat'
                            }
                        }]
                    }
                }
            }, executor_output)

        expected = pipeline_spec_pb2.ExecutorOutput()
        json_format.ParseDict(
            {
                'artifacts': {
                    'dataset_out': {
                        'artifacts': [{
                            'name':
                                'dataset_out',
                            'uri':
                                'foo/multiple-io-2023-11-09-12-04-18-616263/multiple-io/dataset_out',
                            'metadata': {
                                'foo': 'bar'
                            },
                            'type': {
                                'schemaTitle': 'system.Dataset',
                                'schemaVersion': '0.0.1'
                            },
                        }]
                    },
                    'model_out': {
                        'artifacts': [{
                            'name':
                                'model_out',
                            'uri':
                                'foo/multiple-io-2023-11-09-12-04-18-616263/multiple-io/model_out',
                            'metadata': {
                                'baz': 'bat'
                            },
                            'type': {
                                'schemaTitle': 'system.Model',
                                'schemaVersion': '0.0.1'
                            },
                        }]
                    }
                }
            }, expected)

        actual = executor_output_utils.add_type_to_executor_output(
            executor_input=executor_input,
            executor_output=executor_output,
        )
        self.assertEqual(actual, expected)


class TestSpecialDslOutputPathRead(parameterized.TestCase):

    @parameterized.parameters([
        ('foo', 'foo',
         pipeline_spec_pb2.ParameterType.ParameterTypeEnum.STRING),
        ('foo', 'foo',
         pipeline_spec_pb2.ParameterType.ParameterTypeEnum.STRING),
        ('true', True,
         pipeline_spec_pb2.ParameterType.ParameterTypeEnum.BOOLEAN),
        ('True', True,
         pipeline_spec_pb2.ParameterType.ParameterTypeEnum.BOOLEAN),
        ('false', False,
         pipeline_spec_pb2.ParameterType.ParameterTypeEnum.BOOLEAN),
        ('False', False,
         pipeline_spec_pb2.ParameterType.ParameterTypeEnum.BOOLEAN),
        (json.dumps({'x': 'y'}), {
            'x': 'y'
        }, pipeline_spec_pb2.ParameterType.ParameterTypeEnum.STRUCT),
        ('3.14', 3.14,
         pipeline_spec_pb2.ParameterType.ParameterTypeEnum.NUMBER_DOUBLE),
        ('100', 100,
         pipeline_spec_pb2.ParameterType.ParameterTypeEnum.NUMBER_INTEGER),
    ])
    def test(self, written, expected, dtype):
        with tempfile.TemporaryDirectory() as tempdir:
            output_file = os.path.join(tempdir, 'Output')
            with open(output_file, 'w') as f:
                f.write(written)

            actual = executor_output_utils.special_dsl_outputpath_read(
                parameter_name='name',
                output_file=output_file,
                dtype=dtype,
            )

        self.assertEqual(actual, expected)

    def test_exception(self):
        with tempfile.TemporaryDirectory() as tempdir:
            output_file = os.path.join(tempdir, 'Output')
            with open(output_file, 'w') as f:
                f.write(str({'x': 'y'}))
            with self.assertRaisesRegex(
                    ValueError,
                    r"Could not deserialize output 'name' from path"):
                executor_output_utils.special_dsl_outputpath_read(
                    parameter_name='name',
                    output_file=output_file,
                    dtype=pipeline_spec_pb2.ParameterType.ParameterTypeEnum
                    .STRUCT,
                )


def assert_artifacts_equal(
    test_class: unittest.TestCase,
    a1: dsl.Artifact,
    a2: dsl.Artifact,
) -> None:
    test_class.assertEqual(a1.name, a2.name)
    test_class.assertEqual(a1.uri, a2.uri)
    test_class.assertEqual(a1.metadata, a2.metadata)
    test_class.assertEqual(a1.schema_title, a2.schema_title)
    test_class.assertEqual(a1.schema_version, a2.schema_version)
    test_class.assertIsInstance(a1, type(a2))


def assert_artifact_lists_equal(
    test_class: unittest.TestCase,
    l1: List[dsl.Artifact],
    l2: List[dsl.Artifact],
) -> None:
    test_class.assertEqual(len(l1), len(l2))
    for a1, a2 in zip(l1, l2):
        assert_artifacts_equal(test_class, a1, a2)


if __name__ == '__main__':
    unittest.main()

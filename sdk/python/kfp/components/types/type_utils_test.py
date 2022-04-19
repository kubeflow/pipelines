# Copyright 2020-2022 The Kubeflow Authors
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
import unittest
from typing import Any, Dict, List, Union

import kfp
from absl.testing import parameterized
from kfp.components import v1_structures
from kfp.components.types import artifact_types
from kfp.components.types import type_utils
from kfp.components.types.type_utils import InconsistentTypeException
from kfp.pipeline_spec import pipeline_spec_pb2 as pb

_PARAMETER_TYPES = [
    'String',
    'str',
    'Integer',
    'int',
    'Float',
    'Double',
    'bool',
    'Boolean',
    'Dict',
    'List',
    'JsonObject',
    'JsonArray',
    {
        'JsonObject': {
            'data_type': 'proto:tfx.components.trainer.TrainArgs'
        }
    },
    'PipelineTaskFinalStatus',
]

_KNOWN_ARTIFACT_TYPES = [
    'Model',
    'Dataset',
    'Schema',
    'Metrics',
    'ClassificationMetrics',
    'SlicedClassificationMetrics',
    'HTML',
    'Markdown',
]

_UNKNOWN_ARTIFACT_TYPES = [None, 'Arbtrary Model', 'dummy']


class _ArbitraryClass:
    pass


class _VertexDummy(artifact_types.Artifact):
    TYPE_NAME = 'google.VertexDummy'
    VERSION = '0.0.2'

    def __init__(self):
        super().__init__(uri='uri', name='name', metadata={'dummy': '123'})


class TypeUtilsTest(parameterized.TestCase):

    @parameterized.parameters(
        [(item, True) for item in _PARAMETER_TYPES] +
        [(item, False)
         for item in _KNOWN_ARTIFACT_TYPES + _UNKNOWN_ARTIFACT_TYPES])
    def test_is_parameter_type_true(self, type_name, expected_result):
        self.assertEqual(expected_result,
                         type_utils.is_parameter_type(type_name))

    @parameterized.parameters(
        {
            'artifact_class_or_type_name':
                'Model',
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.Model', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                artifact_types.Model,
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.Model', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                'Dataset',
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.Dataset', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                artifact_types.Dataset,
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.Dataset', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                'Metrics',
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.Metrics', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                artifact_types.Metrics,
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.Metrics', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                'ClassificationMetrics',
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.ClassificationMetrics',
                    schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                artifact_types.ClassificationMetrics,
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.ClassificationMetrics',
                    schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                'SlicedClassificationMetrics',
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.SlicedClassificationMetrics',
                    schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                artifact_types.SlicedClassificationMetrics,
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.SlicedClassificationMetrics',
                    schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                'arbitrary name',
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.Artifact', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                _ArbitraryClass,
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.Artifact', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                artifact_types.HTML,
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.HTML', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                artifact_types.Markdown,
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.Markdown', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                'some-google-type',
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='system.Artifact', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                'google.VertexModel',
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='google.VertexModel', schema_version='0.0.1')
        },
        {
            'artifact_class_or_type_name':
                _VertexDummy,
            'expected_result':
                pb.ArtifactTypeSchema(
                    schema_title='google.VertexDummy', schema_version='0.0.2')
        },
    )
    def test_get_artifact_type_schema(self, artifact_class_or_type_name,
                                      expected_result):
        self.assertEqual(
            expected_result,
            type_utils.get_artifact_type_schema(artifact_class_or_type_name))

    @parameterized.parameters(
        {
            'given_type': 'Int',
            'expected_type': pb.ParameterType.NUMBER_INTEGER,
        },
        {
            'given_type': 'Integer',
            'expected_type': pb.ParameterType.NUMBER_INTEGER,
        },
        {
            'given_type': int,
            'expected_type': pb.ParameterType.NUMBER_INTEGER,
        },
        {
            'given_type': 'Double',
            'expected_type': pb.ParameterType.NUMBER_DOUBLE,
        },
        {
            'given_type': 'Float',
            'expected_type': pb.ParameterType.NUMBER_DOUBLE,
        },
        {
            'given_type': float,
            'expected_type': pb.ParameterType.NUMBER_DOUBLE,
        },
        {
            'given_type': 'String',
            'expected_type': pb.ParameterType.STRING,
        },
        {
            'given_type': 'Text',
            'expected_type': pb.ParameterType.STRING,
        },
        {
            'given_type': str,
            'expected_type': pb.ParameterType.STRING,
        },
        {
            'given_type': 'Boolean',
            'expected_type': pb.ParameterType.BOOLEAN,
        },
        {
            'given_type': bool,
            'expected_type': pb.ParameterType.BOOLEAN,
        },
        {
            'given_type': 'Dict',
            'expected_type': pb.ParameterType.STRUCT,
        },
        {
            'given_type': dict,
            'expected_type': pb.ParameterType.STRUCT,
        },
        {
            'given_type': 'List',
            'expected_type': pb.ParameterType.LIST,
        },
        {
            'given_type': list,
            'expected_type': pb.ParameterType.LIST,
        },
        {
            'given_type': Dict[str, int],
            'expected_type': pb.ParameterType.STRUCT,
        },
        {
            'given_type': List[Any],
            'expected_type': pb.ParameterType.LIST,
        },
        {
            'given_type': {
                'JsonObject': {
                    'data_type': 'proto:tfx.components.trainer.TrainArgs'
                }
            },
            'expected_type': pb.ParameterType.STRUCT,
        },
    )
    def test_get_parameter_type(self, given_type, expected_type):
        self.assertEqual(expected_type,
                         type_utils.get_parameter_type(given_type))

        # Test get parameter by Python type.
        self.assertEqual(pb.ParameterType.NUMBER_INTEGER,
                         type_utils.get_parameter_type(int))

    def test_get_parameter_type_invalid(self):
        with self.assertRaises(AttributeError):
            type_utils.get_parameter_type_schema(None)

    def test_get_input_artifact_type_schema(self):
        input_specs = [
            v1_structures.InputSpec(name='input1', type='String'),
            v1_structures.InputSpec(name='input2', type='Model'),
            v1_structures.InputSpec(name='input3', type=None),
        ]
        # input not found.
        with self.assertRaises(AssertionError) as cm:
            type_utils.get_input_artifact_type_schema('input0', input_specs)
            self.assertEqual('Input not found.', str(cm))

        # input found, but it doesn't map to an artifact type.
        with self.assertRaises(AssertionError) as cm:
            type_utils.get_input_artifact_type_schema('input1', input_specs)
            self.assertEqual('Input is not an artifact type.', str(cm))

        # input found, and a matching artifact type schema returned.
        self.assertEqual(
            'system.Model',
            type_utils.get_input_artifact_type_schema('input2',
                                                      input_specs).schema_title)

        # input found, and the default artifact type schema returned.
        self.assertEqual(
            'system.Artifact',
            type_utils.get_input_artifact_type_schema('input3',
                                                      input_specs).schema_title)

    @parameterized.parameters(
        {
            'given_type': 'String',
            'expected_type': 'String',
            'is_compatible': True,
        },
        {
            'given_type': 'String',
            'expected_type': 'Integer',
            'is_compatible': False,
        },
        {
            'given_type': {
                'type_a': {
                    'property': 'property_b',
                }
            },
            'expected_type': {
                'type_a': {
                    'property': 'property_b',
                }
            },
            'is_compatible': True,
        },
        {
            'given_type': {
                'type_a': {
                    'property': 'property_b',
                }
            },
            'expected_type': {
                'type_a': {
                    'property': 'property_c',
                }
            },
            'is_compatible': False,
        },
        {
            'given_type': 'Artifact',
            'expected_type': 'Model',
            'is_compatible': True,
        },
        {
            'given_type': 'Metrics',
            'expected_type': 'Artifact',
            'is_compatible': True,
        },
    )
    def test_verify_type_compatibility(
        self,
        given_type: Union[str, dict],
        expected_type: Union[str, dict],
        is_compatible: bool,
    ):
        if is_compatible:
            self.assertTrue(
                type_utils.verify_type_compatibility(
                    given_type=given_type,
                    expected_type=expected_type,
                    error_message_prefix='',
                ))
        else:
            with self.assertRaises(InconsistentTypeException):
                type_utils.verify_type_compatibility(
                    given_type=given_type,
                    expected_type=expected_type,
                    error_message_prefix='',
                )

    @parameterized.parameters(
        {
            'given_type': str,
            'expected_type_name': 'String',
        },
        {
            'given_type': int,
            'expected_type_name': 'Integer',
        },
        {
            'given_type': float,
            'expected_type_name': 'Float',
        },
        {
            'given_type': bool,
            'expected_type_name': 'Boolean',
        },
        {
            'given_type': list,
            'expected_type_name': 'List',
        },
        {
            'given_type': dict,
            'expected_type_name': 'Dict',
        },
        {
            'given_type': Any,
            'expected_type_name': None,
        },
    )
    def test_get_canonical_type_name_for_type(
        self,
        given_type,
        expected_type_name,
    ):
        self.assertEqual(
            expected_type_name,
            type_utils.get_canonical_type_name_for_type(given_type))

    @parameterized.parameters(
        {
            'given_type': 'PipelineTaskFinalStatus',
            'expected_result': True,
        },
        {
            'given_type': 'pipelineTaskFinalstatus',
            'expected_result': False,
        },
        {
            'given_type': int,
            'expected_result': False,
        },
    )
    def test_is_task_final_statu_type(self, given_type, expected_result):
        self.assertEqual(expected_result,
                         type_utils.is_task_final_status_type(given_type))


class TestTypeCheckManager(unittest.TestCase):

    def test_false_to_true(self):
        kfp.TYPE_CHECK = False
        with type_utils.TypeCheckManager(enable=True):
            self.assertEqual(kfp.TYPE_CHECK, True)
        self.assertEqual(kfp.TYPE_CHECK, False)

    def test_false_to_true(self):
        kfp.TYPE_CHECK = True
        with type_utils.TypeCheckManager(enable=False):
            self.assertEqual(kfp.TYPE_CHECK, False)
        self.assertEqual(kfp.TYPE_CHECK, True)


if __name__ == '__main__':
    unittest.main()

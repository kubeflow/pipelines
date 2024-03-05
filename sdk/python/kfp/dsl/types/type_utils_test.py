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
import os
import tempfile
from typing import Any, Dict, List, Union
import unittest

from absl.testing import parameterized
import kfp
from kfp import compiler
from kfp import components
from kfp import dsl
from kfp.dsl import base_component
from kfp.dsl import Dataset
from kfp.dsl import for_loop
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import pipeline_channel
from kfp.dsl import structures
from kfp.dsl import yaml_component
from kfp.dsl.types import artifact_types
from kfp.dsl.types import type_utils
from kfp.dsl.types.type_utils import InconsistentTypeException
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
    schema_title = 'google.VertexDummy'
    schema_version = '0.0.2'

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


class TestGetArtifactTypeSchema(parameterized.TestCase):

    @parameterized.parameters([
        # v2 standard system types
        {
            'schema_title': 'system.Artifact@0.0.1',
            'exp_schema_title': 'system.Artifact',
            'exp_schema_version': '0.0.1',
        },
        {
            'schema_title': 'system.Dataset@0.0.1',
            'exp_schema_title': 'system.Dataset',
            'exp_schema_version': '0.0.1',
        },
        # google type with schema_version
        {
            'schema_title': 'google.VertexDataset@0.0.2',
            'exp_schema_title': 'google.VertexDataset',
            'exp_schema_version': '0.0.2',
        },
    ])
    def test_valid(
        self,
        schema_title: str,
        exp_schema_title: str,
        exp_schema_version: str,
    ):
        artifact_type_schema = type_utils.bundled_artifact_to_artifact_proto(
            schema_title)
        self.assertEqual(artifact_type_schema.schema_title, exp_schema_title)
        self.assertEqual(artifact_type_schema.schema_version,
                         exp_schema_version)


class TestTypeCheckManager(unittest.TestCase):

    def check_false_to_true(self):
        kfp.TYPE_CHECK = False
        with type_utils.TypeCheckManager(enable=True):
            self.assertEqual(kfp.TYPE_CHECK, True)
        self.assertEqual(kfp.TYPE_CHECK, False)

    def test_true_to_false(self):
        kfp.TYPE_CHECK = True
        with type_utils.TypeCheckManager(enable=False):
            self.assertEqual(kfp.TYPE_CHECK, False)
        self.assertEqual(kfp.TYPE_CHECK, True)


class TestCreateBundledArtifacttType(parameterized.TestCase):

    @parameterized.parameters([
        {
            'schema_title': 'system.Artifact',
            'schema_version': '0.0.2',
            'expected': 'system.Artifact@0.0.2'
        },
        {
            'schema_title': 'google.Artifact',
            'schema_version': '0.0.3',
            'expected': 'google.Artifact@0.0.3'
        },
        {
            'schema_title': 'system.Artifact',
            'schema_version': None,
            'expected': 'system.Artifact@0.0.1'
        },
        {
            'schema_title': 'google.Artifact',
            'schema_version': None,
            'expected': 'google.Artifact@0.0.1'
        },
    ])
    def test(self, schema_title: str, schema_version: Union[str, None],
             expected: str):
        actual = type_utils.create_bundled_artifact_type(
            schema_title, schema_version)
        self.assertEqual(actual, expected)


class TestValidateBundledArtifactType(parameterized.TestCase):

    @parameterized.parameters([
        {
            'type_': 'system.Artifact@0.0.1'
        },
        {
            'type_': 'system.Dataset@2.0.1'
        },
        {
            'type_': 'google.Model@2.0.0'
        },
    ])
    def test_valid(self, type_: str):
        type_utils.validate_bundled_artifact_type(type_)

    @parameterized.parameters([
        {
            'type_': 'system.Artifact'
        },
        {
            'type_': '2.0.1'
        },
        {
            'type_': 'google.Model2.0.0'
        },
        {
            'type_': 'google.Model2.0.0'
        },
        {
            'type_': 'google.Model@'
        },
        {
            'type_': 'google.Model@'
        },
        {
            'type_': '@2.0.0'
        },
    ])
    def test_missing_part(self, type_: str):
        with self.assertRaisesRegex(
                TypeError,
                r'Artifacts must have both a schema_title and a schema_version, separated by `@`'
        ):
            type_utils.validate_bundled_artifact_type(type_)

    @parameterized.parameters([
        {
            'type_': 'system@0.0.1'
        },
        {
            'type_': 'google@0.0.1'
        },
        {
            'type_': 'other@0.0.1'
        },
        {
            'type_': 'Artifact@0.0.1'
        },
    ])
    def test_one_part_schema_title(self, type_: str):
        with self.assertRaisesRegex(
                TypeError,
                r'Artifact schema_title must have both a namespace and a name'):
            type_utils.validate_bundled_artifact_type(type_)

    @parameterized.parameters([
        {
            'type_': 'other.Artifact@0.0.1'
        },
    ])
    def test_must_be_system_or_google_namespace(self, type_: str):
        with self.assertRaisesRegex(
                TypeError,
                r'Artifact schema_title must belong to `system` or `google` namespace'
        ):
            type_utils.validate_bundled_artifact_type(type_)

    @parameterized.parameters([
        {
            'type_': 'system.Artifact@0'
        },
        {
            'type_': 'system.Artifact@0.0'
        },
        {
            'type_': 'google.Artifact@0.01'
        },
    ])
    def test_must_be_valid_semantic_version(self, type_: str):
        with self.assertRaisesRegex(
                TypeError,
                r'Artifact schema_version must use three-part semantic versioning'
        ):
            type_utils.validate_bundled_artifact_type(type_)


class TestDeserializeV1ComponentYamlDefault(parameterized.TestCase):

    @parameterized.parameters([
        {
            'type_': 'String',
            'default': 'val',
            'expected_type': str,
            'expected_val': 'val',
        },
        {
            'type_': 'Boolean',
            'default': 'True',
            'expected_type': bool,
            'expected_val': True,
        },
        {
            'type_': 'Boolean',
            'default': 'true',
            'expected_type': bool,
            'expected_val': True,
        },
        {
            'type_': 'Boolean',
            'default': 'False',
            'expected_type': bool,
            'expected_val': False,
        },
        {
            'type_': 'Boolean',
            'default': 'false',
            'expected_type': bool,
            'expected_val': False,
        },
        {
            'type_': 'Float',
            'default': '0.0',
            'expected_type': float,
            'expected_val': 0.0,
        },
        {
            'type_': 'Float',
            'default': '1.0',
            'expected_type': float,
            'expected_val': 1.0,
        },
        {
            'type_': 'Integer',
            'default': '0',
            'expected_type': int,
            'expected_val': 0,
        },
        {
            'type_': 'JsonObject',
            'default': '[]',
            'expected_type': list,
            'expected_val': [],
        },
        {
            'type_': 'JsonObject',
            'default': '[1, 1.0, "a", true]',
            'expected_type': list,
            'expected_val': [1, 1.0, 'a', True],
        },
        {
            'type_': 'JsonObject',
            'default': '{}',
            'expected_type': dict,
            'expected_val': {},
        },
        {
            'type_': 'JsonObject',
            'default': '{"a": 1.0, "b": true}',
            'expected_type': dict,
            'expected_val': {
                'a': 1.0,
                'b': True
            },
        },
    ])
    def test_for_defaults_as_strings(
        self,
        type_: Any,
        default: str,
        expected_type: type,
        expected_val: Any,
    ):
        res = type_utils.deserialize_v1_component_yaml_default(type_, default)
        # check type first since equals check is insufficient since 1.0 == 1
        self.assertIsInstance(res, expected_type)
        self.assertEqual(res, expected_val)

    @parameterized.parameters([
        {
            'type_': 'Boolean',
            'default': True,
            'expected_type': bool,
            'expected_val': True,
        },
        {
            'type_': 'Boolean',
            'default': False,
            'expected_type': bool,
            'expected_val': False,
        },
        {
            'type_': 'Float',
            'default': 0.0,
            'expected_type': float,
            'expected_val': 0.0,
        },
        {
            'type_': 'Float',
            'default': 1.0,
            'expected_type': float,
            'expected_val': 1.0,
        },
        {
            'type_': 'Integer',
            'default': 0,
            'expected_type': int,
            'expected_val': 0,
        },
        {
            'type_': 'JsonObject',
            'default': [],
            'expected_type': list,
            'expected_val': [],
        },
        {
            'type_': 'JsonObject',
            'default': [1, 1.0, 'a', True],
            'expected_type': list,
            'expected_val': [1, 1.0, 'a', True],
        },
        {
            'type_': 'JsonObject',
            'default': {},
            'expected_type': dict,
            'expected_val': {},
        },
        {
            'type_': 'JsonObject',
            'default': {
                'a': 1.0,
                'b': True
            },
            'expected_type': dict,
            'expected_val': {
                'a': 1.0,
                'b': True
            },
        },
    ])
    def test_robustness_to_literals(
        self,
        type_: Any,
        default: str,
        expected_type: type,
        expected_val: Any,
    ):
        res = type_utils.deserialize_v1_component_yaml_default(type_, default)
        # check type first since equals check is insufficient since 1.0 == 1
        self.assertIsInstance(res, expected_type)
        self.assertEqual(res, expected_val)


class TestTypeChecking(parameterized.TestCase):

    @parameterized.parameters(
        # param True
        {
            'argument_value': 'my text',
            'parameter_input_spec': structures.InputSpec(type='String'),
            'is_compatible': True,
        },
        # param False
        {
            'argument_value': 'my text',
            'parameter_input_spec': structures.InputSpec(type='Integer'),
            'is_compatible': False,
        },
        # param Artifact compat, irrespective of version
        {
            'argument_value':
                pipeline_channel.create_pipeline_channel(
                    'dummy_channel', 'system.Artifact@1.0.0'),
            'parameter_input_spec':
                structures.InputSpec('system.Model@0.0.1'),
            'is_compatible':
                True,
        },
        # param Artifact compat, irrespective of version, other way
        {
            'argument_value':
                pipeline_channel.create_pipeline_channel(
                    'dummy_channel', 'system.Metrics@1.0.0'),
            'parameter_input_spec':
                structures.InputSpec('system.Artifact@0.0.1'),
            'is_compatible':
                True,
        },
        # different schema_title incompat, irrespective of version
        {
            'argument_value':
                pipeline_channel.create_pipeline_channel(
                    'dummy_channel', 'system.Metrics@1.0.0'),
            'parameter_input_spec':
                structures.InputSpec('system.Dataset@1.0.0'),
            'is_compatible':
                False,
        },
        # different major version incompat
        {
            'argument_value':
                pipeline_channel.create_pipeline_channel(
                    'dummy_channel', 'system.Metrics@1.0.0'),
            'parameter_input_spec':
                structures.InputSpec('system.Metrics@2.1.1'),
            'is_compatible':
                False,
        },
        # namespace must match
        {
            'argument_value':
                pipeline_channel.create_pipeline_channel(
                    'dummy_channel', 'google.Model@1.0.0'),
            'parameter_input_spec':
                structures.InputSpec('system.Model@1.0.0'),
            'is_compatible':
                False,
        },
        # system.Artifact compatible works across namespace
        {
            'argument_value':
                pipeline_channel.create_pipeline_channel(
                    'dummy_channel', 'google.Model@1.0.0'),
            'parameter_input_spec':
                structures.InputSpec('system.Artifact@1.0.0'),
            'is_compatible':
                True,
        },
        {
            'argument_value':
                pipeline_channel.create_pipeline_channel(
                    'dummy_channel',
                    'system.Artifact@1.0.0',
                    is_artifact_list=True),
            'parameter_input_spec':
                structures.InputSpec(
                    'system.Artifact@1.0.0', is_artifact_list=True),
            'is_compatible':
                True,
        },
        {
            'argument_value':
                pipeline_channel.create_pipeline_channel(
                    'dummy_channel',
                    'system.Artifact@1.0.0',
                    is_artifact_list=True),
            'parameter_input_spec':
                structures.InputSpec(
                    'system.Artifact@1.0.0', is_artifact_list=False),
            'is_compatible':
                False,
        },
        {
            'argument_value':
                pipeline_channel.create_pipeline_channel(
                    'dummy_channel',
                    'system.Artifact@1.0.0',
                    is_artifact_list=False),
            'parameter_input_spec':
                structures.InputSpec(
                    'system.Artifact@1.0.0', is_artifact_list=True),
            'is_compatible':
                False,
        },
        {
            'argument_value':
                pipeline_channel.create_pipeline_channel(
                    'dummy_channel',
                    'system.Artifact@1.0.0',
                    is_artifact_list=True),
            'parameter_input_spec':
                structures.InputSpec('String'),
            'is_compatible':
                False,
        },
        {
            'argument_value':
                pipeline_channel.create_pipeline_channel(
                    'dummy_channel', 'String'),
            'parameter_input_spec':
                structures.InputSpec(
                    'system.Artifact@1.0.0', is_artifact_list=True),
            'is_compatible':
                False,
        },
        {
            'argument_value':
                for_loop.LoopArgumentVariable(
                    loop_argument=for_loop.LoopParameterArgument
                    .from_pipeline_channel(
                        pipeline_channel.create_pipeline_channel(
                            'Output-loop-item', 'List[str]',
                            'list-dict-without-type-maker-5')),
                    subvar_name='a'),
            'parameter_input_spec':
                structures.InputSpec('Integer'),
            'is_compatible':
                True,
        },
        {
            'argument_value':
                for_loop.LoopParameterArgument.from_pipeline_channel(
                    pipeline_channel.create_pipeline_channel(
                        'Output-loop-item', 'List[int]',
                        'list-dict-without-type-maker-5')),
            'parameter_input_spec':
                structures.InputSpec('Integer'),
            'is_compatible':
                True,
        },
    )
    def test_verify_type_compatibility(
        self,
        argument_value: Union[pipeline_channel.PipelineChannel, str, bool, int,
                              float, dict, list],
        parameter_input_spec: structures.InputSpec,
        is_compatible: bool,
    ):
        if is_compatible:
            self.assertTrue(
                type_utils.verify_type_compatibility(
                    given_value=argument_value,
                    expected_spec=parameter_input_spec,
                    error_message_prefix='',
                ))
        else:
            with self.assertRaises(InconsistentTypeException):
                type_utils.verify_type_compatibility(
                    given_value=argument_value,
                    expected_spec=parameter_input_spec,
                    error_message_prefix='',
                )

    def test_list_of_artifacts_across_compilation_valid(self):

        @dsl.component
        def double(num: int, out_dataset: Output[Dataset]):
            ...

        @dsl.component
        def add(in_datasets: Input[List[Dataset]],
                out_dataset: Output[Dataset]):
            ...

        loaded_add = compile_and_load_component(add)

        @dsl.pipeline
        def math_pipeline(in_datasets: Input[List[Dataset]]) -> List[Dataset]:
            # artifact list arg can be provided for artifact list type (of component)
            loaded_add(in_datasets=in_datasets)

            with dsl.ParallelFor([1, 2, 3]) as x:
                t = double(num=x)
            return dsl.Collected(t.outputs['out_dataset'])

        loaded_math_pipeline = compile_and_load_component(math_pipeline)

        @dsl.pipeline
        def outer_math_pipeline(
                in_datasets: Input[List[Dataset]]) -> List[Dataset]:
            # artifact list arg can be provided for artifact list type (of pipeline)
            t = loaded_math_pipeline(in_datasets=in_datasets)
            # artifact list output can be surfaced to artifact list output
            return t.output

    def test_list_of_artifacts_across_compilation_invalid_component_input(self):

        @dsl.component
        def add(in_datasets: Input[Dataset]):
            ...

        loaded_add = compile_and_load_component(add)

        with self.assertRaisesRegex(
                InconsistentTypeException,
                r"Incompatible argument passed to the input 'in_datasets' of component 'add': Argument type 'List\[system\.Dataset@0\.0\.1\]' is incompatible with the input type 'system\.Dataset@0\.0\.1'"
        ):

            @dsl.pipeline
            def math_pipeline(in_datasets: Input[List[Dataset]]):
                # artifact list arg cannot be provided to single artifact type (of component)
                loaded_add(in_datasets=in_datasets)

    def test_list_of_artifacts_across_compilation_invalid_pipeline_input(self):

        @dsl.component
        def double(num: int, out_dataset: Output[Dataset]):
            ...

        @dsl.component
        def add(in_datasets: Input[List[Dataset]],
                out_dataset: Output[Dataset]):
            ...

        @dsl.pipeline
        def math_pipeline(in_datasets: Input[List[Dataset]]) -> List[Dataset]:
            add(in_datasets=in_datasets)

            with dsl.ParallelFor([1, 2, 3]) as x:
                t = double(num=x)
            return dsl.Collected(t.outputs['out_dataset'])

        with self.assertRaisesRegex(
                InconsistentTypeException,
                r"Incompatible argument passed to the input 'in_datasets' of component 'math-pipeline': Argument type 'system\.Dataset@0\.0\.1' is incompatible with the input type 'List\[system.Dataset@0\.0\.1\]'"
        ):

            @dsl.pipeline
            def outer_math_pipeline(
                    in_dataset: Input[Dataset]) -> List[Dataset]:
                # single artifact arg cannot be provided to artifact list type (of pipeline)
                t = math_pipeline(in_datasets=in_dataset)
                return t.output

    def test_list_of_artifacts_across_compilation_invalid_pipeline_output(self):

        @dsl.component
        def double(num: int, out_dataset: Output[Dataset]):
            ...

        @dsl.component
        def add(in_datasets: Input[List[Dataset]],
                out_dataset: Output[Dataset]):
            ...

        @dsl.pipeline
        def math_pipeline(in_datasets: Input[List[Dataset]]) -> List[Dataset]:
            add(in_datasets=in_datasets)

            with dsl.ParallelFor([1, 2, 3]) as x:
                t = double(num=x)
            return dsl.Collected(t.outputs['out_dataset'])

        with self.assertRaisesRegex(
                InconsistentTypeException,
                r"Incompatible return type provided for output of pipeline 'outer-math-pipeline'\. Output of type 'List\[system\.Dataset@0\.0\.1\]' cannot be surfaced as pipeline output type 'system\.Dataset@0\.0\.1"
        ):

            @dsl.pipeline
            def outer_math_pipeline(
                    in_dataset: Input[List[Dataset]]) -> Dataset:
                t = math_pipeline(in_datasets=in_dataset)
                # artifact list output cannot be surfaced as single artifact output
                return t.output


def compile_and_load_component(
        comp: base_component.BaseComponent) -> yaml_component.YamlComponent:

    with tempfile.TemporaryDirectory() as tempdir:
        output_path = os.path.join(tempdir, 'pipeline.yaml')
        compiler.Compiler().compile(comp, output_path)
        return components.load_component_from_file(output_path)


if __name__ == '__main__':
    unittest.main()

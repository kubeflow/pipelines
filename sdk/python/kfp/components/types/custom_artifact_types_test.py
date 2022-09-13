# Copyright 2022 The Kubeflow Authors
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

import inspect
import os
import sys
import tempfile
import textwrap
import typing
from typing import Any
import unittest

from absl.testing import parameterized
import kfp
from kfp import compiler
from kfp import dsl
from kfp.components.types import artifact_types
from kfp.components.types import custom_artifact_types
from kfp.components.types import type_annotations
from kfp.components.types.artifact_types import Artifact
from kfp.components.types.artifact_types import Dataset
from kfp.components.types.type_annotations import Input
from kfp.components.types.type_annotations import InputPath
from kfp.components.types.type_annotations import Output
from kfp.components.types.type_annotations import OutputPath
import typing_extensions
import yaml

Alias = Artifact
artifact_types_alias = artifact_types


class _TestCaseWithThirdPartyPackage(parameterized.TestCase):

    @classmethod
    def setUpClass(cls):

        class VertexDataset:
            schema_title = 'google.VertexDataset'
            schema_version = '0.0.0'

        class_source = textwrap.dedent(inspect.getsource(VertexDataset))

        tmp_dir = tempfile.TemporaryDirectory()
        with open(os.path.join(tmp_dir.name, 'aiplatform.py'), 'w') as f:
            f.write(class_source)
        sys.path.append(tmp_dir.name)
        cls.tmp_dir = tmp_dir

    @classmethod
    def teardownClass(cls):
        sys.path.pop()
        cls.tmp_dir.cleanup()


class TestFuncToRootAnnotationSymbol(_TestCaseWithThirdPartyPackage):

    def test_no_annotations(self):

        def func(a, b):
            pass

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {}
        self.assertEqual(actual, expected)

    def test_input_output_path(self):

        def func(
                a: int,
                b: InputPath('Dataset'),
                c: OutputPath('Dataset'),
        ) -> str:
            return 'dataset'

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {}
        self.assertEqual(actual, expected)

    def test_no_return(self):

        def func(a: int, b: Input[Artifact]):
            pass

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {'b': 'Artifact'}
        self.assertEqual(actual, expected)

    def test_with_return(self):

        def func(a: int, b: Input[Artifact]) -> int:
            return 1

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {'b': 'Artifact'}
        self.assertEqual(actual, expected)

    def test_module_base_symbol(self):

        def func(a: int, b: Input[artifact_types.Artifact]):
            pass

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {'b': 'artifact_types'}
        self.assertEqual(actual, expected)

    def test_alias(self):

        def func(a: int, b: Input[Alias]):
            pass

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {'b': 'Alias'}
        self.assertEqual(actual, expected)

    def test_named_tuple(self):

        def func(
            a: int,
            b: Input[Artifact],
        ) -> typing.NamedTuple('MyNamedTuple', [('a', int), (
                'b', Artifact), ('c', artifact_types.Artifact)]):
            InnerNamedTuple = typing.NamedTuple(
                'MyNamedTuple', [('a', int), ('b', Artifact),
                                 ('c', artifact_types.Artifact)])
            return InnerNamedTuple(a=a, b=b, c=b)  # type: ignore

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {'b': 'Artifact'}
        self.assertEqual(actual, expected)

    def test_google_type_class_symbol(self):
        from aiplatform import VertexDataset

        def func(a: int, b: Input[VertexDataset]) -> int:
            return 1

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {'b': 'VertexDataset'}
        self.assertEqual(actual, expected)

    def test_google_type_module_base_symbol(self):
        import aiplatform

        def func(a: int, b: Input[aiplatform.VertexDataset]):
            pass

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {'b': 'aiplatform'}
        self.assertEqual(actual, expected)

    def test_google_type_alias(self):
        from aiplatform import VertexDataset

        Alias = VertexDataset

        def func(a: int, b: Input[Alias]) -> int:
            return 1

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {'b': 'Alias'}
        self.assertEqual(actual, expected)

    def test_using_dsl_symbol_for_generic(self):

        def func(a: int, b: dsl.Input[Artifact],
                 c: dsl.Output[dsl.Dataset]) -> int:
            return 1

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {'b': 'Artifact', 'c': 'dsl'}
        self.assertEqual(actual, expected)

    def test_using_kfp_symbol_for_generic(self):

        def func(a: int, b: kfp.dsl.Input[Artifact],
                 c: kfp.dsl.Output[dsl.Dataset]) -> int:
            return 1

        actual = custom_artifact_types.func_to_artifact_class_base_symbol(func)
        expected = {'b': 'Artifact', 'c': 'dsl'}
        self.assertEqual(actual, expected)


class TestGetParamToAnnObj(unittest.TestCase):

    def test_no_named_tuple(self):

        def func(
            a: int,
            b: Input[Artifact],
        ) -> int:
            return 1

        actual = custom_artifact_types.get_param_to_annotation_object(func)
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[Artifact,
                                            type_annotations.InputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_named_tuple(self):

        MyNamedTuple = typing.NamedTuple('MyNamedTuple', [('a', int),
                                                          ('b', str)])

        def func(
            a: int,
            b: Input[Artifact],
        ) -> MyNamedTuple:
            InnerNamedTuple = typing.NamedTuple('MyNamedTuple', [('a', int),
                                                                 ('b', str)])
            return InnerNamedTuple(a=a, b='string')  # type: ignore

        actual = custom_artifact_types.get_param_to_annotation_object(func)
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[Artifact,
                                            type_annotations.InputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_input_output_path(self):

        def func(
                a: int,
                b: InputPath('Dataset'),
                c: OutputPath('Dataset'),
        ) -> str:
            return 'dataset'

        actual = custom_artifact_types.get_param_to_annotation_object(func)
        self.assertEqual(actual['a'], int)
        self.assertIsInstance(actual['b'], InputPath)


class TestGetFullQualnameForArtifact(_TestCaseWithThirdPartyPackage):
    # only gets called on artifacts, so don't need to test on all types
    @parameterized.parameters([
        (Alias, 'kfp.components.types.artifact_types.Artifact'),
        (Artifact, 'kfp.components.types.artifact_types.Artifact'),
        (Dataset, 'kfp.components.types.artifact_types.Dataset'),
    ])
    def test(self, obj: Any, expected_qualname: str):
        self.assertEqual(
            custom_artifact_types.get_full_qualname_for_artifact(obj),
            expected_qualname)

    def test_aiplatform_artifact(self):
        import aiplatform
        self.assertEqual(
            custom_artifact_types.get_full_qualname_for_artifact(
                aiplatform.VertexDataset), 'aiplatform.VertexDataset')


class TestGetArtifactImportItemsFromFunction(_TestCaseWithThirdPartyPackage):

    def test_no_annotations(self):

        def func(a, b):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        expected = []
        self.assertEqual(actual, expected)

    def test_no_return(self):
        from aiplatform import VertexDataset

        def func(a: int, b: Input[VertexDataset]):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        expected = ['aiplatform.VertexDataset']
        self.assertEqual(actual, expected)

    def test_with_return(self):
        from aiplatform import VertexDataset

        def func(a: int, b: Input[VertexDataset]) -> int:
            return 1

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        expected = ['aiplatform.VertexDataset']
        self.assertEqual(actual, expected)

    def test_multiline(self):
        from aiplatform import VertexDataset

        def func(
            a: int,
            b: Input[VertexDataset],
        ) -> int:
            return 1

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        expected = ['aiplatform.VertexDataset']
        self.assertEqual(actual, expected)

    def test_alias(self):
        from aiplatform import VertexDataset
        Alias = VertexDataset

        def func(a: int, b: Input[Alias]):
            pass

        with self.assertRaisesRegex(
                TypeError, r'Module or type name aliases are not supported'):
            custom_artifact_types.get_custom_artifact_import_items_from_function(
                func)

    def test_long_form_annotation(self):
        import aiplatform

        def func(a: int, b: Output[aiplatform.VertexDataset]):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        expected = ['aiplatform']
        self.assertEqual(actual, expected)

    def test_aliased_module_throws_error(self):
        import aiplatform as aiplatform_alias

        def func(a: int, b: Output[aiplatform_alias.VertexDataset]):
            pass

        with self.assertRaisesRegex(
                TypeError, r'Module or type name aliases are not supported'):
            custom_artifact_types.get_custom_artifact_import_items_from_function(
                func)

    def test_input_output_path(self):
        from aiplatform import VertexDataset

        def func(
                a: int,
                b: InputPath('Dataset'),
                c: Output[VertexDataset],
                d: OutputPath('Dataset'),
        ) -> str:
            return 'dataset'

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, ['aiplatform.VertexDataset'])


class TestFuncToAnnotationObject(_TestCaseWithThirdPartyPackage):

    def test_no_annotations(self):

        def func(a, b):
            pass

        actual = custom_artifact_types.func_to_annotation_object(func)
        expected = {'a': inspect._empty, 'b': inspect._empty}
        self.assertEqual(actual, expected)

    def test_no_return(self):
        from aiplatform import VertexDataset

        def func(a: int, b: Input[VertexDataset]):
            pass

        actual = custom_artifact_types.func_to_annotation_object(func)

        import aiplatform
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[
                    aiplatform.VertexDataset,
                    kfp.components.types.type_annotations.InputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_with_return(self):
        from aiplatform import VertexDataset

        def func(a: int, b: Input[VertexDataset]) -> int:
            return 1

        import aiplatform
        actual = custom_artifact_types.func_to_annotation_object(func)
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[
                    aiplatform.VertexDataset,
                    kfp.components.types.type_annotations.InputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_alias(self):
        from aiplatform import VertexDataset
        Alias = VertexDataset

        def func(a: int, b: Input[Alias]):
            pass

        actual = custom_artifact_types.func_to_annotation_object(func)

        import aiplatform
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[
                    aiplatform.VertexDataset,
                    kfp.components.types.type_annotations.InputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_long_form_annotation(self):
        import aiplatform

        def func(a: int, b: Output[aiplatform.VertexDataset]):
            pass

        actual = custom_artifact_types.func_to_annotation_object(func)

        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[
                    aiplatform.VertexDataset,
                    kfp.components.types.type_annotations.OutputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_aliased_module(self):
        import aiplatform as aiplatform_alias

        def func(a: int, b: Output[aiplatform_alias.VertexDataset]):
            pass

        actual = custom_artifact_types.func_to_annotation_object(func)

        import aiplatform
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[
                    aiplatform.VertexDataset,
                    kfp.components.types.type_annotations.OutputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_input_output_path(self):
        from aiplatform import VertexDataset

        def func(
                a: int,
                b: InputPath('Dataset'),
                c: Output[VertexDataset],
                d: OutputPath('Dataset'),
        ) -> str:
            return 'dataset'

        actual = custom_artifact_types.func_to_annotation_object(func)

        import aiplatform
        expected = {
            'a':
                int,
            'b':
                kfp.components.types.type_annotations.InputPath('Dataset'),
            'c':
                typing_extensions.Annotated[
                    aiplatform.VertexDataset,
                    kfp.components.types.type_annotations.OutputAnnotation],
            'd':
                kfp.components.types.type_annotations.OutputPath('Dataset'),
        }
        self.assertEqual(actual, expected)


class TestGetCustomArtifactTypeImportStatements(_TestCaseWithThirdPartyPackage):

    def test_no_annotations(self):

        def func(a, b):
            pass

        actual = custom_artifact_types.get_custom_artifact_type_import_statements(
            func)
        expected = []
        self.assertEqual(actual, expected)

    def test_no_return(self):
        from aiplatform import VertexDataset

        def func(a: int, b: Input[VertexDataset]):
            pass

        actual = custom_artifact_types.get_custom_artifact_type_import_statements(
            func)

        expected = ['from aiplatform import VertexDataset']
        self.assertEqual(actual, expected)

    def test_with_return(self):
        from aiplatform import VertexDataset

        def func(a: int, b: kfp.dsl.Input[VertexDataset]) -> int:
            return 1

        import aiplatform
        actual = custom_artifact_types.get_custom_artifact_type_import_statements(
            func)
        expected = ['from aiplatform import VertexDataset']
        self.assertEqual(actual, expected)

    def test_alias(self):
        from aiplatform import VertexDataset
        Alias = VertexDataset

        def func(a: int, b: Input[Alias]):
            pass

        with self.assertRaisesRegex(
                TypeError, r'Module or type name aliases are not supported'):
            custom_artifact_types.get_custom_artifact_type_import_statements(
                func)

    def test_long_form_annotation(self):
        import aiplatform

        def func(a: int, b: dsl.Output[aiplatform.VertexDataset]):
            pass

        actual = custom_artifact_types.get_custom_artifact_type_import_statements(
            func)

        expected = ['import aiplatform']
        self.assertEqual(actual, expected)

    def test_aliased_module(self):
        import aiplatform as aiplatform_alias

        def func(a: int, b: Output[aiplatform_alias.VertexDataset]):
            pass

        with self.assertRaisesRegex(
                TypeError, r'Module or type name aliases are not supported'):
            custom_artifact_types.get_custom_artifact_type_import_statements(
                func)

    def test_input_output_path(self):
        from aiplatform import VertexDataset

        def func(
                a: int,
                b: InputPath('Dataset'),
                c: Output[VertexDataset],
                d: OutputPath('Dataset'),
        ) -> str:
            return 'dataset'

        actual = custom_artifact_types.get_custom_artifact_type_import_statements(
            func)

        expected = ['from aiplatform import VertexDataset']
        self.assertEqual(actual, expected)


class TestImportStatementAdded(_TestCaseWithThirdPartyPackage):

    def test(self):
        import aiplatform
        from aiplatform import VertexDataset

        @dsl.component
        def one(
            a: int,
            b: Output[VertexDataset],
        ):
            pass

        @dsl.component
        def two(a: Input[aiplatform.VertexDataset],):
            pass

        @dsl.pipeline()
        def my_pipeline():
            one_task = one(a=1)
            two_task = two(a=one_task.outputs['b'])

        with tempfile.TemporaryDirectory() as tmpdir:
            output_path = os.path.join(tmpdir, 'pipeline.yaml')

            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=output_path)
            with open(output_path) as f:
                pipeline_spec = yaml.safe_load(f)

        self.assertIn(
            'from aiplatform import VertexDataset',
            ' '.join(pipeline_spec['deploymentSpec']['executors']['exec-one']
                     ['container']['command']))
        self.assertIn(
            'import aiplatform',
            ' '.join(pipeline_spec['deploymentSpec']['executors']['exec-two']
                     ['container']['command']))


if __name__ == '__main__':
    unittest.main()

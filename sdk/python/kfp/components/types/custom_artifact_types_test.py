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


class TestFuncToRootAnnotationSymbol(unittest.TestCase):

    def test_no_annotations(self):

        def func(a, b):
            pass

        actual = custom_artifact_types.func_to_root_annotation_symbols_for_artifacts(
            func)
        expected = {}
        self.assertEqual(actual, expected)

    def test_no_return(self):

        def func(a: int, b: Input[Artifact]):
            pass

        actual = custom_artifact_types.func_to_root_annotation_symbols_for_artifacts(
            func)
        expected = {'b': 'Artifact'}
        self.assertEqual(actual, expected)

    def test_with_return(self):

        def func(a: int, b: Input[Artifact]) -> int:
            return 1

        actual = custom_artifact_types.func_to_root_annotation_symbols_for_artifacts(
            func)
        expected = {'b': 'Artifact'}
        self.assertEqual(actual, expected)

    def test_multiline(self):

        def func(
            a: int,
            b: Input[Artifact],
        ) -> int:
            return 1

        actual = custom_artifact_types.func_to_root_annotation_symbols_for_artifacts(
            func)
        expected = {'b': 'Artifact'}
        self.assertEqual(actual, expected)

    def test_alias(self):

        def func(a: int, b: Input[Alias]):
            pass

        actual = custom_artifact_types.func_to_root_annotation_symbols_for_artifacts(
            func)
        expected = {'b': 'Alias'}
        self.assertEqual(actual, expected)

    def test_long_form_annotation(self):

        def func(a: int, b: Input[artifact_types.Artifact]):
            pass

        actual = custom_artifact_types.func_to_root_annotation_symbols_for_artifacts(
            func)
        expected = {'b': 'artifact_types'}
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

        actual = custom_artifact_types.func_to_root_annotation_symbols_for_artifacts(
            func)
        expected = {'b': 'Artifact'}
        self.assertEqual(actual, expected)

    def test_input_output_path(self):

        def func(
                a: int,
                b: InputPath('Dataset'),
        ) -> OutputPath('Dataset'):
            return 'dataset'

        actual = custom_artifact_types.func_to_root_annotation_symbols_for_artifacts(
            func)
        expected = {'b': 'InputPath'}
        self.assertEqual(actual, expected)


class MyCustomArtifact:
    schema_title = 'my_custom_artifact'
    schema_version = '0.0.0'


class _TestCaseWithThirdPartyPackage(parameterized.TestCase):

    @classmethod
    def setUpClass(cls):

        class ThirdPartyArtifact:
            schema_title = 'custom.my_third_party_artifact'
            schema_version = '0.0.0'

        class_source = textwrap.dedent(inspect.getsource(ThirdPartyArtifact))

        tmp_dir = tempfile.TemporaryDirectory()
        with open(os.path.join(tmp_dir.name, 'my_package.py'), 'w') as f:
            f.write(class_source)
        sys.path.append(tmp_dir.name)
        cls.tmp_dir = tmp_dir

    @classmethod
    def teardownClass(cls):
        sys.path.pop()
        cls.tmp_dir.cleanup()


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
        ) -> OutputPath('Dataset'):
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

    def test_my_package_artifact(self):
        import my_package
        self.assertEqual(
            custom_artifact_types.get_full_qualname_for_artifact(
                my_package.ThirdPartyArtifact), 'my_package.ThirdPartyArtifact')


class TestGetArtifactImportItemsFromFunction(_TestCaseWithThirdPartyPackage):

    def test_no_annotations(self):

        def func(a, b):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        expected = []
        self.assertEqual(actual, expected)

    def test_no_return(self):
        from my_package import ThirdPartyArtifact

        def func(a: int, b: Input[ThirdPartyArtifact]):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        expected = ['my_package.ThirdPartyArtifact']
        self.assertEqual(actual, expected)

    def test_with_return(self):
        from my_package import ThirdPartyArtifact

        def func(a: int, b: Input[ThirdPartyArtifact]) -> int:
            return 1

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        expected = ['my_package.ThirdPartyArtifact']
        self.assertEqual(actual, expected)

    def test_multiline(self):
        from my_package import ThirdPartyArtifact

        def func(
            a: int,
            b: Input[ThirdPartyArtifact],
        ) -> int:
            return 1

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        expected = ['my_package.ThirdPartyArtifact']
        self.assertEqual(actual, expected)

    def test_alias(self):
        from my_package import ThirdPartyArtifact
        Alias = ThirdPartyArtifact

        def func(a: int, b: Input[Alias]):
            pass

        with self.assertRaisesRegex(
                TypeError, r'Module or type name aliases are not supported'):
            custom_artifact_types.get_custom_artifact_import_items_from_function(
                func)

    def test_long_form_annotation(self):
        import my_package

        def func(a: int, b: Output[my_package.ThirdPartyArtifact]):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        expected = ['my_package']
        self.assertEqual(actual, expected)

    def test_aliased_module_throws_error(self):
        import my_package as my_package_alias

        def func(a: int, b: Output[my_package_alias.ThirdPartyArtifact]):
            pass

        with self.assertRaisesRegex(
                TypeError, r'Module or type name aliases are not supported'):
            custom_artifact_types.get_custom_artifact_import_items_from_function(
                func)

    def test_input_output_path(self):
        from my_package import ThirdPartyArtifact

        def func(
            a: int,
            b: InputPath('Dataset'),
            c: Output[ThirdPartyArtifact],
        ) -> OutputPath('Dataset'):
            return 'dataset'

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, ['my_package.ThirdPartyArtifact'])


class TestFuncToAnnotationObject(_TestCaseWithThirdPartyPackage):

    def test_no_annotations(self):

        def func(a, b):
            pass

        actual = custom_artifact_types.func_to_annotation_object(func)
        expected = {'a': inspect._empty, 'b': inspect._empty}
        self.assertEqual(actual, expected)

    def test_no_return(self):
        from my_package import ThirdPartyArtifact

        def func(a: int, b: Input[ThirdPartyArtifact]):
            pass

        actual = custom_artifact_types.func_to_annotation_object(func)

        import my_package
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[
                    my_package.ThirdPartyArtifact,
                    kfp.components.types.type_annotations.InputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_with_return(self):
        from my_package import ThirdPartyArtifact

        def func(a: int, b: Input[ThirdPartyArtifact]) -> int:
            return 1

        import my_package
        actual = custom_artifact_types.func_to_annotation_object(func)
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[
                    my_package.ThirdPartyArtifact,
                    kfp.components.types.type_annotations.InputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_alias(self):
        from my_package import ThirdPartyArtifact
        Alias = ThirdPartyArtifact

        def func(a: int, b: Input[Alias]):
            pass

        actual = custom_artifact_types.func_to_annotation_object(func)

        import my_package
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[
                    my_package.ThirdPartyArtifact,
                    kfp.components.types.type_annotations.InputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_long_form_annotation(self):
        import my_package

        def func(a: int, b: Output[my_package.ThirdPartyArtifact]):
            pass

        actual = custom_artifact_types.func_to_annotation_object(func)

        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[
                    my_package.ThirdPartyArtifact,
                    kfp.components.types.type_annotations.OutputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_aliased_module(self):
        import my_package as my_package_alias

        def func(a: int, b: Output[my_package_alias.ThirdPartyArtifact]):
            pass

        actual = custom_artifact_types.func_to_annotation_object(func)

        import my_package
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[
                    my_package.ThirdPartyArtifact,
                    kfp.components.types.type_annotations.OutputAnnotation]
        }
        self.assertEqual(actual, expected)

    def test_input_output_path(self):
        from my_package import ThirdPartyArtifact

        def func(
            a: int,
            b: InputPath('Dataset'),
            c: Output[ThirdPartyArtifact],
        ) -> OutputPath('Dataset'):
            return 'dataset'

        actual = custom_artifact_types.func_to_annotation_object(func)

        import my_package
        expected = {
            'a':
                int,
            'b':
                kfp.components.types.type_annotations.InputPath('Dataset'),
            'c':
                typing_extensions.Annotated[
                    my_package.ThirdPartyArtifact,
                    kfp.components.types.type_annotations.OutputAnnotation]
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
        from my_package import ThirdPartyArtifact

        def func(a: int, b: Input[ThirdPartyArtifact]):
            pass

        actual = custom_artifact_types.get_custom_artifact_type_import_statements(
            func)

        expected = ['from my_package import ThirdPartyArtifact']
        self.assertEqual(actual, expected)

    def test_with_return(self):
        from my_package import ThirdPartyArtifact

        def func(a: int, b: Input[ThirdPartyArtifact]) -> int:
            return 1

        import my_package
        actual = custom_artifact_types.get_custom_artifact_type_import_statements(
            func)
        expected = ['from my_package import ThirdPartyArtifact']
        self.assertEqual(actual, expected)

    def test_alias(self):
        from my_package import ThirdPartyArtifact
        Alias = ThirdPartyArtifact

        def func(a: int, b: Input[Alias]):
            pass

        with self.assertRaisesRegex(
                TypeError, r'Module or type name aliases are not supported'):
            custom_artifact_types.get_custom_artifact_type_import_statements(
                func)

    def test_long_form_annotation(self):
        import my_package

        def func(a: int, b: Output[my_package.ThirdPartyArtifact]):
            pass

        actual = custom_artifact_types.get_custom_artifact_type_import_statements(
            func)

        expected = ['import my_package']
        self.assertEqual(actual, expected)

    def test_aliased_module(self):
        import my_package as my_package_alias

        def func(a: int, b: Output[my_package_alias.ThirdPartyArtifact]):
            pass

        with self.assertRaisesRegex(
                TypeError, r'Module or type name aliases are not supported'):
            custom_artifact_types.get_custom_artifact_type_import_statements(
                func)

    def test_input_output_path(self):
        from my_package import ThirdPartyArtifact

        def func(
            a: int,
            b: InputPath('Dataset'),
            c: Output[ThirdPartyArtifact],
        ) -> OutputPath('Dataset'):
            return 'dataset'

        actual = custom_artifact_types.get_custom_artifact_type_import_statements(
            func)

        expected = ['from my_package import ThirdPartyArtifact']
        self.assertEqual(actual, expected)


class TestImportStatementAdded(_TestCaseWithThirdPartyPackage):

    def test(self):
        import my_package
        from my_package import ThirdPartyArtifact

        @dsl.component
        def one(
            a: int,
            b: Output[ThirdPartyArtifact],
        ):
            pass

        @dsl.component
        def two(a: Input[my_package.ThirdPartyArtifact],):
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
            'from my_package import ThirdPartyArtifact',
            ' '.join(pipeline_spec['deploymentSpec']['executors']['exec-one']
                     ['container']['command']))
        self.assertIn(
            'import my_package',
            ' '.join(pipeline_spec['deploymentSpec']['executors']['exec-two']
                     ['container']['command']))


if __name__ == '__main__':
    unittest.main()

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

import collections
import inspect
import os
import sys
import tempfile
import textwrap
import typing
from typing import Any
import unittest

from absl.testing import parameterized
from kfp import dsl
from kfp.components import component_factory
from kfp.components.types import artifact_types
from kfp.components.types import type_annotations
from kfp.components.types.artifact_types import Artifact
from kfp.components.types.artifact_types import Dataset
from kfp.components.types.type_annotations import Input
from kfp.components.types.type_annotations import InputPath
from kfp.components.types.type_annotations import Output
from kfp.components.types.type_annotations import OutputPath
import typing_extensions


class TestGetPackagesToInstallCommand(unittest.TestCase):

    def test_with_no_packages_to_install(self):
        packages_to_install = []

        command = component_factory._get_packages_to_install_command(
            packages_to_install)
        self.assertEqual(command, [])

    def test_with_packages_to_install_and_no_pip_index_url(self):
        packages_to_install = ['package1', 'package2']

        command = component_factory._get_packages_to_install_command(
            packages_to_install)
        concat_command = ' '.join(command)
        for package in packages_to_install:
            self.assertTrue(package in concat_command)

    def test_with_packages_to_install_with_pip_index_url(self):
        packages_to_install = ['package1', 'package2']
        pip_index_urls = ['https://myurl.org/simple']

        command = component_factory._get_packages_to_install_command(
            packages_to_install, pip_index_urls)
        concat_command = ' '.join(command)
        for package in packages_to_install + pip_index_urls:
            self.assertTrue(package in concat_command)


Alias = Artifact
artifact_types_alias = artifact_types


class TestGetParamToAnnString(unittest.TestCase):

    def test_no_annotations(self):

        def func(a, b):
            pass

        actual = component_factory.get_param_to_ann_string(func)
        expected = {'a': None, 'b': None, 'return': None}
        self.assertEqual(actual, expected)

    def test_no_return(self):

        def func(a: int, b: Input[Artifact]):
            pass

        actual = component_factory.get_param_to_ann_string(func)
        expected = {'a': 'int', 'b': 'Artifact', 'return': None}
        self.assertEqual(actual, expected)

    def test_with_return(self):

        def func(a: int, b: Input[Artifact]) -> int:
            return 1

        actual = component_factory.get_param_to_ann_string(func)
        expected = {'a': 'int', 'b': 'Artifact', 'return': 'int'}
        self.assertEqual(actual, expected)

    def test_multiline(self):

        def func(
            a: int,
            b: Input[Artifact],
        ) -> int:
            return 1

        actual = component_factory.get_param_to_ann_string(func)
        expected = {'a': 'int', 'b': 'Artifact', 'return': 'int'}
        self.assertEqual(actual, expected)

    def test_alias(self):

        def func(a: int, b: Input[Alias]):
            pass

        actual = component_factory.get_param_to_ann_string(func)
        expected = {'a': 'int', 'b': 'Alias', 'return': None}
        self.assertEqual(actual, expected)

    def test_long_form_annotation(self):

        def func(a: int, b: Input[artifact_types.Artifact]):
            pass

        actual = component_factory.get_param_to_ann_string(func)
        expected = {'a': 'int', 'b': 'artifact_types', 'return': None}
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

        actual = component_factory.get_param_to_ann_string(func)
        expected = {
            'a': 'int',
            'b': 'Artifact',
            'return-a': 'int',
            'return-b': 'Artifact',
            'return-c': 'artifact_types'
        }
        self.assertEqual(actual, expected)

    def test_input_output_path(self):

        def func(
                a: int,
                b: InputPath('Dataset'),
        ) -> OutputPath('Dataset'):
            return 'dataset'

        actual = component_factory.get_param_to_ann_string(func)
        expected = {
            'a': 'int',
            'b': 'InputPath',
            'return': 'OutputPath',
        }
        self.assertEqual(actual, expected)


class MyCustomArtifact:
    TYPE_NAME = 'my_custom_artifact'


class _TestCaseWithThirdPartyPackage(parameterized.TestCase):

    @classmethod
    def setUpClass(cls):

        class ThirdPartyArtifact(artifact_types.Artifact):
            TYPE_NAME = 'custom.my_third_party_artifact'

        class_source = 'from kfp.components.types import artifact_types\n\n' + textwrap.dedent(
            inspect.getsource(ThirdPartyArtifact))

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

        actual = component_factory.get_param_to_ann_obj(func)
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[Artifact,
                                            type_annotations.InputAnnotation],
            'return':
                int
        }
        self.assertEqual(actual, expected)

    def test_named_tuple(self):

        MyNamedTuple = typing.NamedTuple('MyNamedTuple', [('a', int),
                                                          ('b', Artifact)])

        def func(
            a: int,
            b: Input[Artifact],
        ) -> MyNamedTuple:
            InnerNamedTuple = typing.NamedTuple('MyNamedTuple',
                                                [('a', int), ('b', Artifact)])
            return InnerNamedTuple(a=a, b=b)  # type: ignore

        actual = component_factory.get_param_to_ann_obj(func)
        expected = {
            'a':
                int,
            'b':
                typing_extensions.Annotated[Artifact,
                                            type_annotations.InputAnnotation],
            'return-a':
                int,
            'return-b':
                Artifact
        }
        self.assertEqual(actual, expected)

    def test_input_output_path(self):

        def func(
                a: int,
                b: InputPath('Dataset'),
        ) -> OutputPath('Dataset'):
            return 'dataset'

        actual = component_factory.get_param_to_ann_obj(func)
        self.assertEqual(actual['a'], int)
        self.assertIsInstance(actual['b'], InputPath)
        self.assertIsInstance(actual['return'], OutputPath)


class TestIsNamedTupleTypeAnnotation(unittest.TestCase):

    def test_true_typing(self):
        self.assertTrue(
            component_factory.is_typed_named_tuple_annotation(
                typing.NamedTuple('MyNamedTuple', [('a', int),
                                                   ('b', Artifact)])))

    def test_raises_for_collections(self):
        with self.assertRaisesRegex(
                TypeError,
                r'must be typed with field annotations using typing\.NamedTuple'
        ):
            component_factory.is_typed_named_tuple_annotation(
                collections.namedtuple('namedtuple', ['a', 'b']))

    def test_false_for_primitive(self):
        self.assertFalse(component_factory.is_typed_named_tuple_annotation(int))

    def test_false_for_class(self):

        class MyClass:
            pass

        self.assertFalse(
            component_factory.is_typed_named_tuple_annotation(MyClass))


class TestGetFullQualnameForClass(_TestCaseWithThirdPartyPackage):

    @parameterized.parameters([
        (Alias, 'kfp.components.types.artifact_types.Artifact'),
        (Artifact, 'kfp.components.types.artifact_types.Artifact'),
        (Dataset, 'kfp.components.types.artifact_types.Dataset'),
    ])
    def test(self, obj: Any, expected_qualname: str):
        self.assertEqual(
            component_factory.get_full_qualname_for_artifact(obj),
            expected_qualname)

    def test_my_package_artifact(self):
        import my_package
        self.assertEqual(
            component_factory.get_full_qualname_for_artifact(
                my_package.ThirdPartyArtifact), 'my_package.ThirdPartyArtifact')


class GetArtifactImportItemsFromFunction(_TestCaseWithThirdPartyPackage):

    def test_no_annotations(self):

        def func(a, b):
            pass

        actual = component_factory.get_artifact_import_items_from_function(func)
        expected = []
        self.assertEqual(actual, expected)

    def test_no_return(self):
        from my_package import ThirdPartyArtifact

        def func(a: int, b: Input[ThirdPartyArtifact]):
            pass

        actual = component_factory.get_artifact_import_items_from_function(func)
        expected = ['my_package.ThirdPartyArtifact']
        self.assertEqual(actual, expected)

    def test_with_return(self):
        from my_package import ThirdPartyArtifact

        def func(a: int, b: Input[ThirdPartyArtifact]) -> int:
            return 1

        actual = component_factory.get_artifact_import_items_from_function(func)
        expected = ['my_package.ThirdPartyArtifact']
        self.assertEqual(actual, expected)

    def test_multiline(self):
        from my_package import ThirdPartyArtifact

        def func(
            a: int,
            b: Input[ThirdPartyArtifact],
        ) -> int:
            return 1

        actual = component_factory.get_artifact_import_items_from_function(func)
        expected = ['my_package.ThirdPartyArtifact']
        self.assertEqual(actual, expected)

    def test_alias(self):
        from my_package import ThirdPartyArtifact
        Alias = ThirdPartyArtifact

        def func(a: int, b: Input[Alias]):
            pass

        with self.assertRaisesRegex(
                TypeError, r'Module or type name aliases are not supported'):
            component_factory.get_artifact_import_items_from_function(func)

    def test_long_form_annotation(self):
        import my_package

        def func(a: int, b: Output[my_package.ThirdPartyArtifact]):
            pass

        actual = component_factory.get_artifact_import_items_from_function(func)
        expected = ['my_package']
        self.assertEqual(actual, expected)

    def test_aliased_module_throws_error(self):
        import my_package as my_package_alias

        def func(a: int, b: Output[my_package_alias.ThirdPartyArtifact]):
            pass

        with self.assertRaisesRegex(
                TypeError, r'Module or type name aliases are not supported'):
            component_factory.get_artifact_import_items_from_function(func)

    def test_named_tuple_return(self):
        from typing import NamedTuple

        import my_package

        def named_tuple_op(
            artifact: Input[dsl.Artifact]
        ) -> NamedTuple('Outputs', [('model1', my_package.ThirdPartyArtifact),
                                    ('model2', my_package.ThirdPartyArtifact)]):
            import collections

            output = collections.namedtuple('Outputs', ['model1', 'model2'])
            return output(model1=artifact, model2=artifact)

        actual = component_factory.get_artifact_import_items_from_function(
            named_tuple_op)
        expected = ['my_package']
        self.assertEqual(actual, expected)

    def test_named_tuple_defined_outside_of_signature_throws_error(self):
        from typing import NamedTuple

        import my_package

        nt = NamedTuple('Outputs', [('model1', my_package.ThirdPartyArtifact),
                                    ('model2', my_package.ThirdPartyArtifact)])

        def named_tuple_op(artifact: Input[dsl.Artifact]) -> nt:
            import collections

            output = collections.namedtuple('Outputs', ['model1', 'model2'])
            return output(model1=artifact, model2=artifact)

        with self.assertRaisesRegex(TypeError,
                                    r'Error handling NamedTuple return type'):
            component_factory.get_artifact_import_items_from_function(
                named_tuple_op)

    def test_input_output_path(self):
        from my_package import ThirdPartyArtifact

        def func(
            a: int,
            b: InputPath('Dataset'),
            c: Output[ThirdPartyArtifact],
        ) -> OutputPath('Dataset'):
            return 'dataset'

        actual = component_factory.get_artifact_import_items_from_function(func)
        self.assertEqual(actual, ['my_package.ThirdPartyArtifact'])


if __name__ == '__main__':
    unittest.main()

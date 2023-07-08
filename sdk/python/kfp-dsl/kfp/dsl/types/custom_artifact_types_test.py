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
from kfp import dsl
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl.types import artifact_types
from kfp.dsl.types import custom_artifact_types
from kfp.dsl.types.artifact_types import Artifact
from kfp.dsl.types.artifact_types import Dataset
from kfp.dsl.types.type_annotations import InputPath
from kfp.dsl.types.type_annotations import OutputPath

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


class TestGetParamToCustomArtifactClass(_TestCaseWithThirdPartyPackage):

    def test_no_ann(self):

        def func():
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(actual, {})

    def test_primitives(self):

        def func(a: str, b: int) -> str:
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(actual, {})

    def test_input_path(self):

        def func(a: InputPath(str), b: InputPath('Dataset')) -> str:
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(actual, {})

    def test_output_path(self):

        def func(a: OutputPath(str), b: OutputPath('Dataset')) -> str:
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(actual, {})

    def test_input_kfp_artifact(self):

        def func(a: Input[Artifact]):
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(actual, {})

    def test_output_kfp_artifact(self):

        def func(a: Output[Artifact]):
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(actual, {})

    def test_return_kfp_artifact1(self):

        def func() -> Artifact:
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(actual, {})

    def test_return_kfp_artifact2(self):

        def func() -> dsl.Artifact:
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(actual, {})

    def test_named_tuple_primitives(self):

        def func() -> typing.NamedTuple('Outputs', [
            ('a', str),
            ('b', int),
        ]):
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(actual, {})

    def test_input_google_artifact(self):
        import aiplatform
        from aiplatform import VertexDataset

        def func(
            a: Input[aiplatform.VertexDataset],
            b: Input[VertexDataset],
            c: dsl.Input[aiplatform.VertexDataset],
            d: kfp.dsl.Input[VertexDataset],
        ):
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(
            actual, {
                'a': aiplatform.VertexDataset,
                'b': aiplatform.VertexDataset,
                'c': aiplatform.VertexDataset,
                'd': aiplatform.VertexDataset,
            })

    def test_output_google_artifact(self):
        import aiplatform
        from aiplatform import VertexDataset

        def func(
            a: Output[aiplatform.VertexDataset],
            b: Output[VertexDataset],
            c: dsl.Output[aiplatform.VertexDataset],
            d: kfp.dsl.Output[VertexDataset],
        ):
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(
            actual, {
                'a': aiplatform.VertexDataset,
                'b': aiplatform.VertexDataset,
                'c': aiplatform.VertexDataset,
                'd': aiplatform.VertexDataset,
            })

    def test_return_google_artifact1(self):
        import aiplatform
        from aiplatform import VertexDataset

        def func() -> VertexDataset:
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(actual, {'return-': aiplatform.VertexDataset})

    def test_return_google_artifact2(self):
        import aiplatform

        def func() -> aiplatform.VertexDataset:
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(actual, {'return-': aiplatform.VertexDataset})

    def test_named_tuple_google_artifact(self):
        import aiplatform
        from aiplatform import VertexDataset

        def func() -> typing.NamedTuple('Outputs', [
            ('a', aiplatform.VertexDataset),
            ('b', VertexDataset),
        ]):
            pass

        actual = custom_artifact_types.get_param_to_custom_artifact_class(func)
        self.assertEqual(
            actual, {
                'return-a': aiplatform.VertexDataset,
                'return-b': aiplatform.VertexDataset,
            })


class TestGetFullQualnameForArtifact(_TestCaseWithThirdPartyPackage):
    # only gets called on artifacts, so don't need to test on all types
    @parameterized.parameters([
        (Alias, 'kfp.dsl.types.artifact_types.Artifact'),
        (Artifact, 'kfp.dsl.types.artifact_types.Artifact'),
        (Dataset, 'kfp.dsl.types.artifact_types.Dataset'),
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


class TestGetSymbolImportPath(parameterized.TestCase):

    @parameterized.parameters([
        {
            'artifact_class_base_symbol': 'aiplatform',
            'qualname': 'aiplatform.VertexDataset',
            'expected': 'aiplatform'
        },
        {
            'artifact_class_base_symbol': 'VertexDataset',
            'qualname': 'aiplatform.VertexDataset',
            'expected': 'aiplatform.VertexDataset'
        },
        {
            'artifact_class_base_symbol': 'e',
            'qualname': 'a.b.c.d.e',
            'expected': 'a.b.c.d.e'
        },
        {
            'artifact_class_base_symbol': 'c',
            'qualname': 'a.b.c.d.e',
            'expected': 'a.b.c'
        },
    ])
    def test(self, artifact_class_base_symbol: str, qualname: str,
             expected: str):
        actual = custom_artifact_types.get_symbol_import_path(
            artifact_class_base_symbol, qualname)
        self.assertEqual(actual, expected)


class TestGetCustomArtifactBaseSymbolForParameter(_TestCaseWithThirdPartyPackage
                                                 ):

    def test_input_google_artifact(self):
        import aiplatform
        from aiplatform import VertexDataset

        def func(
            a: Input[aiplatform.VertexDataset],
            b: Input[VertexDataset],
            c: dsl.Input[aiplatform.VertexDataset],
            d: kfp.dsl.Input[VertexDataset],
        ):
            pass

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_parameter(
            func, 'a')
        self.assertEqual(actual, 'aiplatform')

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_parameter(
            func, 'b')
        self.assertEqual(actual, 'VertexDataset')

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_parameter(
            func, 'c')
        self.assertEqual(actual, 'aiplatform')

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_parameter(
            func, 'd')
        self.assertEqual(actual, 'VertexDataset')

    def test_output_google_artifact(self):
        import aiplatform
        from aiplatform import VertexDataset

        def func(
            a: Output[aiplatform.VertexDataset],
            b: Output[VertexDataset],
            c: dsl.Output[aiplatform.VertexDataset],
            d: kfp.dsl.Output[VertexDataset],
        ):
            pass

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_parameter(
            func, 'a')
        self.assertEqual(actual, 'aiplatform')

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_parameter(
            func, 'b')
        self.assertEqual(actual, 'VertexDataset')

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_parameter(
            func, 'c')
        self.assertEqual(actual, 'aiplatform')

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_parameter(
            func, 'd')
        self.assertEqual(actual, 'VertexDataset')


class TestGetCustomArtifactBaseSymbolForReturn(_TestCaseWithThirdPartyPackage):

    def test_return_google_artifact1(self):
        from aiplatform import VertexDataset

        def func() -> VertexDataset:
            pass

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_return(
            func, 'return-')
        self.assertEqual(actual, 'VertexDataset')

    def test_return_google_artifact2(self):
        import aiplatform

        def func() -> aiplatform.VertexDataset:
            pass

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_return(
            func, 'return-')
        self.assertEqual(actual, 'aiplatform')

    def test_named_tuple_google_artifact(self):
        import aiplatform
        from aiplatform import VertexDataset

        def func() -> typing.NamedTuple('Outputs', [
            ('a', aiplatform.VertexDataset),
            ('b', VertexDataset),
        ]):
            pass

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_return(
            func, 'return-a')
        self.assertEqual(actual, 'aiplatform')

        actual = custom_artifact_types.get_custom_artifact_base_symbol_for_return(
            func, 'return-b')
        self.assertEqual(actual, 'VertexDataset')


class TestGetCustomArtifactImportItemsFromFunction(
        _TestCaseWithThirdPartyPackage):

    def test_no_ann(self):

        def func():
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, [])

    def test_primitives(self):

        def func(a: str, b: int) -> str:
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, [])

    def test_input_path(self):

        def func(a: InputPath(str), b: InputPath('Dataset')) -> str:
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, [])

    def test_output_path(self):

        def func(a: OutputPath(str), b: OutputPath('Dataset')) -> str:
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, [])

    def test_input_kfp_artifact(self):

        def func(a: Input[Artifact]):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, [])

    def test_output_kfp_artifact(self):

        def func(a: Output[Artifact]):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, [])

    def test_return_kfp_artifact1(self):

        def func() -> Artifact:
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, [])

    def test_return_kfp_artifact2(self):

        def func() -> dsl.Artifact:
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, [])

    def test_named_tuple_primitives(self):

        def func() -> typing.NamedTuple('Outputs', [
            ('a', str),
            ('b', int),
        ]):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, [])

    def test_input_google_artifact(self):
        import aiplatform
        from aiplatform import VertexDataset

        def func(
            a: Input[aiplatform.VertexDataset],
            b: Input[VertexDataset],
            c: dsl.Input[aiplatform.VertexDataset],
            d: kfp.dsl.Input[VertexDataset],
        ):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, ['aiplatform', 'aiplatform.VertexDataset'])

    def test_output_google_artifact(self):
        import aiplatform
        from aiplatform import VertexDataset

        def func(
            a: Output[aiplatform.VertexDataset],
            b: Output[VertexDataset],
            c: dsl.Output[aiplatform.VertexDataset],
            d: kfp.dsl.Output[VertexDataset],
        ):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)

        self.assertEqual(actual, ['aiplatform', 'aiplatform.VertexDataset'])

    def test_return_google_artifact1(self):
        import aiplatform
        from aiplatform import VertexDataset

        def func() -> VertexDataset:
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, ['aiplatform.VertexDataset'])

    def test_return_google_artifact2(self):
        import aiplatform

        def func() -> aiplatform.VertexDataset:
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, ['aiplatform'])

    def test_named_tuple_google_artifact(self):
        import aiplatform
        from aiplatform import VertexDataset

        def func() -> typing.NamedTuple('Outputs', [
            ('a', aiplatform.VertexDataset),
            ('b', VertexDataset),
        ]):
            pass

        actual = custom_artifact_types.get_custom_artifact_import_items_from_function(
            func)
        self.assertEqual(actual, ['aiplatform', 'aiplatform.VertexDataset'])


if __name__ == '__main__':
    unittest.main()

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
import json
import os
import subprocess
import sys
import tempfile
import textwrap
import typing
from typing import Any
import unittest

from absl.testing import parameterized
import kfp
from kfp import dsl
from kfp.dsl import component_factory
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

try:
    import pydantic
except ImportError:
    pydantic = None


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

        if pydantic is not None:
            with open(os.path.join(tmp_dir.name, 'my_pydantic_models.py'),
                      'w') as f:
                f.write(
                    textwrap.dedent("""\
                    import pydantic

                    class MyModel(pydantic.BaseModel):
                        foo: str
                    """))

        sys.path.append(tmp_dir.name)
        cls.tmp_dir = tmp_dir

    @classmethod
    def tearDownClass(cls):
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


@unittest.skipIf(pydantic is None, 'pydantic is not installed')
class TestGetParamToPydanticBasemodelClass(_TestCaseWithThirdPartyPackage):

    def test_no_ann(self):

        def func():
            pass

        actual = custom_artifact_types.get_param_to_pydantic_basemodel_class(
            func)
        self.assertEqual(actual, {})

    def test_primitives(self):

        def func(a: str, b: int) -> str:
            pass

        actual = custom_artifact_types.get_param_to_pydantic_basemodel_class(
            func)
        self.assertEqual(actual, {})

    def test_input_basemodel(self):
        from my_pydantic_models import MyModel

        def func(a: MyModel):
            pass

        actual = custom_artifact_types.get_param_to_pydantic_basemodel_class(
            func)
        self.assertEqual(actual, {'a': MyModel})

    def test_return_basemodel(self):
        from my_pydantic_models import MyModel

        def func() -> MyModel:
            pass

        actual = custom_artifact_types.get_param_to_pydantic_basemodel_class(
            func)
        self.assertEqual(actual, {'return-': MyModel})

    def test_named_tuple_basemodel(self):
        from my_pydantic_models import MyModel

        def func() -> typing.NamedTuple('Outputs', [
            ('a', MyModel),
            ('b', str),
        ]):
            pass

        actual = custom_artifact_types.get_param_to_pydantic_basemodel_class(
            func)
        self.assertEqual(actual, {'return-a': MyModel})

    def test_optional_input_basemodel(self):
        from my_pydantic_models import MyModel

        def func(a: typing.Optional[MyModel] = None):
            pass

        actual = custom_artifact_types.get_param_to_pydantic_basemodel_class(
            func)
        self.assertEqual(actual, {'a': MyModel})

    def test_optional_return_basemodel(self):
        from my_pydantic_models import MyModel

        def func() -> typing.Optional[MyModel]:
            pass

        actual = custom_artifact_types.get_param_to_pydantic_basemodel_class(
            func)
        self.assertEqual(actual, {'return-': MyModel})


@unittest.skipIf(pydantic is None, 'pydantic is not installed')
class TestGetPydanticBasemodelImportItemsFromFunction(
        _TestCaseWithThirdPartyPackage):

    def test_no_ann(self):

        def func():
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_import_items_from_function(
            func)
        self.assertEqual(actual, [])

    def test_input_basemodel(self):
        from my_pydantic_models import MyModel

        def func(a: MyModel):
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_import_items_from_function(
            func)
        self.assertEqual(actual, ['my_pydantic_models.MyModel'])

    def test_import_statement_format(self):
        from my_pydantic_models import MyModel

        def func(a: MyModel) -> MyModel:
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_type_import_statements(
            func)
        self.assertEqual(actual, ['from my_pydantic_models import MyModel'])

    def test_optional_input_basemodel(self):
        from my_pydantic_models import MyModel

        def func(a: typing.Optional[MyModel] = None):
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_import_items_from_function(
            func)
        self.assertEqual(actual, ['my_pydantic_models.MyModel'])

    def test_raises_for_simple_alias(self):
        from my_pydantic_models import MyModel as MyModelAlias

        def func(a: MyModelAlias):
            pass

        with self.assertRaisesRegex(TypeError, 'aliases are not supported'):
            custom_artifact_types.get_pydantic_basemodel_import_items_from_function(
                func)

    def test_raises_for_aliased_module_dotted_access(self):
        import my_pydantic_models as aliased_module

        def func(a: aliased_module.MyModel):
            pass

        with self.assertRaisesRegex(TypeError, 'aliases are not supported'):
            custom_artifact_types.get_pydantic_basemodel_import_items_from_function(
                func)

    def test_passes_for_unaliased_module_dotted_access(self):
        import my_pydantic_models

        def func(a: my_pydantic_models.MyModel):
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_import_items_from_function(
            func)
        self.assertEqual(actual, ['my_pydantic_models'])

    def test_raises_for_return_alias(self):
        from my_pydantic_models import MyModel as MyModelAlias

        def func() -> MyModelAlias:
            pass

        with self.assertRaisesRegex(TypeError, 'aliases are not supported'):
            custom_artifact_types.get_pydantic_basemodel_import_items_from_function(
                func)


@unittest.skipIf(pydantic is None, 'pydantic is not installed')
class TestGetPydanticBasemodelBaseSymbolForParameter(
        _TestCaseWithThirdPartyPackage):

    def test_simple_name(self):
        from my_pydantic_models import MyModel

        def func(a: MyModel):
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_base_symbol_for_parameter(
            func, 'a')
        self.assertEqual(actual, 'MyModel')

    def test_optional_simple_name(self):
        from my_pydantic_models import MyModel

        def func(a: typing.Optional[MyModel] = None):
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_base_symbol_for_parameter(
            func, 'a')
        self.assertEqual(actual, 'MyModel')

    def test_dotted_access(self):
        import my_pydantic_models

        def func(a: my_pydantic_models.MyModel):
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_base_symbol_for_parameter(
            func, 'a')
        self.assertEqual(actual, 'my_pydantic_models')

    def test_alias(self):
        from my_pydantic_models import MyModel as MyModelAlias

        def func(a: MyModelAlias):
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_base_symbol_for_parameter(
            func, 'a')
        self.assertEqual(actual, 'MyModelAlias')

    def test_keyword_only_param(self):
        from my_pydantic_models import MyModel

        def func(*, a: MyModel):
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_base_symbol_for_parameter(
            func, 'a')
        self.assertEqual(actual, 'MyModel')

    def test_positional_only_param(self):
        from my_pydantic_models import MyModel

        def func(a: MyModel, /):
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_base_symbol_for_parameter(
            func, 'a')
        self.assertEqual(actual, 'MyModel')

    def test_pep604_optional_simple_name(self):
        from my_pydantic_models import MyModel

        def func(a: typing.Optional[MyModel] = None):
            pass

        # can't write `MyModel | None` directly in this test file's source
        # since it's collected on Python 3.9 too, where evaluating `|` on a
        # bare type raises TypeError; exercise the AST path itself instead by
        # parsing an equivalent PEP 604 signature as a string.
        import unittest.mock

        pep604_source = 'def func(a: MyModel | None = None):\n    pass\n'
        with unittest.mock.patch.object(
                custom_artifact_types.component_factory,
                '_get_function_source_definition',
                return_value=pep604_source):
            actual = custom_artifact_types.get_pydantic_basemodel_base_symbol_for_parameter(
                func, 'a')
        self.assertEqual(actual, 'MyModel')


@unittest.skipIf(pydantic is None, 'pydantic is not installed')
class TestGetPydanticBasemodelBaseSymbolForReturn(_TestCaseWithThirdPartyPackage
                                                 ):

    def test_simple_name(self):
        from my_pydantic_models import MyModel

        def func() -> MyModel:
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_base_symbol_for_return(
            func, custom_artifact_types.RETURN_PREFIX)
        self.assertEqual(actual, 'MyModel')

    def test_named_tuple_field(self):
        from my_pydantic_models import MyModel

        def func() -> typing.NamedTuple('Outputs', [('output_model', MyModel)]):
            pass

        actual = custom_artifact_types.get_pydantic_basemodel_base_symbol_for_return(
            func, f'{custom_artifact_types.RETURN_PREFIX}output_model')
        self.assertEqual(actual, 'MyModel')


@unittest.skipIf(pydantic is None, 'pydantic is not installed')
class TestValidatePydanticBasemodelIsImportable(_TestCaseWithThirdPartyPackage):

    def test_passes_for_module_level_class(self):
        from my_pydantic_models import MyModel

        # should not raise
        custom_artifact_types.validate_pydantic_basemodel_is_importable(MyModel)

    def test_raises_for_class_defined_in_main(self):

        class MyModel(pydantic.BaseModel):
            foo: str

        MyModel.__module__ = '__main__'
        with self.assertRaisesRegex(TypeError, "defined in the '__main__'"):
            custom_artifact_types.validate_pydantic_basemodel_is_importable(
                MyModel)

    def test_raises_for_function_local_class(self):

        def make_model():

            class MyModel(pydantic.BaseModel):
                foo: str

            return MyModel

        with self.assertRaisesRegex(TypeError,
                                    'defined inside a function or method'):
            custom_artifact_types.validate_pydantic_basemodel_is_importable(
                make_model())

    def test_import_collection_raises_for_class_defined_in_main(self):

        class MyModel(pydantic.BaseModel):
            foo: str

        MyModel.__module__ = '__main__'

        def func(a: MyModel):
            pass

        with self.assertRaisesRegex(TypeError, "defined in the '__main__'"):
            custom_artifact_types.get_pydantic_basemodel_import_items_from_function(
                func)


@unittest.skipIf(pydantic is None, 'pydantic is not installed')
class TestPydanticBasemodelRuntimeIsolation(_TestCaseWithThirdPartyPackage):
    """End-to-end tests that actually execute a compiled lightweight component
    in a fresh `python -m kfp.dsl.executor_main` subprocess whose working
    directory and PYTHONPATH are set explicitly, rather than inherited from the
    test process.

    Unlike `kfp.local.SubprocessRunner`, which runs the same command but
    inherits the developer machine's cwd and installed packages
    wholesale, this reproduces the isolation boundary a real component
    runtime has: a `pydantic.BaseModel` used for component I/O is only
    resolvable if its module is explicitly made importable at runtime
    (e.g., via `packages_to_install` or a custom `base_image`), not
    because of ambient dev-environment state.
    """

    def _write_ephemeral_component(self, func, run_dir: str) -> str:
        command, _ = component_factory._get_command_and_args_for_lightweight_component(
            func)
        ephemeral_component_source = command[-1]
        component_path = os.path.join(run_dir, 'ephemeral_component.py')
        with open(component_path, 'w') as f:
            f.write(ephemeral_component_source)
        return component_path

    def _run_isolated(
            self, func, executor_input: dict,
            extra_pythonpath_entries: list) -> subprocess.CompletedProcess:
        run_dir = tempfile.TemporaryDirectory()
        self.addCleanup(run_dir.cleanup)
        component_path = self._write_ephemeral_component(func, run_dir.name)

        isolated_cwd = tempfile.TemporaryDirectory()
        self.addCleanup(isolated_cwd.cleanup)

        # sys.path in this test process includes self.tmp_dir.name (appended
        # by setUpClass so `from my_pydantic_models import MyModel` resolves
        # in-process); exclude it here so the subprocess's PYTHONPATH is
        # governed solely by extra_pythonpath_entries.
        pythonpath_entries = extra_pythonpath_entries + [
            p for p in sys.path if p and p != self.tmp_dir.name
        ]
        env = {
            'PATH': os.environ.get('PATH', ''),
            'PYTHONPATH': os.pathsep.join(pythonpath_entries),
            '_KFP_RUNTIME': 'true',
        }

        return subprocess.run(
            [
                sys.executable,
                '-m',
                'kfp.dsl.executor_main',
                '--component_module_path',
                component_path,
                '--function_to_execute',
                func.__name__,
                '--executor_input',
                json.dumps(executor_input),
            ],
            cwd=isolated_cwd.name,
            env=env,
            capture_output=True,
            text=True,
        )

    def test_model_on_runtime_path_executes_successfully_in_isolation(self):
        from my_pydantic_models import MyModel

        def func(my_data: MyModel) -> str:
            return my_data.foo

        with tempfile.TemporaryDirectory() as output_dir:
            output_file = os.path.join(output_dir, 'output_metadata.json')
            result = self._run_isolated(
                func,
                executor_input={
                    'inputs': {
                        'parameterValues': {
                            'my_data': {
                                'foo': 'bar'
                            }
                        }
                    },
                    'outputs': {
                        'outputFile': output_file
                    },
                },
                # simulates installing the model's package into the
                # runtime environment, e.g. via `packages_to_install`
                extra_pythonpath_entries=[self.tmp_dir.name],
            )
            self.assertEqual(
                0, result.returncode,
                f'stdout:\n{result.stdout}\nstderr:\n{result.stderr}')
            with open(output_file) as f:
                output_metadata = json.load(f)
            self.assertEqual('bar',
                             output_metadata['parameterValues']['Output'])

    def test_model_missing_from_runtime_path_fails_in_isolation(self):
        from my_pydantic_models import MyModel

        def func(my_data: MyModel) -> str:
            return my_data.foo

        with tempfile.TemporaryDirectory() as output_dir:
            output_file = os.path.join(output_dir, 'output_metadata.json')
            result = self._run_isolated(
                func,
                executor_input={
                    'inputs': {
                        'parameterValues': {
                            'my_data': {
                                'foo': 'bar'
                            }
                        }
                    },
                    'outputs': {
                        'outputFile': output_file
                    },
                },
                # the model's package was never installed into the runtime
                extra_pythonpath_entries=[],
            )
            self.assertNotEqual(0, result.returncode)
            self.assertIn('my_pydantic_models', result.stderr)


if __name__ == '__main__':
    unittest.main()

# Copyright 2022 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the 'License');
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an 'AS IS' BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import dataclasses
import functools
from typing import (Any, Dict, List, Mapping, MutableMapping, MutableSequence,
                    Optional, OrderedDict, Sequence, Set, Tuple, Union)
import unittest

from absl.testing import parameterized
from kfp.components import base_model


class TypeClass(base_model.BaseModel):
    a: str
    b: List[int]
    c: Dict[str, int]
    d: Union[int, str]
    e: Union[int, str, bool]
    f: Optional[int]


class TestBaseModel(unittest.TestCase):

    def test_is_dataclass(self):

        class Child(base_model.BaseModel):
            x: int

        child = Child(x=1)
        self.assertTrue(dataclasses.is_dataclass(child))

    def test_can_create_properties_using_attributes(self):

        class Child(base_model.BaseModel):
            x: Optional[int]

            @property
            def prop(self) -> bool:
                return self.x is not None

        child1 = Child(x=None)
        self.assertEqual(child1.prop, False)

        child2 = Child(x=1)
        self.assertEqual(child2.prop, True)

    def test_unsupported_type_success(self):

        class OtherClass(base_model.BaseModel):
            x: int

        class MyClass(base_model.BaseModel):
            a: OtherClass

    def test_unsupported_type_failures(self):

        with self.assertRaisesRegex(TypeError, r'not a supported'):

            class MyClass(base_model.BaseModel):
                a: tuple

        with self.assertRaisesRegex(TypeError, r'not a supported'):

            class MyClass(base_model.BaseModel):
                a: Tuple

        with self.assertRaisesRegex(TypeError, r'not a supported'):

            class MyClass(base_model.BaseModel):
                a: Set

        with self.assertRaisesRegex(TypeError, r'not a supported'):

            class OtherClass:
                pass

            class MyClass(base_model.BaseModel):
                a: OtherClass

    def test_base_model_validation(self):

        # test exception thrown
        class MyClass(base_model.BaseModel):
            x: int

            def _validate_x(self) -> None:
                if self.x < 2:
                    raise ValueError('x must be greater than 2')

        with self.assertRaisesRegex(ValueError, 'x must be greater than 2'):
            mc = MyClass(x=1)

        # test value modified same type
        class MyClass(base_model.BaseModel):
            x: int

            def _validate_x(self) -> None:
                self.x = max(self.x, 2)

        mc = MyClass(x=1)
        self.assertEqual(mc.x, 2)

        # test value modified new type
        class MyClass(base_model.BaseModel):
            x: Optional[List[int]] = None

            def _validate_x(self) -> None:
                if isinstance(self.x, list) and not self.x:
                    self.x = None

        mc = MyClass(x=[])
        self.assertEqual(mc.x, None)

    def test_can_set_field(self):

        class MyClass(base_model.BaseModel):
            x: int

        mc = MyClass(x=2)
        mc.x = 1
        self.assertEqual(mc.x, 1)

    def test_can_use_default_factory(self):

        class MyClass(base_model.BaseModel):
            x: List[int] = dataclasses.field(default_factory=list)

        mc = MyClass()
        self.assertEqual(mc.x, [])


class TestIsBaseModel(unittest.TestCase):

    def test_true(self):
        self.assertEqual(base_model._is_basemodel(base_model.BaseModel), True)

        class MyClass(base_model.BaseModel):
            pass

        self.assertEqual(base_model._is_basemodel(MyClass), True)

    def test_false(self):
        self.assertEqual(base_model._is_basemodel(int), False)
        self.assertEqual(base_model._is_basemodel(1), False)
        self.assertEqual(base_model._is_basemodel(str), False)


class TestGetOriginPy37(parameterized.TestCase):

    def test_is_same_as_typing_version(self):
        import sys
        if sys.version_info.major == 3 and sys.version_info.minor >= 8:
            import typing
            for field in dataclasses.fields(TypeClass):
                self.assertEqual(
                    base_model._get_origin_py37(field.type),
                    typing.get_origin(field.type))


class TestGetArgsPy37(parameterized.TestCase):

    def test_is_same_as_typing_version(self):
        import sys
        if sys.version_info.major == 3 and sys.version_info.minor >= 8:
            import typing
            for field in dataclasses.fields(TypeClass):
                self.assertEqual(
                    base_model._get_args_py37(field.type),
                    typing.get_args(field.type))


if __name__ == '__main__':
    unittest.main()

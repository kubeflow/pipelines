# Copyright 2021 The Kubeflow Authors
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
"""Tests for kfp.dsl.types.type_annotations."""

import sys
from typing import Any, Dict, List, Optional, Union
import unittest

from absl.testing import parameterized
from kfp import dsl
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl.types import artifact_types
from kfp.dsl.types import type_annotations
from kfp.dsl.types.artifact_types import Model
from kfp.dsl.types.type_annotations import InputAnnotation
from kfp.dsl.types.type_annotations import InputPath
from kfp.dsl.types.type_annotations import OutputAnnotation
from kfp.dsl.types.type_annotations import OutputPath


class AnnotationsTest(parameterized.TestCase):

    @parameterized.parameters([
        Input[Model],
        Output[Model],
        Output[List[Model]],
        Output['MyArtifact'],
    ])
    def test_is_artifact_annotation(self, annotation):
        self.assertTrue(
            type_annotations.is_Input_Output_artifact_annotation(annotation))

    @parameterized.parameters([
        Model,
        int,
        'Dataset',
        List[str],
        Optional[str],
    ])
    def test_is_not_artifact_annotation(self, annotation):
        self.assertFalse(
            type_annotations.is_Input_Output_artifact_annotation(annotation))

    @parameterized.parameters([
        Input[Model],
        Input,
    ])
    def test_is_input_artifact(self, annotation):
        self.assertTrue(
            type_annotations.is_artifact_wrapped_in_Input(annotation))

    @parameterized.parameters([
        Output[Model],
        Output,
    ])
    def test_is_not_input_artifact(self, annotation):
        self.assertFalse(
            type_annotations.is_artifact_wrapped_in_Input(annotation))

    @parameterized.parameters([
        Output[Model],
        Output[List[Model]],
    ])
    def test_is_output_artifact(self, annotation):
        self.assertTrue(
            type_annotations.is_artifact_wrapped_in_Output(annotation))

    @parameterized.parameters([
        Input[Model],
        Input[List[Model]],
        Input,
    ])
    def test_is_not_output_artifact(self, annotation):
        self.assertFalse(
            type_annotations.is_artifact_wrapped_in_Output(annotation))

    def test_get_io_artifact_class(self):
        self.assertEqual(
            type_annotations.get_io_artifact_class(Output[Model]), Model)
        self.assertEqual(
            type_annotations.get_io_artifact_class(Output[List[Model]]), Model)
        self.assertEqual(
            type_annotations.get_io_artifact_class(Input[List[Model]]), Model)

        self.assertEqual(type_annotations.get_io_artifact_class(Input), None)
        self.assertEqual(type_annotations.get_io_artifact_class(Output), None)
        self.assertEqual(type_annotations.get_io_artifact_class(Model), None)
        self.assertEqual(type_annotations.get_io_artifact_class(str), None)

    def test_get_io_artifact_annotation(self):
        self.assertEqual(
            type_annotations.get_input_or_output_marker(Output[Model]),
            OutputAnnotation)
        self.assertEqual(
            type_annotations.get_input_or_output_marker(Output[List[Model]]),
            OutputAnnotation)
        self.assertEqual(
            type_annotations.get_input_or_output_marker(Input[Model]),
            InputAnnotation)
        self.assertEqual(
            type_annotations.get_input_or_output_marker(Input[List[Model]]),
            InputAnnotation)
        self.assertEqual(
            type_annotations.get_input_or_output_marker(Input), InputAnnotation)
        self.assertEqual(
            type_annotations.get_input_or_output_marker(Output),
            OutputAnnotation)

        self.assertEqual(
            type_annotations.get_input_or_output_marker(Model), None)
        self.assertEqual(type_annotations.get_input_or_output_marker(str), None)

    @parameterized.parameters(
        {
            'original_annotation': str,
            'expected_annotation': str,
        },
        {
            'original_annotation': 'MyCustomType',
            'expected_annotation': 'MyCustomType',
        },
        {
            'original_annotation': List[int],
            'expected_annotation': List[int],
        },
        {
            'original_annotation': Optional[str],
            'expected_annotation': str,
        },
        {
            'original_annotation': Optional[Dict[str, float]],
            'expected_annotation': Dict[str, float],
        },
        {
            'original_annotation': Optional[List[Dict[str, Any]]],
            'expected_annotation': List[Dict[str, Any]],
        },
        {
            'original_annotation': Input[Model],
            'expected_annotation': Input[Model],
        },
        {
            'original_annotation': InputPath('Model'),
            'expected_annotation': InputPath('Model'),
        },
        {
            'original_annotation': OutputPath(Model),
            'expected_annotation': OutputPath(Model),
        },
    )
    def test_maybe_strip_optional_from_annotation(self, original_annotation,
                                                  expected_annotation):
        self.assertEqual(
            expected_annotation,
            type_annotations.maybe_strip_optional_from_annotation(
                original_annotation))

    @parameterized.parameters(
        {
            'original_type_name': 'str',
            'expected_type_name': 'str',
        },
        {
            'original_type_name': 'typing.List[int]',
            'expected_type_name': 'List',
        },
        {
            'original_type_name': 'List[int]',
            'expected_type_name': 'List',
        },
        {
            'original_type_name': 'Dict[str, str]',
            'expected_type_name': 'Dict',
        },
        {
            'original_type_name': 'List[Dict[str, str]]',
            'expected_type_name': 'List',
        },
    )
    def test_get_short_type_name(self, original_type_name, expected_type_name):
        self.assertEqual(
            expected_type_name,
            type_annotations.get_short_type_name(original_type_name))


class TestIsArtifact(parameterized.TestCase):

    @parameterized.parameters([{
        'obj': obj
    } for obj in artifact_types._SCHEMA_TITLE_TO_TYPE.values()])
    def test_true_class(self, obj):
        self.assertTrue(type_annotations.is_artifact_class(obj))

    @parameterized.parameters([{
        'obj': obj(name='name', uri='uri', metadata={})
    } for obj in artifact_types._SCHEMA_TITLE_TO_TYPE.values()])
    def test_true_instance(self, obj):
        self.assertTrue(type_annotations.is_artifact_class(obj))

    @parameterized.parameters([{'obj': 'string'}, {'obj': 1}, {'obj': int}])
    def test_false(self, obj):
        self.assertFalse(type_annotations.is_artifact_class(obj))

    def test_false_no_schema_title(self):

        class MissingSchemaTitle:
            schema_version = ''

        self.assertFalse(type_annotations.is_artifact_class(MissingSchemaTitle))

    def test_false_no_schema_version(self):

        class MissingSchemaVersion:
            schema_title = ''

        self.assertFalse(
            type_annotations.is_artifact_class(MissingSchemaVersion))


class ArtifactSubclass(dsl.Artifact):
    pass


class NotArtifactSubclass:
    pass


class TestIsSubclassOfArtifact(parameterized.TestCase):

    @parameterized.parameters([{
        'obj': obj
    } for obj in [
        dsl.Artifact,
        dsl.Dataset,
        dsl.Metrics,
        ArtifactSubclass,
    ]])
    def test_true(self, obj):
        self.assertTrue(type_annotations.issubclass_of_artifact(obj))

    @parameterized.parameters([{
        'obj': obj
    } for obj in [
        dsl.Artifact(),
        dsl.Dataset(),
        1,
        NotArtifactSubclass,
    ]])
    def test_false(self, obj):
        self.assertFalse(type_annotations.issubclass_of_artifact(obj))


class TestIsGenericList(parameterized.TestCase):

    @parameterized.parameters([{
        'obj': obj
    } for obj in [
        List,
        List[str],
        List[dsl.Artifact],
        List[Dict[str, str]],
    ] + ([
        list,
        list[str],
    ] if sys.version_info >= (3, 9, 0) else [])])
    def test_true(self, obj):
        self.assertTrue(type_annotations.is_generic_list(obj))

    @parameterized.parameters([{
        'obj': obj
    } for obj in [
        Optional[List[str]],
        Dict[str, str],
        str,
        int,
        dsl.Artifact,
    ]])
    def test_false(self, obj):
        self.assertFalse(type_annotations.is_generic_list(obj))


class TestGetInnerType(parameterized.TestCase):

    @parameterized.parameters([{
        'annotation': annotation,
        'expected': expected
    } for annotation, expected in [
        (int, None),
        (Optional[int], (int, type(None))),
        (Union[int, None], (int, type(None))),
        (List[str], str),
        (Dict[str, str], (str, str)),
        (List[dsl.Artifact], dsl.Artifact),
    ] + ([
        (list[str], str),
        (dict[str, str], (str, str)),
        (list[dsl.Artifact], dsl.Artifact),
    ] if sys.version_info >= (3, 9, 0) else [])])
    def test(self, annotation, expected):
        actual = type_annotations.get_inner_type(annotation)
        self.assertEqual(actual, expected)


if __name__ == '__main__':
    unittest.main()

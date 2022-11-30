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
"""Tests for kfp.components.types.type_annotations."""

from typing import Any, Dict, List, Optional
import unittest

from absl.testing import parameterized
from kfp.components.types import artifact_types
from kfp.components.types import type_annotations
from kfp.components.types.artifact_types import Model
from kfp.components.types.type_annotations import InputAnnotation
from kfp.components.types.type_annotations import InputPath
from kfp.components.types.type_annotations import OutputAnnotation
from kfp.components.types.type_annotations import OutputPath
from kfp.dsl import Input
from kfp.dsl import Output


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
        self.assertTrue(type_annotations.is_input_artifact(annotation))

    @parameterized.parameters([
        Output[Model],
        Output,
    ])
    def test_is_not_input_artifact(self, annotation):
        self.assertFalse(type_annotations.is_input_artifact(annotation))

    @parameterized.parameters([
        Output[Model],
        Output[List[Model]],
    ])
    def test_is_output_artifact(self, annotation):
        self.assertTrue(type_annotations.is_output_artifact(annotation))

    @parameterized.parameters([
        Input[Model],
        Input[List[Model]],
        Input,
    ])
    def test_is_not_output_artifact(self, annotation):
        self.assertFalse(type_annotations.is_output_artifact(annotation))

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
            type_annotations.get_io_artifact_annotation(Output[Model]),
            OutputAnnotation)
        self.assertEqual(
            type_annotations.get_io_artifact_annotation(Output[List[Model]]),
            OutputAnnotation)
        self.assertEqual(
            type_annotations.get_io_artifact_annotation(Input[Model]),
            InputAnnotation)
        self.assertEqual(
            type_annotations.get_io_artifact_annotation(Input[List[Model]]),
            InputAnnotation)
        self.assertEqual(
            type_annotations.get_io_artifact_annotation(Input), InputAnnotation)
        self.assertEqual(
            type_annotations.get_io_artifact_annotation(Output),
            OutputAnnotation)

        self.assertEqual(
            type_annotations.get_io_artifact_annotation(Model), None)
        self.assertEqual(type_annotations.get_io_artifact_annotation(str), None)

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

        class NotArtifact:
            schema_version = ''

        self.assertFalse(type_annotations.is_artifact_class(NotArtifact))

    def test_false_no_schema_version(self):

        class NotArtifact:
            schema_title = ''

        self.assertFalse(type_annotations.is_artifact_class(NotArtifact))


if __name__ == '__main__':
    unittest.main()

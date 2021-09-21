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
"""Tests for kfp.v2.components.types.type_annotations."""

import unittest
from typing import List, Optional

from kfp.v2.components.types import type_annotations
from kfp.v2.components.types.artifact_types import Model
from kfp.v2.components.types.type_annotations import Input, InputAnnotation, Output, OutputAnnotation


class AnnotationsTest(unittest.TestCase):

    def test_is_artifact_annotation(self):
        self.assertTrue(type_annotations.is_artifact_annotation(Input[Model]))
        self.assertTrue(type_annotations.is_artifact_annotation(Output[Model]))
        self.assertTrue(
            type_annotations.is_artifact_annotation(Output['MyArtifact']))

        self.assertFalse(type_annotations.is_artifact_annotation(Model))
        self.assertFalse(type_annotations.is_artifact_annotation(int))
        self.assertFalse(type_annotations.is_artifact_annotation('Dataset'))
        self.assertFalse(type_annotations.is_artifact_annotation(List[str]))
        self.assertFalse(type_annotations.is_artifact_annotation(Optional[str]))

    def test_is_input_artifact(self):
        self.assertTrue(type_annotations.is_input_artifact(Input[Model]))
        self.assertTrue(type_annotations.is_input_artifact(Input))

        self.assertFalse(type_annotations.is_input_artifact(Output[Model]))
        self.assertFalse(type_annotations.is_input_artifact(Output))

    def test_is_output_artifact(self):
        self.assertTrue(type_annotations.is_output_artifact(Output[Model]))
        self.assertTrue(type_annotations.is_output_artifact(Output))

        self.assertFalse(type_annotations.is_output_artifact(Input[Model]))
        self.assertFalse(type_annotations.is_output_artifact(Input))

    def test_get_io_artifact_class(self):
        self.assertEqual(type_annotations.get_io_artifact_class(Output[Model]),
                         Model)

        self.assertEqual(type_annotations.get_io_artifact_class(Input), None)
        self.assertEqual(type_annotations.get_io_artifact_class(Output), None)
        self.assertEqual(type_annotations.get_io_artifact_class(Model), None)
        self.assertEqual(type_annotations.get_io_artifact_class(str), None)

    def test_get_io_artifact_annotation(self):
        self.assertEqual(
            type_annotations.get_io_artifact_annotation(Output[Model]),
            OutputAnnotation)
        self.assertEqual(
            type_annotations.get_io_artifact_annotation(Input[Model]),
            InputAnnotation)
        self.assertEqual(type_annotations.get_io_artifact_annotation(Input),
                         InputAnnotation)
        self.assertEqual(type_annotations.get_io_artifact_annotation(Output),
                         OutputAnnotation)

        self.assertEqual(type_annotations.get_io_artifact_annotation(Model),
                         None)
        self.assertEqual(type_annotations.get_io_artifact_annotation(str), None)


if __name__ == '__main__':
    unittest.main()

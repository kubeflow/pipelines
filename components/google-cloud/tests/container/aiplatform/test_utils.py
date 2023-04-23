# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Test remote_runner module."""

from inspect import signature
import json
from typing import Optional, Dict, Tuple, Union, ForwardRef
import unittest
from google.cloud import aiplatform
from google.cloud import aiplatform_v1beta1
from google_cloud_pipeline_components.container.aiplatform import utils


class UtilsTests(unittest.TestCase):

  def setUp(self):
    super(UtilsTests, self).setUp()

  def _test_method(
      self, credentials: Optional, sync: bool, input_param: str, input_name: str
  ):
    """Test short description.

    Long descirption

    Args:
        credentials: credentials
        sync: sync
        input_param: input_param
        input_name:input_name
    """

  def test_get_forward_reference_with_annotation_str(self):
    annotation = aiplatform.Model.__name__

    results = utils.get_forward_reference(annotation)
    self.assertEqual(results, aiplatform.Model)

  def test_get_forward_reference_with_annotation_forward_reference(self):
    annotation = ForwardRef(aiplatform.Model.__name__)

    results = utils.get_forward_reference(annotation)
    self.assertEqual(results, aiplatform.Model)

  def test_resolve_annotation_with_annotation_class(self):
    annotation = aiplatform.Model

    results = utils.resolve_annotation(annotation)
    self.assertEqual(results, annotation)

  def test_resolve_annotation_with_annotation_foward_str_reference(self):
    annotation = aiplatform.Model.__name__

    results = utils.resolve_annotation(annotation)
    self.assertEqual(results, aiplatform.Model)

  def test_resolve_annotation_with_annotation_foward_typed_reference(self):
    annotation = ForwardRef(aiplatform.Model.__name__)

    results = utils.resolve_annotation(annotation)
    self.assertEqual(results, aiplatform.Model)

  def test_resolve_annotation_with_annotation_type_union(self):
    annotation = Union[Dict, None]

    results = utils.resolve_annotation(annotation)
    self.assertEqual(results, Dict)

  def test_resolve_annotation_with_annotation_type_empty(self):
    annotation = None

    results = utils.resolve_annotation(annotation)
    self.assertEqual(results, None)

  def test_is_serializable_to_json_with_serializable_type(self):
    annotation = Dict

    results = utils.is_serializable_to_json(annotation)
    self.assertTrue(results)

  def test_is_serializable_to_json_with_not_serializable_type(self):
    annotation = Tuple

    results = utils.is_serializable_to_json(annotation)
    self.assertFalse(results)

  def test_is_mb_sdk_resource_noun_type_with_not_noun_type(self):
    annotation = Tuple

    results = utils.is_serializable_to_json(annotation)
    self.assertFalse(results)

  def test_is_mb_sdk_resource_noun_type_with_resource_noun_type(self):
    mb_sdk_type = aiplatform.Model

    results = utils.is_mb_sdk_resource_noun_type(mb_sdk_type)
    self.assertTrue(results)

  def test_get_proto_plus_deserializer_with_serializable_type(self):
    annotation = aiplatform_v1beta1.types.explanation.ExplanationParameters

    results = utils.get_proto_plus_deserializer(annotation)
    self.assertEqual(results, annotation.from_json)

  def test_get_proto_plus_deserializer_with_not_proto_plus_type(self):
    annotation = Dict

    results = utils.get_proto_plus_deserializer(annotation)
    self.assertEqual(results, None)

  def test_get_deserializer_with_proto_plus_type(self):
    annotation = aiplatform_v1beta1.types.explanation.ExplanationParameters

    results = utils.get_deserializer(annotation)
    self.assertEqual(results, annotation.from_json)

  def test_get_deserializer_with_serializable_type(self):
    annotation = Dict

    results = utils.get_deserializer(annotation)
    self.assertEqual(results, json.loads)

  def test_get_deserializer_with_not_serializable_type(self):
    annotation = Tuple

    results = utils.get_deserializer(annotation)
    self.assertEqual(results, None)

  def test_custom_training_typed_dataset_annotation_defaults_to_using_base_dataset(
      self,
  ):
    dataset_annotation = (
        signature(aiplatform.CustomTrainingJob.run)
        .parameters['dataset']
        .annotation
    )

    assert (
        utils.resolve_annotation(dataset_annotation)
        is aiplatform.datasets.dataset._Dataset
    )

    dataset_annotation = (
        signature(aiplatform.CustomContainerTrainingJob.run)
        .parameters['dataset']
        .annotation
    )

    assert (
        utils.resolve_annotation(dataset_annotation)
        is aiplatform.datasets.dataset._Dataset
    )

    dataset_annotation = (
        signature(aiplatform.CustomPythonPackageTrainingJob.run)
        .parameters['dataset']
        .annotation
    )

    assert (
        utils.resolve_annotation(dataset_annotation)
        is aiplatform.datasets.dataset._Dataset
    )

# Copyright 2021 Google LLC
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
"""Tests for kfp.dsl.artifact."""

import os
import unittest
import yaml
import json
import jsonschema

from google.protobuf import json_format
from google.protobuf.struct_pb2 import Struct

from kfp.dsl import artifact
from kfp.dsl import ontology_artifacts

from google.protobuf import json_format

_TEST_SIMPLE_INSTANCE_SCHEMA = """\
title: TestArtifact
type: object
properties:
  property1:
    type: number
    format: float
    description: >
      A simple number property.
  property2:
    type: string
    description: >
      A simple string property.
"""

class ArtifactTest(unittest.TestCase):

  def test_setting_metadata(self):
    test_artifact = artifact.Artifact(_TEST_SIMPLE_INSTANCE_SCHEMA)
    test_artifact.property1 = 45.5
    test_artifact.metadata['property2'] = 'test string'
    test_artifact.metadata['custom_property'] = 4
    test_artifact.metadata['custom_arr'] = [2, 4]
    expected_json =  json.loads(
      '{ "property1": 45.5, "property2": "test string", "custom_property": 4, "custom_arr": [2, 4]}')
    self.assertEqual(expected_json, test_artifact.metadata)

  def test_property_type_checking(self):
    test_artifact = artifact.Artifact(_TEST_SIMPLE_INSTANCE_SCHEMA)
    test_artifact.property1 = '45.5'
    with self.assertRaises(RuntimeError):
      test_artifact.serialize()

    test_artifact_2 = artifact.Artifact(_TEST_SIMPLE_INSTANCE_SCHEMA)
    test_artifact_2.metadata['property1'] = '45.4'
    with self.assertRaises(RuntimeError):
      test_artifact_2.serialize()

  def test_complete_artifact(self):
    test_artifact = artifact.Artifact(_TEST_SIMPLE_INSTANCE_SCHEMA)
    test_artifact.property1 = 45.5
    test_artifact.metadata['property2'] = 'test string'
    test_artifact.metadata['custom_property'] = 4
    test_artifact.metadata['custom_arr'] = [2, 4]
    test_artifact.name = 'test_artifact'
    test_artifact.uri = 'gcs://some_bucket/some_object'
    with open(os.path.join(os.path.dirname(__file__),
        'test_data', 'expected_complete_artifact.json')) as json_file:
      expected_artifact_json = json.load(json_file)
      self.assertEqual(expected_artifact_json, json.loads(test_artifact.serialize()))


  def test_model_artifact(self):
    model = ontology_artifacts.Model()
    model.framework = 'TFX'
    model.metadata["framework_version"] = '1.12'
    model.metadata['custom_field'] = {
      'custom_field_1': 1, 'custom_field_2': [1, 3, 4]}
    with open(os.path.join(os.path.dirname(__file__),
        'test_data', 'expected_model_artifact.json')) as json_file:
      expected_model_json = json.load(json_file)
      self.assertEqual(expected_model_json, json.loads(model.serialize()))

  def test_dataset_artifact(self):
    dataset = ontology_artifacts.Dataset()
    dataset.metadata['custom_field'] = 2
    dataset.payload_format = 'JSON'
    dataset.metadata['container_format'] = 'TFRecord'
    with open(os.path.join(os.path.dirname(__file__),
        'test_data', 'expected_dataset_artifact.json')) as json_file:
      expected_dataset_json = json.load(json_file)
      self.assertEqual(expected_dataset_json, json.loads(dataset.serialize()))

  # Test to ensure Artifact class can serialize and deserialize.
  def test_artifact_serialize_deserialize(self):
    with open(os.path.join(os.path.dirname(__file__),
        'test_data', 'expected_complete_artifact.json')) as json_file:
      read_artifact = json_file.read()

    deserialized_artifact = artifact.Artifact.deserialize(read_artifact)
    expected_artifact_json = json.loads(read_artifact)
    self.assertEqual(expected_artifact_json, json.loads(deserialized_artifact.serialize()))

  # Test to ontology Artifact class can serialize and deserialize.
  def test_model_serialize_deserialize(self):
    with open(os.path.join(os.path.dirname(__file__),
        'test_data', 'expected_model_artifact.json')) as json_file:
      read_artifact = json_file.read()

    deserialized_artifact = artifact.Artifact.deserialize(read_artifact)
    expected_artifact_json = json.loads(read_artifact)
    self.assertEqual(expected_artifact_json, json.loads(deserialized_artifact.serialize()))

  def test_metrics(self):
    metrics = ontology_artifacts.Metrics()
    metrics.accuracy = 98.4
    metrics.mean_squared_error = 32.1
    metrics.log_metric('custom_metric1', 24)
    metrics.metadata['custom_metric2'] = 24
    metrics.name = 'test_artifact'

    with open(os.path.join(os.path.dirname(__file__),
        'test_data', 'expected_metrics.json')) as json_file:
      expected_json = json.load(json_file)
      self.assertEqual(expected_json, json.loads(metrics.serialize()))

  def test_classification_metrics(self):
    cls_metrics = ontology_artifacts.ClassificationMetrics()
    cls_metrics.name = "test_me"
    cls_metrics.log_roc_reading(24.3, 24.5, 98.4)
    cls_metrics.set_confusion_matrix_categories(['dog', 'cat', 'horses'])
    cls_metrics.log_confusion_matrix_row('dog', [2, 6, 0])
    cls_metrics.log_confusion_matrix_cell('cat', 'dog', 3)
    cls_metrics.metadata['test'] = 1
    with open(os.path.join(os.path.dirname(__file__),
        'test_data', 'expected_classification_metrics.json')) as json_file:
      expected_json = json.load(json_file)
      self.assertEqual(expected_json, json.loads(cls_metrics.serialize()))

  def test_bulk_load_classification_metrics(self):
    cls_metrics = ontology_artifacts.ClassificationMetrics()
    cls_metrics.load_roc_readings([
      [53.6, 52.6, 85.1],
      [53.6, 52.6, 85.1],
      [53.6, 52.6, 85.1]])
    cls_metrics.load_confusion_matrix(['dog', 'cat', 'horses'], [[2, 6, 0], [3, 5,6], [5,7,8]])
    with open(os.path.join(os.path.dirname(__file__),
        'test_data', 'expected_bulk_loaded_classification_metrics.json')) as json_file:
      expected_json = json.load(json_file)
      self.assertEqual(expected_json, json.loads(cls_metrics.serialize()))

  def test_sliced_classification_metrics(self):
    sliced_metrics = ontology_artifacts.SlicedClassificationMetrics()
    sliced_metrics.name = "test_me"
    sliced_metrics.log_roc_reading('dog', 24.3, 24.5, 98.4)
    sliced_metrics.set_confusion_matrix_categories('overall', ['dog', 'cat', 'horses'])
    sliced_metrics.log_confusion_matrix_row('overall', 'dog', [2, 6, 0])
    sliced_metrics.log_confusion_matrix_cell('overall', 'cat', 'dog', 3)
    sliced_metrics.metadata['test'] = 1
    with open(os.path.join(os.path.dirname(__file__),
        'test_data', 'expected_sliced_classification_metrics.json')) as json_file:
      expected_json = json.load(json_file)
      self.assertEqual(expected_json, json.loads(sliced_metrics.serialize()))

if __name__ == '__main__':
  unittest.main()

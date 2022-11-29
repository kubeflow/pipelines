# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
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
"""Test System Labels Module."""

import json
import os

from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import gcp_labels_util

import unittest

_INVALID__SYSTEM_LABELS = "{key1: \"value1\"}"
_SYSTEM_LABELS = {"key1": "value1", "key2": "value2"}
_EXISTING_LABELS = {"key3": "value3", "key2": "value4"}
_COMBINED_LABELS = {"key1": "value1", "key3": "value3", "key2": "value4"}


class SystemLabelsTests(unittest.TestCase):
  """Tests for System Labels."""

  def test_attach_system_labels_combine_existing_and_system(self):
    os.environ[gcp_labels_util.SYSTEM_LABEL_ENV_VAR] = json.dumps(
        _SYSTEM_LABELS)
    self.assertEqual(
        gcp_labels_util.attach_system_labels(_EXISTING_LABELS),
        _COMBINED_LABELS)

  def test_attach_system_labels_no_existing_returns_system_labels(self):
    os.environ[gcp_labels_util.SYSTEM_LABEL_ENV_VAR] = json.dumps(
        _SYSTEM_LABELS)
    self.assertEqual(gcp_labels_util.attach_system_labels(), _SYSTEM_LABELS)

  def test_attach_system_labels_empty_existing_returns_system_labels(self):
    os.environ[gcp_labels_util.SYSTEM_LABEL_ENV_VAR] = json.dumps(
        _SYSTEM_LABELS)
    self.assertEqual(gcp_labels_util.attach_system_labels({}), _SYSTEM_LABELS)

  def test_attach_system_labels_no_system_labels_returns_existing(self):
    if gcp_labels_util.SYSTEM_LABEL_ENV_VAR in os.environ:
      os.environ.pop(gcp_labels_util.SYSTEM_LABEL_ENV_VAR)
    self.assertEqual(
        gcp_labels_util.attach_system_labels(_EXISTING_LABELS),
        _EXISTING_LABELS)

  def test_attach_system_labels_no_system_labels_returns_none(self):
    if gcp_labels_util.SYSTEM_LABEL_ENV_VAR in os.environ:
      os.environ.pop(gcp_labels_util.SYSTEM_LABEL_ENV_VAR)
    self.assertIsNone(gcp_labels_util.attach_system_labels())

  def test_attach_system_labels_no_system_labels_returns_empty(self):
    if gcp_labels_util.SYSTEM_LABEL_ENV_VAR in os.environ:
      os.environ.pop(gcp_labels_util.SYSTEM_LABEL_ENV_VAR)
    self.assertEqual(gcp_labels_util.attach_system_labels({}), {})

  def test_attach_system_labels_invalid_system_labels_returns_existing(self):
    os.environ[gcp_labels_util.SYSTEM_LABEL_ENV_VAR] = _INVALID__SYSTEM_LABELS
    self.assertEqual(
        gcp_labels_util.attach_system_labels(_EXISTING_LABELS),
        _EXISTING_LABELS)

  def test_attach_system_labels_invalid_system_labels_returns_none(self):
    os.environ[gcp_labels_util.SYSTEM_LABEL_ENV_VAR] = _INVALID__SYSTEM_LABELS
    self.assertIsNone(gcp_labels_util.attach_system_labels())

  def test_attach_system_labels_invalid_system_labels_returns_empty(self):
    os.environ[gcp_labels_util.SYSTEM_LABEL_ENV_VAR] = _INVALID__SYSTEM_LABELS
    self.assertEqual(gcp_labels_util.attach_system_labels({}), {})

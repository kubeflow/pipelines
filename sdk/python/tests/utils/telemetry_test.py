# Copyright 2020 Google LLC
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
"""Tests for kfp.utils.telemetry."""

from kfp.utils import telemetry

import unittest

_TEST_CUSTOM_URI = 'https://raw.githubusercontent.com/my_org/my_repo/master/' \
                   'component.yaml'

_TEST_OOB_COMPONENT_URI = 'https://raw.githubusercontent.com/kubeflow/' \
                          'pipelines/master/components/gcp/dataflow/' \
                          'launch_python/component.yaml'


class TelemetryTest(unittest.TestCase):
  
  def testOobUri(self):
    self.assertEqual(
        telemetry.get_component_name(_TEST_OOB_COMPONENT_URI),
        'dataflow_launch_python'
    )
  
  def testCustomUriBypassing(self):
    self.assertIsNone(telemetry.get_component_name(_TEST_CUSTOM_URI))

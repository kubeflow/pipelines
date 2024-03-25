# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
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
"""Test Vertex AI Cancel Job Launcher Client module."""

import os

from google_cloud_pipeline_components.container.v1.cancel_job import launcher
from google_cloud_pipeline_components.container.v1.cancel_job import remote_runner

import unittest
from unittest import mock


class LauncherCancelJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherCancelJobUtilsTests, self).setUp()
    self._gcp_resources = 'test_gcp_resource'

  @mock.patch.object(remote_runner, 'cancel_job', autospec=True)
  def test_launcher_on_cancel_job_type(self, mock_cancel_job):
    launcher.main(self._gcp_resources)
    mock_cancel_job.assert_called_once_with(
        gcp_resources=self._gcp_resources,
    )

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
"""Test Execution Context module."""

import signal
from unittest import mock
from google_cloud_pipeline_components.container.utils.execution_context import ExecutionContext
import unittest


class ExecutionContextTests(unittest.TestCase):
  "Tests for Execution Context"

  @mock.patch.object(signal, 'signal', autospec=True)
  def test_signal_handler_added(self, mock_signal):
    cancel_handler = mock.Mock()
    context = ExecutionContext(on_cancel=cancel_handler)

    mock_signal.assert_called_once_with(signal.SIGTERM,
                                        context._exit_gracefully)

  def test_exit_gracefully_cancel(self):
    cancel_handler = mock.Mock()
    context = ExecutionContext(on_cancel=cancel_handler)

    context._exit_gracefully()

    cancel_handler.assert_called_once()

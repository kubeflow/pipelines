# Copyright 2023 The Kubeflow Authors
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
"""Utilities for testing local execution."""

import datetime
import unittest
from unittest import mock

from absl.testing import parameterized
from google.protobuf import json_format
from kfp import components
from kfp import dsl
from kfp.local import config as local_config


class LocalRunnerEnvironmentTestCase(parameterized.TestCase):

    def setUp(self):
        from kfp.dsl import pipeline_task
        pipeline_task.TEMPORARILY_BLOCK_LOCAL_EXECUTION = False
        # start each test case without an uninitialized environment
        local_config.LocalExecutionConfig.instance = None

    def tearDown(self) -> None:
        from kfp.dsl import pipeline_task
        pipeline_task.TEMPORARILY_BLOCK_LOCAL_EXECUTION = True


class MockedDatetimeTestCase(unittest.TestCase):

    def setUp(self):
        # set up patch, cleanup, and start
        patcher = mock.patch('kfp.local.executor_input_utils.datetime.datetime')
        self.addCleanup(patcher.stop)
        self.mock_datetime = patcher.start()

        # set mock return values
        mock_now = mock.MagicMock(
            wraps=datetime.datetime(2023, 10, 10, 13, 32, 59, 420710))
        self.mock_datetime.now.return_value = mock_now
        mock_now.strftime.return_value = '2023-10-10-13-32-59-420710'


def compile_and_load_component(
    base_component: dsl.base_component.BaseComponent,
) -> dsl.yaml_component.YamlComponent:
    """Compiles a component to PipelineSpec and reloads it as a
    YamlComponent."""
    return components.load_component_from_text(
        json_format.MessageToJson(base_component.pipeline_spec))

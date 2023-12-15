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

import contextlib
import datetime
import functools
import os
import pathlib
import shutil
import tempfile
from typing import Iterator
import unittest
from unittest import mock

from absl.testing import parameterized
from google.protobuf import json_format
from kfp import components
from kfp import dsl
from kfp.local import config as local_config

_LOCAL_KFP_PACKAGE_PATH = os.path.join(
    os.path.dirname(__file__),
    os.path.pardir,
    os.path.pardir,
)


class LocalRunnerEnvironmentTestCase(parameterized.TestCase):
    """Test class that uses an isolated filesystem and updates the
    dsl.component decorator to install from the local KFP source, rather than
    the latest release."""

    def setUp(self):
        # start each test case without an uninitialized environment
        local_config.LocalExecutionConfig.instance = None
        with contextlib.ExitStack() as stack:
            stack.enter_context(isolated_filesystem())
            self._working_dir = pathlib.Path.cwd()
            self.addCleanup(stack.pop_all().close)

    @classmethod
    def setUpClass(cls):
        from kfp.dsl import pipeline_task
        pipeline_task.TEMPORARILY_BLOCK_LOCAL_EXECUTION = False
        cls.original_component, dsl.component = dsl.component, functools.partial(
            dsl.component, kfp_package_path=_LOCAL_KFP_PACKAGE_PATH)

    @classmethod
    def tearDownClass(cls):
        from kfp.dsl import pipeline_task
        pipeline_task.TEMPORARILY_BLOCK_LOCAL_EXECUTION = True
        dsl.component = cls.original_component


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


@contextlib.contextmanager
def isolated_filesystem() -> Iterator[str]:
    cwd = os.getcwd()
    dt = tempfile.mkdtemp()
    os.chdir(dt)

    try:
        yield dt
    finally:
        os.chdir(cwd)

        with contextlib.suppress(OSError):
            shutil.rmtree(dt)

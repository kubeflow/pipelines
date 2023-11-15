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
import inspect
import os
import pathlib
import shutil
import tempfile
from types import ModuleType
from typing import Iterator, Optional, Union
import unittest
from unittest import mock

from absl.testing import parameterized
from google.protobuf import json_format
from google.protobuf import message
from kfp import components
from kfp import dsl
from kfp.dsl import base_component
from kfp.local import config

_LOCAL_KFP_PACKAGE_PATH = os.path.join(
    os.path.dirname(__file__),
    os.path.pardir,
    os.path.pardir,
)


def make_dsl_patch(new_dsl: ModuleType) -> mock.patch:
    """Patches the DSL module where it is used.

    Pushing the complexity here simplifies the tests where this is used.
    """
    caller_frame = inspect.currentframe().f_back
    caller_module = inspect.getmodule(caller_frame)

    return mock.patch(f'{caller_module.__name__}.dsl', new=new_dsl)


def make_local_kfp_install_patch() -> mock.patch:
    """Use the current local version of KFP to run tests.

    - Ensures the executor code at HEAD of the branch is under test
    - Prevents installing the latest release, rather than the current changes, and using that source for all subsequent tests (this is possible when SubprocessRunner(use_venv=False))
    - Allows us to bump the version number for a release PR and the tests should still pass
    """
    new_dsl = functools.partial(
        dsl.component, kfp_package_path=_LOCAL_KFP_PACKAGE_PATH)
    return make_dsl_patch(new_dsl)


class LocalRunnerEnvironmentTestCase(parameterized.TestCase):
    """Test class that (a) uses an isolated filesystem, (b) updates the
    dsl.component decorator to install from the local KFP source, rather than
    the latest release, and (c) resets the local runner config to force
    stronger independence between tests."""

    def setUp(self):
        with contextlib.ExitStack() as stack:
            stack.enter_context(isolated_filesystem())
            self._working_dir = pathlib.Path.cwd()
            self.addCleanup(stack.pop_all().close)

    def tearDown(self):
        config.LocalExecutionConfig.instance = None

    @classmethod
    def setUpClass(cls):
        cls.dsl_patch = make_local_kfp_install_patch()
        cls.dsl_patch.start()
        base_component.TEMPORARILY_BLOCK_LOCAL_EXECUTION = False

    @classmethod
    def tearDownClass(cls):
        cls.dsl_patch.stop()


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


def write_proto_to_json_file(
    proto_message: message.Message,
    file_path: str,
) -> None:
    """Writes proto_message to file_path as JSON."""
    json_string = json_format.MessageToJson(proto_message)

    with open(file_path, 'w') as json_file:
        json_file.write(json_string)


def compile_and_load_component(
    base_component: dsl.base_component.BaseComponent,
) -> dsl.yaml_component.YamlComponent:
    """Compiles a component to PipelineSpec and reloads it as a
    YamlComponent."""
    return components.load_component_from_text(
        json_format.MessageToJson(base_component.pipeline_spec))


@contextlib.contextmanager
def isolated_filesystem(
        temp_dir: Optional[Union[str, os.PathLike]] = None) -> Iterator[str]:
    cwd = os.getcwd()
    dt = tempfile.mkdtemp(dir=temp_dir)
    os.chdir(dt)

    try:
        yield dt
    finally:
        os.chdir(cwd)

        if temp_dir is None:
            with contextlib.suppress(OSError):
                shutil.rmtree(dt)

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
import functools
import os
import pathlib
import tempfile
from typing import Any, Callable, Dict
import unittest
from unittest import mock

from absl.testing import parameterized
from google.protobuf import json_format
from google.protobuf import message
from kfp import components
from kfp import dsl
from kfp.local import config as local_config
from kfp.local import docker_task_handler

_LOCAL_KFP_PACKAGE_PATH = os.path.join(
    os.path.dirname(__file__),
    os.path.pardir,
    os.path.pardir,
)


def modify_volumes_decorator(
        original_method: Callable[..., Any]) -> Callable[..., Any]:

    def wrapper(self, *args, **kwargs) -> Dict[str, Any]:
        original_volumes = original_method(self, *args, **kwargs)
        LOCAL_KFP_VOLUME = {
            _LOCAL_KFP_PACKAGE_PATH: {
                'bind': _LOCAL_KFP_PACKAGE_PATH,
                'mode': 'rw'
            }
        }
        original_volumes.update(LOCAL_KFP_VOLUME)
        return original_volumes

    return wrapper


class LocalRunnerEnvironmentTestCase(parameterized.TestCase):
    """Test class that uses an isolated filesystem and updates the
    dsl.component decorator to install from the local KFP source, rather than
    the latest release."""

    def setUp(self):
        # ENTER: start each test case without an uninitialized environment
        local_config.LocalExecutionConfig.instance = None

        # ENTER: use tempdir for all tests
        self.working_dir = pathlib.Path.cwd()
        self.temp_dir = tempfile.TemporaryDirectory()
        os.chdir(self.temp_dir.name)

        # ENTER: mount KFP dir to enable install from source for docker runner
        self.original_get_volumes_to_mount = docker_task_handler.DockerTaskHandler.get_volumes_to_mount
        docker_task_handler.DockerTaskHandler.get_volumes_to_mount = modify_volumes_decorator(
            docker_task_handler.DockerTaskHandler.get_volumes_to_mount)

    def tearDown(self):
        # EXIT: use tempdir for all tests
        self.temp_dir.cleanup()
        os.chdir(self.working_dir)

        # EXIT: mount KFP dir to enable install from source for docker runner
        docker_task_handler.DockerTaskHandler.get_volumes_to_mount = self.original_get_volumes_to_mount

    @classmethod
    def setUpClass(cls):
        # ENTER: use local KFP package path for subprocess runner
        cls.original_component, dsl.component = dsl.component, functools.partial(
            dsl.component, kfp_package_path=_LOCAL_KFP_PACKAGE_PATH)

    @classmethod
    def tearDownClass(cls):
        # EXIT: use local KFP package path for subprocess runner
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

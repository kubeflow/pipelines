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
import shutil
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


def create_modify_volumes_decorator(isolated_kfp_path: str):
    """Creates a volume decorator for a specific isolated KFP path."""

    def modify_volumes_decorator(
            original_method: Callable[..., Any]) -> Callable[..., Any]:

        def wrapper(self, *args, **kwargs) -> Dict[str, Any]:
            original_volumes = original_method(self, *args, **kwargs)
            LOCAL_KFP_VOLUME = {
                isolated_kfp_path: {
                    'bind': isolated_kfp_path,
                    'mode': 'rw'
                }
            }
            original_volumes.update(LOCAL_KFP_VOLUME)
            return original_volumes

        return wrapper

    return modify_volumes_decorator


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

        # ENTER: create isolated KFP source copy for this test to prevent build conflicts
        self.isolated_kfp_dir = tempfile.mkdtemp(prefix='kfp_test_')
        shutil.copytree(
            _LOCAL_KFP_PACKAGE_PATH,
            os.path.join(self.isolated_kfp_dir, 'kfp_source'),
            ignore=shutil.ignore_patterns('*.pyc', '__pycache__', '.git',
                                          'build', 'dist', '*.egg-info'))
        self.isolated_kfp_package_path = os.path.join(self.isolated_kfp_dir,
                                                      'kfp_source')

        # ENTER: use isolated KFP package path for this test
        self.original_component, dsl.component = dsl.component, functools.partial(
            dsl.component, kfp_package_path=self.isolated_kfp_package_path)

        # ENTER: mount isolated KFP dir to enable install from source for docker runner
        self.original_get_volumes_to_mount = docker_task_handler.DockerTaskHandler.get_volumes_to_mount
        modify_volumes_decorator = create_modify_volumes_decorator(
            self.isolated_kfp_package_path)
        docker_task_handler.DockerTaskHandler.get_volumes_to_mount = modify_volumes_decorator(
            docker_task_handler.DockerTaskHandler.get_volumes_to_mount)

    def tearDown(self):
        # EXIT: restore original component decorator
        dsl.component = self.original_component

        # EXIT: clean up isolated KFP source directory
        shutil.rmtree(self.isolated_kfp_dir, ignore_errors=True)

        # EXIT: use tempdir for all tests
        self.temp_dir.cleanup()
        os.chdir(self.working_dir)

        # EXIT: mount KFP dir to enable install from source for docker runner
        docker_task_handler.DockerTaskHandler.get_volumes_to_mount = self.original_get_volumes_to_mount


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

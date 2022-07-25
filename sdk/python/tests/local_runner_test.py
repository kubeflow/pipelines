# Copyright 2021 The Kubeflow Authors
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

from typing import Callable
import unittest

from kfp.deprecated import LocalClient
from kfp.deprecated import run_pipeline_func_locally
import kfp.deprecated as kfp

InputPath = kfp.components.InputPath()
OutputPath = kfp.components.OutputPath()

BASE_IMAGE = "python:3.7"


def light_component(base_image: str = BASE_IMAGE,):
    """Decorator of kfp light component with customized parameters.

    Usage:
    ```python
    @light_component(base_image="python:3.7")
    def a_component(src: kfp.components.InputPath(), ...):
        ...
    ```
    """

    def wrapper(func: Callable):
        return kfp.components.create_component_from_func(
            func=func,
            base_image=base_image,
        )

    return wrapper


@light_component()
def hello(name: str):
    print(f"hello {name}")


@light_component()
def local_loader(src: str, dst: kfp.components.OutputPath()):
    import os
    import shutil

    if os.path.exists(src):
        shutil.copyfile(src, dst)


@light_component()
def flip_coin(dst: kfp.components.OutputPath()):
    import random

    result = "head" if random.randint(0, 1) == 0 else "tail"
    with open(dst, "w") as f:
        f.write(result)


@light_component()
def list(dst: kfp.components.OutputPath()):
    import json

    with open(dst, "w") as f:
        json.dump(["hello", "world", "kfp"], f)


@light_component()
def component_connect_demo(src: kfp.components.InputPath(),
                           dst: kfp.components.OutputPath()):
    with open(src, "r") as f:
        line = f.readline()
        print(f"read first line: {line}")
        with open(dst, "w") as fw:
            fw.write(f"{line} copied")


class LocalRunnerTest(unittest.TestCase):

    def setUp(self):
        import tempfile

        with tempfile.NamedTemporaryFile('w', delete=False) as f:
            self.temp_file_path = f.name
            f.write("hello world")

    def test_run_local(self):

        def _pipeline(name: str):
            hello(name)

        run_pipeline_func_locally(
            _pipeline,
            {"name": "world"},
            execution_mode=LocalClient.ExecutionMode("local"),
        )

    def test_local_file(self):

        def _pipeline(file_path: str):
            local_loader(file_path)

        run_result = run_pipeline_func_locally(
            _pipeline,
            {"file_path": self.temp_file_path},
            execution_mode=LocalClient.ExecutionMode("local"),
        )
        output_file_path = run_result.get_output_file("local-loader")

        with open(output_file_path, "r") as f:
            line = f.readline()
            assert "hello" in line

    def test_condition(self):

        def _pipeline():
            _flip = flip_coin()
            with kfp.dsl.Condition(_flip.output == "head"):
                hello("head")

            with kfp.dsl.Condition(_flip.output == "tail"):
                hello("tail")

        run_pipeline_func_locally(
            _pipeline, {}, execution_mode=LocalClient.ExecutionMode("local"))

    def test_for(self):

        @light_component()
        def cat(item, dst: OutputPath):
            with open(dst, "w") as f:
                f.write(item)

        def _pipeline():
            with kfp.dsl.ParallelFor(list().output) as item:
                cat(item)

        run_pipeline_func_locally(
            _pipeline, {}, execution_mode=LocalClient.ExecutionMode("local"))

    def test_connect(self):

        def _pipeline():
            _local_loader = local_loader(self.temp_file_path)
            component_connect_demo(_local_loader.output)

        run_result = run_pipeline_func_locally(
            _pipeline, {}, execution_mode=LocalClient.ExecutionMode("local"))
        output_file_path = run_result.get_output_file("component-connect-demo")

        with open(output_file_path, "r") as f:
            line = f.readline()
            assert "copied" in line

    def test_command_argument_in_any_format(self):

        def echo():
            return kfp.dsl.ContainerOp(
                name="echo",
                image=BASE_IMAGE,
                command=[
                    "echo", "hello world", ">", "/tmp/outputs/output_file"
                ],
                arguments=[],
                file_outputs={"output": "/tmp/outputs/output_file"},
            )

        def _pipeline():
            _echo = echo()
            component_connect_demo(_echo.output)

        run_pipeline_func_locally(
            _pipeline, {}, execution_mode=LocalClient.ExecutionMode("local"))

    @unittest.skip('docker is not installed in CI environment.')
    def test_execution_mode_exclude_op(self):

        @light_component(base_image="image_not_exist")
        def cat_on_image_not_exist(name: str, dst: OutputPath):
            with open(dst, "w") as f:
                f.write(name)

        def _pipeline():
            cat_on_image_not_exist("exclude ops")

        run_result = run_pipeline_func_locally(
            _pipeline,
            {},
            execution_mode=LocalClient.ExecutionMode(mode="docker"),
        )
        output_file_path = run_result.get_output_file("cat-on-image-not-exist")
        import os

        assert not os.path.exists(output_file_path)

        run_result = run_pipeline_func_locally(
            _pipeline,
            {},
            execution_mode=LocalClient.ExecutionMode(
                mode="docker", ops_to_exclude=["cat-on-image-not-exist"]),
        )
        output_file_path = run_result.get_output_file("cat-on-image-not-exist")

        with open(output_file_path, "r") as f:
            line = f.readline()
            assert "exclude ops" in line

    @unittest.skip('docker is not installed in CI environment.')
    def test_docker_options(self):

        @light_component()
        def check_option(dst: OutputPath):
            import os
            with open(dst, "w") as f:
                f.write(os.environ["foo"])

        def _pipeline():
            check_option()

        run_result = run_pipeline_func_locally(
            _pipeline, {},
            execution_mode=LocalClient.ExecutionMode(
                mode="docker", docker_options=["-e", "foo=bar"]))
        assert run_result.success
        output_file_path = run_result.get_output_file("check-option")

        with open(output_file_path, "r") as f:
            line = f.readline()
            assert "bar" in line

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
"""Tests for subprocess_local_task_handler.py."""
import contextlib
import io
import unittest

from absl.testing import parameterized
from kfp import dsl
from kfp import local
from kfp.local import subprocess_task_handler
from kfp.local import testing_utilities


class TestSubprocessRunner(testing_utilities.LocalRunnerEnvironmentTestCase):

    def test_basic(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def comp():
            print('foobar!')

        buffer = io.StringIO()

        with contextlib.redirect_stdout(buffer):
            comp()

        output = buffer.getvalue().strip()

        self.assertContainsSubsequence(output, 'foobar!')

    def test_image_warning(self):
        with self.assertWarnsRegex(
                RuntimeWarning,
                r"You may be attemping to run a task that uses custom or non-Python base image 'my_custom_image' in a Python environment\. This may result in incorrect dependencies and/or incorrect behavior\."
        ):
            subprocess_task_handler.SubprocessTaskHandler(
                image='my_custom_image',
                # avoid catching the Container Component and
                # Containerized Python Component validation errors
                full_command=['kfp.dsl.executor_main'],
                pipeline_root='pipeline_root',
                runner=local.SubprocessRunner(use_venv=True),
            )

    def test_cannot_run_container_component(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.container_component
        def comp():
            return dsl.ContainerSpec(
                image='alpine',
                command=['echo'],
                args=['foo'],
            )

        with self.assertRaisesRegex(
                RuntimeError,
                r'The SubprocessRunner only supports running Lightweight Python Components\. You are attempting to run a Container Component\.',
        ):
            comp()

    def test_cannot_run_containerized_python_component(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component(target_image='foo')
        def comp():
            pass

        with self.assertRaisesRegex(
                RuntimeError,
                r'The SubprocessRunner only supports running Lightweight Python Components\. You are attempting to run a Containerized Python Component\.',
        ):
            comp()


class TestRunLocalSubproces(unittest.TestCase):

    def test_simple_program(self):
        buffer = io.StringIO()

        with contextlib.redirect_stdout(buffer):
            subprocess_task_handler.run_local_subprocess([
                'echo',
                'foo!',
            ])

        output = buffer.getvalue().strip()

        self.assertEqual(output, 'foo!')


class TestUseCurrentPythonExecutable(
        testing_utilities.LocalRunnerEnvironmentTestCase):

    def test(self):
        full_command = ['python3 -c "from kfp import dsl"']
        actual = subprocess_task_handler.replace_python_executable(
            full_command=full_command,
            new_executable='/foo/bar/python3',
        )
        expected = ['/foo/bar/python3 -c "from kfp import dsl"']
        self.assertEqual(actual, expected)


class TestUseVenv(testing_utilities.LocalRunnerEnvironmentTestCase):

    @parameterized.parameters([
        ({
            'runner': local.SubprocessRunner(use_venv=True),
        }),
        ({
            'runner': local.SubprocessRunner(use_venv=True),
        }),
    ])
    def test_use_venv_true(self, **kwargs):
        local.init(**kwargs)

        @dsl.component(packages_to_install=['cloudpickle'])
        def installer_component():
            import cloudpickle
            print('Cloudpickle is installed:', cloudpickle)

        installer_component()

        # since the module was installed in the virtual environment, it should not exist in the current environment
        with self.assertRaisesRegex(ModuleNotFoundError,
                                    r"No module named 'cloudpickle'"):
            import cloudpickle


if __name__ == '__main__':
    unittest.main()

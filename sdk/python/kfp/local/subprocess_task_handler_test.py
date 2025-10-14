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
import functools
import io
import os
from typing import NamedTuple, Optional
import unittest
from unittest import mock

from absl.testing import parameterized
from kfp import dsl
from kfp import local
from kfp.dsl import Artifact
from kfp.dsl import Dataset
from kfp.dsl import Output
from kfp.local import subprocess_task_handler
from kfp.local import testing_utilities
import pytest

# NOTE: When testing SubprocessRunner, use_venv=True throughout to avoid
# modifying current code under test.
# If the dsl.component mocks are modified in a way that makes them not work,
# the code may install kfp from PyPI rather from source. To mitigate the
# impact of such an error we should not install into the main test process'
# environment.

root_dir = os.path.dirname(
    os.path.dirname(
        os.path.dirname(os.path.dirname(os.path.dirname(__file__)))))
kfp_pipeline_spec_path = os.path.join(root_dir, 'api', 'v2alpha1', 'python')


@pytest.fixture(autouse=True)
def set_packages_for_test_classes(monkeypatch, request):
    if request.cls.__name__ in {
            'TestSubprocessRunner', 'TestRunLocalSubproces',
            'TestUseCurrentPythonExecutable', 'TestUseVenv',
            'TestLightweightPythonComponentLogic'
    }:
        original_dsl_component = dsl.component
        monkeypatch.setattr(
            dsl, 'component',
            functools.partial(
                original_dsl_component,
                packages_to_install=[kfp_pipeline_spec_path]))


class TestSubprocessRunner(testing_utilities.LocalRunnerEnvironmentTestCase):

    @mock.patch('sys.stdout', new_callable=io.StringIO)
    def test_basic(self, mock_stdout):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def comp():
            print('foobar!')

        comp()

        output = mock_stdout.getvalue().strip()

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

        @dsl.component(target_image='foo', packages_to_install=[])
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

        @dsl.component(
            packages_to_install=[kfp_pipeline_spec_path, 'cloudpickle'])
        def installer_component():
            import cloudpickle
            print('Cloudpickle is installed:', cloudpickle)

        installer_component()

        # since the module was installed in the virtual environment, it should not exist in the current environment
        with self.assertRaisesRegex(ModuleNotFoundError,
                                    r"No module named 'cloudpickle'"):
            import cloudpickle


class TestLightweightPythonComponentLogic(
        testing_utilities.LocalRunnerEnvironmentTestCase):

    def test_single_output_simple_case(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(x: str) -> str:
            return x

        actual = identity(x='hello').output
        expected = 'hello'
        self.assertIsInstance(actual, str)
        self.assertEqual(actual, expected)

    def test_many_primitives_in_and_out(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(
            string: str,
            integer: int,
            decimal: float,
            boolean: bool,
            l: list,
            d: dict,
        ) -> NamedTuple(
                'Outputs',
                string=str,
                integer=int,
                decimal=float,
                boolean=bool,
                l=list,
                d=dict):
            Outputs = NamedTuple(
                'Outputs',
                string=str,
                integer=int,
                decimal=float,
                boolean=bool,
                l=list,
                d=dict)
            return Outputs(
                string=string,
                integer=integer,
                decimal=decimal,
                boolean=boolean,
                l=l,
                d=d,
            )

        task = identity(
            string='foo',
            integer=1,
            decimal=3.14,
            boolean=True,
            l=['a', 'b'],
            d={'x': 'y'})
        self.assertIsInstance(task.outputs['string'], str)
        self.assertEqual(task.outputs['string'], 'foo')

        self.assertIsInstance(task.outputs['integer'], int)
        self.assertEqual(task.outputs['integer'], 1)

        self.assertIsInstance(task.outputs['decimal'], float)
        self.assertEqual(task.outputs['decimal'], 3.14)

        self.assertIsInstance(task.outputs['boolean'], bool)
        self.assertTrue(task.outputs['boolean'])

        self.assertIsInstance(task.outputs['l'], list)
        self.assertEqual(task.outputs['l'], ['a', 'b'])

        self.assertIsInstance(task.outputs['d'], dict)
        self.assertEqual(task.outputs['d'], {'x': 'y'})

    def test_single_output_not_available(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))
        from typing import NamedTuple

        @dsl.component
        def return_twice(x: str) -> NamedTuple('Outputs', x=str, y=str):
            Outputs = NamedTuple('Output', x=str, y=str)
            return Outputs(x=x, y=x)

        local_task = return_twice(x='foo')
        with self.assertRaisesRegex(
                AttributeError,
                r'The task has multiple outputs\. Please reference the output by its name\.'
        ):
            local_task.output

    def test_single_artifact_output_traditional(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def artifact_maker(x: str, a: Output[Artifact]):
            with open(a.path, 'w') as f:
                f.write(x)

            a.metadata['foo'] = 'bar'

        actual = artifact_maker(x='hello').output
        self.assertIsInstance(actual, Artifact)
        self.assertEqual(actual.name, 'a')
        self.assertTrue(actual.uri.endswith('/a'))
        self.assertEqual(actual.metadata, {'foo': 'bar'})
        with open(actual.path) as f:
            contents = f.read()
            self.assertEqual(contents, 'hello')

    def test_single_artifact_output_pythonic(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def artifact_maker(x: str) -> Artifact:
            artifact = Artifact(
                name='a', uri=dsl.get_uri('a'), metadata={'foo': 'bar'})
            with open(artifact.path, 'w') as f:
                f.write(x)

            return artifact

        actual = artifact_maker(x='hello').output
        self.assertIsInstance(actual, Artifact)
        self.assertEqual(actual.name, 'a')
        self.assertTrue(actual.uri.endswith('/a'))
        self.assertEqual(actual.metadata, {'foo': 'bar'})
        with open(actual.path) as f:
            contents = f.read()
            self.assertEqual(contents, 'hello')

    def test_multiple_artifact_outputs_traditional(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def double_artifact_maker(
            x: str,
            y: str,
            a: Output[Artifact],
            b: Output[Dataset],
        ):
            with open(a.path, 'w') as f:
                f.write(x)

            with open(b.path, 'w') as f:
                f.write(y)

            a.metadata['foo'] = 'bar'
            b.metadata['baz'] = 'bat'

        local_task = double_artifact_maker(x='hello', y='goodbye')

        actual_a = local_task.outputs['a']
        actual_b = local_task.outputs['b']

        self.assertIsInstance(actual_a, Artifact)
        self.assertEqual(actual_a.name, 'a')
        self.assertTrue(actual_a.uri.endswith('/a'))
        with open(actual_a.path) as f:
            contents = f.read()
            self.assertEqual(contents, 'hello')
        self.assertEqual(actual_a.metadata, {'foo': 'bar'})

        self.assertIsInstance(actual_b, Dataset)
        self.assertEqual(actual_b.name, 'b')
        self.assertTrue(actual_b.uri.endswith('/b'))
        self.assertEqual(actual_b.metadata, {'baz': 'bat'})
        with open(actual_b.path) as f:
            contents = f.read()
            self.assertEqual(contents, 'goodbye')

    def test_multiple_artifact_outputs_pythonic(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def double_artifact_maker(
            x: str,
            y: str,
        ) -> NamedTuple(
                'Outputs', a=Artifact, b=Dataset):
            a = Artifact(
                name='a', uri=dsl.get_uri('a'), metadata={'foo': 'bar'})
            b = Dataset(name='b', uri=dsl.get_uri('b'), metadata={'baz': 'bat'})

            with open(a.path, 'w') as f:
                f.write(x)

            with open(b.path, 'w') as f:
                f.write(y)

            Outputs = NamedTuple('Outputs', a=Artifact, b=Dataset)
            return Outputs(a=a, b=b)

        local_task = double_artifact_maker(x='hello', y='goodbye')

        actual_a = local_task.outputs['a']
        actual_b = local_task.outputs['b']

        self.assertIsInstance(actual_a, Artifact)
        self.assertEqual(actual_a.name, 'a')
        self.assertTrue(actual_a.uri.endswith('/a'))
        with open(actual_a.path) as f:
            contents = f.read()
            self.assertEqual(contents, 'hello')
        self.assertEqual(actual_a.metadata, {'foo': 'bar'})

        self.assertIsInstance(actual_b, Dataset)
        self.assertEqual(actual_b.name, 'b')
        self.assertTrue(actual_b.uri.endswith('/b'))
        with open(actual_b.path) as f:
            contents = f.read()
            self.assertEqual(contents, 'goodbye')
        self.assertEqual(actual_b.metadata, {'baz': 'bat'})

    def test_str_input_uses_default(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(x: str = 'hi') -> str:
            return x

        actual = identity().output
        expected = 'hi'
        self.assertIsInstance(actual, str)
        self.assertEqual(actual, expected)

    def test_placeholder_default_resolved(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(x: str = dsl.PIPELINE_TASK_NAME_PLACEHOLDER) -> str:
            return x

        actual = identity().output
        expected = 'identity'
        self.assertIsInstance(actual, str)
        self.assertEqual(actual, expected)

    def test_outputpath(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def my_comp(out_param: dsl.OutputPath(str),) -> int:
            with open(out_param, 'w') as f:
                f.write('Hello' * 2)
            return 1

        task = my_comp()

        self.assertEqual(task.outputs['out_param'], 'HelloHello')
        self.assertEqual(task.outputs['Output'], 1)

    def test_outputpath_result_not_written(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        # use dsl.OutputPath(int) for more thorough testing
        # want to ensure that the code that converts protobuf number to
        # Python int permits unwritten outputs
        @dsl.component
        def my_comp(out_param: dsl.OutputPath(int)):
            pass

        task = my_comp()
        self.assertEmpty(task.outputs)

    def test_optional_param(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def my_comp(string: Optional[str] = None) -> str:
            return 'is none' if string is None else 'not none'

        task = my_comp()
        self.assertEqual(task.output, 'is none')


if __name__ == '__main__':
    unittest.main()

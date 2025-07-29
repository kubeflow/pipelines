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
"""Tests for task_dispatcher.py.

The difference between these tests and the E2E test are that E2E tests
focus on how the runner should behave to be local execution conformant,
whereas these tests focus on how the task dispatcher should behave,
irrespective of the runner. While there will inevitably some overlap, we
should seek to minimize it.
"""
import functools
import io
import os
import re
import unittest
from unittest import mock

from absl.testing import parameterized
from kfp import dsl
from kfp import local
from kfp.dsl import Artifact
from kfp.dsl import Model
from kfp.dsl import Output
from kfp.local import testing_utilities
import pytest

# NOTE: uses SubprocessRunner throughout to test the taks dispatcher behavior
# NOTE: When testing SubprocessRunner, use_venv=True throughout to avoid
# modifying current code under test.
# If the dsl.component mocks are modified in a way that makes them not work,
# the code may install kfp from PyPI rather from source. To mitigate the
# impact of such an error we should not install into the main test process'
# environment.


@pytest.fixture(autouse=True)
def set_packages_for_test_classes(monkeypatch, request):
    if request.cls.__name__ in {
            'TestLocalExecutionValidation', 'TestSupportOfComponentTypes',
            'TestSupportOfComponentTypes', 'TestExceptionHandlingAndLogging',
            'TestPipelineRootPaths'
    }:
        root_dir = os.path.dirname(
            os.path.dirname(
                os.path.dirname(os.path.dirname(os.path.dirname(__file__)))))
        kfp_pipeline_spec_path = os.path.join(root_dir, 'api', 'v2alpha1',
                                              'python')
        original_dsl_component = dsl.component
        monkeypatch.setattr(
            dsl, 'component',
            functools.partial(
                original_dsl_component,
                packages_to_install=[kfp_pipeline_spec_path]))


class TestLocalExecutionValidation(
        testing_utilities.LocalRunnerEnvironmentTestCase):

    def test_env_not_initialized(self):

        @dsl.component
        def identity(x: str) -> str:
            return x

        with self.assertRaisesRegex(
                RuntimeError,
                r"Local environment not initialized\. Please run 'kfp\.local\.init\(\)' before executing tasks locally\."
        ):
            identity(x='foo')


class TestArgumentValidation(parameterized.TestCase):

    def test_no_argument_no_default(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(x: str) -> str:
            return x

        with self.assertRaisesRegex(
                TypeError, r'identity\(\) missing 1 required argument: x'):

            identity()

    def test_default_wrong_type(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(x: str) -> str:
            return x

        with self.assertRaisesRegex(
                dsl.types.type_utils.InconsistentTypeException,
                r"Incompatible argument passed to the input 'x' of component 'identity': Argument type 'NUMBER_INTEGER' is incompatible with the input type 'STRING'"
        ):
            identity(x=1)

    def test_extra_argument(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(x: str) -> str:
            return x

        with self.assertRaisesRegex(
                TypeError,
                r'identity\(\) got an unexpected keyword argument "y"\.'):
            identity(x='foo', y='bar')

    def test_input_artifact_provided(self):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def artifact_identity(a: Artifact) -> Artifact:
            return a

        with self.assertRaisesRegex(
                ValueError,
                r"Input artifacts are not supported. Got input artifact of type 'Artifact'."
        ):
            artifact_identity(a=Artifact(name='a', uri='gs://bucket/foo'))


class TestSupportOfComponentTypes(
        testing_utilities.LocalRunnerEnvironmentTestCase):

    def test_can_run_loaded_component(self):
        # use venv to avoid installing non-local KFP into test process
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(x: str) -> str:
            return x

        loaded_identity = testing_utilities.compile_and_load_component(identity)

        actual = loaded_identity(x='hello').output
        expected = 'hello'
        # since == is overloaded for dsl.Condition, if local execution is not
        # "hit", then actual will be a channel and actual == expected evaluates
        # to ConditionOperation. Since ConditionOperation is truthy,
        # this may result in a false negative test result. For this reason,
        # we perform an isinstance check first.
        self.assertIsInstance(actual, str)
        self.assertEqual(actual, expected)


class TestExceptionHandlingAndLogging(
        testing_utilities.LocalRunnerEnvironmentTestCase):

    @mock.patch('sys.stdout', new_callable=io.StringIO)
    def test_user_code_throws_exception_if_raise_on_error(self, mock_stdout):
        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            raise_on_error=True,
        )

        @dsl.component
        def fail_comp():
            raise Exception('String to match on')

        with self.assertRaisesRegex(
                RuntimeError,
                r"Task \x1b\[96m'fail-comp'\x1b\[0m finished with status \x1b\[91mFAILURE\x1b\[0m",
        ):
            fail_comp()

        self.assertIn(
            'Exception: String to match on',
            mock_stdout.getvalue(),
        )

    @mock.patch('sys.stdout', new_callable=io.StringIO)
    def test_user_code_no_exception_if_not_raise_on_error(self, mock_stdout):
        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            raise_on_error=False,
        )

        @dsl.component
        def fail_comp():
            raise Exception('String to match on')

        task = fail_comp()
        self.assertDictEqual(task.outputs, {})

        self.assertRegex(
            mock_stdout.getvalue(),
            r"\d+:\d+:\d+\.\d+ - ERROR - Task \x1b\[96m'fail-comp'\x1b\[0m finished with status \x1b\[91mFAILURE\x1b\[0m",
        )
        self.assertIn(
            'Exception: String to match on',
            mock_stdout.getvalue(),
        )

    @mock.patch('sys.stdout', new_callable=io.StringIO)
    def test_all_logs(self, mock_stdout):
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def many_type_component(
            num: int,
            model: Output[Model],
        ) -> str:
            print('Inside of my component!')
            model.metadata['foo'] = 'bar'
            return 'hello' * num

        many_type_component(num=2)

        # inner process logs correctly nested inside outer process logs
        outer_log_regex_sections = [
            r"\d+:\d+:\d+\.\d+ - INFO - Executing task \x1b\[96m'many-type-component'\x1b\[0m\n",
            r'\d+:\d+:\d+\.\d+ - INFO - Streamed logs:\n\n',
            r'.*',
            r'Looking for component ',
            r'.*',
            r'Loading KFP component ',
            r'.*',
            r'Got executor_input:',
            r'.*',
            r'Inside of my component!',
            r'.*',
            r'Wrote executor output file to',
            r'.*',
            r"\d+:\d+:\d+\.\d+ - INFO - Task \x1b\[96m'many-type-component'\x1b\[0m finished with status \x1b\[92mSUCCESS\x1b\[0m\n",
            r"\d+:\d+:\d+\.\d+ - INFO - Task \x1b\[96m'many-type-component'\x1b\[0m outputs:\n    Output: 'hellohello'\n    model: Model\( name='model',\n                  uri='[a-zA-Z0-9/_\.-]+/local_outputs/many-type-component-\d+-\d+-\d+-\d+-\d+-\d+-\d+/many-type-component/model',\n                  metadata={'foo': 'bar'} \)\n",
        ]

        self.assertRegex(
            mock_stdout.getvalue(),
            # use dotall os that .* include newline characters
            re.compile(''.join(outer_log_regex_sections), re.DOTALL),
        )


class TestPipelineRootPaths(testing_utilities.LocalRunnerEnvironmentTestCase):

    def test_relpath(self):
        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            pipeline_root='relpath_root')

        # define in test to force install from source
        @dsl.component
        def identity(x: str) -> str:
            return x

        task = identity(x='foo')
        self.assertIsInstance(task.output, str)
        self.assertEqual(task.output, 'foo')

    def test_abspath(self):
        import tempfile
        with tempfile.TemporaryDirectory() as tmpdir:
            local.init(
                runner=local.SubprocessRunner(use_venv=True),
                pipeline_root=os.path.join(tmpdir, 'asbpath_root'))

            # define in test to force install from source
            @dsl.component
            def identity(x: str) -> str:
                return x

            task = identity(x='foo')
            self.assertIsInstance(task.output, str)
            self.assertEqual(task.output, 'foo')


if __name__ == '__main__':
    unittest.main()

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
from kfp.local import task_dispatcher
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


class TestEnvironmentVariables(testing_utilities.LocalRunnerEnvironmentTestCase
                              ):
    """Tests for set_env_variable support in local execution."""

    def test_env_variable_available_in_subprocess(self):
        """Test that env vars set via set_env_variable are available in the
        subprocess."""
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def read_env_var() -> str:
            import os
            return os.environ.get('MY_TEST_VAR', 'NOT_SET')

        @dsl.pipeline
        def env_pipeline() -> str:
            task = read_env_var()
            task.set_env_variable('MY_TEST_VAR', 'hello_from_env')
            return task.output

        result = env_pipeline()
        self.assertIsInstance(result.output, str)
        self.assertEqual(result.output, 'hello_from_env')

    def test_multiple_env_variables(self):
        """Test that multiple env vars can be set."""
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def read_env_vars() -> str:
            import os
            var1 = os.environ.get('VAR_ONE', 'NOT_SET')
            var2 = os.environ.get('VAR_TWO', 'NOT_SET')
            return f'{var1}:{var2}'

        @dsl.pipeline
        def multi_env_pipeline() -> str:
            task = read_env_vars()
            task.set_env_variable('VAR_ONE', 'first')
            task.set_env_variable('VAR_TWO', 'second')
            return task.output

        result = multi_env_pipeline()
        self.assertIsInstance(result.output, str)
        self.assertEqual(result.output, 'first:second')


class TestRunnerEnvironmentMerging(unittest.TestCase):

    def test_docker_runner_environment_takes_precedence(self):
        runner = local.DockerRunner(environment={
            'FOO': 'runner-override',
            'RUNNER_ONLY': '1',
        })
        prepared_runner, merged_env = task_dispatcher._prepare_runner_and_env_vars(
            runner,
            {
                'FOO': 'component-value',
                'COMPONENT_ONLY': '1',
            },
        )

        self.assertIsNot(prepared_runner, runner)
        self.assertNotIn('environment', prepared_runner.container_run_args)
        self.assertEqual(
            merged_env,
            {
                'FOO': 'runner-override',
                'RUNNER_ONLY': '1',
                'COMPONENT_ONLY': '1',
            },
        )

    def test_docker_runner_environment_list_is_normalized(self):
        runner = local.DockerRunner(environment=['FOO=runner', 'BAR=baz'])
        _, merged_env = task_dispatcher._prepare_runner_and_env_vars(
            runner, {'FOO': 'component'})
        self.assertEqual(merged_env, {'FOO': 'runner', 'BAR': 'baz'})


class TestOneOf(testing_utilities.LocalRunnerEnvironmentTestCase):
    """Tests for dsl.OneOf support in local execution."""

    def test_oneof_selects_true_branch(self):
        """Test that OneOf picks the output from the executed branch."""
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def return_value(val: str) -> str:
            return val

        @dsl.pipeline
        def oneof_pipeline(flag: str) -> str:
            with dsl.If(flag == 'yes'):
                true_task = return_value(val='from_true_branch')
            with dsl.Else():
                false_task = return_value(val='from_false_branch')
            return dsl.OneOf(true_task.output, false_task.output)

        result = oneof_pipeline(flag='yes')
        self.assertIsInstance(result.output, str)
        self.assertEqual(result.output, 'from_true_branch')

    def test_oneof_selects_false_branch(self):
        """Test that OneOf picks the output from the else branch."""
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def return_value(val: str) -> str:
            return val

        @dsl.pipeline
        def oneof_pipeline(flag: str) -> str:
            with dsl.If(flag == 'yes'):
                true_task = return_value(val='from_true_branch')
            with dsl.Else():
                false_task = return_value(val='from_false_branch')
            return dsl.OneOf(true_task.output, false_task.output)

        result = oneof_pipeline(flag='no')
        self.assertIsInstance(result.output, str)
        self.assertEqual(result.output, 'from_false_branch')


class TestRetryLogic(testing_utilities.LocalRunnerEnvironmentTestCase):
    """Tests for set_retry support in local execution."""

    def test_retry_succeeds_after_failures(self):
        """Test that a failing task retries and eventually succeeds."""
        local.init(runner=local.SubprocessRunner(use_venv=True))

        import tempfile
        counter_file = os.path.join(tempfile.mkdtemp(), 'counter.txt')

        @dsl.component
        def flaky_component(counter_path: str) -> str:
            import os
            count = 0
            if os.path.exists(counter_path):
                with open(counter_path, 'r') as f:
                    count = int(f.read().strip())
            count += 1
            with open(counter_path, 'w') as f:
                f.write(str(count))
            if count < 3:
                raise RuntimeError(f'Attempt {count}: not yet!')
            return f'succeeded on attempt {count}'

        @dsl.pipeline
        def retry_pipeline(counter_path: str) -> str:
            task = flaky_component(counter_path=counter_path)
            task.set_retry(
                num_retries=3, backoff_duration='0s', backoff_factor=1.0)
            return task.output

        result = retry_pipeline(counter_path=counter_file)
        self.assertIsInstance(result.output, str)
        self.assertEqual(result.output, 'succeeded on attempt 3')

    def test_retry_exhausted_fails(self):
        """Test that exhausting retries results in failure."""
        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            raise_on_error=False,
        )

        @dsl.component
        def always_fail() -> str:
            raise RuntimeError('always fails')

        @dsl.pipeline
        def retry_fail_pipeline() -> str:
            task = always_fail()
            task.set_retry(
                num_retries=1, backoff_duration='0s', backoff_factor=1.0)
            return task.output

        result = retry_fail_pipeline()
        self.assertDictEqual(result.outputs, {})


class TestExitHandler(testing_utilities.LocalRunnerEnvironmentTestCase):
    """Tests for dsl.ExitHandler support in local execution."""

    def test_exit_handler_runs_on_success(self):
        """Test that exit task runs after successful body."""
        local.init(runner=local.SubprocessRunner(use_venv=True))

        import tempfile
        marker_file = os.path.join(tempfile.mkdtemp(), 'exit_marker.txt')

        @dsl.component
        def body_task() -> str:
            return 'body_done'

        @dsl.component
        def exit_task(marker_path: str) -> str:
            with open(marker_path, 'w') as f:
                f.write('exit_ran')
            return 'exit_done'

        @dsl.pipeline
        def exit_pipeline(marker_path: str):
            exit_t = exit_task(marker_path=marker_path)
            with dsl.ExitHandler(exit_t):
                body_task()

        exit_pipeline(marker_path=marker_file)
        with open(marker_file, 'r') as f:
            self.assertEqual(f.read(), 'exit_ran')

    def test_exit_handler_runs_on_failure(self):
        """Test that exit task runs even when body task fails."""
        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            raise_on_error=False,
        )

        import tempfile
        marker_file = os.path.join(tempfile.mkdtemp(), 'exit_marker.txt')

        @dsl.component
        def fail_body() -> str:
            raise RuntimeError('body failed')

        @dsl.component
        def exit_task(marker_path: str) -> str:
            with open(marker_path, 'w') as f:
                f.write('exit_ran_after_failure')
            return 'exit_done'

        @dsl.pipeline
        def exit_pipeline(marker_path: str):
            exit_t = exit_task(marker_path=marker_path)
            with dsl.ExitHandler(exit_t):
                fail_body()

        exit_pipeline(marker_path=marker_file)
        with open(marker_file, 'r') as f:
            self.assertEqual(f.read(), 'exit_ran_after_failure')

    def test_exit_handler_with_pipeline_task_final_status(self):
        """Test that PipelineTaskFinalStatus is passed correctly to exit
        task."""
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def body_task() -> str:
            return 'body_done'

        @dsl.component
        def exit_with_status(status: dsl.PipelineTaskFinalStatus) -> str:
            return status.state

        @dsl.pipeline
        def status_pipeline() -> str:
            exit_t = exit_with_status()
            with dsl.ExitHandler(exit_t):
                body_task()
            return exit_t.output

        result = status_pipeline()
        self.assertIsInstance(result.output, str)
        self.assertEqual(result.output, 'SUCCEEDED')

    def test_exit_handler_final_status_on_failure(self):
        """Test PipelineTaskFinalStatus reports FAILED when body fails."""
        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            raise_on_error=False,
        )

        import tempfile
        status_file = os.path.join(tempfile.mkdtemp(), 'status.txt')

        @dsl.component
        def fail_body() -> str:
            raise RuntimeError('body failed')

        @dsl.component
        def exit_with_status(
            status: dsl.PipelineTaskFinalStatus,
            status_path: str,
        ):
            with open(status_path, 'w') as f:
                f.write(status.state)

        @dsl.pipeline
        def status_pipeline(status_path: str):
            exit_t = exit_with_status(status_path=status_path)
            with dsl.ExitHandler(exit_t):
                fail_body()

        status_pipeline(status_path=status_file)
        with open(status_file, 'r') as f:
            self.assertEqual(f.read(), 'FAILED')


class TestK8sOnlyMethodsWarn(testing_utilities.LocalRunnerEnvironmentTestCase):
    """Tests that K8s-only methods emit warnings instead of errors during local
    execution."""

    def test_k8s_methods_warn_and_task_still_executes(self):
        """Test that K8s-only config methods emit warnings but the task still
        runs."""
        import warnings as warnings_mod
        local.init(runner=local.SubprocessRunner(use_venv=True))

        @dsl.component
        def identity(x: str) -> str:
            return x

        task = identity(x='hello')
        with warnings_mod.catch_warnings(record=True) as w:
            warnings_mod.simplefilter('always')
            task.set_cpu_limit('1')
            task.set_memory_limit('512Mi')
            task.set_display_name('my_task')
        warning_messages = [str(warning.message) for warning in w]
        self.assertTrue(
            any('.set_cpu_limit()' in msg for msg in warning_messages))
        self.assertTrue(
            any('.set_memory_limit()' in msg for msg in warning_messages))
        self.assertTrue(
            any('.set_display_name()' in msg for msg in warning_messages))
        self.assertIsInstance(task.output, str)
        self.assertEqual(task.output, 'hello')


class TestIgnoreUpstreamFailure(testing_utilities.LocalRunnerEnvironmentTestCase
                               ):
    """Tests for ignore_upstream_failure support in local execution."""

    def test_downstream_runs_after_upstream_failure(self):
        """Test that a task with ignore_upstream_failure runs even when
        upstream fails."""
        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            raise_on_error=False,
        )

        import tempfile
        marker_file = os.path.join(tempfile.mkdtemp(), 'marker.txt')

        @dsl.component
        def fail_task() -> str:
            raise RuntimeError('upstream failed')

        @dsl.component
        def cleanup_task(marker_path: str) -> str:
            with open(marker_path, 'w') as f:
                f.write('cleanup_ran')
            return 'cleaned up'

        @dsl.pipeline
        def ignore_failure_pipeline(marker_path: str):
            upstream = fail_task()
            cleanup = cleanup_task(marker_path=marker_path)
            cleanup.after(upstream).ignore_upstream_failure()

        ignore_failure_pipeline(marker_path=marker_file)
        with open(marker_file, 'r') as f:
            self.assertEqual(f.read(), 'cleanup_ran')

    def test_ignore_upstream_failure_with_final_status(self):
        """Test ignore_upstream_failure with PipelineTaskFinalStatus."""
        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            raise_on_error=False,
        )

        import tempfile
        status_file = os.path.join(tempfile.mkdtemp(), 'status.txt')

        @dsl.component
        def fail_task() -> str:
            raise RuntimeError('upstream failed')

        @dsl.component
        def handler_task(
            status: dsl.PipelineTaskFinalStatus,
            status_path: str,
        ):
            with open(status_path, 'w') as f:
                f.write(status.state)

        @dsl.pipeline
        def status_pipeline(status_path: str):
            upstream = fail_task()
            handler = handler_task(status_path=status_path)
            handler.after(upstream).ignore_upstream_failure()

        status_pipeline(status_path=status_file)
        with open(status_file, 'r') as f:
            self.assertEqual(f.read(), 'FAILED')


class TestLocalCaching(testing_utilities.LocalRunnerEnvironmentTestCase):
    """Tests for local task output caching (set_caching_options)."""

    def test_cache_hit_skips_second_run(self):
        """With caching enabled, identical pipeline re-run returns cached
        outputs without re-executing the component."""
        import tempfile
        pipeline_root = tempfile.mkdtemp(prefix='kfp-cache-pipeline-')
        counter_file = os.path.join(tempfile.mkdtemp(), 'counter.txt')

        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            pipeline_root=pipeline_root,
            enable_caching=True,
        )

        @dsl.component
        def counting_component(counter_path: str) -> str:
            with open(counter_path, 'a') as f:
                f.write('x')
            return 'hello'

        @dsl.pipeline
        def cache_pipeline(counter_path: str) -> str:
            t = counting_component(counter_path=counter_path)
            return t.output

        r1 = cache_pipeline(counter_path=counter_file)
        self.assertEqual(r1.output, 'hello')
        with open(counter_file) as f:
            self.assertEqual(len(f.read()), 1)

        r2 = cache_pipeline(counter_path=counter_file)
        self.assertEqual(r2.output, 'hello')
        # Second run should be a cache hit — counter file unchanged.
        with open(counter_file) as f:
            self.assertEqual(len(f.read()), 1)

    def test_per_task_disable_bypasses_cache(self):
        """A task with set_caching_options(False) always executes, even when
        local caching is globally enabled."""
        import tempfile
        pipeline_root = tempfile.mkdtemp(prefix='kfp-cache-pipeline-')
        counter_file = os.path.join(tempfile.mkdtemp(), 'counter.txt')

        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            pipeline_root=pipeline_root,
            enable_caching=True,
        )

        @dsl.component
        def counting_component(counter_path: str) -> str:
            with open(counter_path, 'a') as f:
                f.write('x')
            return 'hello'

        @dsl.pipeline
        def no_cache_pipeline(counter_path: str) -> str:
            t = counting_component(counter_path=counter_path)
            t.set_caching_options(False)
            return t.output

        no_cache_pipeline(counter_path=counter_file)
        no_cache_pipeline(counter_path=counter_file)
        with open(counter_file) as f:
            self.assertEqual(len(f.read()), 2)

    def test_different_inputs_miss(self):
        """Different inputs should produce a cache miss, not a hit on the prior
        run's outputs."""
        import tempfile
        pipeline_root = tempfile.mkdtemp(prefix='kfp-cache-pipeline-')
        counter_file = os.path.join(tempfile.mkdtemp(), 'counter.txt')

        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            pipeline_root=pipeline_root,
            enable_caching=True,
        )

        @dsl.component
        def echo(counter_path: str, val: str) -> str:
            with open(counter_path, 'a') as f:
                f.write('x')
            return val

        @dsl.pipeline
        def echo_pipeline(counter_path: str, val: str) -> str:
            return echo(counter_path=counter_path, val=val).output

        r1 = echo_pipeline(counter_path=counter_file, val='a')
        r2 = echo_pipeline(counter_path=counter_file, val='b')
        self.assertEqual(r1.output, 'a')
        self.assertEqual(r2.output, 'b')
        with open(counter_file) as f:
            self.assertEqual(len(f.read()), 2)

    def test_caching_disabled_globally_always_executes(self):
        """With enable_caching=False in init(), each run executes regardless of
        per-task caching options."""
        import tempfile
        pipeline_root = tempfile.mkdtemp(prefix='kfp-cache-pipeline-')
        counter_file = os.path.join(tempfile.mkdtemp(), 'counter.txt')

        # Note: enable_caching defaults to False; pass explicitly for clarity.
        local.init(
            runner=local.SubprocessRunner(use_venv=True),
            pipeline_root=pipeline_root,
            enable_caching=False,
        )

        @dsl.component
        def counting_component(counter_path: str) -> str:
            with open(counter_path, 'a') as f:
                f.write('x')
            return 'hello'

        @dsl.pipeline
        def cache_pipeline(counter_path: str) -> str:
            return counting_component(counter_path=counter_path).output

        cache_pipeline(counter_path=counter_file)
        cache_pipeline(counter_path=counter_file)
        with open(counter_file) as f:
            self.assertEqual(len(f.read()), 2)


if __name__ == '__main__':
    unittest.main()

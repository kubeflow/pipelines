# Copyright 2022 The Kubeflow Authors
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
from dataclasses import dataclass
import functools
import logging
import os
from pathlib import Path
import shutil
import tempfile
from typing import Any, Callable, Optional
import uuid
import warnings

from kfp import dsl
from kfp import local
import pytest

base_image = 'registry.access.redhat.com/ubi9/python-311:latest'
_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')

dsl.component = functools.partial(
    dsl.component, base_image=base_image, kfp_package_path=_KFP_PACKAGE_PATH)
dsl.notebook_component = functools.partial(
    dsl.notebook_component, kfp_package_path=_KFP_PACKAGE_PATH)

from test_data.sdk_compiled_pipelines.valid import sequential_container
import test_data.sdk_compiled_pipelines.valid.arguments_parameters as arguments_parameters
import test_data.sdk_compiled_pipelines.valid.critical.add_numbers as add_numbers_module
import test_data.sdk_compiled_pipelines.valid.critical.collected_parameters as collected_parameters
import test_data.sdk_compiled_pipelines.valid.critical.component_with_optional_inputs as optional_inputs
import test_data.sdk_compiled_pipelines.valid.critical.flip_coin as flip_coin_module
import test_data.sdk_compiled_pipelines.valid.critical.mixed_parameters as mixed_parameters
import test_data.sdk_compiled_pipelines.valid.critical.multiple_parameters_namedtuple as namedtuple_parameters
import test_data.sdk_compiled_pipelines.valid.critical.notebook_component_simple as notebook_component
import test_data.sdk_compiled_pipelines.valid.critical.pipeline_with_importer_workspace as importer_workspace
import test_data.sdk_compiled_pipelines.valid.critical.producer_consumer_param as producer_consumer
import test_data.sdk_compiled_pipelines.valid.essential.component_with_env_variable as env_variable
import test_data.sdk_compiled_pipelines.valid.essential.concat_message as concat_message_module
import test_data.sdk_compiled_pipelines.valid.essential.container_no_input as container_no_input_module
import test_data.sdk_compiled_pipelines.valid.essential.dict_input as dict_input_module
import test_data.sdk_compiled_pipelines.valid.essential.lightweight_python_functions_with_outputs as lightweight_outputs
from test_data.sdk_compiled_pipelines.valid.hello_world import echo
from test_data.sdk_compiled_pipelines.valid.identity import identity
import test_data.sdk_compiled_pipelines.valid.local_execution.local_execution_pipeline_with_exit_handler as exit_handler
import test_data.sdk_compiled_pipelines.valid.local_execution.local_execution_pipeline_with_ignore_upstream_failure as ignore_upstream_failure
import test_data.sdk_compiled_pipelines.valid.local_execution.local_execution_pipeline_with_k8s_only_methods as k8s_only_methods
import test_data.sdk_compiled_pipelines.valid.local_execution.local_execution_pipeline_with_oneof as oneof
import test_data.sdk_compiled_pipelines.valid.local_execution.local_execution_pipeline_with_retry as retry
import test_data.sdk_compiled_pipelines.valid.local_execution.pipeline_with_caching as caching
import test_data.sdk_compiled_pipelines.valid.local_execution.pipeline_with_collected_artifacts as collected_artifacts
from test_data.sdk_compiled_pipelines.valid.nested_return import nested_return
import test_data.sdk_compiled_pipelines.valid.output_metrics as output_metrics_module
import test_data.sdk_compiled_pipelines.valid.parallel_and_nested.pipeline_with_loops as loops
import test_data.sdk_compiled_pipelines.valid.parameter as parameters
import test_data.sdk_compiled_pipelines.valid.pipeline_with_parallelfor_list_artifacts as parallelfor_list_artifacts
import test_data.sdk_compiled_pipelines.valid.pipeline_with_parallelfor_parallelism as parallelfor_parallelism
import test_data.sdk_compiled_pipelines.valid.pipeline_with_parallelfor_pipeline_param as parallelfor_pipeline_param

arguments_echo = arguments_parameters.echo
add_numbers = add_numbers_module.add_numbers
collected_param_pipeline = collected_parameters.collected_param_pipeline
optional_inputs_pipeline = optional_inputs.pipeline
flip_coin = flip_coin_module.flip_coin
mixed_parameters_pipeline = mixed_parameters.crust
namedtuple_pipeline = namedtuple_parameters.crust
notebook_simple_pipeline = notebook_component.pipeline
run_train_notebook_component = notebook_component.run_train_notebook
importer_workspace_pipeline = importer_workspace.pipeline_with_importer_workspace
producer_consumer_param_pipeline = producer_consumer.producer_consumer_param_pipeline
pipeline_with_env_variable = env_variable.pipeline_with_env_variable
concat_message = concat_message_module.concat_message
container_no_input = container_no_input_module.container_no_input
dict_input = dict_input_module.dict_input
lightweight_with_outputs_pipeline = lightweight_outputs.pipeline
pipeline_with_exit_handler = exit_handler.pipeline_with_exit_handler
pipeline_with_ignore_upstream_failure = ignore_upstream_failure.pipeline_with_ignore_upstream_failure
pipeline_with_k8s_only_methods = k8s_only_methods.pipeline_with_k8s_only_methods
pipeline_with_oneof = oneof.pipeline_with_oneof
pipeline_with_retry = retry.pipeline_with_retry
pipeline_with_caching = caching.pipeline_with_caching
pipeline_with_collected_artifacts = collected_artifacts.pipeline_with_collected_artifacts
output_metrics = output_metrics_module.output_metrics
pipeline_with_loops = loops.my_pipeline
parameter_pipeline = parameters.crust
pipeline_with_parallelfor_list_artifacts = parallelfor_list_artifacts.my_pipeline
pipeline_with_parallelfor_parallelism = parallelfor_parallelism.my_pipeline
pipeline_with_parallelfor_pipeline_param = parallelfor_pipeline_param.my_pipeline
sequential = sequential_container.sequential


@dataclass
class TestData:
    name: str
    pipeline_func: Callable
    pipeline_func_args: Optional[dict]
    expected_output: Any
    subprocess_only: bool = False

    def __str__(self) -> str:
        return (f"Test Data: "
                f"name={self.name} "
                f"pipeline_func={self.pipeline_func.__name__} "
                f"args={self.pipeline_func_args}")

    def __repr__(self) -> str:
        return self.__str__()


def idfn(val):
    return val.name


# Use relative directories that work for both runners
ws_root_base = './test_workspace'
pipeline_root_base = './test_pipeline_outputs'

pipeline_func_data = [
    TestData(
        name='Add Numbers',
        pipeline_func=add_numbers,
        pipeline_func_args={
            'a': 5,
            'b': 5
        },
        expected_output=10,
    ),
    TestData(
        name='Mixed Parameter',
        pipeline_func=mixed_parameters_pipeline,
        pipeline_func_args=None,
        expected_output=None,
    ),
    TestData(
        name='Flip Coin',
        pipeline_func=flip_coin,
        pipeline_func_args=None,
        expected_output=['heads', 'tails'],
    ),
    TestData(
        name='Optional Inputs',
        pipeline_func=optional_inputs_pipeline,
        pipeline_func_args=None,
        expected_output=None,
    ),
    TestData(
        name='Concat Message',
        pipeline_func=concat_message,
        pipeline_func_args={
            'message1': 'Hello ',
            'message2': 'World!'
        },
        expected_output='Hello World!',
    ),
    TestData(
        name='Identity Function',
        pipeline_func=identity,
        pipeline_func_args={'value': 'test_value'},
        expected_output='test_value',
    ),
    TestData(
        name='Lightweight With Outputs',
        pipeline_func=lightweight_with_outputs_pipeline,
        pipeline_func_args={
            'first_message': 'Hello',
            'second_message': ' World',
            'first_number': 10,
            'second_number': 20
        },
        expected_output=None,
    ),
    TestData(
        name='Dict Input',
        pipeline_func=dict_input,
        pipeline_func_args={'struct': {
            'test_key': 'test_value'
        }},
        expected_output=None,
    ),
    TestData(
        name='Parameter Pipeline',
        pipeline_func=parameter_pipeline,
        pipeline_func_args=None,
        expected_output=None,
    ),
    TestData(
        name='Output Metrics',
        pipeline_func=output_metrics,
        pipeline_func_args=None,
        expected_output=None,
    ),
    TestData(
        name='Nested Return',
        pipeline_func=nested_return,
        pipeline_func_args=None,
        expected_output=[{
            'A_a': '1',
            'B_b': '2'
        }, {
            'A_a': '10',
            'B_b': '20'
        }],
    ),
    TestData(
        name='NamedTuple Pipeline',
        pipeline_func=namedtuple_pipeline,
        pipeline_func_args=None,
        expected_output=None,
    ),
    TestData(
        name='Importer Workspace',
        pipeline_func=importer_workspace_pipeline,
        pipeline_func_args=None,
        expected_output=None,
    ),
    TestData(
        name='Pipeline with Loops',
        pipeline_func=pipeline_with_loops,
        pipeline_func_args={'loop_parameter': ['item1', 'item2', 'item3']},
        expected_output=None,
    ),
    TestData(
        name='Pipeline with ParallelFor Pipeline Param',
        pipeline_func=pipeline_with_parallelfor_pipeline_param,
        pipeline_func_args={
            'prefix': 'test-prefix',
            'loop_parameter': ['a', 'b'],
        },
        expected_output=None,
    ),
    TestData(
        name='Pipeline with Env Variable',
        pipeline_func=pipeline_with_env_variable,
        pipeline_func_args={'var_name': 'KFP_TEST_ENV_VAR'},
        expected_output='env_var_value',
    ),
    TestData(
        name='Notebook Component Simple',
        pipeline_func=notebook_simple_pipeline,
        pipeline_func_args={'text': 'hello'},
        expected_output=None,
    ),
    TestData(
        name='Collected Parameters',
        pipeline_func=collected_param_pipeline,
        pipeline_func_args=None,
        expected_output=None,
    ),
    TestData(
        name='Collected Artifacts',
        pipeline_func=pipeline_with_collected_artifacts,
        pipeline_func_args=None,
        expected_output='0,1,2',
    ),
    TestData(
        name='Retry',
        pipeline_func=pipeline_with_retry,
        pipeline_func_args={
            'counter_path':
                os.path.join(tempfile.gettempdir(),
                             f'kfp-retry-{uuid.uuid4().hex}.count'),
        },
        expected_output='attempt=2',
        # Flaky counter file lives on host tmp; Docker containers can't
        # see it without a volume mount, so gate to subprocess.
        subprocess_only=True,
    ),
    TestData(
        name='Exit Handler',
        pipeline_func=pipeline_with_exit_handler,
        pipeline_func_args={'message': 'hi'},
        expected_output=None,
    ),
    TestData(
        name='OneOf',
        pipeline_func=pipeline_with_oneof,
        pipeline_func_args={'pick_a': True},
        expected_output='selected:A',
    ),
    TestData(
        name='K8s-only Methods Ignored Locally',
        pipeline_func=pipeline_with_k8s_only_methods,
        pipeline_func_args={'message': 'hello'},
        expected_output='hello',
    ),
]


@pytest.mark.skipif(
    _KFP_PACKAGE_PATH is None, reason='KFP_PACKAGE_PATH is not configured')
def test_notebook_component_uses_configured_kfp_package_path():
    install_command = run_train_notebook_component.component_spec.implementation.container.command[
        2]

    assert _KFP_PACKAGE_PATH in install_command


docker_specific_pipeline_funcs = [
    TestData(
        name='Producer Consumer Pipeline',
        pipeline_func=producer_consumer_param_pipeline,
        pipeline_func_args=None,
        expected_output=None,
    ),
    TestData(
        name='Container Hello World',
        pipeline_func=echo,
        pipeline_func_args=None,
        expected_output=None,
    ),
    TestData(
        name='Sequential Container Pipeline',
        pipeline_func=sequential,
        pipeline_func_args={
            'param1': 'First',
            'param2': 'Second'
        },
        expected_output=None,
    ),
    TestData(
        name='Container No Args',
        pipeline_func=container_no_input,
        pipeline_func_args=None,
        expected_output=None,
    ),
    TestData(
        name='Container with Arguments',
        pipeline_func=arguments_echo,
        pipeline_func_args={
            'param1': 'arg1',
            'param2': 'arg2'
        },
        expected_output=None,
    ),
    TestData(
        name='Pipeline with ParallelFor Parallelism',
        pipeline_func=pipeline_with_parallelfor_parallelism,
        pipeline_func_args={'loop_parameter': ['item1', 'item2', 'item3']},
        expected_output=None,
    ),
    TestData(
        name='Pipeline with ParallelFor List Artifacts',
        pipeline_func=pipeline_with_parallelfor_list_artifacts,
        pipeline_func_args=None,
        expected_output=None,
    ),
]
docker_specific_pipeline_funcs.extend(pipeline_func_data)


@pytest.mark.regression
class TestDockerRunner:

    @pytest.fixture(autouse=True)
    def setup_and_teardown(self, worker_id):
        ws_root = f'{ws_root_base}_docker_{worker_id}'
        pipeline_root = f'{pipeline_root_base}_docker_{worker_id}'

        # Create directories with proper permissions
        Path(ws_root).mkdir(exist_ok=True)
        Path(pipeline_root).mkdir(exist_ok=True)

        # Set permissions to be writable by all (needed for Docker)
        import stat
        os.chmod(ws_root, stat.S_IRWXU | stat.S_IRWXG | stat.S_IRWXO)
        os.chmod(pipeline_root, stat.S_IRWXU | stat.S_IRWXG | stat.S_IRWXO)

        # Configure DockerRunner - let container run as default user
        local.init(
            runner=local.DockerRunner(),
            raise_on_error=True,
            workspace_root=ws_root,
            pipeline_root=pipeline_root)
        yield
        try:
            if os.path.isdir(ws_root):
                print(f"Deleting WS Root {ws_root}")
                shutil.rmtree(ws_root, ignore_errors=True)
            if os.path.isdir(pipeline_root):
                print(f"Deleting Pipeline Root {pipeline_root}")
                shutil.rmtree(pipeline_root, ignore_errors=True)
        except Exception as e:
            print(f"Failed to delete directory because of {e}")

    @pytest.mark.parametrize(
        'test_data', docker_specific_pipeline_funcs, ids=idfn)
    def test_execution(self, test_data: TestData):
        if test_data.subprocess_only:
            pytest.skip(
                f"{test_data.name} is subprocess-only (not supported in Docker)."
            )
        if test_data.pipeline_func_args is not None:
            pipeline_task = test_data.pipeline_func(
                **test_data.pipeline_func_args)
        else:
            pipeline_task = test_data.pipeline_func()
        if test_data.expected_output is None:
            print('Skipping output check')
        elif type(test_data.expected_output) == list:
            assert pipeline_task.output in test_data.expected_output or pipeline_task.output == test_data.expected_output, 'Output of the pipeline is not the same as expected'
        else:
            assert pipeline_task.output == test_data.expected_output, 'Output of the pipeline is not the same as expected'


@pytest.mark.regression
class TestSubProcessRunner:

    @pytest.fixture(scope='class', autouse=True)
    def setup_and_teardown(self, worker_id):
        ws_root = f'{ws_root_base}_subprocess_{worker_id}'
        pipeline_root = f'{pipeline_root_base}_subprocess_{worker_id}'
        Path(ws_root).mkdir(exist_ok=True)
        Path(pipeline_root).mkdir(exist_ok=True)
        local.init(
            runner=local.SubprocessRunner(),
            raise_on_error=True,
            workspace_root=ws_root,
            pipeline_root=pipeline_root)
        yield
        try:
            if os.path.isdir(ws_root):
                shutil.rmtree(ws_root, ignore_errors=True)
            if os.path.isdir(pipeline_root):
                shutil.rmtree(pipeline_root, ignore_errors=True)
        except Exception as e:
            print(f"Failed to delete directory because of {e}")

    @pytest.mark.parametrize('test_data', pipeline_func_data, ids=idfn)
    def test_execution(self, test_data: TestData):
        if test_data.pipeline_func_args is not None:
            pipeline_task = test_data.pipeline_func(
                **test_data.pipeline_func_args)
        else:
            pipeline_task = test_data.pipeline_func()
        if test_data.expected_output is None:
            print('Skipping output check')
        elif type(test_data.expected_output) == list:
            assert pipeline_task.output in test_data.expected_output or pipeline_task.output == test_data.expected_output, 'Output of the pipeline is not the same as expected'
        else:
            assert pipeline_task.output == test_data.expected_output, 'Output of the pipeline is not the same as expected'


@pytest.mark.regression
class TestIgnoreUpstreamFailureIntegration:
    """Downstream task runs after upstream failure when
    `.ignore_upstream_failure()` is set.

    The overall DAG reports failure, so the test runs with
    `raise_on_error=False` and verifies the downstream ran via a marker
    file.
    """

    def test_downstream_runs_after_upstream_failure(self, tmp_path):
        local.init(
            runner=local.SubprocessRunner(),
            raise_on_error=False,
            pipeline_root=str(tmp_path / 'pipeline_root'),
            workspace_root=str(tmp_path / 'workspace'),
        )
        marker_path = str(tmp_path / 'marker.txt')
        pipeline_with_ignore_upstream_failure(marker_path=marker_path)
        with open(marker_path) as f:
            assert f.read() == 'cleanup_ran'


@pytest.mark.regression
class TestLocalCachingIntegration:
    """Second run with identical inputs hits the local cache.

    Verified by capturing the orchestrator's `cache hit` log line on the
    second run.
    """

    def test_second_run_hits_cache(self, tmp_path, caplog):
        local.init(
            runner=local.SubprocessRunner(),
            raise_on_error=True,
            enable_caching=True,
            cache_root=str(tmp_path / 'cache'),
            pipeline_root=str(tmp_path / 'pipeline_root'),
            workspace_root=str(tmp_path / 'workspace'),
        )

        first = pipeline_with_caching(x=1)
        assert first.output == 2

        with caplog.at_level(logging.INFO):
            second = pipeline_with_caching(x=1)
        assert second.output == 2
        assert any('cache hit' in record.message for record in caplog.records), \
            f'Expected a cache hit log; got: {[r.message for r in caplog.records]}'


@pytest.mark.regression
class TestK8sOnlyWarningsIntegration:
    """Kubernetes-only task-config methods emit a warning when called on a task
    that already executed locally, and do not raise."""

    def test_k8s_methods_warn_after_local_execution(self, tmp_path):
        local.init(
            runner=local.SubprocessRunner(),
            raise_on_error=True,
            pipeline_root=str(tmp_path / 'pipeline_root'),
            workspace_root=str(tmp_path / 'workspace'),
        )

        @dsl.component
        def identity(x: str) -> str:
            return x

        task = identity(x='hello')
        assert task.output == 'hello'

        with warnings.catch_warnings(record=True) as caught:
            warnings.simplefilter('always')
            task.set_cpu_limit('1000m')
            task.set_memory_limit('256Mi')
            task.set_accelerator_type('nvidia.com/gpu')

        messages = [str(w.message) for w in caught]
        assert any('set_cpu_limit' in m for m in messages), messages
        assert any('set_memory_limit' in m for m in messages), messages
        assert any('set_accelerator_type' in m for m in messages), messages

    def test_k8s_methods_warn_during_pipeline_execution(self, tmp_path):
        """Normal pipeline flow: K8s-only resource fields in the compiled spec
        must produce a warning at task execution time.

        The DSL-level `@warn_if_final` only fires for tasks in FINAL
        state. Inside a pipeline decorator the task is still FUTURE when
        `set_cpu_limit(...)` runs, so the warning must come from the
        local orchestrator inspecting the compiled ResourceSpec.
        """
        local.init(
            runner=local.SubprocessRunner(),
            raise_on_error=True,
            pipeline_root=str(tmp_path / 'pipeline_root'),
            workspace_root=str(tmp_path / 'workspace'),
        )

        with warnings.catch_warnings(record=True) as caught:
            warnings.simplefilter('always')
            result = pipeline_with_k8s_only_methods(message='hello')

        assert result.output == 'hello'
        messages = [str(w.message) for w in caught]
        resource_msgs = [m for m in messages if 'current local runner' in m]
        assert resource_msgs, (
            f'Expected a resource-settings warning; got: {messages}')
        joined = ' '.join(resource_msgs)
        assert 'cpu_limit' in joined, resource_msgs
        assert 'memory_limit' in joined, resource_msgs
        assert 'accelerator' in joined, resource_msgs

    def test_display_name_warns_during_pipeline_execution(self, tmp_path):
        """Normal pipeline flow: display names are UI metadata and have no
        local execution effect, so they should warn at task execution time."""
        local.init(
            runner=local.SubprocessRunner(),
            raise_on_error=True,
            pipeline_root=str(tmp_path / 'pipeline_root'),
            workspace_root=str(tmp_path / 'workspace'),
        )

        @dsl.component
        def identity(x: str) -> str:
            return x

        @dsl.pipeline
        def display_name_pipeline() -> str:
            task = identity(x='hello')
            task.set_display_name('Human Label')
            return task.output

        with warnings.catch_warnings(record=True) as caught:
            warnings.simplefilter('always')
            result = display_name_pipeline()

        assert result.output == 'hello'
        messages = [str(w.message) for w in caught]
        display_name_msgs = [
            m for m in messages
            if 'display_name' in m and 'current local runner' in m
        ]
        assert display_name_msgs, (
            f'Expected a display-name warning; got: {messages}')


@pytest.mark.regression
class TestExitHandlerFailurePropagation:
    """A failing exit-handler task must propagate as pipeline FAILURE, not
    silently succeed.

    Regression for a bug where exit-task failures were swallowed when
    the body phase succeeded.
    """

    def test_failing_exit_task_propagates_failure(self, tmp_path):
        local.init(
            runner=local.SubprocessRunner(),
            raise_on_error=True,
            pipeline_root=str(tmp_path / 'pipeline_root'),
            workspace_root=str(tmp_path / 'workspace'),
        )

        @dsl.component
        def body_ok() -> str:
            return 'ok'

        @dsl.component
        def exit_fails(status: dsl.PipelineTaskFinalStatus) -> str:
            raise RuntimeError(f'exit handler failed; saw {status.state}')

        @dsl.pipeline
        def failing_exit_pipeline():
            exit_task = exit_fails()
            with dsl.ExitHandler(exit_task):
                body_ok()

        # Body succeeds; exit task raises. Pipeline must surface FAILURE,
        # which (under raise_on_error=True) manifests as a RuntimeError.
        with pytest.raises(RuntimeError, match='finished with status'):
            failing_exit_pipeline()


@pytest.mark.regression
class TestCacheEnvVarInvalidation:
    """Cache entries must be invalidated when a container's env vars change.

    Regression for a bug where cache key ignored `container.env`, so
    changing `set_env_variable('X', 'one')` → `set_env_variable('X',
    'two')` returned the stale prior result.
    """

    def test_env_var_change_invalidates_cache(self, tmp_path, caplog):
        local.init(
            runner=local.SubprocessRunner(),
            raise_on_error=True,
            enable_caching=True,
            cache_root=str(tmp_path / 'cache'),
            pipeline_root=str(tmp_path / 'pipeline_root'),
            workspace_root=str(tmp_path / 'workspace'),
        )

        @dsl.component
        def read_var() -> str:
            import os
            return os.environ.get('MYVAR', '<unset>')

        @dsl.pipeline
        def p_one() -> str:
            t = read_var()
            t.set_env_variable('MYVAR', 'one')
            return t.output

        @dsl.pipeline
        def p_two() -> str:
            t = read_var()
            t.set_env_variable('MYVAR', 'two')
            return t.output

        first = p_one()
        assert first.output == 'one'

        with caplog.at_level(logging.INFO):
            second = p_two()
        assert second.output == 'two', (
            'Changing env var should invalidate cache; got stale value')
        assert not any(
            'cache hit' in record.message for record in caplog.records), (
                f'Changing env should be a cache miss; got: '
                f'{[r.message for r in caplog.records]}')

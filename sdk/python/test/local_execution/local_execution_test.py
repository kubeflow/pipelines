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
import os
from pathlib import Path
import shutil
from typing import Any, Callable, Optional

from kfp import dsl
from kfp import local
import pytest

base_image = "registry.access.redhat.com/ubi9/python-311:latest"
_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')

dsl.component = functools.partial(
    dsl.component, base_image=base_image, kfp_package_path=_KFP_PACKAGE_PATH)

from test_data.sdk_compiled_pipelines.valid.arguments_parameters import \
    echo as arguments_echo
from test_data.sdk_compiled_pipelines.valid.critical.add_numbers import \
    add_numbers
from test_data.sdk_compiled_pipelines.valid.critical.component_with_optional_inputs import \
    pipeline as optional_inputs_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.flip_coin import flip_coin
from test_data.sdk_compiled_pipelines.valid.critical.mixed_parameters import \
    crust as mixed_parameters_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.multiple_parameters_namedtuple import \
    crust as namedtuple_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.pipeline_with_importer_workspace import \
    pipeline_with_importer_workspace as importer_workspace_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.producer_consumer_param import \
    producer_consumer_param_pipeline
from test_data.sdk_compiled_pipelines.valid.dict_input import dict_input
from test_data.sdk_compiled_pipelines.valid.essential.concat_message import \
    concat_message
from test_data.sdk_compiled_pipelines.valid.essential.container_no_input import \
    container_no_input
from test_data.sdk_compiled_pipelines.valid.essential.lightweight_python_functions_with_outputs import \
    pipeline as lightweight_with_outputs_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_with_loops import \
    my_pipeline as pipeline_with_loops
from test_data.sdk_compiled_pipelines.valid.hello_world import echo
from test_data.sdk_compiled_pipelines.valid.identity import identity
from test_data.sdk_compiled_pipelines.valid.nested_return import nested_return
from test_data.sdk_compiled_pipelines.valid.output_metrics import \
    output_metrics
from test_data.sdk_compiled_pipelines.valid.parameter import \
    crust as parameter_pipeline
from test_data.sdk_compiled_pipelines.valid.pipeline_with_parallelfor_list_artifacts import \
    my_pipeline as pipeline_with_parallelfor_list_artifacts
from test_data.sdk_compiled_pipelines.valid.pipeline_with_parallelfor_parallelism import \
    my_pipeline as pipeline_with_parallelfor_parallelism
from test_data.sdk_compiled_pipelines.valid.sequential_v1 import sequential


@dataclass
class TestData:
    name: str
    pipeline_func: Callable
    pipeline_func_args: Optional[dict]
    expected_output: Any

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
]

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
    def setup_and_teardown(self):
        ws_root = f'{ws_root_base}_docker'
        pipeline_root = f'{pipeline_root_base}_docker'

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
        if test_data.pipeline_func_args is not None:
            pipeline_task = test_data.pipeline_func(
                **test_data.pipeline_func_args)
        else:
            pipeline_task = test_data.pipeline_func()
        if test_data.expected_output is None:
            print("Skipping output check")
        elif type(test_data.expected_output) == list:
            assert pipeline_task.output in test_data.expected_output or pipeline_task.output == test_data.expected_output, "Output of the pipeline is not the same as expected"
        else:
            assert pipeline_task.output == test_data.expected_output, "Output of the pipeline is not the same as expected"


@pytest.mark.regression
class TestSubProcessRunner:

    @pytest.fixture(scope="class", autouse=True)
    def setup_and_teardown(self):
        ws_root = f'{ws_root_base}_subprocess'
        pipeline_root = f'{pipeline_root_base}_subprocess'
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
            print("Skipping output check")
        elif type(test_data.expected_output) == list:
            assert pipeline_task.output in test_data.expected_output or pipeline_task.output == test_data.expected_output, "Output of the pipeline is not the same as expected"
        else:
            assert pipeline_task.output == test_data.expected_output, "Output of the pipeline is not the same as expected"

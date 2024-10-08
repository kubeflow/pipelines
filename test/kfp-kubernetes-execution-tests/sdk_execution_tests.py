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
import asyncio
import dataclasses
import functools
import os
import sys
from typing import Any, Dict, List, Tuple

from kfp import client
from kfp import dsl
import kfp_server_api
import pytest
import yaml

KFP_ENDPOINT = os.environ['KFP_ENDPOINT']
TIMEOUT_SECONDS = os.environ['TIMEOUT_SECONDS']
CURRENT_DIR = os.path.abspath(os.path.dirname(__file__))
PROJECT_ROOT = os.path.abspath(
    os.path.join(CURRENT_DIR, *([os.path.pardir] * 2)))
CONFIG_PATH = os.path.join(
    PROJECT_ROOT,
    'kubernetes_platform',
    'python',
    'test',
    'snapshot',
    'test_data_config.yaml',
)

kfp_client = client.Client(host=KFP_ENDPOINT)


@dataclasses.dataclass
class TestCase:
    name: str
    module_path: str
    yaml_path: str
    function_name: str
    arguments: Dict[str, Any]


def create_test_case_parameters() -> List[TestCase]:
    parameters: List[TestCase] = []
    with open(CONFIG_PATH) as f:
        config = yaml.safe_load(f)
    test_data_dir = os.path.join(
        PROJECT_ROOT,
        'kubernetes_platform',
        'python',
        'test',
        'snapshot',
        'data',
    )

    parameters.extend(
        TestCase(
            name=test_case['name'] + '-' + test_case['module'],
            module_path=os.path.join(test_data_dir,
                                     f'{test_case["module"]}.py'),
            yaml_path=os.path.join(test_data_dir,
                                   f'{test_case["module"]}.yaml'),
            function_name=test_case['name'],
            arguments=test_case.get('arguments'),
        ) for test_case in config['test_cases'])

    return parameters


def wait(
        run_result: client.client.RunPipelineResult
) -> kfp_server_api.V2beta1Run:
    return kfp_client.wait_for_run_completion(
        run_id=run_result.run_id, timeout=int(TIMEOUT_SECONDS))


def import_obj_from_file(python_path: str, obj_name: str) -> Any:
    sys.path.insert(0, os.path.dirname(python_path))
    module_name = os.path.splitext(os.path.split(python_path)[1])[0]
    module = __import__(module_name, fromlist=[obj_name])
    if not hasattr(module, obj_name):
        raise ValueError(
            f'Object "{obj_name}" not found in module {python_path}.')
    return getattr(module, obj_name)


def run(test_case: TestCase) -> Tuple[str, client.client.RunPipelineResult]:
    full_path = os.path.join(PROJECT_ROOT, test_case.module_path)
    pipeline_func = import_obj_from_file(full_path, test_case.function_name)
    run_result = kfp_client.create_run_from_pipeline_func(
        pipeline_func,
        enable_caching=True,
        arguments=test_case.arguments,
    )
    run_url = f'{KFP_ENDPOINT}/#/runs/details/{run_result.run_id}'
    print(
        f'- Created run {test_case.name}\n\tModule: {test_case.module_path}\n\tURL: {run_url}\n'
    )
    return run_url, run_result


def get_kfp_package_path() -> str:
    if os.environ.get('PULL_NUMBER') is not None:
        path = f'git+https://github.com/kubeflow/pipelines.git@refs/pull/{os.environ.get("PULL_NUMBER")}/merge#subdirectory=sdk/python'
    else:
        path = 'git+https://github.com/kubeflow/pipelines.git@master#subdirectory=sdk/python'
    print(f'Using the following KFP package path for tests: {path}')
    return path


dsl.component = functools.partial(
    dsl.component, kfp_package_path=get_kfp_package_path())


@pytest.mark.asyncio_cooperative
@pytest.mark.parametrize('test_case', create_test_case_parameters())
async def test(test_case: TestCase) -> None:
    """Asynchronously runs all samples and test that they succeed."""
    event_loop = asyncio.get_running_loop()
    try:
        run_url, run_result = run(test_case)
    except Exception as e:
        raise RuntimeError(
            f'Error triggering pipeline {test_case.name}.') from e

    api_run = await event_loop.run_in_executor(None, wait, run_result)
    assert api_run.state == 'SUCCEEDED', f'Pipeline {test_case.name} ended with incorrect status: {api_run.state}. More info: {run_url}'


if __name__ == '__main__':
    pytest.main()

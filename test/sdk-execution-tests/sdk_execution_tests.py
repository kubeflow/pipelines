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

import dataclasses
import functools
import os
import sys
import kfp_server_api
import kubernetes.client
import kubernetes.client.rest
import kubernetes.config
import pytest
import yaml
from typing import Any, Dict, List, Tuple
from minio import Minio
from kfp import client
from kfp import dsl

KFP_ENDPOINT = os.environ['KFP_ENDPOINT']
KFP_NAMESPACE = os.getenv('KFP_NAMESPACE', 'kubeflow')
TIMEOUT_SECONDS = os.environ['TIMEOUT_SECONDS']
CURRENT_DIR = os.path.abspath(os.path.dirname(__file__))
PROJECT_ROOT = os.path.abspath(
    os.path.join(CURRENT_DIR, *([os.path.pardir] * 2)))
CONFIG_PATH = os.path.join(PROJECT_ROOT, 'sdk', 'python', 'test_data',
                           'test_data_config.yaml')

METADATA_HOST = '127.0.0.1'
METADATA_PORT = 8080

kfp_client = client.Client(host=KFP_ENDPOINT)
kubernetes.config.load_kube_config()

minio_client = Minio(
    endpoint="127.0.0.1:9000",
    access_key="minio",
    secret_key="minio123",
    secure=False,
)

BUCKET_NAME = "mlpipeline"

print("Checking Minio connectivity...")
for b in minio_client.list_buckets():
    print(f"Found bucket: {b.name}")

@dataclasses.dataclass
class TestCase:
    name: str
    module_path: str
    yaml_path: str
    function_name: str
    arguments: Dict[str, Any]
    expected_state: str


def create_test_case_parameters() -> List[TestCase]:
    parameters: List[TestCase] = []
    with open(CONFIG_PATH) as f:
        config = yaml.safe_load(f)
    for name, test_group in config.items():
        test_data_dir = os.path.join(PROJECT_ROOT, test_group['test_data_dir'])

        parameters.extend(
            TestCase(
                name=name + '-' + test_case['module'],
                module_path=os.path.join(test_data_dir,
                                         f'{test_case["module"]}.py'),
                yaml_path=os.path.join(test_data_dir,
                                       f'{test_case["module"]}.yaml'),
                function_name=test_case['name'],
                arguments=test_case.get('arguments'),
                expected_state=test_case.get('expected_state', 'SUCCEEDED'),
            ) for test_case in test_group['test_cases'] if test_case['execute'])

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
    # Ensure that we are installing pipeline-spec from source during pipeline execution.
    if 'KFP_PIPELINE_SPEC_PACKAGE_PATH' not in os.environ:
        os.environ['KFP_PIPELINE_SPEC_PACKAGE_PATH'] = get_kfp_pipeline_spec_path()

    full_path = os.path.join(PROJECT_ROOT, test_case.module_path)
    pipeline_func = import_obj_from_file(full_path, test_case.function_name)
    run_result = kfp_client.create_run_from_pipeline_func(
        pipeline_func,
        enable_caching=False,
        arguments=test_case.arguments,
    )
    run_url = f'{KFP_ENDPOINT}/#/runs/details/{run_result.run_id}'
    print(
        f'- Created run {test_case.name}\n\tModule: {test_case.module_path}\n\tURL: {run_url}\n'
    )

    return run_url, run_result


def cleanup_run_resources(run_id: str):
    print(f"Cleaning up resources for run {run_id}")

    # Clean up Argo Workflow
    try:
        print(f'Deleting the Argo Workflow for run {run_id}')
        kubernetes.client.CustomObjectsApi(
        ).delete_collection_namespaced_custom_object(
            'argoproj.io',
            'v1alpha1',
            KFP_NAMESPACE,
            'workflows',
            label_selector=f'pipeline/runid={run_id}')
    except kubernetes.client.rest.ApiException as e:
        print(
            f'Failed to delete the Argo Workflow for run {run_id}: {e}',
            file=sys.stderr)


def get_kfp_package_path() -> str:
    path = get_package_path("sdk/python")
    print(f'Using the following KFP package path for tests: {path}')
    return path

def get_kfp_pipeline_spec_path() -> str:
    path = get_package_path("api/v2alpha1/python")
    print(f'Using the following KFP pipeline spec path for tests: {path}')
    return path

def get_package_path(subdir: str) -> str:
    repo_name = os.environ.get('REPO_NAME', 'kubeflow/pipelines')
    if os.environ.get('PULL_NUMBER'):
        path = f'git+https://github.com/{repo_name}.git@refs/pull/{os.environ["PULL_NUMBER"]}/merge#subdirectory={subdir}'
    else:
        path = f'git+https://github.com/{repo_name}.git@master#subdirectory={subdir}'
    return path


dsl.component = functools.partial(
    dsl.component, kfp_package_path=get_kfp_package_path(),  packages_to_install=[get_kfp_pipeline_spec_path()])


@pytest.mark.parametrize('test_case', create_test_case_parameters())
def test(test_case: TestCase) -> None:
    try:
        run_url, run_result = run(test_case)
    except Exception as e:
        raise RuntimeError(
            f'Error triggering pipeline {test_case.name}.') from e

    api_run = wait(run_result)
    assert api_run.state == test_case.expected_state, f'Pipeline {test_case.name} ended with incorrect status: {api_run.state}. More info: {run_url}'

    cleanup_run_resources(api_run.run_id)

if __name__ == '__main__':
    pytest.main()

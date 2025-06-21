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
import subprocess
import sys
import kfp_server_api
import kubernetes.client
import kubernetes.client.rest
import kubernetes.config
import pytest
import yaml
from typing import Any, Dict, List, Tuple
import boto3
from botocore.exceptions import ClientError
from ml_metadata import metadata_store
from ml_metadata.proto import metadata_store_pb2
from ml_metadata.metadata_store.metadata_store import ListOptions
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

# SeaweedFS S3-compatible client configuration
s3_client = boto3.client(
    's3',
    aws_access_key_id="minio",
    aws_secret_access_key="minio123",
    endpoint_url="http://127.0.0.1:9000",  # SeaweedFS S3 API via port-forward
    region_name='us-east-1'  # Required but not used by SeaweedFS
)

BUCKET_NAME = "mlpipeline"

print("Checking SeaweedFS connectivity...")
try:
    response = s3_client.list_buckets()
    print(f"Connected to SeaweedFS. Found {len(response.get('Buckets', []))} buckets")
    for bucket in response.get('Buckets', []):
        print(f"Found bucket: {bucket['Name']}")
except Exception as e:
    print(f"Warning: Could not connect to SeaweedFS: {e}")

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


def get_run_artifacts(run_id: str):
    mlmd_connection_config = metadata_store_pb2.MetadataStoreClientConfig(
        host=METADATA_HOST,
        port=METADATA_PORT,
    )
    mlmd_store = metadata_store.MetadataStore(mlmd_connection_config)
    contexts = mlmd_store.get_contexts(list_options=ListOptions(filter_query=f"name = '{run_id}'"))
    if len(contexts) != 1:
        print("ERROR: Unable to find pipelinerun context in MLMD", file=sys.stderr)
        return []

    context = contexts[0]
    return [a for a in mlmd_store.get_artifacts_by_context(context.id)]


def cleanup_run_resources(run_id: str):
    print(f"Cleaning up resources for run {run_id}")

    artifacts = get_run_artifacts(run_id)
    print(f"Found {len(artifacts)} artifacts for run {run_id}")
    
    # Clean up any Artifacts from SeaweedFS object store
    for artifact in artifacts:
        try:
            # Handle both minio:// and s3:// URI schemes
            uri = artifact.uri
            if uri.startswith(f"minio://{BUCKET_NAME}/"):
                object_key = uri.removeprefix(f"minio://{BUCKET_NAME}/")
            elif uri.startswith(f"s3://{BUCKET_NAME}/"):
                object_key = uri.removeprefix(f"s3://{BUCKET_NAME}/")
            else:
                print(f"Skipping artifact with unknown URI scheme: {uri}")
                continue
                
            print(f"Deleting artifact {object_key} for run {run_id}")
            s3_client.delete_object(Bucket=BUCKET_NAME, Key=object_key)
            
        except ClientError as err:
            # Handle S3/SeaweedFS specific errors
            error_code = err.response['Error']['Code'] 
            if error_code == 'NoSuchKey':
                print(f"Artifact {object_key} already deleted or doesn't exist")
            else:
                print(f"SeaweedFS error: {err} for run {run_id}")
        except Exception as err:
            print(f"Unexpected error deleting artifact: {err} for run {run_id}")

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
    repo_name = os.environ.get('REPO_NAME', 'kubeflow/pipelines')
    if os.environ.get('PULL_NUMBER'):
        path = f'git+https://github.com/{repo_name}.git@refs/pull/{os.environ["PULL_NUMBER"]}/merge#subdirectory=sdk/python'
    else:
        path = f'git+https://github.com/{repo_name}.git@master#subdirectory=sdk/python'
    print(f'Using the following KFP package path for tests: {path}')
    return path


dsl.component = functools.partial(
    dsl.component, kfp_package_path=get_kfp_package_path())


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

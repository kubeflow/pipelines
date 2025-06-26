# Copyright 2024 The Kubeflow Authors
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
from concurrent.futures import as_completed
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
import inspect
import os
from pprint import pprint
from typing import List
import unittest

import collected_parameters
import component_with_optional_inputs
import hello_world
import kfp
from kfp.dsl.graph_component import GraphComponent
from kubernetes import client
from kubernetes import config
from kubernetes import utils
from modelcar import modelcar
import parallel_after_dependency
import parallel_consume_upstream
import pipeline_container_no_input
import pipeline_with_env
import pipeline_with_placeholders
import pipeline_with_secret_as_env
import pipeline_with_secret_as_volume
import producer_consumer_param
import subdagio
import two_step_pipeline_containerized
import yaml
import pipeline_with_retry
import pipeline_with_input_status_state

_MINUTE = 60  # seconds
_DEFAULT_TIMEOUT = 10 * _MINUTE
SAMPLES_DIR = os.path.realpath(os.path.dirname(os.path.dirname(__file__)))
PRE_REQ_DIR = os.path.join(SAMPLES_DIR, 'v2', 'pre-requisites')
PREREQS = [os.path.join(PRE_REQ_DIR, 'test-secrets.yaml')]

_KFP_NAMESPACE = os.getenv('KFP_NAMESPACE', 'kubeflow')


@dataclass
class TestCase:
    pipeline_func: GraphComponent
    timeout: int = _DEFAULT_TIMEOUT


def deploy_k8s_yaml(namespace: str, yaml_file: str):
    config.load_kube_config()
    api_client = client.ApiClient()
    try:
        utils.create_from_yaml(api_client, yaml_file, namespace=namespace)
        print(f'Resource(s) from {yaml_file} deployed successfully.')
    except Exception as e:
        raise RuntimeError(f'Exception when deploying from YAML: {e}')


def delete_k8s_yaml(namespace: str, yaml_file: str):
    config.load_kube_config()
    v1 = client.CoreV1Api()
    apps_v1 = client.AppsV1Api()

    try:
        with open(yaml_file, 'r') as f:
            yaml_docs = yaml.safe_load_all(f)

            for doc in yaml_docs:
                if not doc:
                    continue  # Skip empty documents

                kind = doc.get('kind', '').lower()
                name = doc['metadata']['name']

                print(f'Deleting {kind} named {name}...')

                # There's no utils.delete_from_yaml
                # as a workaround we manually fetch required data
                if kind == 'deployment':
                    apps_v1.delete_namespaced_deployment(name, namespace)
                elif kind == 'service':
                    v1.delete_namespaced_service(name, namespace)
                elif kind == 'configmap':
                    v1.delete_namespaced_config_map(name, namespace)
                elif kind == 'pod':
                    v1.delete_namespaced_pod(name, namespace)
                elif kind == 'secret':
                    v1.delete_namespaced_secret(name, namespace)
                elif kind == 'persistentvolumeclaim':
                    v1.delete_namespaced_persistent_volume_claim(
                        name, namespace)
                elif kind == 'namespace':
                    client.CoreV1Api().delete_namespace(name)
                else:
                    print(f'Skipping unsupported resource type: {kind}')

        print(f'Resource(s) from {yaml_file} deleted successfully.')
    except Exception as e:
        print(f'Exception when deleting from YAML: {e}')


class SampleTest(unittest.TestCase):
    _kfp_host_and_port = os.getenv('KFP_API_HOST_AND_PORT',
                                   'http://localhost:8888')
    _kfp_ui_and_port = os.getenv('KFP_UI_HOST_AND_PORT',
                                 'http://localhost:8080')
    _client = kfp.Client(host=_kfp_host_and_port, ui_host=_kfp_ui_and_port)

    @classmethod
    def setUpClass(cls):
        """Runs once before all tests."""
        print('Deploying pre-requisites....')
        for p in PREREQS:
            deploy_k8s_yaml(_KFP_NAMESPACE, p)
        print('Done deploying pre-requisites.')

    @classmethod
    def tearDownClass(cls):
        """Runs once after all tests in this class."""
        print('Cleaning up resources....')
        for p in PREREQS:
            delete_k8s_yaml(_KFP_NAMESPACE, p)
        print('Done clean up.')

    def test(self):
        test_cases: List[TestCase] = [
            TestCase(pipeline_func=hello_world.pipeline_hello_world),
            TestCase(pipeline_func=producer_consumer_param
                     .producer_consumer_param_pipeline),
            TestCase(pipeline_func=pipeline_container_no_input
                     .pipeline_container_no_input),
            TestCase(pipeline_func=two_step_pipeline_containerized
                     .two_step_pipeline_containerized),
            TestCase(pipeline_func=component_with_optional_inputs.pipeline),
            TestCase(pipeline_func=pipeline_with_env.pipeline_with_env),

            # The following tests are not working. Tracking issue: https://github.com/kubeflow/pipelines/issues/11053
            # TestCase(pipeline_func=pipeline_with_importer.pipeline_with_importer),
            # TestCase(pipeline_func=pipeline_with_volume.pipeline_with_volume),
            TestCase(pipeline_func=pipeline_with_secret_as_volume
                     .pipeline_secret_volume),
            TestCase(
                pipeline_func=pipeline_with_secret_as_env.pipeline_secret_env),
            TestCase(pipeline_func=subdagio.parameter.crust),
            TestCase(pipeline_func=subdagio.parameter_cache.crust),
            TestCase(pipeline_func=subdagio.mixed_parameters.crust),
            TestCase(
                pipeline_func=subdagio.multiple_parameters_namedtuple.crust),
            TestCase(pipeline_func=subdagio.parameter_oneof.crust),
            TestCase(pipeline_func=subdagio.artifact_cache.crust),
            TestCase(pipeline_func=subdagio.artifact.crust),
            TestCase(
                pipeline_func=subdagio.multiple_artifacts_namedtuple.crust),
            TestCase(pipeline_func=pipeline_with_placeholders
                     .pipeline_with_placeholders),
            TestCase(pipeline_func=modelcar.pipeline_modelcar_test),
            TestCase(
                pipeline_func=parallel_consume_upstream.loop_consume_upstream),
            TestCase(pipeline_func=parallel_after_dependency
                     .loop_with_after_dependency_set),
            TestCase(
                pipeline_func=collected_parameters.collected_param_pipeline),
            TestCase(pipeline_func=pipeline_with_retry.retry_pipeline),
            TestCase(pipeline_func=pipeline_with_input_status_state.status_state_pipeline),
        ]

        with ThreadPoolExecutor() as executor:
            futures = [
                executor.submit(self.run_test_case, test_case.pipeline_func,
                                test_case.timeout) for test_case in test_cases
            ]
            for future in as_completed(futures):
                future.result()

    def run_test_case(self, pipeline_func: GraphComponent, timeout: int):
        with self.subTest(pipeline=pipeline_func, msg=pipeline_func.name):
            print(
                f'Running pipeline: {inspect.getmodule(pipeline_func.pipeline_func).__name__}/{pipeline_func.name}.'
            )
            run_result = self._client.create_run_from_pipeline_func(
                pipeline_func=pipeline_func)

            run_response = run_result.wait_for_run_completion(timeout)

            pprint(run_response.run_details)
            print('Run details page URL:')
            print(
                f'{self._kfp_ui_and_port}/#/runs/details/{run_response.run_id}')

            self.assertEqual(run_response.state, 'SUCCEEDED')
            print(
                f'Pipeline, {inspect.getmodule(pipeline_func.pipeline_func).__name__}/{pipeline_func.name}, succeeded.'
            )


if __name__ == '__main__':
    unittest.main()

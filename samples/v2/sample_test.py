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
import os
import unittest
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from pprint import pprint
from typing import List

import kfp
from kfp.dsl.graph_component import GraphComponent
import component_with_optional_inputs
import pipeline_with_env
import hello_world
import producer_consumer_param
import pipeline_container_no_input
import two_step_pipeline_containerized

_MINUTE = 60  # seconds
_DEFAULT_TIMEOUT = 5 * _MINUTE


@dataclass
class TestCase:
    pipeline_func: GraphComponent
    timeout: int = _DEFAULT_TIMEOUT


class SampleTest(unittest.TestCase):
    _kfp_host_and_port = os.getenv('KFP_API_HOST_AND_PORT', 'http://localhost:8888')
    _kfp_ui_and_port = os.getenv('KFP_UI_HOST_AND_PORT', 'http://localhost:8080')
    _client = kfp.Client(host=_kfp_host_and_port, ui_host=_kfp_ui_and_port)

    def test(self):
        test_cases: List[TestCase] = [
            TestCase(pipeline_func=hello_world.pipeline_hello_world),
            TestCase(pipeline_func=producer_consumer_param.producer_consumer_param_pipeline),
            TestCase(pipeline_func=pipeline_container_no_input.pipeline_container_no_input),
            TestCase(pipeline_func=two_step_pipeline_containerized.two_step_pipeline_containerized),
            TestCase(pipeline_func=component_with_optional_inputs.pipeline),
            TestCase(pipeline_func=pipeline_with_env.pipeline_with_env),

            # The following tests are not working. Tracking issue: https://github.com/kubeflow/pipelines/issues/11053
            # TestCase(pipeline_func=pipeline_with_importer.pipeline_with_importer),
            # TestCase(pipeline_func=pipeline_with_volume.pipeline_with_volume),
            # TestCase(pipeline_func=pipeline_with_secret_as_volume.pipeline_secret_volume),
            # TestCase(pipeline_func=pipeline_with_secret_as_env.pipeline_secret_env),
        ]

        with ThreadPoolExecutor() as executor:
            futures = [
                executor.submit(self.run_test_case, test_case.pipeline_func, test_case.timeout)
                for test_case in test_cases
            ]
            for future in as_completed(futures):
                future.result()

    def run_test_case(self, pipeline_func: GraphComponent, timeout: int):
        with self.subTest(pipeline=pipeline_func, msg=pipeline_func.name):
            run_result = self._client.create_run_from_pipeline_func(pipeline_func=pipeline_func)

            run_response = run_result.wait_for_run_completion(timeout)

            pprint(run_response.run_details)
            print("Run details page URL:")
            print(f"{self._kfp_ui_and_port}/#/runs/details/{run_response.run_id}")

            self.assertEqual(run_response.state, "SUCCEEDED")


if __name__ == '__main__':
    unittest.main()

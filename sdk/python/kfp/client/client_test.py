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

import unittest
import os
import yaml

from absl.testing import parameterized
from kfp.client import client


class TestValidatePipelineName(parameterized.TestCase):

    @parameterized.parameters([
        'pipeline',
        'my-pipeline',
        'my-pipeline-1',
        '1pipeline',
        'pipeline1',
    ])
    def test_valid(self, name: str):
        client.validate_pipeline_resource_name(name)

    @parameterized.parameters([
        'my_pipeline',
        "person's-pipeline",
        'my pipeline',
        'pipeline.yaml',
    ])
    def test_invalid(self, name: str):
        with self.assertRaisesRegex(ValueError, r'Invalid pipeline name:'):
            client.validate_pipeline_resource_name(name)


PIPELINES_TEST_DATA_DIR = os.path.join(
    os.path.abspath(os.path.join(__file__, "../..")),
    'compiler', 'test_data', 'pipelines')

PIPELINE_TEST_CASES = [
    'pipeline_with_importer',
    'pipeline_with_ontology',
    'pipeline_with_if_placeholder',
    'pipeline_with_concat_placeholder',
    'pipeline_with_resource_spec',
    'pipeline_with_various_io_types',
    'pipeline_with_reused_component',
    'pipeline_with_after',
    'pipeline_with_condition',
    'pipeline_with_nested_conditions',
    'pipeline_with_nested_conditions_yaml',
    'pipeline_with_loops',
    'pipeline_with_nested_loops',
    'pipeline_with_loops_and_conditions',
    'pipeline_with_params_containing_format',
    'lightweight_python_functions_pipeline',
    'lightweight_python_functions_with_outputs',
    'xgboost_sample_pipeline',
    'pipeline_with_metrics_outputs',
    'pipeline_with_exit_handler',
    'pipeline_with_env',
    'component_with_optional_inputs',
    'pipeline_with_gcpc_types',
    'pipeline_with_placeholders',
    'pipeline_with_task_final_status',
    'pipeline_with_task_final_status_yaml',
    'component_with_pip_index_urls',
]
class TestOverrideCachingOptions(parameterized.TestCase):
    @parameterized.parameters(PIPELINE_TEST_CASES)
    def test_override_caching(self, pipeline_base_name: str):
        pipeline_path = os.path.join(PIPELINES_TEST_DATA_DIR, f'{pipeline_base_name}.yaml')
        with open(pipeline_path) as f:
            yaml_dict = yaml.safe_load(f)
            test_client = client.Client(namespace='dummy_namespace')
            test_client._override_caching_options(yaml_dict, False)
            for task in yaml_dict['root']['dag']['tasks']:
                if 'cachingOptions' in yaml_dict:
                    assert yaml_dict['root']['dag']['tasks'][task]['cachingOptions']['enableCache'] == False

if __name__ == '__main__':
    unittest.main()

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

CONFIG = {
    'pipelines': {
        'test_cases': [
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
            'container_component_with_no_inputs',
            'two_step_pipeline_containerized',
            'pipeline_with_multiple_exit_handlers',
        ],
        'test_data_dir': 'sdk/python/kfp/compiler/test_data/pipelines',
        'config': {
            'read': False,
            'write': True
        }
    },
    'components': {
        'test_cases': [
            'add_numbers',
            'component_with_pip_install',
            'concat_message',
            'dict_input',
            'identity',
            'input_artifact',
            'nested_return',
            'output_metrics',
            'preprocess',
            'container_no_input',
            'container_io',
            'container_with_artifact_output',
            'container_with_concat_placeholder',
            'container_with_if_placeholder',
        ],
        'test_data_dir': 'sdk/python/kfp/compiler/test_data/components',
        'config': {
            'read': True,
            'write': True
        }
    },
    'v1_components': {
        'test_cases': [
            'concat_placeholder_component',
            'if_placeholder_component',
            'add_component',
            'ingestion_component',
        ],
        'test_data_dir': 'sdk/python/kfp/compiler/test_data/v1_component_yaml',
        'config': {
            'read': True,
            'write': False
        }
    }
}

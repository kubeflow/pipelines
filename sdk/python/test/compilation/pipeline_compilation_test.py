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
import os.path
import tempfile
from typing import Callable, Optional

from kfp.compiler import Compiler
import pytest

from test_data.sdk_compiled_pipelines.valid.arguments_parameters import \
    echo as arguments_parameters_echo
from test_data.sdk_compiled_pipelines.valid.artifacts_complex import \
    math_pipeline as artifacts_complex_pipeline
from test_data.sdk_compiled_pipelines.valid.artifacts_simple import \
    math_pipeline as artifacts_simple_pipeline
from test_data.sdk_compiled_pipelines.valid.collected_artifacts import \
    collected_artifact_pipeline
from test_data.sdk_compiled_pipelines.valid.component_with_metadata_fields import \
    dataset_joiner
from test_data.sdk_compiled_pipelines.valid.component_with_task_final_status import \
    exit_comp as task_final_status_pipeline
from test_data.sdk_compiled_pipelines.valid.components_with_optional_artifacts import \
    pipeline as optional_artifacts_pipeline
from test_data.sdk_compiled_pipelines.valid.conditional_producer_and_consumers import \
    math_pipeline as conditional_producer_consumers_pipeline
from test_data.sdk_compiled_pipelines.valid.container_io import container_io
from test_data.sdk_compiled_pipelines.valid.container_with_artifact_output import \
    container_with_artifact_output
from test_data.sdk_compiled_pipelines.valid.container_with_concat_placeholder import \
    container_with_concat_placeholder
from test_data.sdk_compiled_pipelines.valid.container_with_if_placeholder import \
    container_with_if_placeholder
from test_data.sdk_compiled_pipelines.valid.container_with_if_placeholder import \
    container_with_if_placeholder as pipeline_with_if_placeholder_pipeline
from test_data.sdk_compiled_pipelines.valid.container_with_placeholder_in_fstring import \
    container_with_placeholder_in_fstring
from test_data.sdk_compiled_pipelines.valid.containerized_python_component import \
    concat_message as containerized_concat_message
from test_data.sdk_compiled_pipelines.valid.create_pod_metadata_complex import \
    pipeline_with_pod_metadata as create_pod_metadata_complex
from test_data.sdk_compiled_pipelines.valid.critical.add_numbers import \
    add_numbers
from test_data.sdk_compiled_pipelines.valid.critical.artifact_cache import \
    crust as artifact_cache_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.artifact_crust import \
    crust as artifact_crust_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.collected_parameters import \
    collected_param_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.component_with_optional_inputs import \
    pipeline
from test_data.sdk_compiled_pipelines.valid.critical.container_component_with_no_inputs import \
    pipeline as container_no_inputs_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.flip_coin import \
    flipcoin_pipeline as flip_coin
from test_data.sdk_compiled_pipelines.valid.critical.loop_consume_upstream import \
    loop_consume_upstream
from test_data.sdk_compiled_pipelines.valid.critical.missing_kubernetes_optional_inputs import \
    missing_kubernetes_optional_inputs_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.mixed_parameters import \
    crust as mixed_parameters_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.modelcar.modelcar import \
    pipeline_modelcar_test
from test_data.sdk_compiled_pipelines.valid.critical.multiple_artifacts_namedtuple import \
    crust as multiple_artifacts_namedtuple_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.multiple_parameters_namedtuple import \
    crust as multiple_params_namedtuple_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.nested_pipeline_opt_input_child_level import \
    nested_pipeline_opt_input_child_level
from test_data.sdk_compiled_pipelines.valid.critical.nested_pipeline_opt_inputs_nil import \
    nested_pipeline_opt_inputs_nil
from test_data.sdk_compiled_pipelines.valid.critical.nested_pipeline_opt_inputs_parent_level import \
    nested_pipeline_opt_inputs_parent_level
from test_data.sdk_compiled_pipelines.valid.critical.parallel_for_after_dependency import \
    loop_with_after_dependency_set
from test_data.sdk_compiled_pipelines.valid.critical.parameter_cache import \
    crust as parameter_cache_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.parameter_oneof import \
    crust as parameter_oneof_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.parameters_simple import \
    math_pipeline as parameters_simple_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.pipeline_with_artifact_upload_download import \
    my_pipeline as artifact_upload_download_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.pipeline_with_env import \
    my_pipeline as env_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.pipeline_with_importer_workspace import \
    pipeline_with_importer_workspace
from test_data.sdk_compiled_pipelines.valid.critical.pipeline_with_input_status_state import \
    status_state_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.pipeline_with_placeholders import \
    pipeline_with_placeholders
from test_data.sdk_compiled_pipelines.valid.critical.pipeline_with_pod_metadata import \
    pipeline_with_pod_metadata
from test_data.sdk_compiled_pipelines.valid.critical.pipeline_with_secret_as_env import \
    pipeline_secret_env
from test_data.sdk_compiled_pipelines.valid.critical.pipeline_with_workspace import \
    pipeline_with_workspace
from test_data.sdk_compiled_pipelines.valid.critical.producer_consumer_param import \
    producer_consumer_param_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.pythonic_artifacts_test_pipeline import \
    pythonic_artifacts_test_pipeline
from test_data.sdk_compiled_pipelines.valid.critical.two_step_pipeline_containerized import \
    my_pipeline as two_step_containerized_pipeline
from test_data.sdk_compiled_pipelines.valid.cross_loop_after_topology import \
    my_pipeline as cross_loop_after_topology_pipeline
from test_data.sdk_compiled_pipelines.valid.dict_input import dict_input
from test_data.sdk_compiled_pipelines.valid.env_var import test_env_exists
from test_data.sdk_compiled_pipelines.valid.essential.component_with_pip_index_urls import \
    pipeline as pip_index_urls_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.component_with_pip_install import \
    component_with_pip_install as pip_install_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.component_with_pip_install_in_venv import \
    component_with_pip_install as pip_install_venv_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.concat_message import \
    concat_message
from test_data.sdk_compiled_pipelines.valid.essential.container_no_input import \
    container_no_input
from test_data.sdk_compiled_pipelines.valid.essential.lightweight_python_functions_pipeline import \
    pipeline as lightweight_python_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.lightweight_python_functions_with_outputs import \
    pipeline as lightweight_python_with_outputs_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_in_pipeline import \
    my_pipeline as pipeline_in_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_in_pipeline_complex import \
    my_pipeline as pipeline_in_pipeline_complex
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_in_pipeline_loaded_from_yaml import \
    my_pipeline as pipeline_in_pipeline_loaded_from_yaml
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_with_after import \
    my_pipeline as after_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_with_condition import \
    my_pipeline as condition_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_with_if_placeholder import \
    pipeline_none
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_with_loops import \
    my_pipeline as loops_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_with_metrics_outputs import \
    my_pipeline as metrics_outputs_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_with_nested_conditions import \
    my_pipeline as nested_conditions_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_with_nested_conditions_yaml import \
    my_pipeline as pipeline_with_nested_conditions_yaml
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_with_outputs import \
    my_pipeline as outputs_pipeline
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_with_params_containing_format import \
    my_pipeline as pipeline_with_params_containing_format
from test_data.sdk_compiled_pipelines.valid.essential.pipeline_with_reused_component import \
    my_pipeline as reused_component_pipeline
from test_data.sdk_compiled_pipelines.valid.failing.pipeline_with_exit_handler import \
    my_pipeline as exit_handler_pipeline
from test_data.sdk_compiled_pipelines.valid.failing.pipeline_with_multiple_exit_handlers import \
    my_pipeline as multiple_exit_handlers_pipeline
from test_data.sdk_compiled_pipelines.valid.hello_world import echo
from test_data.sdk_compiled_pipelines.valid.identity import identity
from test_data.sdk_compiled_pipelines.valid.if_elif_else_complex import \
    lucky_number_pipeline
from test_data.sdk_compiled_pipelines.valid.if_elif_else_with_oneof_parameters import \
    outer_pipeline as if_elif_else_oneof_params_pipeline
from test_data.sdk_compiled_pipelines.valid.if_else_with_oneof_artifacts import \
    outer_pipeline as if_else_oneof_artifacts_pipeline
from test_data.sdk_compiled_pipelines.valid.if_else_with_oneof_parameters import \
    flip_coin_pipeline as if_else_oneof_params_pipeline
from test_data.sdk_compiled_pipelines.valid.input_artifact import \
    input_artifact
from test_data.sdk_compiled_pipelines.valid.long_running import \
    wait_awhile as long_running_pipeline
from test_data.sdk_compiled_pipelines.valid.metrics_visualization_v2 import \
    metrics_visualization_pipeline
from test_data.sdk_compiled_pipelines.valid.nested_return import nested_return
from test_data.sdk_compiled_pipelines.valid.nested_with_parameters import \
    math_pipeline as nested_with_parameters_pipeline
from test_data.sdk_compiled_pipelines.valid.output_metrics import \
    output_metrics
from test_data.sdk_compiled_pipelines.valid.parameters_complex import \
    math_pipeline as parameters_complex_pipeline
from test_data.sdk_compiled_pipelines.valid.pipeline_as_exit_task import \
    my_pipeline as pipeline_as_exit_task
from test_data.sdk_compiled_pipelines.valid.pipeline_producer_consumer import \
    math_pipeline as producer_consumer_parallel_for_pipeline
from test_data.sdk_compiled_pipelines.valid.pipeline_with_concat_placeholder import \
    pipeline_with_concat_placeholder as \
    pipeline_with_concat_placeholder_pipeline
# Final batch of remaining missing pipeline imports
from test_data.sdk_compiled_pipelines.valid.pipeline_with_condition_dynamic_task_output_custom_training_job import \
    pipeline_with_dynamic_condition_output
from test_data.sdk_compiled_pipelines.valid.pipeline_with_dynamic_importer_metadata import \
    my_pipeline as pipeline_with_dynamic_importer_metadata
from test_data.sdk_compiled_pipelines.valid.pipeline_with_google_artifact_type import \
    my_pipeline as pipeline_with_google_artifact_type
from test_data.sdk_compiled_pipelines.valid.pipeline_with_importer import \
    my_pipeline as importer_pipeline
from test_data.sdk_compiled_pipelines.valid.pipeline_with_importer_and_gcpc_types import \
    my_pipeline as pipeline_with_importer_and_gcpc_types
from test_data.sdk_compiled_pipelines.valid.pipeline_with_loops_and_conditions import \
    my_pipeline as loops_and_conditions_pipeline
from test_data.sdk_compiled_pipelines.valid.pipeline_with_metadata_fields import \
    dataset_concatenator as pipeline_with_metadata_fields
from test_data.sdk_compiled_pipelines.valid.pipeline_with_nested_loops import \
    my_pipeline as nested_loops_pipeline
from test_data.sdk_compiled_pipelines.valid.pipeline_with_parallelfor_list_artifacts import \
    my_pipeline as pipeline_with_parallelfor_list_artifacts
from test_data.sdk_compiled_pipelines.valid.pipeline_with_parallelfor_parallelism import \
    my_pipeline as pipeline_with_parallelfor_parallelism
from test_data.sdk_compiled_pipelines.valid.pipeline_with_retry import \
    my_pipeline as retry_pipeline
from test_data.sdk_compiled_pipelines.valid.pipeline_with_secret_as_volume import \
    pipeline_secret_volume
from test_data.sdk_compiled_pipelines.valid.pipeline_with_string_machine_fields_pipeline_input import \
    pipeline as pipeline_with_string_machine_fields_pipeline_input
from test_data.sdk_compiled_pipelines.valid.pipeline_with_string_machine_fields_task_output import \
    pipeline as pipeline_with_string_machine_fields_task_output
from test_data.sdk_compiled_pipelines.valid.pipeline_with_task_final_status import \
    my_pipeline as pipeline_with_task_final_status
from test_data.sdk_compiled_pipelines.valid.pipeline_with_task_using_ignore_upstream_failure import \
    my_pipeline as pipeline_with_task_using_ignore_upstream_failure
from test_data.sdk_compiled_pipelines.valid.pipeline_with_utils import \
    pipeline_with_utils
from test_data.sdk_compiled_pipelines.valid.pipeline_with_various_io_types import \
    my_pipeline as various_io_types_pipeline
from test_data.sdk_compiled_pipelines.valid.pipeline_with_volume import \
    pipeline_with_volume
from test_data.sdk_compiled_pipelines.valid.pipeline_with_volume_no_cache import \
    pipeline_with_volume_no_cache
from test_data.sdk_compiled_pipelines.valid.preprocess import preprocess
from test_data.sdk_compiled_pipelines.valid.pythonic_artifact_with_single_return import \
    make_language_model_pipeline as pythonic_artifact_with_single_return
from test_data.sdk_compiled_pipelines.valid.pythonic_artifacts_with_list_of_artifacts import \
    make_and_join_datasets as pythonic_artifacts_with_list_of_artifacts
from test_data.sdk_compiled_pipelines.valid.pythonic_artifacts_with_multiple_returns import \
    split_datasets_and_return_first as pythonic_artifacts_multiple_returns
from test_data.sdk_compiled_pipelines.valid.sequential_v2 import sequential
from test_data.sdk_compiled_pipelines.valid.two_step_pipeline import \
    my_pipeline as two_step_pipeline
from test_data.sdk_compiled_pipelines.valid.xgboost_sample_pipeline import \
    xgboost_pipeline

from ..test_utils.comparison_utils import ComparisonUtils
from ..test_utils.file_utils import FileUtils


@pytest.mark.compilation
@pytest.mark.regression
class TestPipelineCompilation:
    _VALID_PIPELINE_FILES = FileUtils.VALID_PIPELINE_FILES

    @dataclass
    class TestData:
        pipeline_name: str
        pipeline_func: Callable
        pipline_func_args: Optional[dict]
        compiled_file_name: str
        expected_compiled_file_path: str

        def __str__(self) -> str:
            return (f"Compilation Data: name={self.pipeline_name} "
                    f"compiled_file_name={self.compiled_file_name} "
                    f"expected_file={self.expected_compiled_file_path}")

        def __repr__(self) -> str:
            return self.__str__()

    @pytest.mark.parametrize(
        'pipeline_data', [
            TestData(
                pipeline_name='add-numbers',
                pipeline_func=add_numbers,
                pipline_func_args=None,
                compiled_file_name='add_numbers.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/add_numbers.yaml'
            ),
            TestData(
                pipeline_name='hello-world',
                pipeline_func=echo,
                pipline_func_args=None,
                compiled_file_name='hello_world.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/hello_world.yaml'
            ),
            TestData(
                pipeline_name='simple-two-step-pipeline',
                pipeline_func=two_step_pipeline,
                pipline_func_args={'text': 'Hello KFP!'},
                compiled_file_name='two_step_pipeline.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/two_step_pipeline.yaml'
            ),
            TestData(
                pipeline_name='single-condition-pipeline',
                pipeline_func=condition_pipeline,
                pipline_func_args=None,
                compiled_file_name='condition_pipeline.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_with_condition.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-loops',
                pipeline_func=loops_pipeline,
                pipline_func_args={'loop_parameter': ['item1', 'item2']},
                compiled_file_name='loops_pipeline.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_with_loops.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-outputs',
                pipeline_func=outputs_pipeline,
                pipline_func_args=None,
                compiled_file_name='outputs_pipeline.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_with_outputs.yaml'
            ),
            TestData(
                pipeline_name='collected-param-pipeline',
                pipeline_func=collected_param_pipeline,
                pipline_func_args=None,
                compiled_file_name='collected_parameters.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/collected_parameters.yaml'
            ),
            TestData(
                pipeline_name='component-optional-input',
                pipeline_func=pipeline,
                pipline_func_args=None,
                compiled_file_name='component_with_optional_inputs.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/component_with_optional_inputs.yaml'
            ),
            TestData(
                pipeline_name='mixed_parameters-pipeline',
                pipeline_func=mixed_parameters_pipeline,
                pipline_func_args=None,
                compiled_file_name='mixed_parameters.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/mixed_parameters.yaml'
            ),
            TestData(
                pipeline_name='producer-consumer-param-pipeline',
                pipeline_func=producer_consumer_param_pipeline,
                pipline_func_args=None,
                compiled_file_name='producer_consumer_param_pipeline.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/producer_consumer_param_pipeline.yaml'
            ),
            TestData(
                pipeline_name='parameter_cache-pipeline',
                pipeline_func=parameter_cache_pipeline,
                pipline_func_args=None,
                compiled_file_name='parameter_cache.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/parameter_cache.yaml'
            ),
            TestData(
                pipeline_name='parameter_oneof-pipeline',
                pipeline_func=parameter_oneof_pipeline,
                pipline_func_args=None,
                compiled_file_name='parameter_oneof.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/parameter_oneof.yaml'
            ),
            TestData(
                pipeline_name='missing-kubernetes-optional-inputs',
                pipeline_func=missing_kubernetes_optional_inputs_pipeline,
                pipline_func_args=None,
                compiled_file_name='missing_kubernetes_optional_inputs.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/missing_kubernetes_optional_inputs.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-env',
                pipeline_func=env_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_env.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_env.yaml'
            ),
            TestData(
                pipeline_name='test-pipeline',
                pipeline_func=retry_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_retry.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_retry.yaml'
            ),
            TestData(
                pipeline_name='loop-with-after-dependency-set',
                pipeline_func=loop_with_after_dependency_set,
                pipline_func_args=None,
                compiled_file_name='parallel_for_after_dependency.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/parallel_for_after_dependency.yaml'
            ),
            TestData(
                pipeline_name='multiple_parameters_namedtuple-pipeline',
                pipeline_func=multiple_params_namedtuple_pipeline,
                pipline_func_args=None,
                compiled_file_name='multiple_parameters_namedtuple.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/multiple_parameters_namedtuple.yaml'
            ),
            TestData(
                pipeline_name='multiple_artifacts_namedtuple-pipeline',
                pipeline_func=multiple_artifacts_namedtuple_pipeline,
                pipline_func_args=None,
                compiled_file_name='multiple_artifacts_namedtuple.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/multiple_artifacts_namedtuple.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-modelcar-model',
                pipeline_func=pipeline_modelcar_test,
                pipline_func_args=None,
                compiled_file_name='modelcar.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/modelcar.yaml'
            ),
            TestData(
                pipeline_name='artifact_cache-pipeline',
                pipeline_func=artifact_cache_pipeline,
                pipline_func_args=None,
                compiled_file_name='artifact_cache.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/artifact_cache.yaml'
            ),
            TestData(
                pipeline_name='artifact_crust-pipeline',
                pipeline_func=artifact_crust_pipeline,
                pipline_func_args=None,
                compiled_file_name='artifact_crust.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/artifact_crust.yaml'
            ),
            TestData(
                pipeline_name='v2-container-component-no-input',
                pipeline_func=container_no_inputs_pipeline,
                pipline_func_args=None,
                compiled_file_name='container_component_with_no_inputs.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/container_component_with_no_inputs.yaml'
            ),
            TestData(
                pipeline_name='loop-consume-upstream',
                pipeline_func=loop_consume_upstream,
                pipline_func_args=None,
                compiled_file_name='loop_consume_upstream.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/loop_consume_upstream.yaml'
            ),
            TestData(
                pipeline_name='math-pipeline',
                pipeline_func=parameters_simple_pipeline,
                pipline_func_args=None,
                compiled_file_name='parameters_simple.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/parameters_simple.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-datasets',
                pipeline_func=artifact_upload_download_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_artifact_upload_download.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_artifact_upload_download.yaml'
            ),
            TestData(
                pipeline_name='status-state-pipeline',
                pipeline_func=status_state_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_input_status_state.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_input_status_state.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-placeholders',
                pipeline_func=pipeline_with_placeholders,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_placeholders.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_placeholders.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-pod-metadata',
                pipeline_func=pipeline_with_pod_metadata,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_pod_metadata.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_pod_metadata.yaml'
            ),
            TestData(
                pipeline_name='pipeline-secret-env',
                pipeline_func=pipeline_secret_env,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_secret_as_env.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_secret_as_env.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-workspace',
                pipeline_func=pipeline_with_workspace,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_workspace.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_workspace.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-importer-workspace',
                pipeline_func=pipeline_with_importer_workspace,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_importer_workspace.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_importer_workspace.yaml'
            ),
            TestData(
                pipeline_name='containerized-two-step-pipeline',
                pipeline_func=two_step_containerized_pipeline,
                pipline_func_args={'text': 'Hello KFP Containerized!'},
                compiled_file_name='two_step_pipeline_containerized.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/two_step_pipeline_containerized.yaml'
            ),
            TestData(
                pipeline_name='nested-pipeline-opt-input-child-level',
                pipeline_func=nested_pipeline_opt_input_child_level,
                pipline_func_args=None,
                compiled_file_name='nested_pipeline_opt_input_child_level.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/nested_pipeline_opt_input_child_level_compiled.yaml'
            ),
            TestData(
                pipeline_name='split-datasets-and-return-first',
                pipeline_func=pythonic_artifacts_test_pipeline,
                pipline_func_args=None,
                compiled_file_name='pythonic_artifacts_test_pipeline.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pythonic_artifacts_test_pipeline.yaml'
            ),
            TestData(
                pipeline_name='optional-artifact-pipeline',
                pipeline_func=optional_artifacts_pipeline,
                pipline_func_args=None,
                compiled_file_name='components_with_optional_artifacts.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/components_with_optional_artifacts.yaml'
            ),
            TestData(
                pipeline_name='my-test-pipeline-beta',
                pipeline_func=lightweight_python_pipeline,
                pipline_func_args={
                    'message': 'Hello KFP!',
                    'input_dict': {
                        'A': 1,
                        'B': 2
                    }
                },
                compiled_file_name='lightweight_python_functions_pipeline.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/lightweight_python_functions_pipeline.yaml'
            ),
            TestData(
                pipeline_name='xgboost-sample-pipeline',
                pipeline_func=xgboost_pipeline,
                pipline_func_args=None,
                compiled_file_name='xgboost_sample_pipeline.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/xgboost_sample_pipeline.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-after',
                pipeline_func=after_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_after.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_with_after.yaml'
            ),
            TestData(
                pipeline_name='metrics-visualization-pipeline',
                pipeline_func=metrics_visualization_pipeline,
                pipline_func_args=None,
                compiled_file_name='metrics_visualization_v2.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/metrics_visualization_v2.yaml'
            ),
            TestData(
                pipeline_name='nested-conditions-pipeline',
                pipeline_func=nested_conditions_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_nested_conditions.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_with_nested_conditions.yaml'
            ),
            TestData(
                pipeline_name='container-io',
                pipeline_func=container_io,
                pipline_func_args={'text': 'Hello Container!'},
                compiled_file_name='container_io.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_io.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-exit-handler',
                pipeline_func=exit_handler_pipeline,
                pipline_func_args={'message': 'Hello Exit Handler!'},
                compiled_file_name='pipeline_with_exit_handler.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/failing/pipeline_with_exit_handler.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-importer',
                pipeline_func=importer_pipeline,
                pipline_func_args={
                    'dataset2': 'gs://ml-pipeline-playground/shakespeare2.txt'
                },
                compiled_file_name='pipeline_with_importer.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_importer.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-nested-loops',
                pipeline_func=nested_loops_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_nested_loops.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_nested_loops.yaml'
            ),
            TestData(
                pipeline_name='concat-message',
                pipeline_func=concat_message,
                pipline_func_args={
                    'message1': 'Hello',
                    'message2': ' World!'
                },
                compiled_file_name='concat_message.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/concat_message.yaml'
            ),
            TestData(
                pipeline_name='preprocess',
                pipeline_func=preprocess,
                pipline_func_args={
                    'message': 'test',
                    'input_dict_parameter': {
                        'A': 1
                    },
                    'input_list_parameter': ['a', 'b']
                },
                compiled_file_name='preprocess.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/preprocess.yaml'
            ),
            TestData(
                pipeline_name='sequential',
                pipeline_func=sequential,
                pipline_func_args={'url': 'gs://sample-data/test.txt'},
                compiled_file_name='sequential_v2.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/sequential_v2.yaml'
            ),
            TestData(
                pipeline_name='math-pipeline',
                pipeline_func=artifacts_simple_pipeline,
                pipline_func_args=None,
                compiled_file_name='artifacts_simple.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/artifacts_simple.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-multiple-exit-handlers',
                pipeline_func=multiple_exit_handlers_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_multiple_exit_handlers.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/failing/pipeline_with_multiple_exit_handlers.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-reused-component',
                pipeline_func=reused_component_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_reused_component.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_with_reused_component.yaml'
            ),
            TestData(
                pipeline_name='math-pipeline',
                pipeline_func=artifacts_complex_pipeline,
                pipline_func_args=None,
                compiled_file_name='artifacts_complex.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/artifacts_complex.yaml'
            ),
            TestData(
                pipeline_name='math-pipeline',
                pipeline_func=conditional_producer_consumers_pipeline,
                pipline_func_args={'threshold': 2},
                compiled_file_name='conditional_producer_and_consumers.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/conditional_producer_and_consumers.yaml'
            ),
            TestData(
                pipeline_name='collected-artifact-pipeline',
                pipeline_func=collected_artifact_pipeline,
                pipline_func_args=None,
                compiled_file_name='collected_artifacts.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/collected_artifacts.yaml'
            ),
            TestData(
                pipeline_name='split-datasets-and-return-first',
                pipeline_func=pythonic_artifacts_multiple_returns,
                pipline_func_args=None,
                compiled_file_name='pythonic_artifacts_with_multiple_returns.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pythonic_artifacts_with_multiple_returns.yaml'
            ),
            TestData(
                pipeline_name='identity',
                pipeline_func=identity,
                pipline_func_args={'value': 'test'},
                compiled_file_name='identity.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/identity.yaml'
            ),
            TestData(
                pipeline_name='input-artifact',
                pipeline_func=input_artifact,
                pipline_func_args=None,
                compiled_file_name='input_artifact.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/input_artifact.yaml'
            ),
            TestData(
                pipeline_name='nested-return',
                pipeline_func=nested_return,
                pipline_func_args=None,
                compiled_file_name='nested_return.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/nested_return.yaml'
            ),
            TestData(
                pipeline_name='pipeline-in-pipeline',
                pipeline_func=pipeline_in_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_in_pipeline.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_in_pipeline.yaml'
            ),
            TestData(
                pipeline_name='container-with-concat-placeholder',
                pipeline_func=container_with_concat_placeholder,
                pipline_func_args={'text1': 'Hello'},
                compiled_file_name='container_with_concat_placeholder.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_with_concat_placeholder.yaml'
            ),
            TestData(
                pipeline_name='my-test-pipeline-output',
                pipeline_func=lightweight_python_with_outputs_pipeline,
                pipline_func_args={
                    'first_message': 'Hello KFP!',
                    'second_message': 'Welcome',
                    'first_number': 3,
                    'second_number': 4
                },
                compiled_file_name='lightweight_python_functions_with_outputs.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/lightweight_python_functions_with_outputs.yaml'
            ),
            TestData(
                pipeline_name='output-metrics',
                pipeline_func=output_metrics,
                pipline_func_args=None,
                compiled_file_name='output_metrics.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/output_metrics.yaml'
            ),
            TestData(
                pipeline_name='dict-input',
                pipeline_func=dict_input,
                pipline_func_args={
                    'struct': {
                        'key1': 'value1',
                        'key2': 'value2'
                    }
                },
                compiled_file_name='dict_input.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/dict_input.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-volume',
                pipeline_func=pipeline_with_volume,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_volume.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_volume.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-volume-no-cache',
                pipeline_func=pipeline_with_volume_no_cache,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_volume_no_cache.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_volume_no_cache.yaml'
            ),
            TestData(
                pipeline_name='container-with-if-placeholder',
                pipeline_func=container_with_if_placeholder,
                pipline_func_args=None,
                compiled_file_name='container_with_if_placeholder.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_with_if_placeholder.yaml'
            ),
            TestData(
                pipeline_name='container-with-placeholder-in-fstring',
                pipeline_func=container_with_placeholder_in_fstring,
                pipline_func_args=None,
                compiled_file_name='container_with_placeholder_in_fstring.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_with_placeholder_in_fstring.yaml'
            ),
            TestData(
                pipeline_name='pipeline-in-pipeline-complex',
                pipeline_func=pipeline_in_pipeline_complex,
                pipline_func_args=None,
                compiled_file_name='pipeline_in_pipeline_complex.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_in_pipeline_complex.yaml'
            ),
            TestData(
                pipeline_name='lucky-number-pipeline',
                pipeline_func=lucky_number_pipeline,
                pipline_func_args={
                    'add_drumroll': True,
                    'repeat_if_lucky_number': True,
                    'trials': [1, 2, 3]
                },
                compiled_file_name='if_elif_else_complex.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/if_elif_else_complex.yaml'
            ),
            TestData(
                pipeline_name='if-elif-else-with-oneof-parameters',
                pipeline_func=if_elif_else_oneof_params_pipeline,
                pipline_func_args=None,
                compiled_file_name='if_elif_else_with_oneof_parameters.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/if_elif_else_with_oneof_parameters.yaml'
            ),
            TestData(
                pipeline_name='if-else-with-oneof-parameters',
                pipeline_func=if_else_oneof_params_pipeline,
                pipline_func_args=None,
                compiled_file_name='if_else_with_oneof_parameters.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/if_else_with_oneof_parameters.yaml'
            ),
            TestData(
                pipeline_name='if-else-with-oneof-artifacts',
                pipeline_func=if_else_oneof_artifacts_pipeline,
                pipline_func_args=None,
                compiled_file_name='if_else_with_oneof_artifacts.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/if_else_with_oneof_artifacts.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-loops-and-conditions',
                pipeline_func=loops_and_conditions_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_loops_and_conditions.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_loops_and_conditions.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-various-io-types',
                pipeline_func=various_io_types_pipeline,
                pipline_func_args={
                    'input1': 'Hello',
                    'input4': 'World'
                },
                compiled_file_name='pipeline_with_various_io_types.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_various_io_types.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-metrics-outputs',
                pipeline_func=metrics_outputs_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_metrics_outputs.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_with_metrics_outputs.yaml'
            ),
            TestData(
                pipeline_name='containerized-concat-message',
                pipeline_func=containerized_concat_message,
                pipline_func_args={
                    'message1': 'Hello',
                    'message2': ' Containerized!'
                },
                compiled_file_name='containerized_python_component.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/containerized_python_component.yaml'
            ),
            TestData(
                pipeline_name='container-with-artifact-output',
                pipeline_func=container_with_artifact_output,
                pipline_func_args={'num_epochs': 10},
                compiled_file_name='container_with_artifact_output.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_with_artifact_output.yaml'
            ),
            TestData(
                pipeline_name='math-pipeline',
                pipeline_func=nested_with_parameters_pipeline,
                pipline_func_args=None,
                compiled_file_name='nested_with_parameters.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/nested_with_parameters.yaml'
            ),
            TestData(
                pipeline_name='math-pipeline',
                pipeline_func=parameters_complex_pipeline,
                pipline_func_args=None,
                compiled_file_name='parameters_complex.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/parameters_complex.yaml'
            ),
            TestData(
                pipeline_name='dataset-joiner',
                pipeline_func=dataset_joiner,
                pipline_func_args=None,
                compiled_file_name='component_with_metadata_fields.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/component_with_metadata_fields.yaml'
            ),
            TestData(
                pipeline_name='my-test-pipeline',
                pipeline_func=pip_index_urls_pipeline,
                pipline_func_args=None,
                compiled_file_name='component_with_pip_index_urls.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/component_with_pip_index_urls.yaml'
            ),
            TestData(
                pipeline_name='flip-coin',
                pipeline_func=flip_coin,
                pipline_func_args=None,
                compiled_file_name='flip_coin.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/flip_coin.yaml'
            ),
            TestData(
                pipeline_name='component-with-pip-install-in-venv',
                pipeline_func=pip_install_venv_pipeline,
                pipline_func_args=None,
                compiled_file_name='component_with_pip_install_in_venv.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/component_with_pip_install_in_venv.yaml'
            ),
            TestData(
                pipeline_name='component-with-pip-install',
                pipeline_func=pip_install_pipeline,
                pipline_func_args=None,
                compiled_file_name='component_with_pip_install.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/component_with_pip_install.yaml'
            ),
            TestData(
                pipeline_name='component-with-task-final-status',
                pipeline_func=task_final_status_pipeline,
                pipline_func_args=None,
                compiled_file_name='component_with_task_final_status_GH-12033.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/component_with_task_final_status_GH-12033.yaml'
            ),
            TestData(
                pipeline_name='math-pipeline',
                pipeline_func=producer_consumer_parallel_for_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_producer_consumer.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_producer_consumer.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-secret-as-volume',
                pipeline_func=pipeline_secret_volume,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_secret_as_volume.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_secret_as_volume.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-utils',
                pipeline_func=pipeline_with_utils,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_utils.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_utils.yaml'
            ),
            TestData(
                pipeline_name='echo',
                pipeline_func=arguments_parameters_echo,
                pipline_func_args={
                    'param1': 'hello',
                    'param2': 'world'
                },
                compiled_file_name='arguments_parameters.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/arguments_parameters.yaml'
            ),
            TestData(
                pipeline_name='container-no-input',
                pipeline_func=container_no_input,
                pipline_func_args=None,
                compiled_file_name='container_no_input.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/container_no_input.yaml'
            ),
            TestData(
                pipeline_name='test-env-exists',
                pipeline_func=test_env_exists,
                pipline_func_args={'env_var': 'HOME'},
                compiled_file_name='env_var.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/env-var.yaml'
            ),
            TestData(
                pipeline_name='create-pod-metadata-complex',
                pipeline_func=create_pod_metadata_complex,
                pipline_func_args=None,
                compiled_file_name='create_pod_metadata_complex.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/create_pod_metadata_complex.yaml'
            ),
            TestData(
                pipeline_name='nested-pipeline-opt-inputs-nil',
                pipeline_func=nested_pipeline_opt_inputs_nil,
                pipline_func_args=None,
                compiled_file_name='nested_pipeline_opt_inputs_nil.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/nested_pipeline_opt_inputs_nil_compiled.yaml'
            ),
            TestData(
                pipeline_name='nested-pipeline-opt-inputs-parent-level',
                pipeline_func=nested_pipeline_opt_inputs_parent_level,
                pipline_func_args=None,
                compiled_file_name='nested_pipeline_opt_inputs_parent_level.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/nested_pipeline_opt_inputs_parent_level_compiled.yaml'
            ),
            TestData(
                pipeline_name='cross-loop-after-topology',
                pipeline_func=cross_loop_after_topology_pipeline,
                pipline_func_args=None,
                compiled_file_name='cross_loop_after_topology.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/cross_loop_after_topology.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-task-final-status-conditional',
                pipeline_func=pipeline_as_exit_task,
                pipline_func_args={'message': 'Hello Exit Task!'},
                compiled_file_name='pipeline_as_exit_task.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_as_exit_task.yaml'
            ),
            TestData(
                pipeline_name='pipeline-in-pipeline-loaded-from-yaml',
                pipeline_func=pipeline_in_pipeline_loaded_from_yaml,
                pipline_func_args=None,
                compiled_file_name='pipeline_in_pipeline_loaded_from_yaml.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_in_pipeline_loaded_from_yaml.yaml'
            ),
            TestData(
                pipeline_name='dynamic-importer-metadata-pipeline',
                pipeline_func=pipeline_with_dynamic_importer_metadata,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_dynamic_importer_metadata.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_dynamic_importer_metadata.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-google-artifact-types',
                pipeline_func=pipeline_with_google_artifact_type,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_google_artifact_type.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_google_artifact_type.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-metadata-fields',
                pipeline_func=pipeline_with_metadata_fields,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_metadata_fields.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_metadata_fields.yaml'
            ),
            TestData(
                pipeline_name='make-and-join-datasets',
                pipeline_func=pythonic_artifacts_with_list_of_artifacts,
                pipline_func_args={'texts': ['text1', 'text2']},
                compiled_file_name='pythonic_artifacts_with_list_of_artifacts.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pythonic_artifacts_with_list_of_artifacts.yaml'
            ),
            TestData(
                pipeline_name='make-language-model-pipeline',
                pipeline_func=pythonic_artifact_with_single_return,
                pipline_func_args=None,
                compiled_file_name='pythonic_artifact_with_single_return.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pythonic_artifact_with_single_return.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-condition-dynamic-task-output',
                pipeline_func=pipeline_with_dynamic_condition_output,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_condition_dynamic_task_output_custom_training_job.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_condition_dynamic_task_output_custom_training_job.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-importer-and-gcpc-types',
                pipeline_func=pipeline_with_importer_and_gcpc_types,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_importer_and_gcpc_types.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_importer_and_gcpc_types.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-nested-conditions-yaml',
                pipeline_func=pipeline_with_nested_conditions_yaml,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_nested_conditions_yaml.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_with_nested_conditions_yaml.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-parallelfor-parallelism',
                pipeline_func=pipeline_with_parallelfor_parallelism,
                pipline_func_args={
                    'loop_parameter': ['item1', 'item2', 'item3']
                },
                compiled_file_name='pipeline_with_parallelfor_parallelism.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_parallelfor_parallelism.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-params-containing-format',
                pipeline_func=pipeline_with_params_containing_format,
                pipline_func_args={'name': 'KFP'},
                compiled_file_name='pipeline_with_params_containing_format.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/pipeline_with_params_containing_format.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-string-machine-fields-pipeline-input',
                pipeline_func=pipeline_with_string_machine_fields_pipeline_input,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_string_machine_fields_pipeline_input.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_string_machine_fields_pipeline_input.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-string-machine-fields-task-output',
                pipeline_func=pipeline_with_string_machine_fields_task_output,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_string_machine_fields_task_output.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_string_machine_fields_task_output.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-task-using-ignore-upstream-failure',
                pipeline_func=pipeline_with_task_using_ignore_upstream_failure,
                pipline_func_args={'sample_input': 'test message'},
                compiled_file_name='pipeline_with_task_using_ignore_upstream_failure.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_task_using_ignore_upstream_failure.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-concat-placeholder',
                pipeline_func=pipeline_with_concat_placeholder_pipeline,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_concat_placeholder.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_concat_placeholder.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-if-placeholder',
                pipeline_func=pipeline_with_if_placeholder_pipeline,
                pipline_func_args=None,
                compiled_file_name='container_with_if_placeholder.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_with_if_placeholder.yaml'
            ),
            TestData(
                pipeline_name='wait-awhile',
                pipeline_func=long_running_pipeline,
                pipline_func_args=None,
                compiled_file_name='long_running.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/long-running.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-task-final-status',
                pipeline_func=pipeline_with_task_final_status,
                pipline_func_args={'message': 'Hello World!'},
                compiled_file_name='pipeline_with_task_final_status.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_task_final_status.yaml'
            ),
            TestData(
                pipeline_name='pipeline-parallelfor-artifacts',
                pipeline_func=pipeline_with_parallelfor_list_artifacts,
                pipline_func_args=None,
                compiled_file_name='pipeline_with_parallelfor_list_artifacts.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_parallelfor_list_artifacts_GH-12033.yaml'
            ),
            TestData(
                pipeline_name='pipeline-with-if-placeholder-supply-none',
                pipeline_func=pipeline_none,
                pipline_func_args=None,
                compiled_file_name='placeholder_none_input_value.yaml',
                expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/essential/placeholder_with_if_placeholder_none_input_value.yaml'
            ),
        ],
        ids=str)
    def test_compilation(self, pipeline_data: TestData):
        temp_compiled_pipeline_file = os.path.join(
            tempfile.gettempdir(), pipeline_data.compiled_file_name)
        try:
            Compiler().compile(
                pipeline_func=pipeline_data.pipeline_func,
                pipeline_name=pipeline_data.pipeline_name,
                pipeline_parameters=pipeline_data.pipline_func_args,
                package_path=temp_compiled_pipeline_file,
            )
            print(f'Pipeline Created at : {temp_compiled_pipeline_file}')
            print(
                f'Parsing expected yaml {pipeline_data.expected_compiled_file_path} for comparison'
            )
            expected_pipeline_specs, expected_platform_specs = FileUtils.read_yaml_file(
                pipeline_data.expected_compiled_file_path)
            print(
                f'Parsing compiled yaml {temp_compiled_pipeline_file} for comparison'
            )
            generated_pipeline_specs, generated_platform_specs = FileUtils.read_yaml_file(
                temp_compiled_pipeline_file)
            print('Verify that the generated yaml matches expected yaml or not')
            ComparisonUtils.compare_pipeline_spec_dicts(
                actual=generated_pipeline_specs,
                expected=expected_pipeline_specs,
                name=pipeline_data.pipeline_name,
                runtime_params=pipeline_data.pipline_func_args,
            )
            ComparisonUtils.compare_pipeline_spec_dicts(
                actual=generated_platform_specs,
                expected=expected_platform_specs)
        finally:
            print(f'Deleting temp compiled file: {temp_compiled_pipeline_file}')
            os.remove(temp_compiled_pipeline_file)
            print(f'Deleted temp compiled file: {temp_compiled_pipeline_file}')

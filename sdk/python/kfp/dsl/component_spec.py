# Copyright 2021 Google LLC
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
"""Functions for creating IR ComponentSpec instance."""

from typing import List, Union

from kfp.components import _structures as structures
from kfp.dsl import _pipeline_param
from kfp.dsl import dsl_utils
from kfp.dsl import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2


def additional_input_name_for_pipelineparam(
    param_or_name: Union[_pipeline_param.PipelineParam, str]) -> str:
  """Gets the name for an additional (compiler-injected) input."""

  # Adding a prefix to avoid (reduce chance of) name collision between the
  # original component inputs and the injected input.
  return 'pipelineparam--' + (
      param_or_name.full_name if isinstance(
          param_or_name, _pipeline_param.PipelineParam) else param_or_name)


def build_component_spec_from_structure(
    component_spec: structures.ComponentSpec,
    executor_label: str,
    actual_inputs: List[str],
) -> pipeline_spec_pb2.ComponentSpec:
  """Builds an IR ComponentSpec instance from structures.ComponentSpec.

  Args:
    component_spec: The structure component spec.
    executor_label: The executor label.
    actual_inputs: The actual arugments passed to the task. This is used as a
      short term workaround to support optional inputs in component spec IR.

  Returns:
    An instance of IR ComponentSpec.
  """
  result = pipeline_spec_pb2.ComponentSpec()
  result.executor_label = executor_label

  for input_spec in component_spec.inputs or []:
    # skip inputs not present
    if input_spec.name not in actual_inputs:
      continue
    if type_utils.is_parameter_type(input_spec.type):
      result.input_definitions.parameters[
          input_spec.name].type = type_utils.get_parameter_type(input_spec.type)
    else:
      result.input_definitions.artifacts[
          input_spec.name].artifact_type.instance_schema = (
              type_utils.get_artifact_type_schema(input_spec.type))

  for output_spec in component_spec.outputs or []:
    if type_utils.is_parameter_type(output_spec.type):
      result.output_definitions.parameters[
          output_spec.name].type = type_utils.get_parameter_type(
              output_spec.type)
    else:
      result.output_definitions.artifacts[
          output_spec.name].artifact_type.instance_schema = (
              type_utils.get_artifact_type_schema(output_spec.type))

  return result


def build_component_inputs_spec(
    component_spec: pipeline_spec_pb2.ComponentSpec,
    pipeline_params: List[_pipeline_param.PipelineParam],
    is_root_component: bool,
) -> None:
  """Builds component inputs spec from pipeline params.

  Args:
    component_spec: The component spec to fill in its inputs spec.
    pipeline_params: The list of pipeline params.
    is_root_component: Whether the component is the root.
  """
  for param in pipeline_params:
    input_name = (
        param.full_name if is_root_component else
        additional_input_name_for_pipelineparam(param))

    if type_utils.is_parameter_type(param.param_type):
      component_spec.input_definitions.parameters[
          input_name].type = type_utils.get_parameter_type(param.param_type)
    elif input_name not in getattr(component_spec.input_definitions,
                                   'parameters', []):
      component_spec.input_definitions.artifacts[
          input_name].artifact_type.instance_schema = (
              type_utils.get_artifact_type_schema(param.param_type))


def build_component_outputs_spec(
    component_spec: pipeline_spec_pb2.ComponentSpec,
    pipeline_params: List[_pipeline_param.PipelineParam],
) -> None:
  """Builds component outputs spec from pipeline params.

  Args:
    component_spec: The component spec to fill in its outputs spec.
    pipeline_params: The list of pipeline params.
  """
  for param in pipeline_params or []:
    output_name = param.full_name
    if type_utils.is_parameter_type(param.param_type):
      component_spec.output_definitions.parameters[
          output_name].type = type_utils.get_parameter_type(param.param_type)
    elif output_name not in getattr(component_spec.output_definitions,
                                    'parameters', []):
      component_spec.output_definitions.artifacts[
          output_name].artifact_type.instance_schema = (
              type_utils.get_artifact_type_schema(param.param_type))


def build_task_inputs_spec(
    task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    pipeline_params: List[_pipeline_param.PipelineParam],
    tasks_in_current_dag: List[str],
    is_parent_component_root: bool,
) -> None:
  """Builds task inputs spec from pipeline params.

  Args:
    task_spec: The task spec to fill in its inputs spec.
    pipeline_params: The list of pipeline params.
    tasks_in_current_dag: The list of tasks names for tasks in the same dag.
    is_parent_component_root: Whether the task is in the root component.
  """
  for param in pipeline_params or []:

    input_name = additional_input_name_for_pipelineparam(param)
    if type_utils.is_parameter_type(param.param_type):
      if param.op_name and dsl_utils.sanitize_task_name(
          param.op_name) in tasks_in_current_dag:
        task_spec.inputs.parameters[
            input_name].task_output_parameter.producer_task = (
                dsl_utils.sanitize_task_name(param.op_name))
        task_spec.inputs.parameters[
            input_name].task_output_parameter.output_parameter_key = (
                param.name)
      else:
        task_spec.inputs.parameters[input_name].component_input_parameter = (
            param.full_name if is_parent_component_root else input_name)
    else:
      if param.op_name and dsl_utils.sanitize_task_name(
          param.op_name) in tasks_in_current_dag:
        task_spec.inputs.artifacts[
            input_name].task_output_artifact.producer_task = (
                dsl_utils.sanitize_task_name(param.op_name))
        task_spec.inputs.artifacts[
            input_name].task_output_artifact.output_artifact_key = (
                param.name)
      else:
        task_spec.inputs.artifacts[input_name].component_input_artifact = (
            param.full_name if is_parent_component_root else input_name)


def update_task_inputs_spec(
    task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    parent_component_inputs: pipeline_spec_pb2.ComponentInputsSpec,
    pipeline_params: List[_pipeline_param.PipelineParam],
    tasks_in_current_dag: List[str],
) -> None:
  """Updates task inputs spec.

  A task input may reference an output outside its immediate DAG.
  For instance::

    random_num = random_num_op(...)
    with dsl.Condition(random_num.output > 5):
      print_op('%s > 5' % random_num.output)

  In this example, `dsl.Condition` forms a sub-DAG with one task from `print_op`
  inside the sub-DAG. The task of `print_op` references output from `random_num`
  task, which is outside the sub-DAG. When compiling to IR, such cross DAG
  reference is disallowed. So we need to "punch a hole" in the sub-DAG to make
  the input available in the sub-DAG component inputs if it's not already there,
  Next, we can call this method to fix the tasks inside the sub-DAG to make them
  reference the component inputs instead of directly referencing the original
  producer task.

  Args:
    task_spec: The task spec to fill in its inputs spec.
    parent_component_inputs: The input spec of the task's parent component.
    pipeline_params: The list of pipeline params.
    tasks_in_current_dag: The list of tasks names for tasks in the same dag.
  """
  if not hasattr(task_spec, 'inputs'):
    return

  for input_name in getattr(task_spec.inputs, 'parameters', []):

    if task_spec.inputs.parameters[input_name].WhichOneof(
        'kind') == 'task_output_parameter' and (
            task_spec.inputs.parameters[input_name].task_output_parameter
            .producer_task not in tasks_in_current_dag):

      param = _pipeline_param.PipelineParam(
          name=task_spec.inputs.parameters[input_name].task_output_parameter
          .output_parameter_key,
          op_name=dsl_utils.remove_task_name_prefix(
              task_spec.inputs.parameters[input_name].task_output_parameter
              .producer_task))
      component_input_parameter = (
          additional_input_name_for_pipelineparam(param))
      assert component_input_parameter in parent_component_inputs.parameters

      task_spec.inputs.parameters[
          input_name].component_input_parameter = component_input_parameter

  for input_name in getattr(task_spec.inputs, 'artifacts', []):

    if task_spec.inputs.artifacts[input_name].WhichOneof(
        'kind') == 'task_output_artifact' and (
            task_spec.inputs.artifacts[input_name].task_output_artifact
            .producer_task not in tasks_in_current_dag):

      param = _pipeline_param.PipelineParam(
          name=task_spec.inputs.artifacts[input_name].task_output_artifact
          .output_artifact_key,
          op_name=dsl_utils.remove_task_name_prefix(
              task_spec.inputs.artifacts[input_name].task_output_artifact
              .producer_task))
      component_input_artifact = (
          additional_input_name_for_pipelineparam(param))
      assert component_input_artifact in parent_component_inputs.artifacts

      task_spec.inputs.artifacts[
          input_name].component_input_artifact = component_input_artifact


def pop_input_from_component_spec(
    component_spec: pipeline_spec_pb2.ComponentSpec,
    input_name: str,
) -> None:
  """Removes an input from component spec input_definitions.

  Args:
    component_spec: The component spec to update in place.
    input_name: The name of the input, which could be an artifact or paremeter.
  """
  component_spec.input_definitions.artifacts.pop(input_name)
  component_spec.input_definitions.parameters.pop(input_name)

  if component_spec.input_definitions == pipeline_spec_pb2.ComponentInputsSpec(
  ):
    component_spec.ClearField('input_definitions')


def pop_input_from_task_spec(
    task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    input_name: str,
) -> None:
  """Removes an input from task spec inputs.

  Args:
    task_spec: The pipeline task spec to update in place.
    input_name: The name of the input, which could be an artifact or paremeter.
  """
  task_spec.inputs.artifacts.pop(input_name)
  task_spec.inputs.parameters.pop(input_name)

  if task_spec.inputs == pipeline_spec_pb2.TaskInputsSpec():
    task_spec.ClearField('inputs')

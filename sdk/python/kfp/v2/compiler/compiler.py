# Copyright 2020 Google LLC
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
"""KFP DSL v2 compiler.

This is an experimental implementation of KFP compiler that compiles KFP
pipeline into Pipeline IR:
https://docs.google.com/document/d/1PUDuSQ8vmeKSBloli53mp7GIvzekaY7sggg6ywy35Dk/
"""

import inspect
import warnings
from typing import Any, Callable, List, Mapping, Optional

import kfp
from kfp.compiler._k8s_helper import sanitize_k8s_name
from kfp.components import _python_op
from kfp.v2 import dsl
from kfp.v2.compiler import compiler_utils
from kfp.v2.dsl import component_spec as dsl_component_spec
from kfp.v2.dsl import dsl_utils
from kfp.v2.dsl import importer_node
from kfp.v2.dsl import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2

from google.protobuf import json_format


class Compiler(object):
  """Experimental DSL compiler that targets the PipelineSpec IR.

  It compiles pipeline function into PipelineSpec json string.
  PipelineSpec is the IR protobuf message that defines a pipeline:
  https://github.com/kubeflow/pipelines/blob/237795539f7b85bac77435e2464367226ee19391/api/v2alpha1/pipeline_spec.proto#L8
  In this initial implementation, we only support components authored through
  Component yaml spec. And we don't support advanced features like conditions,
  static and dynamic loops, etc.

  Example:
    How to use the compiler to construct pipeline_spec json:

      @dsl.pipeline(
        name='name',
        description='description'
      )
      def my_pipeline(a: int = 1, b: str = "default value"):
        ...

      kfp.v2.compiler.Compiler().compile(my_pipeline, 'path/to/pipeline.json')
  """

  def _create_pipeline_spec(
      self,
      args: List[dsl.PipelineParam],
      pipeline: dsl.Pipeline,
  ) -> pipeline_spec_pb2.PipelineSpec:
    """Creates the pipeline spec object.

    Args:
      args: The list of pipeline arguments.
      pipeline: The instantiated pipeline object.

    Returns:
      A PipelineSpec proto representing the compiled pipeline.

    Raises:
      NotImplementedError if the argument is of unsupported types.
    """
    compiler_utils.validate_pipeline_name(pipeline.name)

    pipeline_spec = pipeline_spec_pb2.PipelineSpec()

    pipeline_spec.pipeline_info.name = pipeline.name
    pipeline_spec.sdk_version = 'kfp-{}'.format(kfp.__version__)
    # Schema version 2.0.0 is required for kfp-pipeline-spec>0.1.3.1
    pipeline_spec.schema_version = '2.0.0'

    pipeline_spec.root.CopyFrom(
        dsl_component_spec.build_root_spec_from_pipeline_params(args))

    deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig()

    for op in pipeline.ops.values():
      task_name = op.task_spec.task_info.name
      component_name = op.task_spec.component_ref.name
      executor_label = op.component_spec.executor_label

      pipeline_spec.root.dag.tasks[task_name].CopyFrom(op.task_spec)
      pipeline_spec.components[component_name].CopyFrom(op.component_spec)
      if compiler_utils.is_v2_component(op):
        compiler_utils.refactor_v2_container_spec(op.container_spec)

      deployment_config.executors[executor_label].container.CopyFrom(
          op.container_spec)

      task = pipeline_spec.root.dag.tasks[task_name]
      # A task may have explicit depdency on other tasks even though they may
      # not have inputs/outputs dependency. e.g.: op2.after(op1)
      if op.dependent_names:
        op.dependent_names = [
            dsl_utils.sanitize_task_name(name) for name in op.dependent_names
        ]
        task.dependent_tasks.extend(op.dependent_names)

      # Check if need to insert importer node
      for input_name in task.inputs.artifacts:
        if not task.inputs.artifacts[
            input_name].task_output_artifact.producer_task:
          type_schema = type_utils.get_input_artifact_type_schema(
              input_name, op._metadata.inputs)

          importer_name = importer_node.generate_importer_base_name(
              dependent_task_name=task_name, input_name=input_name)
          importer_task_spec = importer_node.build_importer_task_spec(
              importer_name)
          importer_comp_spec = importer_node.build_importer_component_spec(
              importer_base_name=importer_name,
              input_name=input_name,
              input_type_schema=type_schema)
          importer_task_name = importer_task_spec.task_info.name
          importer_comp_name = importer_task_spec.component_ref.name
          importer_exec_label = importer_comp_spec.executor_label
          pipeline_spec.root.dag.tasks[importer_task_name].CopyFrom(
              importer_task_spec)
          pipeline_spec.components[importer_comp_name].CopyFrom(
              importer_comp_spec)

          task.inputs.artifacts[
              input_name].task_output_artifact.producer_task = (
                  importer_task_name)
          task.inputs.artifacts[
              input_name].task_output_artifact.output_artifact_key = (
                  importer_node.OUTPUT_KEY)

          # Retrieve the pre-built importer spec
          importer_spec = op.importer_specs[input_name]
          deployment_config.executors[importer_exec_label].importer.CopyFrom(
              importer_spec)

    pipeline_spec.deployment_spec.update(
        json_format.MessageToDict(deployment_config))

    return pipeline_spec

  def _create_pipeline(
      self,
      pipeline_func: Callable[..., Any],
      pipeline_root: Optional[str] = None,
      pipeline_name: Optional[str] = None,
      pipeline_parameters_override: Optional[Mapping[str, Any]] = None,
  ) -> pipeline_spec_pb2.PipelineJob:
    """Creates a pipeline instance and constructs the pipeline spec from it.

    Args:
      pipeline_func: Pipeline function with @dsl.pipeline decorator.
      pipeline_root: The root of the pipeline outputs. Optional.
      pipeline_name: The name of the pipeline. Optional.
      pipeline_parameters_override: The mapping from parameter names to values.
        Optional.

    Returns:
      A PipelineJob proto representing the compiled pipeline.
    """

    # Create the arg list with no default values and call pipeline function.
    # Assign type information to the PipelineParam
    pipeline_meta = _python_op._extract_component_interface(pipeline_func)
    pipeline_name = pipeline_name or pipeline_meta.name

    pipeline_root = pipeline_root or getattr(pipeline_func, 'output_directory',
                                             None)
    if not pipeline_root:
      warnings.warn('pipeline_root is None or empty. A valid pipeline_root '
                    'must be provided at job submission.')

    args_list = []
    signature = inspect.signature(pipeline_func)
    for arg_name in signature.parameters:
      arg_type = None
      for pipeline_input in pipeline_meta.inputs or []:
        if arg_name == pipeline_input.name:
          arg_type = pipeline_input.type
          break
      args_list.append(
          dsl.PipelineParam(
              sanitize_k8s_name(arg_name, True), param_type=arg_type))

    with dsl.Pipeline(pipeline_name) as dsl_pipeline:
      pipeline_func(*args_list)

    # Fill in the default values.
    args_list_with_defaults = []
    if pipeline_meta.inputs:
      args_list_with_defaults = [
          dsl.PipelineParam(
              sanitize_k8s_name(input_spec.name, True),
              param_type=input_spec.type,
              value=input_spec.default) for input_spec in pipeline_meta.inputs
      ]

    pipeline_spec = self._create_pipeline_spec(
        args_list_with_defaults,
        dsl_pipeline,
    )

    pipeline_parameters = {
        arg.name: arg.value for arg in args_list_with_defaults
    }
    # Update pipeline parameters override if there were any.
    pipeline_parameters.update(pipeline_parameters_override or {})
    runtime_config = compiler_utils.build_runtime_config_spec(
        output_directory=pipeline_root, pipeline_parameters=pipeline_parameters)
    pipeline_job = pipeline_spec_pb2.PipelineJob(runtime_config=runtime_config)
    pipeline_job.pipeline_spec.update(json_format.MessageToDict(pipeline_spec))

    return pipeline_job

  def compile(self,
              pipeline_func: Callable[..., Any],
              output_path: str,
              pipeline_root: Optional[str] = None,
              pipeline_name: Optional[str] = None,
              pipeline_parameters: Optional[Mapping[str, Any]] = None,
              type_check: bool = True) -> None:
    """Compile the given pipeline function into pipeline job json.

    Args:
      pipeline_func: Pipeline function with @dsl.pipeline decorator.
      output_path: The output pipeline job .json file path. for example,
        "~/pipeline_job.json"
      pipeline_root: The root of the pipeline outputs. Optional. The
        pipeline_root value can be specified either from this `compile()` method
        or through the `@dsl.pipeline` decorator. If it's specified in both
        places, the value provided here prevails.
      pipeline_name: The name of the pipeline. Optional.
      pipeline_parameters: The mapping from parameter names to values. Optional.
      type_check: Whether to enable the type check or not, default: True.
    """
    type_check_old_value = kfp.TYPE_CHECK
    try:
      kfp.TYPE_CHECK = type_check
      pipeline_job = self._create_pipeline(
          pipeline_func=pipeline_func,
          pipeline_root=pipeline_root,
          pipeline_name=pipeline_name,
          pipeline_parameters_override=pipeline_parameters)
      self._write_pipeline(pipeline_job, output_path)
    finally:
      kfp.TYPE_CHECK = type_check_old_value

  def _write_pipeline(self, pipeline_job: pipeline_spec_pb2.PipelineJob,
      output_path: str) -> None:
    """Dump pipeline spec into json file.

    Args:
      pipeline_job: IR pipeline job spec.
      ouput_path: The file path to be written.

    Raises:
      ValueError: if the specified output path doesn't end with the acceptable
      extentions.
    """
    json_text = json_format.MessageToJson(pipeline_job)

    if output_path.endswith('.json'):
      with open(output_path, 'w') as json_file:
        json_file.write(json_text)
    else:
      raise ValueError(
          'The output path {} should ends with ".json".'.format(output_path))

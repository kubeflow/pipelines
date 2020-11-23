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
from typing import Any, Callable, List, Mapping, Optional

import kfp
from kfp.compiler._k8s_helper import sanitize_k8s_name
from kfp.v2 import dsl
from kfp.v2.compiler import compiler_utils
from kfp.v2.components import python_op
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
    if not pipeline.name:
      raise ValueError('Pipeline name is required.')

    pipeline_spec = pipeline_spec_pb2.PipelineSpec(
        runtime_parameters=compiler_utils.build_runtime_parameter_spec(args))

    pipeline_spec.pipeline_info.name = pipeline.name
    pipeline_spec.sdk_version = 'kfp-{}'.format(kfp.__version__)
    pipeline_spec.schema_version = 'v2alpha1'

    deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig()
    importer_tasks = []

    for op in pipeline.ops.values():
      component_spec = op._metadata
      task = pipeline_spec.tasks.add()
      task.CopyFrom(op.task_spec)
      deployment_config.executors[task.executor_label].container.CopyFrom(
          op.container_spec)

      # Check if need to insert importer node
      for input_name in task.inputs.artifacts:
        if not task.inputs.artifacts[input_name].producer_task:
          type_schema = type_utils.get_input_artifact_type_schema(
              input_name, component_spec.inputs)

          importer_task = importer_node.build_importer_task_spec(
              dependent_task=task,
              input_name=input_name,
              input_type_schema=type_schema)
          importer_tasks.append(importer_task)

          task.inputs.artifacts[
              input_name].producer_task = importer_task.task_info.name
          task.inputs.artifacts[
              input_name].output_artifact_key = importer_node.OUTPUT_KEY

          # Retrieve the pre-built importer spec
          importer_spec = op.importer_spec[input_name]
          deployment_config.executors[
              importer_task.executor_label].importer.CopyFrom(importer_spec)

    pipeline_spec.deployment_config.Pack(deployment_config)
    pipeline_spec.tasks.extend(importer_tasks)

    return pipeline_spec

  def _create_pipeline(
      self,
      pipeline_func: Callable[..., Any],
      pipeline_name: Optional[str] = None,
  ) -> pipeline_spec_pb2.PipelineSpec:
    """Creates a pipeline instance and constructs the pipeline spec from it.

    Args:
      pipeline_func: Pipeline function with @dsl.pipeline decorator.
      pipeline_name: The name of the pipeline. Optional.

    Returns:
      The IR representation (pipeline spec) of the pipeline.
    """

    # Create the arg list with no default values and call pipeline function.
    # Assign type information to the PipelineParam
    pipeline_meta = python_op._extract_component_interface(pipeline_func)
    pipeline_name = pipeline_name or pipeline_meta.name

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

    return pipeline_spec

  def _create_pipeline_job(
      self,
      pipeline_spec: pipeline_spec_pb2.PipelineSpec,
      pipeline_root: str,
      pipeline_parameters: Optional[Mapping[str, Any]] = None,
  ) -> pipeline_spec_pb2.PipelineJob:
    """Creates the pipeline job spec object.

    Args:
      pipeline_spec: The pipeline spec object.
      pipeline_root: The root of the pipeline outputs.
      pipeline_parameters: The mapping from parameter names to values. Optional.

    Returns:
      A PipelineJob proto representing the compiled pipeline.
    """
    runtime_config = compiler_utils.build_runtime_config_spec(
        pipeline_root=pipeline_root)
    pipeline_job = pipeline_spec_pb2.PipelineJob(runtime_config=runtime_config)
    pipeline_job.pipeline_spec.update(json_format.MessageToDict(pipeline_spec))

    return pipeline_job

  def compile(self,
              pipeline_func: Callable[..., Any],
              pipeline_root: str,
              output_path: str,
              pipeline_name: Optional[str] = None,
              pipeline_parameters: Optional[Mapping[str, Any]] = None,
              type_check: bool = True) -> None:
    """Compile the given pipeline function into pipeline job json.

    Args:
      pipeline_func: Pipeline function with @dsl.pipeline decorator.
      pipeline_root: The root of the pipeline ouputs.
      output_path: The output pipeline spec .json file path. for example,
        "~/a.json"
      pipeline_name: The name of the pipeline. Optional.
      pipeline_parameters: The mapping from parameter names to values. Optional.
      type_check: Whether to enable the type check or not, default: True.
    """
    type_check_old_value = kfp.TYPE_CHECK
    try:
      kfp.TYPE_CHECK = type_check
      pipeline = self._create_pipeline(pipeline_func, pipeline_name)
      pipeline_job = self._create_pipeline_job(
          pipeline_spec=pipeline,
          pipeline_root=pipeline_root,
          pipeline_parameters=pipeline_parameters)
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

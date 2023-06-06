# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Private utilities for component authoring."""

import copy
import json
import re
from typing import Any, Dict, List, Optional

from google_cloud_pipeline_components import _image
from kfp import components
from kfp import dsl
from kfp.components import placeholders

from google.protobuf import json_format


# note: this is a slight dependency on KFP SDK implementation details
# other code should not similarly depend on the stability of kfp.placeholders
DOCS_INTEGRATED_OUTPUT_RENAMING_PREFIX = "output__"


def build_serverless_customjob_container_spec(
    *,
    project: str,
    location: str,
    custom_job_payload: Dict[str, Any],
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
) -> dsl.ContainerSpec:
  """Builds a container spec that launches a custom job.

  Args:
    project: Project to run the job in.
    location: Location to run the job in.
    custom_job_payload: Payload to pass to the custom job. This dictionary is
      serialized and passed as the custom job ``--payload``.
    gcp_resources: GCP resources that can be used to track the job.

  Returns:
    Container spec that launches a custom job with the specified payload.
  """
  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          "python3",
          "-u",
          "-m",
          "google_cloud_pipeline_components.container.v1.custom_job.launcher",
      ],
      args=[
          "--type",
          "CustomJob",
          "--payload",
          container_component_dumps(custom_job_payload),
          "--project",
          project,
          "--location",
          location,
          "--gcp_resources",
          gcp_resources,
      ],
  )


def container_component_dumps(obj: Any) -> Any:
  """Dump object to JSON string with KFP SDK placeholders included and, if the placeholder does not correspond to a runtime string, quotes escaped.

  Limitations:
    - Cannot handle placeholders as dictionary keys

  Example usage:

    @dsl.container_component
    def comp(val: str, other_val: int):
      return dsl.ContainerSpec(
          image='alpine',
          command=['echo'],
          args=[utils.container_component_dumps({'key': val, 'other_key':
          other_val})],
      )

  Args:
      obj: JSON serializable object, which may container KFP SDK placeholder
        objects.

  Returns:
    JSON string, possibly with placeholder strings.
  """

  def collect_string_fields(obj: Any) -> List[str]:
    non_str_fields = []

    def inner_func(obj):
      if isinstance(obj, list):
        for e in obj:
          inner_func(e)
      elif isinstance(obj, dict):
        for _, v in obj.items():
          # InputValuePlaceholder keys will be caught at dict construction as `TypeError: unhashable type: InputValuePlaceholder`
          inner_func(v)
      elif (
          isinstance(obj, placeholders.InputValuePlaceholder)
          and obj._ir_type != "STRING"  # pylint: disable=protected-access
      ):
        non_str_fields.append(obj.input_name)

    inner_func(obj)

    return non_str_fields

  def custom_placeholder_encoder(obj: Any) -> str:
    if isinstance(obj, placeholders.Placeholder):
      return str(obj)
    raise TypeError(
        f"Object of type {obj.__class__.__name__!r} is not JSON serializable."
    )

  def unquote_nonstring_placeholders(
      json_string: str, strip_quotes_fields: List[str]
  ) -> str:
    for key in strip_quotes_fields:
      pattern = rf"\"\{{\{{\$\.inputs\.parameters\[(?:'|\"|\"){key}(?:'|\"|\")]\}}\}}\""
      repl = f"{{{{$.inputs.parameters['{key}']}}}}"
      json_string = re.sub(pattern, repl, json_string)
    return json_string

  string_fields = collect_string_fields(obj)
  json_string = json.dumps(obj, default=custom_placeholder_encoder)
  return unquote_nonstring_placeholders(json_string, string_fields)


def gcpc_output_name_converter(
    new_name: str,
    original_name: Optional[str] = None,
):
  """Replace the output with original_name with a new_name in a component decorated with an @dsl.container_component decorator.

  Enables authoring components that have an input and output with the same
  key/name.

  Example usage:

    @utils.gcpc_output_name_converter('output__gcp_resources', 'gcp_resources')
    @dsl.container_component
    def my_component(
        param: str,
        output__param: dsl.OutputPath(str),
    ):
        '''Has an input `param` and creates an output `param`'''
        return dsl.ContainerSpec(
            image='alpine',
            command=['echo'],
            args=[output__param],
        )
  """
  original_name = (
      original_name
      if original_name is not None
      else DOCS_INTEGRATED_OUTPUT_RENAMING_PREFIX + new_name
  )

  def converter(comp):
    def get_modified_pipeline_spec(
        pipeline_spec,
        original_name: str,
        new_name: str,
    ):
      root_component_spec = pipeline_spec.root
      num_components = len(pipeline_spec.components)
      component_spec_key, _ = dict(pipeline_spec.components).popitem()
      inner_component_spec = pipeline_spec.components[component_spec_key]
      is_primitive_component = (
          num_components == 1
          and "comp-" + pipeline_spec.pipeline_info.name == component_spec_key
          and root_component_spec.input_definitions
          == inner_component_spec.input_definitions
          and root_component_spec.output_definitions
          == inner_component_spec.output_definitions
      )
      if not is_primitive_component:
        raise ValueError(
            f"The {gcpc_output_name_converter.__name__!r} decorator can only be"
            " used on primitive container components. You are trying to use it"
            " on a pipeline."
        )

      executor_key, _ = dict(
          pipeline_spec.deployment_spec["executors"]
      ).popitem()
      container_spec = pipeline_spec.deployment_spec["executors"][executor_key][
          "container"
      ]
      command = container_spec.get_or_create_list("command")
      args = container_spec.get_or_create_list("args")
      if "--executor_input" in args and "--function_to_execute" in args:
        raise ValueError(
            f"The {gcpc_output_name_converter.__name__!r} decorator can only be"
            " used on primitive container components. You are trying to use it"
            " on a Python component."
        )

      def replace_output_name_in_componentspec_interface(
          component_spec,
          original_name: str,
          new_name: str,
      ):
        # copy so that iterable doesn't change size on iteration
        for output_name in copy.copy(
            list(component_spec.output_definitions.parameters.keys())
        ):
          if output_name == original_name:
            component_spec.output_definitions.parameters[new_name].CopyFrom(
                component_spec.output_definitions.parameters.pop(original_name)
            )

        # copy so that iterable doesn't change size on iteration
        for output_name in copy.copy(
            list(component_spec.output_definitions.artifacts.keys())
        ):
          if output_name == original_name:
            component_spec.output_definitions.artifacts[new_name].CopyFrom(
                component_spec.output_definitions.artifacts.pop(original_name)
            )

      def replace_output_name_in_dag_outputs(
          component_spec,
          original_name: str,
          new_name: str,
      ):
        # copy so that iterable doesn't change size on iteration
        for parameter_name in copy.copy(
            list(component_spec.dag.outputs.parameters.keys())
        ):
          if parameter_name == original_name:
            modified_dag_output_parameter_spec = (
                component_spec.dag.outputs.parameters.pop(original_name)
            )
            modified_dag_output_parameter_spec.value_from_parameter.output_parameter_key = (
                new_name
            )
            component_spec.dag.outputs.parameters[new_name].CopyFrom(
                modified_dag_output_parameter_spec
            )

      def replace_output_name_in_executor(
          command: list,
          args: list,
          original_name: str,
          new_name: str,
      ):
        def placeholder_replacer(string: str) -> str:
          param_pattern = rf"\{{\{{\$\.outputs\.parameters\[(?:''|'|\")({original_name})(?:''|'|\")]"
          param_replacement = f"{{{{$.outputs.parameters['{new_name}']"

          artifact_pattern = rf"\{{\{{\$\.outputs\.artifacts\[(?:''|'|\")({original_name})(?:''|'|\")]"
          artifact_replacement = f"{{{{$.outputs.artifacts['{new_name}']"

          string = re.sub(
              param_pattern,
              param_replacement,
              string,
          )
          return re.sub(
              artifact_pattern,
              artifact_replacement,
              string,
          )

        for i, s in enumerate(command):
          command[i] = placeholder_replacer(s)

        for i, s in enumerate(args):
          args[i] = placeholder_replacer(s)

      replace_output_name_in_componentspec_interface(
          root_component_spec,
          original_name,
          new_name,
      )
      replace_output_name_in_componentspec_interface(
          inner_component_spec,
          original_name,
          new_name,
      )
      replace_output_name_in_dag_outputs(
          root_component_spec,
          original_name,
          new_name,
      )
      replace_output_name_in_executor(
          command,
          args,
          original_name,
          new_name,
      )

      return pipeline_spec

    reloaded_component = components.load_component_from_text(
        json_format.MessageToJson(
            get_modified_pipeline_spec(
                comp.pipeline_spec,
                original_name,
                new_name,
            )
        )
    )
    reloaded_component.__doc__ = comp.pipeline_func.__doc__
    reloaded_component.__annotations__ = comp.pipeline_func.__annotations__
    return reloaded_component

  return converter

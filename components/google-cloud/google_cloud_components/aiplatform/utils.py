# Copyright 2021 Google LLC. All Rights Reserved.
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
"""Module for creating pipeline components based on AI Platform SDK."""

import inspect
import kfp
from kfp import components

INIT_KEY = "init"
METHOD_KEY = "method"


def convert_method_to_component(method, should_serialize_init=False):
  """Creates a corresponding KFP Component for a given MBD SDK method."""
  method_name = method.__name__

  if inspect.ismethod(method):
    cls_name = method.__self__.__name__
    init_signature = inspect.signature(method.__self__.__init__)
  else:
    cls = getattr(
        inspect.getmodule(method),
        method.__qualname__.split(".<locals>", 1)[0].rsplit(".", 1)[0], None)
    cls_name = cls.__name__
    init_signature = inspect.signature(cls.__init__)

  init_arg_names = set(
      init_signature.parameters.keys()) if should_serialize_init else set([])

  def make_args(sa):
    additional_args = []
    for key, args in sa.items():
      for arg_key, value in args.items():
        additional_args.append(f"    - --{key}.{arg_key}={value}")
    return "\n".join(additional_args)

  def component_yaml_generator(**kwargs):
    """Function to create the actual component yaml for the input kwargs."""
    inputs = ["inputs:"]
    input_args = []
    input_kwargs = {}

    serialized_args = {"init": {}, "method": {}}

    for key, value in kwargs.items():
      prefix_key = "init" if key in init_arg_names else "method"
      if isinstance(value, kfp.dsl.PipelineParam):
        name = key
        inputs.append("- {name: %s, type: Artifact}" % (name))
        input_args.append("""
    - --%s
    - {inputUri: %s}
""" % (f"{prefix_key}.{key}", key))
        input_kwargs[key] = value
      else:
        serialized_args[prefix_key][key] = value

    inputs = "\n".join(inputs) if len(inputs) > 1 else ""
    input_args = "\n".join(input_args) if input_args else ""
    component_text = """
name: %s-%s
%s
outputs:
- {name: resource_name_output, type: Artifact}
implementation:
  container:
    image: gcr.io/sashaproject-1/AIPlatform_component:latest
    command:
    - python3
    - remote_runner.py
    - --cls_name=%s
    - --method_name=%s
%s
    args:
    - --resource_name_output_uri
    - {outputUri: resource_name_output}
%s
""" % (cls_name, method_name, inputs, cls_name, method_name,
       make_args(serialized_args), input_args)

    print(component_text)

    return components.load_component_from_text(component_text)(**input_kwargs)

  return component_yaml_generator

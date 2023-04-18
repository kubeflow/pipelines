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
"""Google Cloud Pipeline Components private utilities."""

import json
import re
from typing import Any, List

# note: this is a slight dependency on KFP SDK implementation details
# other code should not similarly depend on the stability of kfp.placeholders
from kfp.components import placeholders


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

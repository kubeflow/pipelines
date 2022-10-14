# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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

import json
import re


def _camel_case_to_snake_case(s):
  return re.sub(r'(?<!^)(?=[A-Z])', '_', s).lower()


def camel_case_to_snake_case_recursive(j):
  if isinstance(j, list):
    return [camel_case_to_snake_case_recursive(i) for i in j]

  if isinstance(j, dict):
    return {
        _camel_case_to_snake_case(k): camel_case_to_snake_case_recursive(v)
        for (k, v) in j.items()
    }
  return j


# TODO(IronPan) This library can be removed once ifPresent is supported within concat[] in component YAML V2.
# Currently the component YAML will generate the payload with all API fields presented,
# and those fields will be left empty if user doesn't specify them in the Python.
def __remove_empty(j):
  """Remove the empty fields in the Json."""
  if isinstance(j, list):
    res = []
    for i in j:
      # Don't remove empty primitive types. Only remove other empty types.
      if isinstance(i, int) or isinstance(i, float) or isinstance(
          i, str) or isinstance(i, bool):
        res.append(i)
      elif __remove_empty(i):
        res.append(__remove_empty(i))
    return res

  if isinstance(j, dict):
    final_dict = {}
    for k, v in j.items():
      if v:
        final_dict[k] = __remove_empty(v)
    return final_dict
  return j


def recursive_remove_empty(j):
  """Recursively remove the empty fields in the Json until there is no empty fields and sub-fields."""
  # Handle special case where an empty "explanation_spec" "metadata" "outputs"
  # should not be removed. Introduced for b/245453693.
  temp_explanation_spec_metadata_outputs = None
  if ('explanation_spec'
      in j) and ('metadata' in j['explanation_spec'] and
                 'outputs' in j['explanation_spec']['metadata']):
    temp_explanation_spec_metadata_outputs = j['explanation_spec']['metadata'][
        'outputs']

  needs_update = True
  while needs_update:
    new_j = __remove_empty(j)
    needs_update = json.dumps(new_j) != json.dumps(j)
    j = new_j

  if temp_explanation_spec_metadata_outputs is not None:
    if 'explanation_spec' not in j:
      j['explanation_spec'] = {}
    if 'metadata' not in j['explanation_spec']:
      j['explanation_spec']['metadata'] = {}
    j['explanation_spec']['metadata'][
        'outputs'] = temp_explanation_spec_metadata_outputs

  return j

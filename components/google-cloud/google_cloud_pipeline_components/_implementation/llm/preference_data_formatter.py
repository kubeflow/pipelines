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
"""Utility function to format the preference data."""

from kfp import dsl

from google_cloud_pipeline_components import _image


# pylint: disable=g-import-not-at-top
@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def format_preference_input_data(
    model_a_inference_dir_uri: str,
    model_b_inference_dir_uri: str,
    instruction: str,
) -> str:
  """Format the inference data from model a and model b and merge them as the input for auto sxs evaluation.

  Args:
    model_a_inference_dir_uri: Where the model a judgments data was saved in the
      previous step.
    model_b_inference_dir_uri: Where the model b judgments data was saved in the
      previous step.
    instruction: instruction to the task.

  Returns:
    The path to the new output file that saved the formatted input data for
    AutoSxs arbiter.
  """
  import json
  import hashlib
  import os
  import re
  import glob

  model_a_inference_dir_uri = re.sub(
      '^gs://', '/gcs/', model_a_inference_dir_uri
  )
  model_b_inference_dir_uri = re.sub(
      '^gs://', '/gcs/', model_b_inference_dir_uri
  )

  model_a_inference_data_map = {}
  model_b_inference_data_map = {}
  files_in_folder_a = glob.glob(
      os.path.join(model_a_inference_dir_uri, 'text*')
  )
  files_in_folder_b = glob.glob(
      os.path.join(model_b_inference_dir_uri, 'text*')
  )
  assert (
      len(files_in_folder_a) == 1 & len(files_in_folder_b) == 1
  ), 'There should be one inference data file for each model'
  with open(files_in_folder_a[0], 'r') as inputs:
    for line in inputs:
      line_json = json.loads(line)
      hash_obj = hashlib.md5(
          json.dumps(line_json['inputs']['inputs_pretokenized']).encode()
      )
      hash_int = int(hash_obj.hexdigest(), 16)
      model_a_inference_data_map[str(hash_int)] = line_json

  with open(files_in_folder_b[0], 'r') as inputs:
    for line in inputs:
      line_json = json.loads(line)
      hash_obj = hashlib.md5(
          json.dumps(line_json['inputs']['inputs_pretokenized']).encode()
      )
      hash_int = int(hash_obj.hexdigest(), 16)
      model_b_inference_data_map[str(hash_int)] = line_json

  formatted_data_json = []
  for key, model_a_inference_item in model_a_inference_data_map.items():
    if key in model_b_inference_data_map:
      model_b_inference_item = model_b_inference_data_map[key]
      updated_line_json = {}
      updated_line_json['inference_instruction'] = instruction
      updated_line_json['content'] = model_a_inference_item['inputs'][
          'inputs_pretokenized'
      ]
      updated_line_json['inference_context'] = model_a_inference_item['inputs'][
          'inputs_pretokenized'
      ]
      updated_line_json['response_a'] = model_a_inference_item['prediction']
      updated_line_json['response_b'] = model_b_inference_item['prediction']
      formatted_data_json.append(updated_line_json)

  output_uri = files_in_folder_a[0].replace(
      '.jsonl', '_formatted_for_autosxs.jsonl'
  )
  with open(output_uri, 'w') as f:
    for line in formatted_data_json:
      f.write(json.dumps(line))
      f.write('\n')
  return output_uri


# pylint: disable=g-import-not-at-top
@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def format_preference_data(input_uri: str) -> str:
  """Format the input for preference data.

  Args:
    input_uri: Where the judgments data was saved in the previous step.

  Returns:
    The path to the new output file that saved the formatted preference data.
    It's under the same folder as the original data file.
  """
  import json
  import re

  input_uri = re.sub('^gs://', '/gcs/', input_uri)
  output_uri = input_uri.replace('.jsonl', '_formatted_for_rlaif.jsonl')
  formatted_data_json = []
  with open(input_uri, 'r') as inputs:
    for line in inputs:
      line_json = json.loads(line)
      if line_json['choice'] not in ['A', 'B']:
        continue
      updated_line_json = {}
      updated_line_json['input_text'] = line_json['content']
      updated_line_json['candidate_0'] = line_json['response_a']
      updated_line_json['candidate_1'] = line_json['response_b']
      updated_line_json['choice'] = 0 if line_json['choice'] == 'A' else 1
      formatted_data_json.append(updated_line_json)

  with open(output_uri, 'w') as f:
    for line in formatted_data_json:
      f.write(json.dumps(line))
      f.write('\n')
  return output_uri

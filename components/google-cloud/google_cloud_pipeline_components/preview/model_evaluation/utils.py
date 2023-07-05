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

from typing import Any, Dict, List, Optional, Tuple, Union

from google.cloud.aiplatform_v1.types.model_evaluation_slice import ModelEvaluationSlice
from google.protobuf import json_format
from google.protobuf import wrappers_pb2


def create_slice_specs_list(
    list_of_feature_and_value: List[
        Dict[str, Union[float, int, str, List[float], bool]]
    ]
) -> List[ModelEvaluationSlice.Slice.SliceSpec]:
  """Creates a list of ModelEvaluationSlice.Slice.SliceSpec from a list of dictionary inputs.

  Args:
    list_of_feature_and_value: A list of feature_and_value. Each
      feature_and_value is a dictionary of feature names to values. The feature
      value can be a float, int, or str for
      ModelEvaluationSlice.Slice.SliceSpec.Value; a bool for `all_values` or a
      list for ModelEvaluationSlice.Slice.SliceSpec.Range.

  Returns:
    A list of ModelEvaluationSlice.Slice.SliceSpec proto.

  Raises:
    ValueError: if the format of a feature's value is invalid.
  """
  slice_specs_list = []
  for feature_and_value in list_of_feature_and_value:
    configs = {}
    for feature, value in feature_and_value.items():
      if isinstance(value, bool):
        # Bool must be checked first, bool is a child of int in Python.
        configs[feature] = ModelEvaluationSlice.Slice.SliceSpec.SliceConfig(
            all_values=wrappers_pb2.BoolValue(value=value)
        )
      elif isinstance(value, int) or isinstance(value, float):
        configs[feature] = ModelEvaluationSlice.Slice.SliceSpec.SliceConfig(
            value=ModelEvaluationSlice.Slice.SliceSpec.Value(
                float_value=float(value)
            )
        )
      elif isinstance(value, str):
        configs[feature] = ModelEvaluationSlice.Slice.SliceSpec.SliceConfig(
            value=ModelEvaluationSlice.Slice.SliceSpec.Value(string_value=value)
        )
      elif isinstance(value, list):
        configs[feature] = ModelEvaluationSlice.Slice.SliceSpec.SliceConfig(
            range=ModelEvaluationSlice.Slice.SliceSpec.Range(
                low=value[0], high=value[1]
            )
        )
      else:
        raise ValueError(
            'Please provide a valid format of value for feature: {}. The'
            ' accepted formats are: bool, float, int, str and list.'.format(
                feature
            )
        )
    slice_spec = ModelEvaluationSlice.Slice.SliceSpec(configs=configs)
    slice_specs_list.append(json_format.MessageToDict(slice_spec._pb))
  return slice_specs_list


def create_bias_configs_list(
    list_of_slice_a_and_slice_b: List[
        List[Dict[str, Union[float, int, str, List[float]]]]
    ],
) -> List[Any]:
  """Creates a list of BiasConfig from a list of tuple inputs.

  Args:
    list_of_slice_a_and_slice_b: A list of slice_a_and_slice_b. Each
      slice_a_and_slice_b is a list which contains 1 or two elelments. Each
      element in the list is a dictionary of feature names to values that
      represents the slice config for 'slice_a' or 'slice_b'. 'slice_b' is
      optional. The feature value can be a float, int, or str for
      ModelEvaluationSlice.Slice.SliceSpec.Value; a list for
      ModelEvaluationSlice.Slice.SliceSpec.Range. Following are example inputs:
      Ex 1. Only provide the config of slice_a: `list_of_slice_a_and_slice_b =
      [[{'education': 'low'}]]`. Ex 2. Provide both configs of slice_a and
      slice_b: `list_of_slice_a_and_slice_b = [[{'education': 'low'},
        {'education': 'high'}]]`.

  Returns:
    A list of BiasConfig.

  Raises:
    ValueError: if a feature's value is `all_values` or the format of the
      feature's value is invalid.
  """
  bias_configs_list = []
  for slice_a_and_slice_b in list_of_slice_a_and_slice_b:
    slice_a = slice_a_and_slice_b[0]
    if len(slice_a_and_slice_b) > 1:
      slice_b = slice_a_and_slice_b[1]
    else:
      slice_b = None
    bias_config = {}
    configs_a = {}
    for feature, value in slice_a.items():
      if isinstance(value, bool):
        # Bool must be checked first, bool is a child of int in Python.
        raise ValueError(
            '`all_values` SliceConfig is not allowed for bias detection.'
        )
      elif isinstance(value, int) or isinstance(value, float):
        configs_a[feature] = ModelEvaluationSlice.Slice.SliceSpec.SliceConfig(
            value=ModelEvaluationSlice.Slice.SliceSpec.Value(
                float_value=float(value)
            )
        )
      elif isinstance(value, str):
        configs_a[feature] = ModelEvaluationSlice.Slice.SliceSpec.SliceConfig(
            value=ModelEvaluationSlice.Slice.SliceSpec.Value(string_value=value)
        )
      elif isinstance(value, list):
        configs_a[feature] = ModelEvaluationSlice.Slice.SliceSpec.SliceConfig(
            range=ModelEvaluationSlice.Slice.SliceSpec.Range(
                low=value[0], high=value[1]
            )
        )
      else:
        raise ValueError(
            'Please provide a valid format of value for feature: {}. The'
            ' accepted formats are: bool, float, int, str and list.'.format(
                feature
            )
        )
    slice_spec_a = ModelEvaluationSlice.Slice.SliceSpec(configs=configs_a)
    slice_spec_a_dict = json_format.MessageToDict(slice_spec_a._pb)
    bias_config['slices'] = [slice_spec_a_dict]
    if slice_b is not None:
      configs_b = {}
      for feature, value in slice_b.items():
        if isinstance(value, bool):
          # Bool must be checked first, bool is a child of int in Python.
          raise ValueError(
              '`all_values` SliceConfig is not allowed for bias detection.'
          )
        elif isinstance(value, int) or isinstance(value, float):
          configs_b[feature] = ModelEvaluationSlice.Slice.SliceSpec.SliceConfig(
              value=ModelEvaluationSlice.Slice.SliceSpec.Value(
                  float_value=float(value)
              )
          )
        elif isinstance(value, str):
          configs_b[feature] = ModelEvaluationSlice.Slice.SliceSpec.SliceConfig(
              value=ModelEvaluationSlice.Slice.SliceSpec.Value(
                  string_value=value
              )
          )
        elif isinstance(value, list):
          configs_b[feature] = ModelEvaluationSlice.Slice.SliceSpec.SliceConfig(
              range=ModelEvaluationSlice.Slice.SliceSpec.Range(
                  low=value[0], high=value[1]
              )
          )
        else:
          raise ValueError(
              'Please provide a valid format of value for feature: {}. The'
              ' accepted formats are: bool, float, int, str and list.'.format(
                  feature
              )
          )
      slice_spec_b = ModelEvaluationSlice.Slice.SliceSpec(configs=configs_b)
      slice_spec_b_dict = json_format.MessageToDict(slice_spec_b._pb)
      bias_config['slices'].append(slice_spec_b_dict)
    bias_configs_list.append(bias_config)
  return bias_configs_list

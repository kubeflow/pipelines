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
"""Facebook Prophet AI Platform Pipeline Components."""

import json
import os

from kfp import dsl
from kfp.components import load_component_from_file
from kfp.v2.dsl import Dataset

__all__ = ['FitProphetModelOp', 'ProphetPredictOp']

_FitProphetModelOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'fit_model_component.yaml'))

_ProphetPredictOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'prediction_component.yaml'))

# Convert compile-time list arguments to JSON strings
def SerializeListArgs(args: tuple, kwargs: dict):
  processed_args = [json.dumps(arg) if type(arg) is list else arg for arg in args]
  for key, value in kwargs.items():
    if type(value) is list:
      kwargs[key] = json.dumps(value)
  return tuple(processed_args), kwargs


def FitProphetModelOp(data_source, *args, **kwargs):
  args, kwargs = SerializeListArgs(args, kwargs)
  if type(data_source) is str:
    data_source = dsl.importer(artifact_uri=data_source, artifact_class=Dataset).output
  return _FitProphetModelOp(data_source, *args, **kwargs)

def ProphetPredictOp(*args, **kwargs):
  args, kwargs = SerializeListArgs(args, kwargs)
  if 'future_data_source' in kwargs and type(kwargs['future_data_source']) is str:
    kwargs['future_data_source'] = dsl.importer(artifact_uri=kwargs['future_data_source'], artifact_class=Dataset).output
  return _ProphetPredictOp(*args, **kwargs)

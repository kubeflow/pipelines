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
from typing import List
from typing import Optional
from typing import Union

from kfp import dsl
from kfp.components import load_component_from_file
from kfp.v2.dsl import Dataset
from kfp.v2.dsl import Model

__all__ = ['FitProphetModelOp', 'ProphetPredictOp']

FitProphetModelOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'fit_model_component.yaml'))

ProphetPredictOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'prediction_component.yaml'))

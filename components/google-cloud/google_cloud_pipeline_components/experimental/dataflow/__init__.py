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
"""Google Cloud Pipeline Dataflow components."""

import os

from kfp.components import load_component_from_file

__all__ = [
    'DataflowFlexTemplateJobOp',
    'DataflowPythonJobOp',
]

# TODO(wwoo): remove try block after experimental components are migrated to v2.
try:
  from .flex_template import component as dataflow_flex_template_component  # type: ignore

  DataflowFlexTemplateJobOp = (
      dataflow_flex_template_component.dataflow_flex_template
  )
except ImportError:

  def _raise_unsupported(*args, **kwargs):
    raise ImportError(
        'DataflowFlexTemplateJobOp requires KFP SDK v2.0.0b1 or higher.'
    )

  DataflowFlexTemplateJobOp = _raise_unsupported

DataflowPythonJobOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'python_job/component.yaml')
)

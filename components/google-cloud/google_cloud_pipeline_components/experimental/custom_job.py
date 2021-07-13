# Copyright 2021 The Kubeflow Authors
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
"""Module for supporting Google Vertex AI Custom Job."""

import tempfile
from typing import Callable, Dict, List, Optional, Tuple, Union
from kfp import components
from kfp.components.structures import ComponentSpec, ContainerImplementation, ContainerSpec, InputPathPlaceholder, InputSpec, InputValuePlaceholder, OutputPathPlaceholder, OutputSpec, OutputUriPlaceholder, InputUriPlaceholder

_DEFAULT_CUSTOM_JOB_MACHINE_TYPE = 'n1-standard-4'


def custom_job(component_op):
  payload = {}
  custom_job_component_spec = ComponentSpec(
      name=component_op.component_spec.name,
      inputs=component_op.component_spec.inputs + [
          InputSpec(name='gcp_project', type='String'),
          InputSpec(name='gcp_region', type='String')
      ],
      outputs=component_op.component_spec.outputs +
      [OutputSpec(name='gcp_resources', type='gcp_resources')],
      implementation=ContainerImplementation(
          container=ContainerSpec(
              image='gcr.io/managed-pipeline-test/gcp-launcher:v7',
              command=['python', '-u', '-m', 'launcher.py'],
              args=[
                  '--type',
                  'CustomJob',
                  '--gcp_project',
                  InputValuePlaceholder(input_name='gcp_project'),
                  '--gcp_region',
                  InputValuePlaceholder(input_name='gcp_region'),
                  '--payload',
                  payload,
                  '--gcp_resources',
                  OutputUriPlaceholder(output_name='gcp_resources'),
              ],
          )))
  component_path = tempfile.mktemp()
  custom_job_component_spec.save(component_path)

  return components.load_component_from_file(component_path)


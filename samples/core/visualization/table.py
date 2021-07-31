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

import kfp.dsl as dsl
from kfp.components import create_component_from_func

# Advanced function
# Demonstrates imports, helper functions and multiple outputs
from typing import NamedTuple


@create_component_from_func
def table_visualization(train_file_path: str = 'https://raw.githubusercontent.com/zijianjoy/pipelines/5651f41071816594b2ed27c88367f5efb4c60b50/samples/core/visualization/table.csv') -> NamedTuple('VisualizationOutput', [('mlpipeline_ui_metadata', 'UI_metadata')]):
  """Provide number to visualize as table metrics."""
  import json
    
  header = ['Average precision ', 'Precision', 'Recall']
  metadata = {
      'outputs' : [{
          'type': 'table',
          'storage': 'gcs',
          'format': 'csv',
          'header': header,
          'source': train_file_path
          }]
      }

  from collections import namedtuple
  visualization_output = namedtuple('VisualizationOutput', [
    'mlpipeline_ui_metadata'])
  return visualization_output(json.dumps(metadata))


@dsl.pipeline(
    name='table-pipeline',
    description='A sample pipeline to generate scalar metrics for UI visualization.'
)
def table_pipeline():
    table_visualization_task = table_visualization()
    # You can also upload samples/core/visualization/table.csv to Google Cloud Storage.
    # And call the component function with gcs path parameter like below:
    # table_visualization_task = table_visualization('gs://<bucket-name>/<path>/table.csv')

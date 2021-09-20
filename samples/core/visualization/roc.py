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
def roc_visualization(roc_csv_uri: str='https://raw.githubusercontent.com/kubeflow/pipelines/master/samples/core/visualization/roc.csv') -> NamedTuple('VisualizationOutput', [('mlpipeline_ui_metadata', 'UI_metadata')]):
  """Provide roc curve csv file to visualize as metrics."""
  import json

  metadata = {
    'outputs': [{
      'type': 'roc',
      'format': 'csv',
      'schema': [
        {'name': 'fpr', 'type': 'NUMBER'},
        {'name': 'tpr', 'type': 'NUMBER'},
        {'name': 'thresholds', 'type': 'NUMBER'},
      ],
      'source': roc_csv_uri
    }]
  }

  from collections import namedtuple
  visualization_output = namedtuple('VisualizationOutput', [
    'mlpipeline_ui_metadata'])
  return visualization_output(json.dumps(metadata))


@dsl.pipeline(
    name='roc-curve-pipeline',
    description='A sample pipeline to generate ROC Curve for UI visualization.'
)
def roc_curve_pipeline():
    roc_visualization_task = roc_visualization()
    # You can also upload samples/core/visualization/roc.csv to Google Cloud Storage.
    # And call the component function with gcs path parameter like below:
    # roc_visualization_task2 = roc_visualization('gs://<bucket-name>/<path>/roc.csv')

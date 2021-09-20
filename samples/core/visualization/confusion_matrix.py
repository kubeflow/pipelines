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
def confusion_visualization(matrix_uri: str = 'https://raw.githubusercontent.com/kubeflow/pipelines/master/samples/core/visualization/confusion_matrix.csv') -> NamedTuple('VisualizationOutput', [('mlpipeline_ui_metadata', 'UI_metadata')]):
    """Provide confusion matrix csv file to visualize as metrics."""
    import json
    
    metadata = {
        'outputs' : [{
          'type': 'confusion_matrix',
          'format': 'csv',
          'schema': [
            {'name': 'target', 'type': 'CATEGORY'},
            {'name': 'predicted', 'type': 'CATEGORY'},
            {'name': 'count', 'type': 'NUMBER'},
          ],
          'source': matrix_uri,
          'labels': ['rose', 'lily', 'iris'],
        }]
    }
    
    from collections import namedtuple
    visualization_output = namedtuple('VisualizationOutput', [
        'mlpipeline_ui_metadata'])
    return visualization_output(json.dumps(metadata))
        
@dsl.pipeline(
    name='confusion-matrix-pipeline',
    description='A sample pipeline to generate Confusion Matrix for UI visualization.'
)
def confusion_matrix_pipeline():
    confusion_visualization_task = confusion_visualization()
    # You can also upload samples/core/visualization/confusion_matrix.csv to Google Cloud Storage.
    # And call the component function with gcs path parameter like below:
    # confusion_visualization_task2 = confusion_visualization('gs://<bucket-name>/<path>/confusion_matrix.csv')

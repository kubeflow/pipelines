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
def html_visualization(gcsPath: str) -> NamedTuple('VisualizationOutput', [('mlpipeline_ui_metadata', 'UI_metadata')]):
    import json

    metadata = {
        'outputs': [{
            'type': 'web-app',
            'storage': 'inline',
            'source': '<h1>Hello, World!</h1>',
        }]
    }

    # Temporarily hack for empty string scenario: https://github.com/kubeflow/pipelines/issues/5830
    if gcsPath and gcsPath != 'BEGIN-KFP-PARAM[]END-KFP-PARAM':
        metadata.get('outputs').append({
            'type': 'web-app',
            'storage': 'gcs',
            'source': gcsPath,
        })

    from collections import namedtuple
    visualization_output = namedtuple('VisualizationOutput', [
        'mlpipeline_ui_metadata'])
    return visualization_output(json.dumps(metadata))

@dsl.pipeline(
    name='html-pipeline',
    description='A sample pipeline to generate HTML for UI visualization.'
)
def html_pipeline():
    html_visualization_task = html_visualization("")
    # html_visualization_task = html_visualization_op("gs://jamxl-kfp-bucket/v2-compatible/html/hello-world.html")
    # Replace the parameter gcsPath with actual google cloud storage path with html file.
    # For example: Upload hello-world.html in the same folder to gs://bucket-name/hello-world.html.
    # Then uncomment the following line.
    # html_visualization_task = html_visualization_op("gs://bucket-name/hello-world.html")

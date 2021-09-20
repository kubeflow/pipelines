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
import kfp.components as comp

# Advanced function
# Demonstrates imports, helper functions and multiple outputs
from typing import NamedTuple

@comp.create_component_from_func
def markdown_visualization() -> NamedTuple('VisualizationOutput', [('mlpipeline_ui_metadata', 'UI_metadata')]):
    import json

    # Exports a sample tensorboard:
    metadata = {
        'outputs': [
            {
                # Markdown that is hardcoded inline
                'storage': 'inline',
                'source': '''# Inline Markdown

* [Kubeflow official doc](https://www.kubeflow.org/).
''',
                'type': 'markdown',
            },
            {
                # Markdown that is read from a file
                'source': 'https://raw.githubusercontent.com/kubeflow/pipelines/master/README.md',
                # Alternatively, use Google Cloud Storage for sample.
                # 'source': 'gs://jamxl-kfp-bucket/v2-compatible/markdown/markdown_example.md',
                'type': 'markdown',
            }]
    }

    from collections import namedtuple
    divmod_output = namedtuple('VisualizationOutput', [
                               'mlpipeline_ui_metadata'])
    return divmod_output(json.dumps(metadata))


@dsl.pipeline(
    name='markdown-pipeline',
    description='A sample pipeline to generate markdown for UI visualization.'
)
def markdown_pipeline():
    # Passing a task output reference as operation arguments
    # For an operation with a single return value, the output reference can be accessed using `task.output` or `task.outputs['output_name']` syntax
    markdown_visualization_task = markdown_visualization()

# Copyright 2020 The Kubeflow Authors
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
"""Utility function for building Importer Node spec."""

from typing import Any, Mapping, Optional, Type, Union

from kfp.components import importer_component
from kfp.components import pipeline_channel
from kfp.components import pipeline_task
from kfp.components import placeholders
from kfp.components import structures
from kfp.components.types import artifact_types

INPUT_KEY = 'uri'
OUTPUT_KEY = 'artifact'


def importer(
    artifact_uri: Union[pipeline_channel.PipelineParameterChannel, str],
    artifact_class: Type[artifact_types.Artifact],
    reimport: bool = False,
    metadata: Optional[Mapping[str, Any]] = None,
) -> pipeline_task.PipelineTask:
    """Imports an existing artifact for use in a downstream component.

    Args:
      artifact_uri: The URI of the artifact to import.
      artifact_class: The artifact class being imported.
      reimport: Whether to reimport the artifact.
      metadata: Properties of the artifact.

    Returns:
      A task with the artifact accessible via its ``.output`` attribute.

    Examples::

      @dsl.pipeline(name='pipeline-with-importer')
      def pipeline_with_importer():

          importer1 = importer(
              artifact_uri='gs://ml-pipeline-playground/shakespeare1.txt',
              artifact_class=Dataset,
              reimport=False)
          train(dataset=importer1.output)
    """
    component_spec = structures.ComponentSpec(
        name='importer',
        implementation=structures.Implementation(
            importer=structures.ImporterSpec(
                artifact_uri=placeholders.InputValuePlaceholder(
                    INPUT_KEY).to_placeholder_string(),
                type_schema=artifact_class.TYPE_NAME,
                reimport=reimport,
                metadata=metadata)),
        inputs={INPUT_KEY: structures.InputSpec(type='String')},
        outputs={
            OUTPUT_KEY: structures.OutputSpec(type=artifact_class.__name__)
        },
    )

    importer = importer_component.ImporterComponent(
        component_spec=component_spec)
    return importer(uri=artifact_uri)

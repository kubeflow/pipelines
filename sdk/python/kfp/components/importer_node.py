# Copyright 2020-2022 The Kubeflow Authors
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

import sys
from typing import Any, Dict, List, Mapping, Optional, Type, Union

from kfp.components import importer_component
from kfp.components import pipeline_channel
from kfp.components import pipeline_task
from kfp.components import placeholders
from kfp.components import structures
from kfp.components.types import artifact_types
from kfp.components.types import type_utils

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

    metadata_pipeline_param_channels = set()

    def traverse_dict_and_create_metadata_inputs(d: Any) -> Any:
        if isinstance(d, pipeline_channel.PipelineParameterChannel):
            metadata_pipeline_param_channels.add(d)
            result = placeholders.InputValuePlaceholder(
                input_name=d.name).to_placeholder_string()
            return result
        elif isinstance(d, dict):
            return {
                traverse_dict_and_create_metadata_inputs(k):
                traverse_dict_and_create_metadata_inputs(v)
                for k, v in d.items()
            }

        elif isinstance(d, list):
            return [traverse_dict_and_create_metadata_inputs(el) for el in d]
        else:
            return d

    metadata_with_placeholders = traverse_dict_and_create_metadata_inputs(
        metadata)

    component_inputs: Dict[str, structures.InputSpec] = {}
    call_inputs: Dict[str, Any] = {}

    for pipeline_param_channel in metadata_pipeline_param_channels:
        unique_name = make_placeholder_unique(
            pipeline_param_channel.name,
            list(component_inputs),
            '-',
        )
        component_inputs[unique_name] = structures.InputSpec(
            type=pipeline_param_channel.channel_type)
        call_inputs[unique_name] = pipeline_param_channel

    component_spec = structures.ComponentSpec(
        name='importer',
        implementation=structures.Implementation(
            importer=structures.ImporterSpec(
                artifact_uri=placeholders.InputValuePlaceholder(
                    INPUT_KEY)._to_placeholder_string(),
                schema_title=type_utils.create_bundled_artifact_type(
                    artifact_class.schema_title, artifact_class.schema_version),
                schema_version=artifact_class.schema_version,
                reimport=reimport,
                metadata=metadata_with_placeholders)),
        inputs={
            INPUT_KEY: structures.InputSpec(type='String'),
            **component_inputs
        },
        outputs={
            OUTPUT_KEY:
                structures.OutputSpec(
                    type=type_utils.create_bundled_artifact_type(
                        artifact_class.schema_title,
                        artifact_class.schema_version))
        },
    )
    importer = importer_component.ImporterComponent(
        component_spec=component_spec)
    return importer(uri=artifact_uri, **call_inputs)


def make_placeholder_unique(
    name: str,
    collection: List[str],
    delimiter: str,
) -> str:
    """Makes a unique name by adding index."""
    unique_name = name
    if unique_name in collection:
        for i in range(2, sys.maxsize**10):
            unique_name = name + delimiter + str(i)
            if unique_name not in collection:
                break
    return unique_name
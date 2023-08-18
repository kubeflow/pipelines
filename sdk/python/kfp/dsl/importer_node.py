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

from typing import Any, Dict, Mapping, Optional, Type, Union

from kfp.dsl import importer_component
from kfp.dsl import pipeline_channel
from kfp.dsl import pipeline_task
from kfp.dsl import placeholders
from kfp.dsl import structures
from kfp.dsl import utils
from kfp.dsl.types import artifact_types
from kfp.dsl.types import type_utils

URI_KEY = 'uri'
METADATA_KEY = 'metadata'
DEFAULT_OUTPUT_KEY = 'artifact'
DEFAULT_COMPONENT_NAME = 'importer'


def importer(
    artifact_uri: Union[pipeline_channel.PipelineParameterChannel, str],
    artifact_class: Type[artifact_types.Artifact],
    output_key: Optional[str] = DEFAULT_OUTPUT_KEY,
    reimport: bool = False,
    metadata: Optional[Mapping[str, Any]] = None,
    component_name: Optional[str] = DEFAULT_COMPONENT_NAME,
) -> pipeline_task.PipelineTask:
    """Imports an existing artifact for use in a downstream component.

    Args:
      artifact_uri: The URI of the artifact to import.
      artifact_class: The artifact class being imported.
      output_key: The name given to the output artifact.
      reimport: Whether to reimport the artifact.
      metadata: Properties of the artifact.
      component_name: The name of this importer component.

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

    component_inputs: Dict[str, structures.InputSpec] = {}
    call_inputs: Dict[str, Any] = {}

    def traverse_dict_and_create_metadata_inputs(d: Any) -> Any:
        if isinstance(d, pipeline_channel.PipelineParameterChannel):
            reversed_call_inputs = {
                pipeline_param_chan: name
                for name, pipeline_param_chan in call_inputs.items()
            }

            # minimizes importer spec interface by not creating new
            # inputspec/parameters if the same input is used multiple places
            # in metadata
            unique_name = reversed_call_inputs.get(
                d,
                utils.make_name_unique_by_adding_index(
                    METADATA_KEY,
                    list(call_inputs),
                    '-',
                ),
            )

            call_inputs[unique_name] = d
            component_inputs[unique_name] = structures.InputSpec(
                type=d.channel_type)

            return placeholders.InputValuePlaceholder(
                input_name=unique_name)._to_string()

        elif isinstance(d, dict):
            # use this instead of list comprehension to ensure compiles are identical across Python versions
            res = {}
            for k, v in d.items():
                new_k = traverse_dict_and_create_metadata_inputs(k)
                new_v = traverse_dict_and_create_metadata_inputs(v)
                res[new_k] = new_v
            return res

        elif isinstance(d, list):
            return [traverse_dict_and_create_metadata_inputs(el) for el in d]

        elif isinstance(d, str):
            # extract pipeline channels from f-strings, if any
            pipeline_channels = pipeline_channel.extract_pipeline_channels_from_any(
                d)

            # pass the channel back into the recursive function to create the placeholder, component inputs, and call inputs, then replace the channel with the placeholder
            for channel in pipeline_channels:
                input_placeholder = traverse_dict_and_create_metadata_inputs(
                    channel)
                d = d.replace(channel.pattern, input_placeholder)
            return d

        else:
            return d

    metadata_with_placeholders = traverse_dict_and_create_metadata_inputs(
        metadata)

    component_spec = structures.ComponentSpec(
        name=component_name,
        implementation=structures.Implementation(
            importer=structures.ImporterSpec(
                artifact_uri=placeholders.InputValuePlaceholder(
                    URI_KEY)._to_string(),
                schema_title=type_utils.create_bundled_artifact_type(
                    artifact_class.schema_title, artifact_class.schema_version),
                schema_version=artifact_class.schema_version,
                reimport=reimport,
                metadata=metadata_with_placeholders)),
        inputs={
            URI_KEY: structures.InputSpec(type='String'),
            **component_inputs
        },
        outputs={
            output_key:
                structures.OutputSpec(
                    type=type_utils.create_bundled_artifact_type(
                        artifact_class.schema_title,
                        artifact_class.schema_version))
        },
    )
    importer = importer_component.ImporterComponent(
        component_spec=component_spec)
    return importer(uri=artifact_uri, **call_inputs)

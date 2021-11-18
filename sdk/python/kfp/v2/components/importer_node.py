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

from typing import Union, Type

from kfp.v2.components import pipeline_task
from kfp.v2.components import pipeline_channel
from kfp.v2.components import placeholders
from kfp.v2.components import structures
from kfp.v2.components import importer_component
from kfp.v2.components.types import artifact_types

INPUT_KEY = 'uri'
OUTPUT_KEY = 'artifact'


def importer(artifact_uri: Union[pipeline_channel.PipelineParameterChannel,
                                 str],
             artifact_class: Type[artifact_types.Artifact],
             reimport: bool = False) -> pipeline_task.PipelineTask:
    """dsl.importer for importing an existing artifact. Only for v2 pipeline.

    Args:
      artifact_uri: The artifact uri to import from.
      artifact_type_schema: The user specified artifact type schema of the
        artifact to be imported.
      reimport: Whether to reimport the artifact. Defaults to False.

    Returns:
      A PipelineTask instance.

    Raises:
      ValueError if the passed in artifact_uri is neither a PipelineParam nor a
        constant string value.
    """
    component_spec = structures.ComponentSpec(
        name='importer',
        implementation=structures.Implementation(
            importer=structures.ImporterSpec(
                artifact_uri=placeholders.input_parameter_placeholder(
                    INPUT_KEY),
                type_schema=artifact_class.TYPE_NAME,
                reimport=reimport)),
        inputs={INPUT_KEY: structures.InputSpec(type='String')},
        outputs={OUTPUT_KEY: structures.OutputSpec(type='Artifact')},
    )

    importer = importer_component.ImporterComponent(
        component_spec=component_spec)
    return importer(uri=artifact_uri)

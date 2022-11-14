"""The `kfp.dsl` module contains domain-specific language objects used to
compose pipelines."""
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

__all__ = [
    'component',
    'container_component',
    'pipeline',
    'importer',
    'ContainerSpec',
    'Condition',
    'ExitHandler',
    'ParallelFor',
    'Input',
    'Output',
    'InputPath',
    'OutputPath',
    'IfPresentPlaceholder',
    'ConcatPlaceholder',
    'PipelineTaskFinalStatus',
    'PIPELINE_JOB_NAME_PLACEHOLDER',
    'PIPELINE_JOB_RESOURCE_NAME_PLACEHOLDER',
    'PIPELINE_JOB_ID_PLACEHOLDER',
    'PIPELINE_TASK_NAME_PLACEHOLDER',
    'PIPELINE_TASK_ID_PLACEHOLDER',
    'Artifact',
    'ClassificationMetrics',
    'Dataset',
    'HTML',
    'Markdown',
    'Metrics',
    'Model',
    'SlicedClassificationMetrics',
    'PipelineTask',
]

try:
    from typing import Annotated
except ImportError:
    from typing_extensions import Annotated

from typing import TypeVar

from kfp.components.component_decorator import component
from kfp.components.container_component_decorator import container_component
from kfp.components.importer_node import importer
from kfp.components.pipeline_context import pipeline
from kfp.components.pipeline_task import PipelineTask
from kfp.components.placeholders import ConcatPlaceholder
from kfp.components.placeholders import IfPresentPlaceholder
from kfp.components.structures import ContainerSpec
from kfp.components.task_final_status import PipelineTaskFinalStatus
from kfp.components.tasks_group import Condition
from kfp.components.tasks_group import ExitHandler
from kfp.components.tasks_group import ParallelFor
from kfp.components.types.artifact_types import Artifact
from kfp.components.types.artifact_types import ClassificationMetrics
from kfp.components.types.artifact_types import Dataset
from kfp.components.types.artifact_types import HTML
from kfp.components.types.artifact_types import Markdown
from kfp.components.types.artifact_types import Metrics
from kfp.components.types.artifact_types import Model
from kfp.components.types.artifact_types import SlicedClassificationMetrics
from kfp.components.types.type_annotations import InputAnnotation
from kfp.components.types.type_annotations import InputPath
from kfp.components.types.type_annotations import OutputAnnotation
from kfp.components.types.type_annotations import OutputPath

# hack: constants and custom type generics have to be defined here to be captured by autodoc and autodocsumm used in ./docs/conf.py

PIPELINE_JOB_NAME_PLACEHOLDER = '{{$.pipeline_job_name}}'
"""A placeholder used to obtain a pipeline job name within a task at pipeline runtime.

    Example:
      ::

        @dsl.pipeline
        def my_pipeline():
            print_op(
                msg='Job name:',
                value=dsl.PIPELINE_JOB_NAME_PLACEHOLDER,
            )
"""

PIPELINE_JOB_RESOURCE_NAME_PLACEHOLDER = '{{$.pipeline_job_resource_name}}'
"""A placeholder used to obtain a pipeline job resource name within a task at pipeline runtime.

    Example:
      ::

        @dsl.pipeline
        def my_pipeline():
            print_op(
                msg='Job resource name:',
                value=dsl.PIPELINE_JOB_RESOURCE_NAME_PLACEHOLDER,
            )
"""

PIPELINE_JOB_ID_PLACEHOLDER = '{{$.pipeline_job_uuid}}'
"""A placeholder used to obtain a pipeline job ID within a task at pipeline runtime.

    Example:
      ::

        @dsl.pipeline
        def my_pipeline():
            print_op(
                msg='Job ID:',
                value=dsl.PIPELINE_JOB_ID_PLACEHOLDER,
            )
"""

PIPELINE_TASK_NAME_PLACEHOLDER = '{{$.pipeline_task_name}}'
"""A placeholder used to obtain a task name within a task at pipeline runtime.

    Example:
      ::

        @dsl.pipeline
        def my_pipeline():
            print_op(
                msg='Task name:',
                value=dsl.PIPELINE_TASK_NAME_PLACEHOLDER,
            )
"""

PIPELINE_TASK_ID_PLACEHOLDER = '{{$.pipeline_task_uuid}}'
"""A placeholder used to obtain a task ID within a task at pipeline runtime.

    Example:
      ::

        @dsl.pipeline
        def my_pipeline():
            print_op(
                msg='Task ID:',
                value=dsl.PIPELINE_TASK_ID_PLACEHOLDER,
            )
"""
PIPELINE_JOB_CREATE_TIME_UTC_PLACEHOLDER = '{{$.pipeline_job_create_time_utc}}'
"""A placeholder used to obtain the time that a pipeline job was created.

    Example:
      ::

        @dsl.pipeline
        def my_pipeline():
            print_op(
                msg='Job created at:',
                value=dsl.PIPELINE_JOB_CREATE_TIME_UTC,
            )
"""
PIPELINE_JOB_SCHEDULE_TIME_UTC_PLACEHOLDER = '{{$.pipeline_job_schedule_time_utc}}'
"""A placeholder used to obtain the time for which a pipeline job is scheduled.

    Example:
      ::

        @dsl.pipeline
        def my_pipeline():
            print_op(
                msg='Job scheduled at:',
                value=dsl.PIPELINE_JOB_SCHEDULE_TIME_UTC,
            )
"""

T = TypeVar('T')
Input = Annotated[T, InputAnnotation]
"""Type generic used to represent an input artifact of type ``T``, where ``T`` is an artifact class.

Use ``Input[Artifact]`` or ``Output[Artifact]`` to indicate whether the enclosed artifact is a component input or output.

Args:
    T: The type of the input artifact.

Example:
  ::

    @dsl.component
    def artifact_producer(model: Output[Artifact]):
        with open(model.path, 'w') as f:
            f.write('my model')

    @dsl.component
    def artifact_consumer(model: Input[Artifact]):
        print(model)

    @dsl.pipeline
    def my_pipeline():
        producer_task = artifact_producer()
        artifact_consumer(model=producer_task.output)
"""

Output = Annotated[T, OutputAnnotation]
"""A type generic used to represent an output artifact of type ``T``, where ``T`` is an artifact class. The argument typed with this annotation is provided at runtime by the executing backend and does not need to be passed as an input by the pipeline author (see example).

Use ``Input[Artifact]`` or ``Output[Artifact]`` to indicate whether the enclosed artifact is a component input or output.

Args:
    T: The type of the output artifact.

Example:
  ::

    @dsl.component
    def artifact_producer(model: Output[Artifact]):
        with open(model.path, 'w') as f:
            f.write('my model')

    @dsl.component
    def artifact_consumer(model: Input[Artifact]):
        print(model)

    @dsl.pipeline
    def my_pipeline():
        producer_task = artifact_producer()
        artifact_consumer(model=producer_task.output)
"""

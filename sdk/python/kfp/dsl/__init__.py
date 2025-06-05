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

# runtime dependencies
__all__ = [
    'Input',
    'Output',
    'InputPath',
    'OutputPath',
    'PipelineTaskFinalStatus',
    'Artifact',
    'ClassificationMetrics',
    'Dataset',
    'HTML',
    'Markdown',
    'Metrics',
    'Model',
    'SlicedClassificationMetrics',
    'get_uri',
    'PIPELINE_JOB_NAME_PLACEHOLDER',
    'PIPELINE_JOB_RESOURCE_NAME_PLACEHOLDER',
    'PIPELINE_JOB_ID_PLACEHOLDER',
    'PIPELINE_TASK_NAME_PLACEHOLDER',
    'PIPELINE_TASK_ID_PLACEHOLDER',
    'PIPELINE_TASK_EXECUTOR_OUTPUT_PATH_PLACEHOLDER',
    'PIPELINE_TASK_EXECUTOR_INPUT_PLACEHOLDER',
    'PIPELINE_ROOT_PLACEHOLDER',
    'PIPELINE_JOB_CREATE_TIME_UTC_PLACEHOLDER',
    'PIPELINE_JOB_SCHEDULE_TIME_UTC_PLACEHOLDER',
]
import os

from kfp.dsl.task_final_status import PipelineTaskFinalStatus
from kfp.dsl.types.artifact_types import Artifact
from kfp.dsl.types.artifact_types import ClassificationMetrics
from kfp.dsl.types.artifact_types import Dataset
from kfp.dsl.types.artifact_types import get_uri
from kfp.dsl.types.artifact_types import HTML
from kfp.dsl.types.artifact_types import Markdown
from kfp.dsl.types.artifact_types import Metrics
from kfp.dsl.types.artifact_types import Model
from kfp.dsl.types.artifact_types import SlicedClassificationMetrics
from kfp.dsl.types.type_annotations import InputAnnotation
from kfp.dsl.types.type_annotations import InputPath
from kfp.dsl.types.type_annotations import OutputAnnotation
from kfp.dsl.types.type_annotations import OutputPath

try:
    from typing import Annotated
except ImportError:
    from typing_extensions import Annotated

from typing import TypeVar

# hack: constants and custom type generics have to be defined here to be captured by autodoc and autodocsumm used in ./docs/conf.py
PIPELINE_JOB_NAME_PLACEHOLDER = '{{$.pipeline_job_name}}'
"""A placeholder used to obtain a pipeline job name within a task at pipeline runtime.
    In Kubeflow Pipelines, this maps to the pipeline run display name.

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
    In Kubeflow Pipelines, this maps to the pipeline run name in the underlying pipeline engine (e.g. an Argo Workflow
    object name).

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
    In Kubeflow Pipelines, this maps to the pipeline run UUID.

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
    In Kubeflow Pipelines, this maps to the component name.

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
    In Kubeflow Pipelines, this maps to the component's ML Metadata (MLMD) execution ID.

    Example:
      ::

        @dsl.pipeline
        def my_pipeline():
            print_op(
                msg='Task ID:',
                value=dsl.PIPELINE_TASK_ID_PLACEHOLDER,
            )
"""

PIPELINE_TASK_EXECUTOR_OUTPUT_PATH_PLACEHOLDER = '{{$.outputs.output_file}}'
"""A placeholder used to obtain the path to the executor_output.json file within the task container.

    Example:
      ::

        @dsl.pipeline
        def my_pipeline():
            create_artifact_with_metadata(
                metadata={'foo': 'bar'},
                executor_output_destination=dsl.PIPELINE_TASK_EXECUTOR_OUTPUT_PATH_PLACEHOLDER,
            )
"""

PIPELINE_TASK_EXECUTOR_INPUT_PLACEHOLDER = '{{$}}'
"""A placeholder used to obtain executor input message passed to the task.

    Example:
      ::

        @dsl.pipeline
        def my_pipeline():
            custom_container_op(
                executor_input=dsl.PIPELINE_TASK_EXECUTOR_INPUT_PLACEHOLDER,
            )
"""

PIPELINE_ROOT_PLACEHOLDER = '{{$.pipeline_root}}'
"""A placeholder used to obtain the pipeline root.

    Example:
      ::

        @dsl.pipeline
        def my_pipeline():
            store_model(
                tmp_dir=dsl.PIPELINE_ROOT_PLACEHOLDER+'/tmp',
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

# compile-time only dependencies
if os.environ.get('_KFP_RUNTIME', 'false') != 'true':
    from kfp.dsl.component_decorator import component
    from kfp.dsl.container_component_decorator import container_component
    # TODO: Collected should be moved to pipeline_channel.py, consistent with OneOf
    from kfp.dsl.for_loop import Collected
    from kfp.dsl.importer_node import importer
    from kfp.dsl.pipeline_channel import OneOf
    from kfp.dsl.pipeline_config import KubernetesWorkspaceConfig
    from kfp.dsl.pipeline_config import PipelineConfig
    from kfp.dsl.pipeline_config import WorkspaceConfig
    from kfp.dsl.pipeline_context import pipeline
    from kfp.dsl.pipeline_task import PipelineTask
    from kfp.dsl.placeholders import ConcatPlaceholder
    from kfp.dsl.placeholders import IfPresentPlaceholder
    from kfp.dsl.structures import ContainerSpec
    from kfp.dsl.tasks_group import Condition
    from kfp.dsl.tasks_group import Elif
    from kfp.dsl.tasks_group import Else
    from kfp.dsl.tasks_group import ExitHandler
    from kfp.dsl.tasks_group import If
    from kfp.dsl.tasks_group import ParallelFor
    __all__.extend([
        'component', 'container_component', 'pipeline', 'importer',
        'ContainerSpec', 'Condition', 'If', 'Elif', 'Else', 'OneOf',
        'ExitHandler', 'ParallelFor', 'Collected', 'IfPresentPlaceholder',
        'ConcatPlaceholder', 'PipelineTask', 'PipelineConfig',
        'WorkspaceConfig', 'KubernetesWorkspaceConfig'
    ])

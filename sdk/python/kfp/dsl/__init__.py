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
    'TaskConfig',
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
    'WORKSPACE_PATH_PLACEHOLDER',
    'run_notebook',
]
import os

from kfp.dsl.notebook_helpers import run_notebook
from kfp.dsl.task_config import TaskConfig
from kfp.dsl.task_final_status import PipelineTaskFinalStatus
from kfp.dsl.types.artifact_types import (
    HTML,
    Artifact,
    ClassificationMetrics,
    Dataset,
    Markdown,
    Metrics,
    Model,
    SlicedClassificationMetrics,
    get_uri,
)
from kfp.dsl.types.type_annotations import (
    EmbeddedAnnotation,
    InputAnnotation,
    InputPath,
    OutputAnnotation,
    OutputPath,
)

try:
    from typing import Annotated
except ImportError:
    from typing_extensions import Annotated

from typing import TypeVar

# hack: constants and custom type generics have to be defined here to be captured by autodoc and autodocsumm used in ./docs/conf.py
PIPELINE_JOB_NAME_PLACEHOLDER = '{{$.pipeline_job_name}}'
"""A placeholder used to obtain a pipeline job name within a task at pipeline
runtime. In Kubeflow Pipelines, this maps to the pipeline run display name.

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
"""A placeholder used to obtain a pipeline job resource name within a task at
pipeline runtime. In Kubeflow Pipelines, this maps to the pipeline run name in
the underlying pipeline engine (e.g. an Argo Workflow object name).

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
"""A placeholder used to obtain a pipeline job ID within a task at pipeline
runtime. In Kubeflow Pipelines, this maps to the pipeline run UUID.

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
"""A placeholder used to obtain a task ID within a task at pipeline runtime. In
Kubeflow Pipelines, this maps to the component's ML Metadata (MLMD) execution
ID.

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
"""A placeholder used to obtain the path to the executor_output.json file
within the task container.

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
PIPELINE_JOB_SCHEDULE_TIME_UTC_PLACEHOLDER = (
    '{{$.pipeline_job_schedule_time_utc}}'
)
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

WORKSPACE_PATH_PLACEHOLDER = '{{$.workspace_path}}'
"""A placeholder used to obtain the path to the shared workspace within a
component.

Example:
  ::

    @dsl.pipeline(
        pipeline_config=dsl.PipelineConfig(
            workspace=dsl.WorkspaceConfig(size='100Gi'),
        ),
    )
    def my_pipeline():
        clone_repo_task = clone_repo(
            workspacePath=dsl.WORKSPACE_PATH_PLACEHOLDER, repo='https://github.com/example/repo',
        )
"""

T = TypeVar('T')
Input = Annotated[T, InputAnnotation]
"""Type generic used to represent an input artifact of type ``T``, where ``T``
is an artifact class.

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
# Runtime-only input for accessing embedded artifacts extracted at runtime.
EmbeddedInput = Annotated[T, EmbeddedAnnotation]
"""A type generic used to represent an output artifact of type ``T``, where
``T`` is an artifact class. The argument typed with this annotation is provided
at runtime by the executing backend and does not need to be passed as an input
by the pipeline author (see example).

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

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from kfp.dsl.component_decorator import component
    from kfp.dsl.component_task_config import (
        TaskConfigField,
        TaskConfigPassthrough,
    )
    from kfp.dsl.container_component_decorator import container_component
    from kfp.dsl.for_loop import Collected
    from kfp.dsl.importer_node import importer
    from kfp.dsl.notebook_component_decorator import notebook_component
    from kfp.dsl.pipeline_channel import OneOf
    from kfp.dsl.pipeline_config import (
        KubernetesWorkspaceConfig,
        PipelineConfig,
        WorkspaceConfig,
    )
    from kfp.dsl.pipeline_context import pipeline
    from kfp.dsl.pipeline_task import PipelineTask
    from kfp.dsl.placeholders import ConcatPlaceholder, IfPresentPlaceholder
    from kfp.dsl.structures import ContainerSpec
    from kfp.dsl.tasks_group import (
        Condition,
        Elif,
        Else,
        ExitHandler,
        If,
        ParallelFor,
    )

_LAZY_COMPILE_TIME_SYMBOLS = {
    'component': ('kfp.dsl.component_decorator', 'component'),
    'TaskConfigField': ('kfp.dsl.component_task_config', 'TaskConfigField'),
    'TaskConfigPassthrough': (
        'kfp.dsl.component_task_config',
        'TaskConfigPassthrough',
    ),
    'container_component': (
        'kfp.dsl.container_component_decorator',
        'container_component',
    ),
    'Collected': ('kfp.dsl.for_loop', 'Collected'),
    'importer': ('kfp.dsl.importer_node', 'importer'),
    'notebook_component': (
        'kfp.dsl.notebook_component_decorator',
        'notebook_component',
    ),
    'OneOf': ('kfp.dsl.pipeline_channel', 'OneOf'),
    'KubernetesWorkspaceConfig': (
        'kfp.dsl.pipeline_config',
        'KubernetesWorkspaceConfig',
    ),
    'PipelineConfig': ('kfp.dsl.pipeline_config', 'PipelineConfig'),
    'WorkspaceConfig': ('kfp.dsl.pipeline_config', 'WorkspaceConfig'),
    'pipeline': ('kfp.dsl.pipeline_context', 'pipeline'),
    'PipelineTask': ('kfp.dsl.pipeline_task', 'PipelineTask'),
    'ConcatPlaceholder': ('kfp.dsl.placeholders', 'ConcatPlaceholder'),
    'IfPresentPlaceholder': (
        'kfp.dsl.placeholders',
        'IfPresentPlaceholder',
    ),
    'ContainerSpec': ('kfp.dsl.structures', 'ContainerSpec'),
    'Condition': ('kfp.dsl.tasks_group', 'Condition'),
    'Elif': ('kfp.dsl.tasks_group', 'Elif'),
    'Else': ('kfp.dsl.tasks_group', 'Else'),
    'ExitHandler': ('kfp.dsl.tasks_group', 'ExitHandler'),
    'If': ('kfp.dsl.tasks_group', 'If'),
    'ParallelFor': ('kfp.dsl.tasks_group', 'ParallelFor'),
}

__all__.extend(list(_LAZY_COMPILE_TIME_SYMBOLS.keys()))


def __getattr__(name: str):
    if name in _LAZY_COMPILE_TIME_SYMBOLS:
        module_name, attr_name = _LAZY_COMPILE_TIME_SYMBOLS[name]
        import importlib

        mod = importlib.import_module(module_name)
        val = getattr(mod, attr_name)
        globals()[name] = val
        return val
    raise AttributeError(f"module '{__name__}' has no attribute '{name}'")


def __dir__():
    return sorted(list(globals().keys()) + __all__)

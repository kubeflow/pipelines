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
    'ContainerSpec',
    'importer',
    'pipeline',
    'PipelineTask',
    'PipelineTaskFinalStatus',
    'Condition',
    'ExitHandler',
    'ParallelFor',
    'Artifact',
    'ClassificationMetrics',
    'Dataset',
    'HTML',
    'Markdown',
    'Metrics',
    'Model',
    'SlicedClassificationMetrics',
    'Input',
    'Output',
    'InputPath',
    'OutputPath',
    'PIPELINE_JOB_NAME_PLACEHOLDER',
    'PIPELINE_JOB_RESOURCE_NAME_PLACEHOLDER',
    'PIPELINE_JOB_ID_PLACEHOLDER',
    'PIPELINE_TASK_NAME_PLACEHOLDER',
    'PIPELINE_TASK_ID_PLACEHOLDER',
    'IfPresentPlaceholder',
    'ConcatPlaceholder',
]

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
from kfp.components.types.type_annotations import Input
from kfp.components.types.type_annotations import InputPath
from kfp.components.types.type_annotations import Output
from kfp.components.types.type_annotations import OutputPath

PIPELINE_JOB_NAME_PLACEHOLDER = '{{$.pipeline_job_name}}'
"""A placeholder used to obtain a pipeline job name within a task at pipeline runtime.

    Example:
      ::

        @dsl.pipeline(name='my-pipeline')
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

        @dsl.pipeline(name='my-pipeline')
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

        @dsl.pipeline(name='my-pipeline')
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

        @dsl.pipeline(name='my-pipeline')
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

        @dsl.pipeline(name='my-pipeline')
        def my_pipeline():
            print_op(
                msg='Task ID:',
                value=dsl.PIPELINE_TASK_ID_PLACEHOLDER,
            )
"""

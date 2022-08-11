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

from kfp.components.component_decorator import component

from kfp.components.importer_node import importer

from kfp.components.pipeline_channel import (
    PipelineArtifactChannel,
    PipelineChannel,
    PipelineParameterChannel,
)
from kfp.components.pipeline_context import pipeline
from kfp.components.pipeline_task import PipelineTask
from kfp.components.task_final_status import PipelineTaskFinalStatus
from kfp.components.tasks_group import (
    Condition,
    ExitHandler,
    ParallelFor,
)
from kfp.components.types.artifact_types import (
    Artifact,
    ClassificationMetrics,
    Dataset,
    HTML,
    Markdown,
    Metrics,
    Model,
    SlicedClassificationMetrics,
)
from kfp.components.types.type_annotations import (
    Input,
    Output,
    InputPath,
    OutputPath,
)

PIPELINE_JOB_NAME_PLACEHOLDER = '{{$.pipeline_job_name}}'
PIPELINE_JOB_RESOURCE_NAME_PLACEHOLDER = '{{$.pipeline_job_resource_name}}'
PIPELINE_JOB_ID_PLACEHOLDER = '{{$.pipeline_job_uuid}}'
PIPELINE_TASK_NAME_PLACEHOLDER = '{{$.pipeline_task_name}}'
PIPELINE_TASK_ID_PLACEHOLDER = '{{$.pipeline_task_uuid}}'

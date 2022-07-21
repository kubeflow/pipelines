# Copyright 2022 The Kubeflow Authors
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
"""Definition for PipelineTaskFinalStatus."""

import dataclasses
from typing import Optional


@dataclasses.dataclass
class PipelineTaskFinalStatus:
    """A final status of a pipeline task. Annotate a component parameter with
    this class to obtain a handle to a task's status (see example).

    This is the Python representation of the proto message `PipelineTaskFinalStatus <https://github.com/kubeflow/pipelines/blob/d8b9439ef92b88da3420df9e8c67db0f1e89d4ef/api/v2alpha1/pipeline_spec.proto#L929-L951>`_.

    Examples:
      ::

        @dsl.component
        def task_status(user_input: str, status: PipelineTaskFinalStatus):
            print('Pipeline status: ', status.state)
            print('Job resource name: ', status.pipeline_job_resource_name)
            print('Pipeline task name: ', status.pipeline_task_name)
            print('Error code: ', status.error_code)
            print('Error message: ', status.error_message)

        @dsl.pipeline(name='my_pipeline')
        def my_pipeline():
            task = task_status(user_input='my_input')
    """
    state: str
    """Final state of the task. The value could be one of ``'SUCCEEDED'``, ``'FAILED'`` or ``'CANCELLED'``."""

    pipeline_job_resource_name: str
    """Pipeline job resource name, in the format of ``projects/{project}/locations/{location}/pipelineJobs/{pipeline_job}``."""

    pipeline_task_name: str
    """Name of the task that produced this status."""

    error_code: Optional[int]
    """The `google.rpc.Code <github.com/googleapis/googleapis/blob/master/google/rpc/code.proto>`_ in case of error. If state is ``'SUCCEEDED'``, this is ``None``."""

    error_message: Optional[str]
    """In case of error, the detailed error message. If state is ``'SUCCEEDED'``, this is ``None``."""

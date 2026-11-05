# Copyright 2024 The Kubeflow Authors
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
"""Structured lifecycle events for LocalRunner execution callbacks."""

import dataclasses
from typing import Any, Callable, Dict, Optional, Union

from kfp.local import status as _status_module


@dataclasses.dataclass
class PipelineStartedEvent:
    """Fired when a pipeline execution begins.

    Args:
        pipeline_name: The name of the pipeline as defined in its
            ``@pipeline`` decorator.
    """

    pipeline_name: str


@dataclasses.dataclass
class TaskStartedEvent:
    """Fired immediately before a task's container starts executing.

    Args:
        task_name: The human-readable task name (e.g. ``'my-preprocessing-op'``).
        pipeline_name: Name of the enclosing pipeline.
    """

    task_name: str
    pipeline_name: str


@dataclasses.dataclass
class TaskSucceededEvent:
    """Fired when a task completes successfully.

    Args:
        task_name: The human-readable task name.
        pipeline_name: Name of the enclosing pipeline.
        outputs: A mapping of output name to output value as returned by the
            task. Empty dict if the task produced no outputs.
    """

    task_name: str
    pipeline_name: str
    outputs: Dict[str, Any]


@dataclasses.dataclass
class TaskFailedEvent:
    """Fired when a task exits with a failure status.

    Args:
        task_name: The human-readable task name.
        pipeline_name: Name of the enclosing pipeline.
        error_message: The error or log message associated with the failure,
            if available.
    """

    task_name: str
    pipeline_name: str
    error_message: Optional[str] = None


@dataclasses.dataclass
class PipelineFinishedEvent:
    """Fired when a pipeline execution completes (success or failure).

    Args:
        pipeline_name: The name of the pipeline.
        status: ``Status.SUCCESS`` or ``Status.FAILURE``.
    """

    pipeline_name: str
    status: _status_module.Status


#: Union type covering all five lifecycle events.
TaskLifecycleEvent = Union[
    PipelineStartedEvent,
    TaskStartedEvent,
    TaskSucceededEvent,
    TaskFailedEvent,
    PipelineFinishedEvent,
]

#: Type alias for the callback signature accepted by :func:`kfp.local.init`.
TaskLifecycleCallback = Callable[[TaskLifecycleEvent], None]


def fire_event(event: TaskLifecycleEvent) -> None:
    """Deliver *event* to the configured callback, if any.

    Reads the callback from the ``LocalExecutionConfig`` singleton so callers
    do not need to thread the callback through the call stack.  Swallows
    exceptions raised by the callback to avoid disrupting the pipeline run;
    the exception is logged at WARNING level.
    """
    import logging

    from kfp.local.config import LocalExecutionConfig

    cfg = LocalExecutionConfig.instance
    if cfg is None or cfg.on_event is None:
        return
    try:
        cfg.on_event(event)
    except Exception as exc:  # pragma: no cover
        logging.warning(
            f'on_event callback raised an exception for '
            f'{type(event).__name__}: {exc}'
        )

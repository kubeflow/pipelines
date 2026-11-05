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
"""Structured progress events emitted during local pipeline execution."""

import dataclasses
import enum
from typing import Optional


class EventType(enum.Enum):
    """The type of a local-execution lifecycle event."""

    PIPELINE_STARTED = 'PIPELINE_STARTED'
    TASK_STARTED = 'TASK_STARTED'
    TASK_COMPLETED = 'TASK_COMPLETED'
    TASK_FAILED = 'TASK_FAILED'
    PIPELINE_COMPLETED = 'PIPELINE_COMPLETED'


@dataclasses.dataclass
class ExecutionEvent:
    """A structured event emitted during local pipeline execution.

    Attributes:
        event_type: The lifecycle stage this event describes.
        pipeline_name: The name of the root pipeline being executed.
        task_name: The name of the task this event concerns.  ``None``
            for pipeline-level events (``PIPELINE_STARTED`` and
            ``PIPELINE_COMPLETED``).
        error: The error message when ``event_type`` is
            ``TASK_FAILED``; ``None`` otherwise.
    """

    event_type: EventType
    pipeline_name: str
    task_name: Optional[str] = None
    error: Optional[str] = None


def emit(event: ExecutionEvent) -> None:
    """Calls the user-supplied progress callback, if one is configured.

    This is a no-op when ``kfp.local.init()`` was called without a
    ``progress_callback``.
    """
    # Import here to avoid a circular dependency between progress and config.
    from kfp.local import config  # noqa: PLC0415

    cfg = config.LocalExecutionConfig.instance
    if cfg is not None and cfg.progress_callback is not None:
        cfg.progress_callback(event)

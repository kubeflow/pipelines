# Copyright 2025 The Kubeflow Authors
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
"""Report intermediate execution stages from a running component.

This module is intentionally stdlib-only. Runtime containers install ``kfp``
with ``--no-deps`` and set ``_KFP_RUNTIME=true``, which disables most SDK
imports, so importing anything heavier here would break components at runtime.
"""

import json
import os
from typing import Any, Optional

# The launcher passes the write-end of its stage pipe to the component as file
# descriptor 3 (the first exec.Cmd.ExtraFiles entry). It is absent during local
# execution and tests, where set_task_stage() becomes a no-op.
_STAGE_FD = 3

# Lazily opened, line-buffered writer wrapping the stage pipe. It is cached so
# repeated calls reuse the same file object instead of re-opening FD 3.
_stage_writer = None

# Flips to False the first time we learn FD 3 is unusable (absent pipe, or the
# launcher closed the read-end), making every later call a cheap no-op.
_fd_available = True


def set_task_stage(
    stage: str,
    message: str,
    state: str = 'RUNNING',
    progress: Optional[float] = None,
    **custom: Any,
) -> None:
    """Reports an intermediate stage for the currently running task.

    Call this from within a component function to surface progress for
    long-running work. Each call reports one stage, keyed by ``stage``; calling
    again with the same ``stage`` updates that stage. The KFP backend stores the
    stages under ``status_metadata.custom_properties["stages"]`` on the task,
    retrievable via ``GetRun(view=FULL)`` or ``GetTask``.

    Outside the launcher (local execution, unit tests) file descriptor 3 is not
    wired up, so this function silently does nothing and components need no
    conditional logic.

    Args:
        stage: Stage identifier, used as the key under ``stages`` (e.g.
            ``"load_data"``, ``"train"``).
        message: Human-readable description of the stage (e.g.
            ``"Epoch 7/50, loss=0.23"``).
        state: One of ``"RUNNING"``, ``"COMPLETED"``, ``"FAILED"``,
            ``"SKIPPED"``. Defaults to ``"RUNNING"``.
        progress: Optional completion fraction, included only when provided.
        **custom: Additional fields passed through as-is into the stage detail.
    """
    global _stage_writer, _fd_available

    if not _fd_available:
        return

    if _stage_writer is None:
        try:
            _stage_writer = os.fdopen(_STAGE_FD, 'w')
        except OSError:
            # FD 3 is not a usable pipe (local execution, tests). No-op.
            _fd_available = False
            return

    record = {'stage': stage, 'message': message, 'state': state}
    if progress is not None:
        record['progress'] = progress
    record.update(custom)

    try:
        _stage_writer.write(json.dumps(record) + '\n')
        _stage_writer.flush()
    except (BrokenPipeError, ValueError, OSError):
        # BrokenPipeError: the launcher exited and closed the read-end.
        # ValueError: the writer was closed underneath us.
        # Either way nobody is listening, so stop trying.
        _fd_available = False

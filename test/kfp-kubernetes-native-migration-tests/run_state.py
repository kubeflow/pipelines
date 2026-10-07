# Copyright 2026 The Kubeflow Authors
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
"""Wait for asynchronous run-state persistence in migration tests."""

import time


def wait_for_run_state(client, run_id, timeout=60, poll_interval=1):
    """Wait for a reported run state without retrying failures."""
    if timeout <= 0 or poll_interval <= 0:
        raise ValueError('timeout and poll_interval must be positive')

    deadline = time.monotonic() + timeout
    state = None
    while time.monotonic() < deadline:
        run = client.get_run(run_id=run_id)
        state = getattr(run, 'state', None)
        if state in ('FAILED', 'CANCELED'):
            raise AssertionError(f'Run {run_id!r} entered {state}')
        # An initial Workflow report can precede its Argo phase. The API
        # omits the unspecified enum from JSON until a phase is persisted.
        if state not in (None, '', 'RUNTIME_STATE_UNSPECIFIED'):
            return run

        remaining = deadline - time.monotonic()
        if remaining > 0:
            time.sleep(min(poll_interval, remaining))

    raise TimeoutError(
        f'Run {run_id!r} did not report a state within {timeout}s '
        f'(last state: {state!r}). Check workflow-controller and '
        'persistence-agent logs.')

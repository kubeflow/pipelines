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
"""Wait for asynchronous run state without relaxing migration assertions."""

import time


def wait_for_run_state(client, run_id, timeout=60, poll_interval=1):
    """Return the same run once its initially unspecified state is reported."""
    deadline = time.monotonic() + timeout
    while True:
        run = client.get_run(run_id=run_id)
        assert run.run_id == run_id, "Run should have correct ID"
        # A persisted run may precede the first Workflow status report. The
        # zero-valued API enum can therefore be omitted from the JSON response.
        if getattr(run, 'state',
                   None) not in (None, '', 'RUNTIME_STATE_UNSPECIFIED'):
            return run
        remaining = deadline - time.monotonic()
        assert remaining > 0, (
            f"Run {run_id} did not report state within {timeout} seconds")
        time.sleep(min(poll_interval, remaining))

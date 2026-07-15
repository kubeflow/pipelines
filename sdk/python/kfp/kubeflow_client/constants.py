# Copyright The Kubeflow Authors
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
"""Run state constants for PipelinesClient."""

__all__ = [
    'RUN_SUCCEEDED',
    'RUN_FAILED',
    'RUN_SKIPPED',
    'RUN_CANCELED',
    'RUN_CANCELING',
    'RUN_RUNNING',
    'RUN_PENDING',
    'RUN_PAUSED',
    'RUN_COMPLETE',
    'TERMINAL_STATES',
]

RUN_SUCCEEDED = 'succeeded'
RUN_FAILED = 'failed'
RUN_SKIPPED = 'skipped'
RUN_CANCELED = 'canceled'
RUN_CANCELING = 'canceling'
RUN_RUNNING = 'running'
RUN_PENDING = 'pending'
RUN_PAUSED = 'paused'

RUN_COMPLETE = RUN_SUCCEEDED

TERMINAL_STATES = frozenset({
    RUN_SUCCEEDED,
    RUN_FAILED,
    RUN_SKIPPED,
    RUN_CANCELED,
})

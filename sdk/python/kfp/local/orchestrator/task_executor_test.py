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
"""Tests for kfp.local.orchestrator.task_executor."""

import unittest

from absl.testing import parameterized
from kfp.local.orchestrator import task_executor


class TestRetryDelay(parameterized.TestCase):

    @parameterized.parameters(
        (0, 30.0),
        (1, 60.0),
        (2, 120.0),
    )
    def test_exponential_backoff(self, attempt: int, expected: float):
        delay = task_executor._retry_delay(
            attempt=attempt,
            backoff_duration=30,
            backoff_factor=2.0,
            backoff_max_duration=3600,
        )
        self.assertEqual(delay, expected)

    @parameterized.parameters(0, 1, 2, 5)
    def test_zero_factor_waits_backoff_duration_every_time(self, attempt: int):
        delay = task_executor._retry_delay(
            attempt=attempt,
            backoff_duration=30,
            backoff_factor=0.0,
            backoff_max_duration=3600,
        )
        self.assertEqual(delay, 30.0)

    def test_delay_is_capped_at_max_duration(self):
        delay = task_executor._retry_delay(
            attempt=10,
            backoff_duration=30,
            backoff_factor=2.0,
            backoff_max_duration=120,
        )
        self.assertEqual(delay, 120.0)


if __name__ == '__main__':
    unittest.main()

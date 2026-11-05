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
"""Tests for progress.py and progress_callback integration."""

import unittest
from typing import List
from unittest import mock

from kfp import dsl, local
from kfp.local import progress, testing_utilities


class TestExecutionEventDataclass(unittest.TestCase):
    """Unit tests for the ExecutionEvent dataclass."""

    def test_pipeline_started_event(self):
        event = progress.ExecutionEvent(
            event_type=progress.EventType.PIPELINE_STARTED,
            pipeline_name='my-pipeline',
        )
        self.assertEqual(event.event_type, progress.EventType.PIPELINE_STARTED)
        self.assertEqual(event.pipeline_name, 'my-pipeline')
        self.assertIsNone(event.task_name)
        self.assertIsNone(event.error)

    def test_task_failed_event_carries_error(self):
        event = progress.ExecutionEvent(
            event_type=progress.EventType.TASK_FAILED,
            pipeline_name='my-pipeline',
            task_name='preprocess',
            error='exit code 1',
        )
        self.assertEqual(event.event_type, progress.EventType.TASK_FAILED)
        self.assertEqual(event.task_name, 'preprocess')
        self.assertEqual(event.error, 'exit code 1')

    def test_event_type_values(self):
        expected = {
            'PIPELINE_STARTED',
            'TASK_STARTED',
            'TASK_COMPLETED',
            'TASK_FAILED',
            'PIPELINE_COMPLETED',
        }
        actual = {e.value for e in progress.EventType}
        self.assertEqual(actual, expected)


class TestEmitWithNoConfig(unittest.TestCase):
    """emit() must be a no-op when LocalExecutionConfig is not initialised."""

    def test_emit_does_not_raise_without_config(self):
        # Temporarily clear the singleton so emit() sees no config.
        from kfp.local import config as local_config

        original = local_config.LocalExecutionConfig.instance
        local_config.LocalExecutionConfig.instance = None
        try:
            progress.emit(
                progress.ExecutionEvent(
                    event_type=progress.EventType.PIPELINE_STARTED,
                    pipeline_name='p',
                )
            )
        finally:
            local_config.LocalExecutionConfig.instance = original


class TestProgressCallbackIntegration(
        testing_utilities.LocalRunnerEnvironmentTestCase):
    """Integration tests: verify callback receives correct events."""

    def test_no_callback_does_not_break_existing_behaviour(self):
        """Omitting progress_callback must not change run results."""

        @dsl.component
        def add(a: int, b: int) -> int:
            return a + b

        @dsl.pipeline
        def add_pipeline(x: int, y: int) -> int:
            return add(a=x, b=y).output

        # init() without progress_callback
        local.init(runner=local.SubprocessRunner(use_venv=False))
        result = add_pipeline(x=1, y=2)
        self.assertIsNotNone(result)

    def test_callback_receives_pipeline_and_task_events_on_success(self):
        """On a successful run the callback receives STARTED, task events,
        and COMPLETED in order."""
        received: List[progress.ExecutionEvent] = []

        @dsl.component
        def double(x: int) -> int:
            return x * 2

        @dsl.pipeline
        def double_pipeline(n: int) -> int:
            return double(x=n).output

        local.init(
            runner=local.SubprocessRunner(use_venv=False),
            progress_callback=received.append,
        )
        double_pipeline(n=3)

        event_types = [e.event_type for e in received]
        self.assertIn(progress.EventType.PIPELINE_STARTED, event_types)
        self.assertIn(progress.EventType.PIPELINE_COMPLETED, event_types)
        self.assertIn(progress.EventType.TASK_STARTED, event_types)
        self.assertIn(progress.EventType.TASK_COMPLETED, event_types)
        self.assertNotIn(progress.EventType.TASK_FAILED, event_types)

        # pipeline events bracket the task events
        pipeline_started_idx = event_types.index(progress.EventType.PIPELINE_STARTED)
        pipeline_completed_idx = event_types.index(progress.EventType.PIPELINE_COMPLETED)
        self.assertLess(pipeline_started_idx, pipeline_completed_idx)

        # all events carry the pipeline name
        for event in received:
            self.assertIsNotNone(event.pipeline_name)

        # task events carry the task name
        for event in received:
            if event.event_type in (
                    progress.EventType.TASK_STARTED,
                    progress.EventType.TASK_COMPLETED,
                    progress.EventType.TASK_FAILED,
            ):
                self.assertIsNotNone(event.task_name)

    def test_callback_receives_task_failed_event_on_failure(self):
        """When a task fails the callback receives a TASK_FAILED event."""
        received: List[progress.ExecutionEvent] = []

        @dsl.component
        def will_fail() -> None:
            raise RuntimeError('deliberate failure')

        @dsl.pipeline
        def failing_pipeline() -> None:
            will_fail()

        local.init(
            runner=local.SubprocessRunner(use_venv=False),
            raise_on_error=False,
            progress_callback=received.append,
        )
        failing_pipeline()

        event_types = [e.event_type for e in received]
        self.assertIn(progress.EventType.TASK_FAILED, event_types)

        failed_events = [
            e for e in received
            if e.event_type == progress.EventType.TASK_FAILED
        ]
        self.assertEqual(len(failed_events), 1)
        self.assertIsNotNone(failed_events[0].task_name)

    def test_callback_exception_propagates(self):
        """An exception raised by the callback propagates to the caller."""

        def exploding_callback(event):
            if event.event_type == progress.EventType.TASK_STARTED:
                raise ValueError('callback error')

        @dsl.component
        def noop() -> None:
            pass

        @dsl.pipeline
        def noop_pipeline() -> None:
            noop()

        local.init(
            runner=local.SubprocessRunner(use_venv=False),
            progress_callback=exploding_callback,
        )
        with self.assertRaises(ValueError):
            noop_pipeline()


if __name__ == '__main__':
    unittest.main()

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
"""Tests for the LocalRunner lifecycle event callback mechanism."""

import unittest
from typing import List
from unittest import mock

from kfp.local import events
from kfp.local.config import LocalExecutionConfig, SubprocessRunner
from kfp.local.status import Status


class TestEventDataclasses(unittest.TestCase):
    """Verify that event objects hold the right fields."""

    def test_pipeline_started_event(self):
        e = events.PipelineStartedEvent(pipeline_name='my-pipeline')
        self.assertEqual(e.pipeline_name, 'my-pipeline')

    def test_task_started_event(self):
        e = events.TaskStartedEvent(task_name='preprocess', pipeline_name='pipe')
        self.assertEqual(e.task_name, 'preprocess')
        self.assertEqual(e.pipeline_name, 'pipe')

    def test_task_succeeded_event(self):
        e = events.TaskSucceededEvent(
            task_name='train',
            pipeline_name='pipe',
            outputs={'accuracy': 0.95},
        )
        self.assertEqual(e.task_name, 'train')
        self.assertEqual(e.outputs, {'accuracy': 0.95})

    def test_task_failed_event_with_message(self):
        e = events.TaskFailedEvent(
            task_name='bad-task',
            pipeline_name='pipe',
            error_message='exit code 1',
        )
        self.assertEqual(e.error_message, 'exit code 1')

    def test_task_failed_event_default_message(self):
        e = events.TaskFailedEvent(task_name='bad-task', pipeline_name='pipe')
        self.assertIsNone(e.error_message)

    def test_pipeline_finished_event(self):
        e = events.PipelineFinishedEvent(
            pipeline_name='pipe', status=Status.SUCCESS
        )
        self.assertEqual(e.status, Status.SUCCESS)


class TestFireEvent(unittest.TestCase):
    """Unit tests for fire_event() with mocked LocalExecutionConfig."""

    def _make_config(self, callback):
        cfg = object.__new__(LocalExecutionConfig)
        cfg.on_event = callback
        return cfg

    def test_fire_event_calls_callback(self):
        received: List[events.TaskLifecycleEvent] = []
        cfg = self._make_config(received.append)

        with mock.patch.object(LocalExecutionConfig, 'instance', cfg):
            events.fire_event(
                events.PipelineStartedEvent(pipeline_name='test-pipe')
            )

        self.assertEqual(len(received), 1)
        self.assertIsInstance(received[0], events.PipelineStartedEvent)
        self.assertEqual(received[0].pipeline_name, 'test-pipe')

    def test_fire_event_no_callback_does_nothing(self):
        cfg = self._make_config(None)

        with mock.patch.object(LocalExecutionConfig, 'instance', cfg):
            # Should not raise
            events.fire_event(
                events.TaskStartedEvent(task_name='t', pipeline_name='p')
            )

    def test_fire_event_no_config_does_nothing(self):
        with mock.patch.object(LocalExecutionConfig, 'instance', None):
            # Should not raise
            events.fire_event(
                events.TaskStartedEvent(task_name='t', pipeline_name='p')
            )

    def test_fire_event_all_five_types(self):
        received: List[events.TaskLifecycleEvent] = []
        cfg = self._make_config(received.append)

        all_events = [
            events.PipelineStartedEvent(pipeline_name='p'),
            events.TaskStartedEvent(task_name='t', pipeline_name='p'),
            events.TaskSucceededEvent(
                task_name='t', pipeline_name='p', outputs={}
            ),
            events.TaskFailedEvent(task_name='t', pipeline_name='p'),
            events.PipelineFinishedEvent(
                pipeline_name='p', status=Status.SUCCESS
            ),
        ]

        with mock.patch.object(LocalExecutionConfig, 'instance', cfg):
            for e in all_events:
                events.fire_event(e)

        self.assertEqual(len(received), 5)
        types = [type(e) for e in received]
        self.assertIn(events.PipelineStartedEvent, types)
        self.assertIn(events.TaskStartedEvent, types)
        self.assertIn(events.TaskSucceededEvent, types)
        self.assertIn(events.TaskFailedEvent, types)
        self.assertIn(events.PipelineFinishedEvent, types)


class TestInitAcceptsOnEvent(unittest.TestCase):
    """Verify that kfp.local.init() stores the callback on the config."""

    def test_init_stores_callback(self):
        import tempfile

        from kfp.local import config

        callback = mock.MagicMock()
        runner = SubprocessRunner(use_venv=False)

        with tempfile.TemporaryDirectory() as tmp:
            config.LocalExecutionConfig(
                runner=runner,
                pipeline_root=tmp,
                workspace_root=tmp,
                raise_on_error=False,
                on_event=callback,
            )
            self.assertIs(
                config.LocalExecutionConfig.instance.on_event, callback
            )

    def test_init_default_on_event_is_none(self):
        import tempfile

        from kfp.local import config

        runner = SubprocessRunner(use_venv=False)

        with tempfile.TemporaryDirectory() as tmp:
            config.LocalExecutionConfig(
                runner=runner,
                pipeline_root=tmp,
                workspace_root=tmp,
                raise_on_error=False,
            )
            self.assertIsNone(config.LocalExecutionConfig.instance.on_event)


if __name__ == '__main__':
    unittest.main()

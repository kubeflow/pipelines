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

import json
import os
from pathlib import Path
import runpy
import sys
import unittest
from unittest import mock

from kfp import task_stage


class TestSetTaskStage(unittest.TestCase):

    def setUp(self):
        # Reset the module's lazily-initialized global state so each test starts
        # from a clean slate and can point _STAGE_FD at its own pipe.
        self._orig_fd = task_stage._STAGE_FD
        task_stage._stage_writer = None
        task_stage._fd_available = True

    def tearDown(self):
        task_stage._STAGE_FD = self._orig_fd
        if task_stage._stage_writer is not None:
            try:
                task_stage._stage_writer.close()
            except (OSError, ValueError):
                pass
        task_stage._stage_writer = None
        task_stage._fd_available = True

    def test_writes_json_line_to_pipe(self):
        read_fd, write_fd = os.pipe()
        self.addCleanup(lambda: os.close(read_fd))
        task_stage._STAGE_FD = write_fd

        task_stage.set_task_stage(
            'train',
            'Epoch 7/50',
            state='RUNNING',
            progress=0.14,
            loss=0.23,
        )

        with os.fdopen(read_fd, 'r', closefd=False) as reader:
            line = reader.readline()
        record = json.loads(line)
        self.assertEqual(
            record, {
                'stage': 'train',
                'message': 'Epoch 7/50',
                'state': 'RUNNING',
                'progress': 0.14,
                'loss': 0.23,
            })

    def test_omits_progress_when_not_provided(self):
        read_fd, write_fd = os.pipe()
        self.addCleanup(lambda: os.close(read_fd))
        task_stage._STAGE_FD = write_fd

        task_stage.set_task_stage('load', 'Loaded', state='COMPLETED')

        with os.fdopen(read_fd, 'r', closefd=False) as reader:
            record = json.loads(reader.readline())
        self.assertNotIn('progress', record)
        self.assertEqual(record['state'], 'COMPLETED')

    def test_multiple_calls_write_multiple_lines(self):
        read_fd, write_fd = os.pipe()
        self.addCleanup(lambda: os.close(read_fd))
        task_stage._STAGE_FD = write_fd

        task_stage.set_task_stage('load', 'Loaded', state='COMPLETED')
        task_stage.set_task_stage('train', 'Training', state='RUNNING')

        with os.fdopen(read_fd, 'r', closefd=False) as reader:
            first = json.loads(reader.readline())
            second = json.loads(reader.readline())
        self.assertEqual(first['stage'], 'load')
        self.assertEqual(second['stage'], 'train')

    def test_broken_pipe_is_suppressed(self):
        read_fd, write_fd = os.pipe()
        os.close(read_fd)  # no reader; writing will raise BrokenPipeError
        task_stage._STAGE_FD = write_fd

        # Must not raise.
        task_stage.set_task_stage('train', 'Epoch 1', state='RUNNING')
        self.assertFalse(task_stage._fd_available)

    def test_no_fd_is_noop(self):
        # A file descriptor that is not open yields OSError from os.fdopen.
        task_stage._STAGE_FD = 9999

        # Must not raise.
        task_stage.set_task_stage('train', 'Epoch 1', state='RUNNING')
        self.assertIsNone(task_stage._stage_writer)
        self.assertFalse(task_stage._fd_available)

    def test_noop_after_fd_marked_unavailable(self):
        task_stage._STAGE_FD = 9999
        task_stage.set_task_stage('train', 'first', state='RUNNING')
        # Subsequent calls short-circuit without attempting to reopen FD 3.
        with mock.patch.object(
                task_stage.os,
                'fdopen',
                side_effect=AssertionError('should not reopen')):
            task_stage.set_task_stage('train', 'second', state='RUNNING')


class TestRuntimeExport(unittest.TestCase):
    """set_task_stage must be importable at runtime (_KFP_RUNTIME=true), the
    only place components actually run."""

    def test_exported_from_kfp_under_runtime_flag(self):
        with mock.patch.dict('os.environ', {'_KFP_RUNTIME': 'true'}):
            namespace = runpy.run_path(
                str(Path(task_stage.__file__).with_name('__init__.py')),
                init_globals={'__path__': []})
        self.assertIn('set_task_stage', namespace)
        self.assertTrue(callable(namespace['set_task_stage']))


if __name__ == '__main__':
    unittest.main()

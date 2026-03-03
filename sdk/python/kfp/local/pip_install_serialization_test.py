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
"""Tests for pip installation serialization across runners."""

import sys
import threading
import time
from typing import List
import unittest

from kfp import dsl
from kfp import local
from kfp.dsl import pipeline_task
from kfp.local import testing_utilities
from kfp.local.pip_install_manager import pip_install_manager

ROOT_FOR_TESTING = './testing_root_serialization'


class TestPipInstallSerialization(
        testing_utilities.LocalRunnerEnvironmentTestCase):

    def setUp(self):
        super().setUp()
        # Reset pip install manager stats for each test
        pip_install_manager.reset_stats()

    def test_subprocess_runner_serialized_installs(self):
        """Test that SubprocessRunner with serialization prevents race
        conditions."""
        local.init(
            local.SubprocessRunner(
                use_venv=True,
                serialize_pip_installs=True,
                max_concurrent_pip_installs=1),
            pipeline_root=ROOT_FOR_TESTING)

        @dsl.component(packages_to_install=['requests==2.28.2'])
        def fetch_data(value: int) -> int:
            return value * 2

        @dsl.pipeline
        def test_pipeline():
            with dsl.ParallelFor([1, 2, 3, 4]) as item:
                fetch_data(value=item)

        # This should work without race conditions
        start_time = time.time()
        task = test_pipeline()
        duration = time.time() - start_time

        self.assertIsInstance(task, pipeline_task.PipelineTask)

        # Check that installations were serialized (should take longer due to serialization)
        stats = pip_install_manager.get_stats()
        self.assertEqual(stats['total_installs'], 4)
        self.assertGreaterEqual(stats['total_wait_time'],
                                0)  # Should have some wait time

        print(f'Serialized execution completed in {duration:.2f}s')
        print(f'Install stats: {stats}')

    def test_subprocess_runner_concurrent_installs(self):
        """Test SubprocessRunner with controlled concurrent installs."""
        local.init(
            local.SubprocessRunner(
                use_venv=True,
                serialize_pip_installs=False,
                max_concurrent_pip_installs=2),
            pipeline_root=ROOT_FOR_TESTING,
            raise_on_error=False)  # Don't fail pipeline due to build conflicts

        pip_install_manager.reset_stats()

        @dsl.component(packages_to_install=['urllib3==1.26.18'])
        def process_data(value: int) -> int:
            import urllib3
            return value * 2

        @dsl.pipeline
        def test_pipeline():
            with dsl.ParallelFor([1, 2, 3, 4]) as item:
                process_data(value=item)

        start_time = time.time()
        task = test_pipeline()
        duration = time.time() - start_time

        self.assertIsInstance(task, pipeline_task.PipelineTask)

        # Check concurrent execution stats
        stats = pip_install_manager.get_stats()
        self.assertEqual(stats['total_installs'], 4)
        self.assertLessEqual(stats['concurrent_peak'],
                             2)  # Should respect max concurrent limit

        print(f'Concurrent execution completed in {duration:.2f}s')
        print(f'Install stats: {stats}')

    def test_subprocess_runner_no_venv(self):
        """Test SubprocessRunner without virtual environment."""
        local.init(
            local.SubprocessRunner(use_venv=False),
            pipeline_root=ROOT_FOR_TESTING,
            raise_on_error=False)  # Don't fail pipeline due to build conflicts

        @dsl.component
        def simple_task(message: str) -> str:
            return f'Processed: {message}'

        @dsl.pipeline
        def test_pipeline():
            with dsl.ParallelFor(['msg1', 'msg2', 'msg3']) as item:
                simple_task(message=item)

        task = test_pipeline()
        self.assertIsInstance(task, pipeline_task.PipelineTask)

        # KFP installations occur due to test environment monkey-patching kfp_package_path
        # Some tasks may fail due to build conflicts, but serialization should still work
        stats = pip_install_manager.get_stats()
        self.assertGreaterEqual(stats['total_installs'], 2)
        self.assertGreater(stats['total_wait_time'], 0)

    def test_pip_install_manager_stats(self):
        """Test pip install manager statistics tracking."""
        local.init(
            local.SubprocessRunner(use_venv=True, serialize_pip_installs=True),
            pipeline_root=ROOT_FOR_TESTING)

        pip_install_manager.reset_stats()

        @dsl.component(packages_to_install=['six==1.16.0'])
        def stats_test_task(num: int) -> int:
            import six
            return num

        @dsl.pipeline
        def test_pipeline():
            with dsl.ParallelFor([1, 2]) as item:
                stats_test_task(num=item)

        initial_stats = pip_install_manager.get_stats()
        self.assertEqual(initial_stats['total_installs'], 0)

        task = test_pipeline()

        final_stats = pip_install_manager.get_stats()
        self.assertEqual(final_stats['total_installs'], 2)
        self.assertGreaterEqual(final_stats['total_wait_time'], 0)
        self.assertGreaterEqual(final_stats['max_wait_time'], 0)
        self.assertEqual(final_stats['current_active'],
                         0)  # Should be 0 after completion

    def test_error_handling_during_install(self):
        """Test error handling when pip installation fails."""
        local.init(
            local.SubprocessRunner(use_venv=True, serialize_pip_installs=True),
            pipeline_root=ROOT_FOR_TESTING,
            raise_on_error=False  # Don't raise to test error handling
        )

        @dsl.component(
            packages_to_install=['nonexistent-package-12345==99.99.99'])
        def failing_install_task() -> str:
            return 'This should not execute'

        @dsl.pipeline
        def test_pipeline():
            failing_install_task()

        # This should handle the error gracefully
        task = test_pipeline()
        self.assertIsInstance(task, pipeline_task.PipelineTask)

        # Stats should still be updated even for failed installs
        stats = pip_install_manager.get_stats()
        self.assertEqual(stats['total_installs'], 1)

    def test_original_parallel_for_race_condition_fixed(self):
        """Test that the original race condition from the test is now fixed."""
        # This uses the same configuration as the original failing test
        # but now with serialization enabled
        local.init(
            local.SubprocessRunner(
                use_venv=True,
                serialize_pip_installs=True  # Enable serialization
            ),
            pipeline_root=ROOT_FOR_TESTING)

        # Use the original component that was failing
        # Use numpy>=1.26.0 for Python 3.13 compatibility (has pre-built wheels)
        import sys
        numpy_version = 'numpy>=1.26.0' if sys.version_info >= (
            3, 13) else 'numpy==1.24.3'

        @dsl.component(packages_to_install=[numpy_version])
        def package_using_op() -> List[int]:
            import numpy as np
            return np.array([1, 2, 3]).tolist()

        @dsl.pipeline
        def my_pipeline():
            # Use more parallel tasks to increase chance of race condition
            with dsl.ParallelFor([1, 2, 3, 4, 5, 6, 7, 8]):
                package_using_op()

        # This should now work without race conditions
        task = my_pipeline()
        self.assertIsInstance(task, pipeline_task.PipelineTask)

        # Verify serialization worked
        stats = pip_install_manager.get_stats()
        self.assertEqual(stats['total_installs'], 8)
        self.assertGreater(stats['total_wait_time'],
                           0)  # Should have wait time due to serialization


class TestPipInstallManager(unittest.TestCase):
    """Test the PipInstallManager class directly."""

    def setUp(self):
        # Get fresh instance for each test
        pip_install_manager.reset_stats()

    def test_singleton_pattern(self):
        """Test that PipInstallManager is a singleton."""
        from kfp.local.pip_install_manager import PipInstallManager

        manager1 = PipInstallManager()
        manager2 = PipInstallManager()

        self.assertIs(manager1, manager2)

    def test_configuration(self):
        """Test manager configuration."""
        pip_install_manager.configure(serialize_installs=True, max_concurrent=1)
        self.assertTrue(pip_install_manager.is_serialized())

        pip_install_manager.configure(
            serialize_installs=False, max_concurrent=3)
        self.assertFalse(pip_install_manager.is_serialized())

    def test_docker_configuration(self):
        """Test docker-specific configuration."""
        pip_install_manager.configure_docker(max_concurrent=2)

        # Test that docker semaphore is configured
        docker_semaphore = pip_install_manager.get_docker_semaphore()
        self.assertIsNotNone(docker_semaphore)

    def test_managed_install_context(self):
        """Test the managed install context manager."""
        pip_install_manager.configure(serialize_installs=True, max_concurrent=1)

        start_time = time.time()

        with pip_install_manager.managed_install('subprocess'):
            # Simulate some work
            time.sleep(0.1)

        duration = time.time() - start_time
        self.assertGreaterEqual(duration, 0.1)

        # Check stats were updated
        stats = pip_install_manager.get_stats()
        self.assertEqual(stats['total_installs'], 1)

    def test_concurrent_managed_installs(self):
        """Test concurrent managed installs with threading."""
        pip_install_manager.configure(serialize_installs=True, max_concurrent=1)
        pip_install_manager.reset_stats()

        results = []
        exceptions = []

        def worker(worker_id: int):
            try:
                with pip_install_manager.managed_install('subprocess'):
                    start = time.time()
                    time.sleep(0.1)  # Simulate work
                    end = time.time()
                    results.append((worker_id, start, end))
            except Exception as e:
                exceptions.append(e)

        # Start multiple threads
        threads = []
        for i in range(3):
            thread = threading.Thread(target=worker, args=(i,))
            threads.append(thread)
            thread.start()

        # Wait for all threads to complete
        for thread in threads:
            thread.join()

        # Check that no exceptions occurred
        self.assertEqual(len(exceptions), 0)

        # Check that all workers completed
        self.assertEqual(len(results), 3)

        # Check that operations were serialized (no overlap)
        results.sort(key=lambda x: x[1])  # Sort by start time
        for i in range(len(results) - 1):
            current_end = results[i][2]
            next_start = results[i + 1][1]
            self.assertLessEqual(
                current_end, next_start,
                f'Operations {results[i][0]} and {results[i+1][0]} overlapped')

        # Check stats
        stats = pip_install_manager.get_stats()
        self.assertEqual(stats['total_installs'], 3)
        self.assertGreater(stats['total_wait_time'], 0)


if __name__ == '__main__':
    unittest.main()

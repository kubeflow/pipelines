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
"""Manager for coordinating pip installations across parallel tasks to prevent
race conditions."""

from contextlib import contextmanager
import logging
import threading
import time
from typing import Any, Dict, Optional

logger = logging.getLogger(__name__)


class PipInstallManager:
    """Manages serialized pip installations to avoid race conditions.

    This singleton class coordinates pip installations across parallel
    tasks to prevent conflicts when multiple virtual environments are
    created simultaneously.
    """

    _instance: Optional['PipInstallManager'] = None
    _lock = threading.Lock()

    def __new__(cls) -> 'PipInstallManager':
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None:
                    cls._instance = super(PipInstallManager, cls).__new__(cls)
        return cls._instance

    def __init__(self):
        if not hasattr(self, '_initialized'):
            self._pip_install_semaphore: Optional[threading.Semaphore] = None
            self._docker_install_semaphore: Optional[threading.Semaphore] = None
            self._serialize_installs = True
            self._max_concurrent = 1
            self._docker_max_concurrent = 1
            self._install_stats: Dict[str, Any] = {
                'total_installs': 0,
                'total_wait_time': 0.0,
                'max_wait_time': 0.0,
                'concurrent_peak': 0,
                'current_active': 0
            }
            self._stats_lock = threading.Lock()
            self._initialized = True

    def configure(self, serialize_installs: bool, max_concurrent: int):
        """Configure the pip install manager for subprocess runner.

        Args:
            serialize_installs: Whether to serialize pip installations
            max_concurrent: Maximum concurrent pip installations when not serializing
        """
        with self._lock:
            self._serialize_installs = serialize_installs
            self._max_concurrent = max_concurrent if not serialize_installs else 1

            # Create or recreate semaphore based on configuration
            self._pip_install_semaphore = threading.Semaphore(
                self._max_concurrent)

            logger.info(
                f'PipInstallManager configured: serialize={serialize_installs}, '
                f'max_concurrent={self._max_concurrent}')

    def configure_docker(self, max_concurrent: int):
        """Configure the pip install manager for docker runner.

        Args:
            max_concurrent: Maximum concurrent docker container pip installations
        """
        with self._lock:
            self._docker_max_concurrent = max_concurrent
            self._docker_install_semaphore = threading.Semaphore(max_concurrent)

            logger.info(
                f'Docker PipInstallManager configured: max_concurrent={max_concurrent}'
            )

    def get_install_semaphore(self) -> threading.Semaphore:
        """Get the semaphore for subprocess pip installations."""
        if self._pip_install_semaphore is None:
            with self._lock:
                if self._pip_install_semaphore is None:
                    # Default configuration
                    self._pip_install_semaphore = threading.Semaphore(1)
        return self._pip_install_semaphore

    def get_docker_semaphore(self) -> threading.Semaphore:
        """Get the semaphore for docker pip installations."""
        if self._docker_install_semaphore is None:
            with self._lock:
                if self._docker_install_semaphore is None:
                    # Default configuration
                    self._docker_install_semaphore = threading.Semaphore(1)
        return self._docker_install_semaphore

    def is_serialized(self) -> bool:
        """Check if pip installations are being serialized."""
        return self._serialize_installs

    def get_stats(self) -> Dict[str, Any]:
        """Get installation statistics."""
        with self._stats_lock:
            return self._install_stats.copy()

    def reset_stats(self):
        """Reset installation statistics."""
        with self._stats_lock:
            self._install_stats = {
                'total_installs': 0,
                'total_wait_time': 0.0,
                'max_wait_time': 0.0,
                'concurrent_peak': 0,
                'current_active': 0
            }

    @contextmanager
    def managed_install(self,
                        runner_type: str = 'subprocess',
                        track_stats: bool = True):
        """Context manager for managed pip installations.

        Args:
            runner_type: Type of runner ('subprocess' or 'docker')
            track_stats: Whether to track statistics for this installation. Default is True.
                         Set to False when serializing venv creation without package installation.
        """
        semaphore = (
            self.get_docker_semaphore()
            if runner_type == 'docker' else self.get_install_semaphore())

        is_serialized = self.is_serialized(
        ) if runner_type == 'subprocess' else True

        # Record wait time
        start_wait = time.time()

        if is_serialized:
            logger.debug(f'🔒 Acquiring {runner_type} pip installation lock...')

        with semaphore:
            wait_time = time.time() - start_wait

            # Update statistics only if tracking is enabled
            if track_stats:
                with self._stats_lock:
                    self._install_stats['total_installs'] += 1
                    self._install_stats['total_wait_time'] += wait_time
                    self._install_stats['max_wait_time'] = max(
                        self._install_stats['max_wait_time'], wait_time)
                    self._install_stats['current_active'] += 1
                    self._install_stats['concurrent_peak'] = max(
                        self._install_stats['concurrent_peak'],
                        self._install_stats['current_active'])

            if wait_time > 0.1 and is_serialized:  # Only log if we had to wait
                logger.info(
                    f'⏱️  Waited {wait_time:.2f}s for {runner_type} pip installation lock'
                )

            logger.debug(f'🚀 Installing packages with {runner_type} runner...')

            try:
                yield
                logger.debug(
                    f'✅ Package installation completed with {runner_type} runner'
                )
            except Exception as e:
                logger.error(
                    f'❌ Package installation failed with {runner_type} runner: {e}'
                )
                raise
            finally:
                # Update statistics only if tracking is enabled
                if track_stats:
                    with self._stats_lock:
                        self._install_stats['current_active'] -= 1


# Global instance
pip_install_manager = PipInstallManager()

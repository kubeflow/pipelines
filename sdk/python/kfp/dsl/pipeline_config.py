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
"""Pipeline-level config options."""


class PipelineConfig:
    """PipelineConfig contains pipeline-level config options."""

    def __init__(self):
        self.semaphore_key = None
        self.mutex_name = None

    def set_semaphore_key(self, semaphore_key: str):
        """Set the name of the semaphore to control pipeline concurrency.

        Args:
            semaphore_key (str): Name of the semaphore.
        """
        self.semaphore_key = semaphore_key.strip()

    def set_mutex_name(self, mutex_name: str):
        """Set the name of the mutex to ensure mutual exclusion.

        Args:
            mutex_name (str): Name of the mutex.
        """
        self.mutex_name = mutex_name.strip()

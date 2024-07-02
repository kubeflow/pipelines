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


class PipelineConfig():
    """PipelineConfig contains pipeline-level config options."""

    def __init__(self):
        self.completed_pipeline_ttl_seconds = None

    def set_completed_pipeline_ttl_seconds(self, seconds: int):
        """Configures the duration in seconds after which the pipeline platform will
        do garbage collection. This is dependent on the platform's implementation, but
        typically involves deleting pods and higher level resources associated with the
        pods.

        Args:
          seconds: number of seconds for the platform to do garbage collection after
            the pipeline run is completed.
        """
        self.completed_pipeline_ttl_seconds = seconds
        return self

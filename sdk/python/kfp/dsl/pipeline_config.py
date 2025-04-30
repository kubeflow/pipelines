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
from typing import Optional


class PipelineConfig:
    """PipelineConfig contains pipeline-level config options.

    Attributes:
        max_parallelism: Optional[int]: Limits the maximum number of parallel tasks. Unlimited by default.
            . warning::
            Currently supported only when using the Argo backend.
    """
    max_parallelism: Optional[int]

    def __init__(self, max_parallelism: Optional[int] = None):
        self.max_parallelism = max_parallelism

    # TODO add pipeline level configs

# Copyright 2023 The Kubeflow Authors
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
"""The `kfp.local` module contains objects for running KFP components
locally."""

from kfp.local.config import DockerRunner
from kfp.local.config import init
from kfp.local.config import SubprocessRunner

__all__ = [
    'init',
    'SubprocessRunner',
    'DockerRunner',
]

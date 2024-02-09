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

__version__ = '1.1.0'

__all__ = [
    'CreatePVC',
    'DeletePVC',
    'mount_pvc',
    'use_secret_as_env',
    'use_secret_as_volume',
    'add_node_selector',
    'set_image_pull_secrets'
]

from kfp.kubernetes.node_selector import add_node_selector
from kfp.kubernetes.secret import use_secret_as_env
from kfp.kubernetes.secret import use_secret_as_volume
from kfp.kubernetes.volume import CreatePVC
from kfp.kubernetes.volume import DeletePVC
from kfp.kubernetes.volume import mount_pvc
from kfp.kubernetes.image import set_image_pull_secrets

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

import re
from typing import Any, Dict, Optional

# Workspace size validation regex
_SIZE_REGEX = re.compile(
    r'^(?:(?:0|[1-9]\d*)(?:\.\d+)?)(?:Ki|Mi|Gi|Ti|Pi|Ei|K|M|G|T|P|E)?$')


def _is_valid_workspace_size(value: str) -> bool:
    """Returns True if size is a valid Kubernetes resource quantity string."""
    if not isinstance(value, str):
        return False
    size = value.strip()
    return _SIZE_REGEX.match(size) is not None


class KubernetesWorkspaceConfig:
    """Configuration for Kubernetes-specific workspace settings.

    Use this to override the default PersistentVolumeClaim (PVC) configuration
    used when running pipelines on a Kubernetes cluster.

    Attributes:
        pvcSpecPatch: A dictionary of fields to patch onto the default PVC spec
                      (e.g., 'storageClassName', 'accessModes').
    """

    def __init__(self, pvcSpecPatch: Optional[Dict[str, Any]] = None):
        self.pvcSpecPatch = pvcSpecPatch or {}

    def set_pvcSpecPatch(self, patch: Dict[str, Any]):
        self.pvcSpecPatch = patch


class WorkspaceConfig:
    """Configuration for a shared workspace that persists during the pipeline
    run.

    Attributes:
        size (str): The size of the workspace (e.g., '250Gi'). This is a required field.
        See https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/ for valid quantity formats.
        kubernetes: (Optional) Kubernetes-specific configuration for the underlying PVC.
    """

    def __init__(self,
                 size: str,
                 kubernetes: Optional[KubernetesWorkspaceConfig] = None):
        self.size = size
        self.kubernetes = kubernetes or KubernetesWorkspaceConfig()

    @property
    def size(self) -> str:
        return self._size

    @size.setter
    def size(self, size: str) -> None:
        if not size or not str(size).strip():
            raise ValueError('Workspace size is required and cannot be empty')
        if not _is_valid_workspace_size(str(size)):
            raise ValueError(
                f'Workspace size "{size}" is invalid. Must be a valid Kubernetes resource quantity '
                '(e.g., "10Gi", "500Mi", "1Ti")')
        self._size = str(size).strip()

    def get_workspace(self) -> dict:
        workspace = {'size': self.size}
        if self.kubernetes:
            workspace['kubernetes'] = {
                'pvcSpecPatch': self.kubernetes.pvcSpecPatch
            }
        return workspace

    def set_size(self, size: str):
        self.size = size

    def set_kubernetes_config(self,
                              kubernetes_config: KubernetesWorkspaceConfig):
        self.kubernetes = kubernetes_config


class PipelineConfig:
    """PipelineConfig contains pipeline-level config options."""

    def __init__(self,
                 workspace: Optional[WorkspaceConfig] = None,
                 pipeline_version_concurrency_limit: Optional[int] = None):
        self._pipeline_version_concurrency_limit = None
        self.workspace = workspace
        self.pipeline_version_concurrency_limit = pipeline_version_concurrency_limit

    @property
    def pipeline_version_concurrency_limit(self) -> Optional[int]:
        return self._pipeline_version_concurrency_limit

    @pipeline_version_concurrency_limit.setter
    def pipeline_version_concurrency_limit(self, value: Optional[int]) -> None:
        if value is not None:
            if not isinstance(value, int):
                raise ValueError(
                    'pipeline_version_concurrency_limit must be an integer if specified.'
                )
            if value <= 0:
                raise ValueError(
                    'pipeline_version_concurrency_limit must be a positive integer.'
                )
            self._pipeline_version_concurrency_limit = value
        else:
            self._pipeline_version_concurrency_limit = None

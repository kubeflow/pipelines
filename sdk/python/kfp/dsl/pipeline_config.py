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

from typing import Any, Dict, Optional


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
        if not size or not size.strip():
            raise ValueError('Workspace size is required and cannot be empty')
        self.size = size.strip()
        self.kubernetes = kubernetes or KubernetesWorkspaceConfig()

    def get_workspace(self) -> dict:
        workspace = {'size': self.size}
        if self.kubernetes:
            workspace['kubernetes'] = {
                'pvcSpecPatch': self.kubernetes.pvcSpecPatch
            }
        return workspace

    def set_size(self, size: str):
        if not size or not size.strip():
            raise ValueError('Workspace size is required and cannot be empty')
        self.size = size.strip()

    def set_kubernetes_config(self,
                              kubernetes_config: KubernetesWorkspaceConfig):
        self.kubernetes = kubernetes_config


class PipelineConfig:
    """PipelineConfig contains pipeline-level config options."""

    def __init__(self, workspace: Optional[WorkspaceConfig] = None):
        self.workspace = workspace

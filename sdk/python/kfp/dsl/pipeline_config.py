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
        size (str): The size of the workspace (e.g., '250Gi').
        See https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/ for valid quantity formats.
        kubernetes: (Optional) Kubernetes-specific configuration for the underlying PVC.
    """

    def __init__(self,
                 size: str,
                 kubernetes: Optional[KubernetesWorkspaceConfig] = None):
        self.size = size
        self.kubernetes = kubernetes or KubernetesWorkspaceConfig()

    def get_workspace(self) -> dict:
        workspace = {'size': self.size}
        if self.kubernetes:
            workspace['kubernetes'] = {
                'pvcSpecPatch': self.kubernetes.pvcSpecPatch
            }
        return workspace

    def set_size(self, size: str):
        self.size = size.strip()

    def set_kubernetes_config(self,
                              kubernetes_config: KubernetesWorkspaceConfig):
        self.kubernetes = kubernetes_config


class PipelineConfig:
    """PipelineConfig contains pipeline-level config options."""

    def __init__(self,
                 workspace: Optional[WorkspaceConfig] = None,
                 semaphore_key: Optional[str] = None,
                 mutex_name: Optional[str] = None):
        """Initialize a PipelineConfig instance.

        Args:
            workspace: Optional configuration for a shared workspace that persists during the pipeline run.
                       This includes settings for storage size and Kubernetes-specific configurations.
            semaphore_key: Optional key used for semaphore-based concurrency control. When specified,
                           the pipeline will use this key to coordinate execution with other pipelines
                           using the same semaphore key. Support and behavior of this feature is
                           dependent on the underlying pipeline engine used.
            mutex_name: Optional name used for mutex-based concurrency control. When specified,
                        the pipeline will use this mutex to ensure exclusive execution with other
                        pipelines using the same mutex name. Support and behavior of this feature is
                        dependent on the underlying pipeline engine used.
        """
        self.workspace = workspace
        self.semaphore_key = semaphore_key
        self.mutex_name = mutex_name

    def get_pipeline_config(self) -> dict:
        pipeline_config = {}
        if self.workspace:
            pipeline_config['workspace'] = self.workspace.get_workspace()

        if self.semaphore_key:
            pipeline_config['semaphore_key'] = self.semaphore_key

        if self.mutex_name:
            pipeline_config['mutex_name'] = self.mutex_name

        return pipeline_config

    def set_semaphore_key(self, semaphore_key: str):
        """Set the semaphore key for the pipeline to control pipeline
        concurrency. Support and behavior of this feature is dependent on the
        underlying pipeline engine used.

        Args:
            semaphore_key: The semaphore key to use for the pipeline.
        """
        self.semaphore_key = semaphore_key

    def set_mutex_name(self, mutex_name: str):
        """Set the mutex name for the pipeline to ensure mutual exclusion.
        Support and behavior of this feature is dependent on the underlying
        pipeline engine used.

        Args:
            mutex_name: The mutex name to use for the pipeline.
        """
        self.mutex_name = mutex_name

    def set_workspace(self, workspace: WorkspaceConfig):
        """Set the workspace for the pipeline.

        Args:
            workspace: The workspace to use for the pipeline.
        """
        self.workspace = workspace

# Copyright 2025 The Kubeflow Authors
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

from dataclasses import dataclass, field
from typing import Dict, List, Any, Optional


@dataclass
class K8sMetadata:
    name: str
    namespace: str
    annotations: Dict[str, str] = field(default_factory=dict)
    labels: Dict[str, str] = field(default_factory=dict)
    creationTimestamp: Optional[str] = None
    uid: Optional[str] = None


@dataclass
class K8sPipeline:
    apiVersion: str
    kind: str
    metadata: K8sMetadata
    spec: Dict[str, Any] = field(default_factory=dict)

    def get_name(self) -> str:
        return self.metadata.name

    def get_original_id(self) -> Optional[str]:
        return self.metadata.annotations.get('pipelines.kubeflow.org/original-id')


@dataclass
class K8sPipelineVersion:
    apiVersion: str
    kind: str
    metadata: K8sMetadata
    spec: Dict[str, Any] = field(default_factory=dict)

    def get_name(self) -> str:
        return self.metadata.name

    def get_original_id(self) -> Optional[str]:
        return self.metadata.annotations.get('pipelines.kubeflow.org/original-id')

    def get_pipeline_spec(self) -> Optional[Dict[str, Any]]:
        return self.spec.get('pipelineSpec')


@dataclass
class MigrationResult:
    k8s_pipelines: List[K8sPipeline] = field(default_factory=list)
    k8s_pipeline_versions: List[K8sPipelineVersion] = field(default_factory=list)
    yaml_files: List[str] = field(default_factory=list)

    def get_pipeline_by_name(self, name: str) -> Optional[K8sPipeline]:
        for p in self.k8s_pipelines:
            if p.get_name() == name:
                return p
        return None

    def get_pipeline_versions_by_pipeline_name(self, pipeline_name: str) -> List[K8sPipelineVersion]:
        versions: List[K8sPipelineVersion] = []
        for v in self.k8s_pipeline_versions:
            spec = v.get_pipeline_spec() or {}
            info = spec.get('pipelineInfo', {})
            if info.get('name') == pipeline_name:
                versions.append(v)
        return versions



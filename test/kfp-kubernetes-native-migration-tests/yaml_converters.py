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

from pathlib import Path
from typing import Dict, Any
import yaml

from k8s_models import K8sMetadata, K8sPipeline, K8sPipelineVersion, MigrationResult


def _to_metadata(meta: Dict[str, Any]) -> K8sMetadata:
    return K8sMetadata(
        name=meta.get('name', ''),
        namespace=meta.get('namespace', ''),
        annotations=meta.get('annotations', {}) or {},
        labels=meta.get('labels', {}) or {},
        creationTimestamp=meta.get('creationTimestamp'),
        uid=meta.get('uid')
    )


def parse_yaml_files_to_objects(output_dir: Path) -> MigrationResult:
    """Parse YAML files in output_dir and convert to K8sPipeline objects."""
    result = MigrationResult()
    for yaml_file in output_dir.glob("*.yaml"):
        result.yaml_files.append(str(yaml_file))
        with open(yaml_file) as f:
            docs = list(yaml.safe_load_all(f))
            for doc in docs:
                if not doc or 'kind' not in doc:
                    continue
                kind = doc.get('kind')
                api_version = doc.get('apiVersion', '')
                metadata = _to_metadata(doc.get('metadata', {}))
                spec = doc.get('spec', {}) or {}

                if kind == 'Pipeline':
                    result.k8s_pipelines.append(K8sPipeline(apiVersion=api_version, kind=kind, metadata=metadata, spec=spec))
                elif kind == 'PipelineVersion':
                    result.k8s_pipeline_versions.append(K8sPipelineVersion(apiVersion=api_version, kind=kind, metadata=metadata, spec=spec))                
    return result



# Copyright 2026 The Kubeflow Authors
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
"""Repository-level consistency checks for pinned Argo versions."""

import importlib.util
from pathlib import Path
import re
import unittest

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
SYNC_SCRIPT = Path(__file__).with_name('sync_argo_versions.py')
SPEC = importlib.util.spec_from_file_location('sync_argo_versions', SYNC_SCRIPT)
sync_argo_versions = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(sync_argo_versions)


class ArgoVersionConsistencyTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.current_version = (REPOSITORY_ROOT /
                               'third_party/argo/VERSION').read_text(
                                   encoding='utf-8').strip()

    def assert_version_matches(self, relative_path, pattern):
        contents = (REPOSITORY_ROOT / relative_path).read_text(encoding='utf-8')
        versions = set(re.findall(pattern, contents))
        self.assertEqual(versions, {self.current_version}, relative_path)

    def test_ci_and_documented_versions_match_version_files(self):
        self.assertEqual(
            sync_argo_versions.sync(REPOSITORY_ROOT, check=True),
            [],
        )

    def test_current_runtime_references_match_version(self):
        checks = (
            ('go.mod',
             r'github\.com/argoproj/argo-workflows/v\d+ (v\d+\.\d+\.\d+)'),
            ('backend/Dockerfile', r'ARGO_VERSION=(v\d+\.\d+\.\d+)'),
            (
                'backend/src/common/types.go',
                r'argo-workflows/v\d+@(v\d+\.\d+\.\d+)',
            ),
            ('test/install-argo-cli.sh', r'ARGO_VERSION=(v\d+\.\d+\.\d+)'),
            ('third_party/argo/UPGRADE.md', r'ARGO_TAG=(v\d+\.\d+\.\d+)'),
            (
                'manifests/kustomize/third-party/argo/base/kustomization.yaml',
                r'argo-workflows/.+?ref=(v\d+\.\d+\.\d+)',
            ),
            (
                'manifests/kustomize/third-party/argo/base/'
                'workflow-controller-configmap-patch.yaml',
                r'argo-workflows/blob/(v\d+\.\d+\.\d+)',
            ),
            (
                'manifests/kustomize/third-party/argo/base/'
                'workflow-controller-deployment-patch.yaml',
                r'quay\.io/argoproj/(?:workflow-controller|argoexec):'
                r'(v\d+\.\d+\.\d+)',
            ),
            (
                'manifests/kustomize/third-party/argo/installs/cluster/'
                'kustomization.yaml',
                r'argo-workflows/.+?ref=(v\d+\.\d+\.\d+)',
            ),
            (
                'manifests/kustomize/third-party/argo/installs/namespace/'
                'kustomization.yaml',
                r'argo-workflows/.+?ref=(v\d+\.\d+\.\d+)',
            ),
            (
                'manifests/kustomize/third-party/argo/installs/namespace/'
                'cluster-scoped/kustomization.yaml',
                r'argo-workflows/.+?ref=(v\d+\.\d+\.\d+)',
            ),
        )
        for relative_path, pattern in checks:
            with self.subTest(path=relative_path):
                self.assert_version_matches(relative_path, pattern)


if __name__ == '__main__':
    unittest.main()

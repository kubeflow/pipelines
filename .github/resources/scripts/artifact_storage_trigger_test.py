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
"""Guard live storage acceptance against skipped dependency-only changes."""

import fnmatch
from pathlib import Path
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = '.github/workflows/artifact-storage-acceptance.yml'


def triggered(path, patterns):
    # These filters use literal paths and ** suffix/filename patterns. Apply
    # exclusions in order, as GitHub's pull_request.paths evaluation does.
    included = False
    for pattern in patterns:
        excluded = pattern.startswith('!')
        if fnmatch.fnmatchcase(path, pattern.lstrip('!')):
            included = not excluded
    return included


class ArtifactStorageTriggerTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        workflow = yaml.load(
            (ROOT / WORKFLOW).read_text(), Loader=yaml.BaseLoader)
        cls.patterns = workflow['on']['pull_request']['paths']

    def test_dependency_and_shared_code_changes_run_live_acceptance(self):
        for path in (
                'frontend/package.json',
                'frontend/package-lock.json',
                'frontend/.nvmrc',
                'frontend/server/package.json',
                'frontend/server/package-lock.json',
                'frontend/server/vitest.config.ts',
                'frontend/server/tsconfig.json',
                'frontend/server/app.ts',
                'frontend/server/integration-tests/test-helper.ts',
                'frontend/server/integration-tests/artifact-storage-live.test.ts',
                'frontend/server/handlers/artifacts.ts',
                'frontend/server/handlers/domain-checker.ts',
                'frontend/server/minio-helper.ts',
        ):
            with self.subTest(path=path):
                self.assertTrue((ROOT / path).is_file())
                self.assertTrue(triggered(path, self.patterns))

    def test_local_runtime_dependencies_run_live_acceptance(self):
        for path in (
                WORKFLOW,
                '.github/actions/create-cluster/action.yml',
                '.github/actions/github-disk-cleanup/action.yml',
                '.github/actions/github-disk-cleanup/free-disk-space.sh',
                '.github/actions/upload-artifact-with-retry/action.yml',
                '.github/resources/scripts/configure-docker-hub-mirror.sh',
                '.github/resources/scripts/docker-pull-with-retry.sh',
                '.github/resources/scripts/helper-functions.sh',
                '.github/resources/curl-retry/.curlrc',
                'manifests/kustomize/third-party/seaweedfs/base/seaweedfs/seaweedfs-deployment.yaml',
        ):
            with self.subTest(path=path):
                self.assertTrue((ROOT / path).is_file())
                self.assertTrue(triggered(path, self.patterns))

    def test_documentation_and_unrelated_changes_do_not_run_live_acceptance(
            self):
        for path in (
                'frontend/README.md',
                'frontend/server/integration-tests/README.md',
                'frontend/OWNERS',
                '.github/actions/create-cluster/README.md',
                'docs/agents/ci.md',
                'backend/src/apiserver/main.go',
        ):
            with self.subTest(path=path):
                self.assertFalse(triggered(path, self.patterns))


if __name__ == '__main__':
    unittest.main()

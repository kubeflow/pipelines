#!/usr/bin/env python3
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
"""Tests for generated-file dependency change detection."""

import os
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from generated_files_dependency_change import module_version
from generated_files_dependency_change import generated_paths_are_allowed
from generated_files_dependency_change import requires_validation
from generated_files_dependency_change import supports_auto_regeneration
from generated_files_dependency_change import worktree_changed_paths


ROOT_BASE = """require (
    github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0
    google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.6.2
    google.golang.org/protobuf v1.36.11
)
"""
TOOLS_BASE = "require github.com/go-swagger/go-swagger v0.32.3\n"
REPOSITORY_ROOT = Path(__file__).resolve().parents[3]


def manifests(root: str = ROOT_BASE, tools: str = TOOLS_BASE) -> dict[str, str]:
    return {
        'go.mod': root,
        'backend/api/tools/go.mod': tools,
    }


class GeneratedFilesDependencyChangeTest(unittest.TestCase):

    def test_module_version_reads_require_block(self):
        self.assertEqual(
            module_version(
                ROOT_BASE, 'github.com/grpc-ecosystem/grpc-gateway/v2'
            ),
            'v2.29.0',
        )

    def test_untracked_go_module_updates_skip_validation(self):
        changed = [
            'go.mod',
            'go.sum',
            'test/tools/project-cleaner/go.mod',
            'test/tools/project-cleaner/go.sum',
        ]
        self.assertFalse(requires_validation(
            changed, manifests(), manifests()
        ))

    def test_runtime_coupled_generator_update_requires_validation(self):
        updated = ROOT_BASE.replace('v2.29.0', 'v2.30.0')
        self.assertTrue(requires_validation(
            ['go.mod', 'go.sum'], manifests(), manifests(root=updated)
        ))

    def test_standalone_generator_update_requires_validation(self):
        updated = TOOLS_BASE.replace('v0.32.3', 'v0.33.0')
        self.assertTrue(requires_validation(
            ['backend/api/tools/go.mod'],
            manifests(),
            manifests(tools=updated),
        ))

    def test_existing_generator_input_requires_validation(self):
        self.assertTrue(requires_validation(
            ['backend/api/v2beta1/run.proto'], manifests(), manifests()
        ))

    def test_tracked_module_update_supports_auto_regeneration(self):
        updated = ROOT_BASE.replace('v2.29.0', 'v2.30.0')
        self.assertTrue(supports_auto_regeneration(
            ['go.mod', 'go.sum'], manifests(), manifests(root=updated)
        ))

    def test_untracked_module_update_does_not_support_auto_regeneration(self):
        self.assertFalse(supports_auto_regeneration(
            ['go.mod', 'go.sum'], manifests(), manifests()
        ))

    def test_non_module_change_does_not_support_auto_regeneration(self):
        updated = ROOT_BASE.replace('v2.29.0', 'v2.30.0')
        self.assertFalse(supports_auto_regeneration(
            ['go.mod', 'go.sum', 'backend/api/v2beta1/run.proto'],
            manifests(),
            manifests(root=updated),
        ))

    def test_removed_generator_does_not_support_auto_regeneration(self):
        updated = ROOT_BASE.replace(
            '    github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0\n', ''
        )
        self.assertFalse(supports_auto_regeneration(
            ['go.mod', 'go.sum'], manifests(), manifests(root=updated)
        ))

    def test_generated_paths_follow_git_attribute_pattern_semantics(self):
        isolated_git_environment = {
            'GIT_CONFIG_GLOBAL': os.devnull,
            'GIT_CONFIG_NOSYSTEM': '1',
        }
        with mock.patch.dict(os.environ, isolated_git_environment), \
                tempfile.TemporaryDirectory() as temp_directory:
            repository = Path(temp_directory)
            (repository / '.gitattributes').write_text(
                '*.pb.go dependabot-auto-generated\n'
                'generated/*.json dependabot-auto-generated\n',
                encoding='utf-8',
            )
            subprocess.run(('git', 'init', '-q'), cwd=repository, check=True)
            self.assertTrue(generated_paths_are_allowed([
                'nested/output.pb.go',
                'generated/output.json',
            ], repository))
            self.assertFalse(generated_paths_are_allowed([], repository))
            self.assertFalse(generated_paths_are_allowed([
                'generated/nested/output.json',
            ], repository))

    def test_repository_attributes_are_the_generated_path_source_of_truth(self):
        self.assertTrue(generated_paths_are_allowed([
            'backend/api/v1beta1/go_client/run.pb.gw.go',
            'backend/api/v2beta1/swagger/filter.swagger.json',
            'kubernetes_platform/go/kubernetesplatform/config.pb.go',
        ]))
        self.assertFalse(generated_paths_are_allowed(['go.sum']))
        self.assertFalse(generated_paths_are_allowed([
            'backend/api/v1beta1/swagger/pipeline.upload.swagger.json',
            'kubernetes_platform/python/kfp/kubernetes/common.py',
        ]))
        self.assertTrue(generated_paths_are_allowed([
            'kubernetes_platform/python/kfp/kubernetes/'
            'kubernetes_executor_config_pb2.py',
        ]))

    def test_worktree_paths_include_tracked_deleted_and_untracked_files(self):
        isolated_git_environment = {
            'GIT_CONFIG_GLOBAL': os.devnull,
            'GIT_CONFIG_NOSYSTEM': '1',
        }
        with mock.patch.dict(os.environ, isolated_git_environment), \
                tempfile.TemporaryDirectory() as temp_directory:
            repository = Path(temp_directory)
            subprocess.run(('git', 'init', '-q'), cwd=repository, check=True)
            subprocess.run(
                ('git', 'config', 'user.email', 'test@example.com'),
                cwd=repository,
                check=True,
            )
            subprocess.run(
                ('git', 'config', 'user.name', 'Test'),
                cwd=repository,
                check=True,
            )
            (repository / 'modified.txt').write_text('old\n', encoding='utf-8')
            (repository / 'deleted.txt').write_text('old\n', encoding='utf-8')
            subprocess.run(('git', 'add', '.'), cwd=repository, check=True)
            subprocess.run(('git', 'commit', '-qm', 'base'), cwd=repository, check=True)
            (repository / 'modified.txt').write_text('new\n', encoding='utf-8')
            (repository / 'deleted.txt').unlink()
            (repository / 'untracked.txt').write_text('new\n', encoding='utf-8')

            previous_directory = os.getcwd()
            try:
                os.chdir(repository)
                self.assertEqual(
                    worktree_changed_paths(),
                    ['deleted.txt', 'modified.txt', 'untracked.txt'],
                )
            finally:
                os.chdir(previous_directory)

    def test_generator_version_sources_are_wired_to_automation(self):
        dockerfile = (REPOSITORY_ROOT / 'backend/api/Dockerfile').read_text(
            encoding='utf-8')
        for removed_pin in (
            'GRPC_GATEWAY_VERSION',
            'GO_SWAGGER_VERSION',
            'GRPC_VERSION',
            'PROTOC_GEN_GO_GRPC',
            'PROTOBUF_GO',
        ):
            self.assertNotIn(f'ENV {removed_pin}=', dockerfile)
        self.assertIn('COPY go.mod /tmp/kfp-module/go.mod', dockerfile)
        self.assertIn(
            'COPY backend/api/tools/go.mod '
            '/tmp/api-generator-tools/go.mod',
            dockerfile,
        )

        dependabot = (REPOSITORY_ROOT / '.github/dependabot.yml').read_text(
            encoding='utf-8')
        self.assertRegex(
            dependabot,
            r'(?ms)^  - package-ecosystem: gomod\n'
            r'(?:(?!^  - package-ecosystem:).)*?'
            r'^    directories:\n      - "\*\*/\*"$',
        )

        makefile = (REPOSITORY_ROOT / 'backend/api/Makefile').read_text(
            encoding='utf-8')
        self.assertIn(
            '.image-built: Dockerfile ../../go.mod tools/go.mod', makefile
        )

    def test_workflow_runs_detector_and_preserves_required_check(self):
        workflow = (
            REPOSITORY_ROOT / '.github/workflows/validate-generated-files.yml'
        ).read_text(encoding='utf-8')
        self.assertIn("- 'go.mod'", workflow)
        self.assertIn(
            "- 'backend/api/build_kfp_server_api_python_package.sh'",
            workflow,
        )
        self.assertIn('generated_files_dependency_change.py', workflow)
        self.assertIn('detect-generator-changes:', workflow)
        self.assertIn('generate-and-check:', workflow)
        self.assertIn('validate-generated-files:', workflow)
        self.assertIn('dependabot-generated-files-', workflow)
        self.assertIn(
            'uses: ./.github/actions/upload-artifact-with-retry', workflow
        )
        self.assertIn('overwrite: true', workflow)
        self.assertIn("github.actor == 'dependabot[bot]'", workflow)

        updater_workflow = (
            REPOSITORY_ROOT
            / '.github/workflows/update-dependabot-generated-files.yml'
        ).read_text(encoding='utf-8')
        self.assertIn('workflow_run:', updater_workflow)
        self.assertIn('SSH_PRIVATE_KEY', updater_workflow)
        self.assertNotIn('pull_request_target:', updater_workflow)
        self.assertIn(
            './trusted/.github/actions/download-artifact-with-retry',
            updater_workflow,
        )
        self.assertIn(
            'listPullRequestsAssociatedWithCommit', updater_workflow
        )
        self.assertIn(
            'pullRequest.state === "open"', updater_workflow
        )
        self.assertIn('persist-credentials: false', updater_workflow)
        self.assertNotIn('ssh-key:', updater_workflow)
        self.assertNotIn('git diff --check', updater_workflow)
        self.assertNotIn(
            'source/.github/resources/scripts/generated_files_dependency_change.py',
            updater_workflow,
        )
        self.assertLess(
            updater_workflow.index(
                'Confirm the pull request head is still current'
            ),
            updater_workflow.index('SSH_PRIVATE_KEY:'),
        )

        ci_scripts_workflow = (
            REPOSITORY_ROOT / '.github/workflows/ci-scripts-tests.yml'
        ).read_text(encoding='utf-8')
        self.assertIn(
            "- '.github/workflows/validate-generated-files.yml'",
            ci_scripts_workflow,
        )


if __name__ == '__main__':
    unittest.main()

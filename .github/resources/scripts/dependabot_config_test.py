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

from pathlib import Path
from pathlib import PurePosixPath
import re
import unittest

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
DEPENDABOT_PATH = REPOSITORY_ROOT / '.github/dependabot.yml'
CI_SCRIPTS_WORKFLOW_PATH = (
    REPOSITORY_ROOT / '.github/workflows/ci-scripts-tests.yml')
GENERATED_PYTHON_CLIENTS = {
    '/backend/api/v1beta1/python_http_client',
    '/backend/api/v2beta1/python_http_client',
}


def repository_directory(path: Path) -> str:
    relative_directory = path.parent.relative_to(REPOSITORY_ROOT).as_posix()
    return '/' if relative_directory == '.' else f'/{relative_directory}'


def matches_dependabot_directory(directory: str, pattern: str) -> bool:
    # Dependabot defines **/* as the current directory and all descendants.
    if pattern == '**/*':
        return True
    return PurePosixPath(directory).match(pattern)


class DependabotConfigTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.config = DEPENDABOT_PATH.read_text(encoding='utf-8')
        cls.ci_scripts_workflow = CI_SCRIPTS_WORKFLOW_PATH.read_text(
            encoding='utf-8')

    def update_blocks(self) -> list[tuple[str, str]]:
        return re.findall(
            r'^  - package-ecosystem: (\S+)\n'
            r'(.*?)(?=^  - package-ecosystem:|\Z)',
            self.config,
            flags=re.MULTILINE | re.DOTALL,
        )

    def update_block(self, ecosystem: str) -> str:
        matching_blocks = [
            block for configured_ecosystem, block in self.update_blocks()
            if configured_ecosystem == ecosystem
        ]
        self.assertEqual(
            len(matching_blocks),
            1,
            f'expected exactly one Dependabot ecosystem {ecosystem}',
        )
        return matching_blocks[0]

    def configured_directories(self, ecosystem: str) -> set[str]:
        block = self.update_block(ecosystem)
        singular_match = re.search(r'^    directory: "([^"]+)"$', block,
                                   re.MULTILINE)
        if singular_match:
            return {singular_match.group(1)}

        directories_match = re.search(
            r'^    directories:\n((?:      - "[^"]+"\n)+)',
            block,
            re.MULTILINE,
        )
        self.assertIsNotNone(
            directories_match,
            f'missing directories for Dependabot ecosystem {ecosystem}',
        )
        return set(
            re.findall(r'^      - "([^"]+)"$', directories_match.group(1),
                       re.MULTILINE))

    def configured_labels(self, ecosystem: str) -> list[str]:
        block = self.update_block(ecosystem)
        label_keys = re.findall(
            r'^    (?:labels|[\'\"]labels[\'\"])\s*:',
            block,
            flags=re.MULTILINE,
        )
        self.assertEqual(
            len(label_keys),
            1,
            f'expected exactly one labels key for ecosystem {ecosystem}',
        )
        label_blocks = re.findall(
            r'^    labels:\n((?:      - "[^"]+"\n)+)',
            block,
            flags=re.MULTILINE,
        )
        self.assertEqual(
            len(label_blocks),
            1,
            f'expected exactly one labels block for ecosystem {ecosystem}',
        )
        return re.findall(r'^      - "([^"]+)"$', label_blocks[0], re.MULTILINE)

    def test_all_supported_repository_ecosystems_are_configured(self):
        configured_ecosystems = [
            ecosystem for ecosystem, _ in self.update_blocks()
        ]

        self.assertCountEqual(
            configured_ecosystems,
            ('gomod', 'docker', 'npm', 'pip', 'github-actions', 'pre-commit'))
        self.assertEqual(
            len(configured_ecosystems), len(set(configured_ecosystems)))

    def test_version_and_security_updates_start_held_with_defaults_preserved(
            self):
        ecosystem_labels = {
            'gomod': 'go',
            'docker': 'docker',
            'npm': 'javascript',
            'pip': 'python',
            'github-actions': 'github_actions',
            'pre-commit': 'pre_commit',
        }

        for ecosystem, ecosystem_label in ecosystem_labels.items():
            with self.subTest(ecosystem=ecosystem):
                self.assertNotRegex(
                    self.update_block(ecosystem),
                    r'(?m)^    (?:target-branch|[\'\"]target-branch[\'\"])\s*:',
                )
                self.assertEqual(
                    self.configured_labels(ecosystem),
                    ['dependencies', ecosystem_label, 'do-not-merge/hold'],
                )

    def test_all_go_modules_are_covered(self):
        module_directories = {
            repository_directory(path)
            for path in REPOSITORY_ROOT.rglob('go.mod')
        }

        configured_directories = self.configured_directories('gomod')
        self.assertEqual(configured_directories, {'**/*'})
        self.assertTrue(module_directories)
        self.assertTrue(
            all(
                any(
                    matches_dependabot_directory(module_directory, pattern)
                    for pattern in configured_directories)
                for module_directory in module_directories))

    def test_all_npm_projects_are_covered(self):
        npm_directories = {
            repository_directory(path)
            for path in REPOSITORY_ROOT.rglob('package.json')
            if 'node_modules' not in path.parts
        }

        configured_directories = self.configured_directories('npm')
        self.assertEqual(configured_directories, {'**/*'})
        self.assertTrue(npm_directories)
        self.assertTrue(
            all(
                any(
                    matches_dependabot_directory(npm_directory, pattern)
                    for pattern in configured_directories)
                for npm_directory in npm_directories))

    def test_all_maintained_python_projects_are_covered(self):
        python_manifests = set(REPOSITORY_ROOT.rglob('setup.py'))
        python_manifests.update(REPOSITORY_ROOT.rglob('pyproject.toml'))
        python_manifests.update(REPOSITORY_ROOT.rglob('requirements*.txt'))
        python_directories = {
            repository_directory(path)
            for path in python_manifests
            if repository_directory(path) not in GENERATED_PYTHON_CLIENTS
        }

        self.assertEqual(self.configured_directories('pip'), python_directories)

    def test_workflows_and_reusable_actions_are_covered(self):
        configured_directories = self.configured_directories('github-actions')
        self.assertIn('/', configured_directories)
        action_directories = {
            repository_directory(path)
            for path in (REPOSITORY_ROOT /
                         '.github/actions').rglob('action.y*ml')
        }

        self.assertTrue(action_directories)
        self.assertTrue(
            all(
                any(
                    matches_dependabot_directory(action_directory,
                                                 configured_directory)
                    for configured_directory in configured_directories
                    if configured_directory != '/')
                for action_directory in action_directories))

    def test_pre_commit_configuration_is_covered(self):
        self.assertTrue((REPOSITORY_ROOT / '.pre-commit-config.yaml').is_file())
        self.assertEqual(self.configured_directories('pre-commit'), {'/'})

    def test_new_ecosystems_use_bounded_weekly_updates(self):
        for ecosystem in ('npm', 'pip', 'github-actions', 'pre-commit'):
            with self.subTest(ecosystem=ecosystem):
                block = self.update_block(ecosystem)
                self.assertIn('      interval: weekly', block)
                self.assertIn('    open-pull-requests-limit: 10', block)
                self.assertIn('      prefix: "chore(deps)"', block)

    def test_config_changes_run_ci_script_tests(self):
        self.assertIn("      - '.github/dependabot.yml'",
                      self.ci_scripts_workflow)
        for manifest_pattern in (
                '**/go.mod',
                '**/package.json',
                '**/requirements*.txt',
                '**/setup.py',
                '**/pyproject.toml',
                '**/action.yml',
                '**/action.yaml',
                '.pre-commit-config.yaml',
        ):
            with self.subTest(manifest_pattern=manifest_pattern):
                self.assertIn(f"      - '{manifest_pattern}'",
                              self.ci_scripts_workflow)


if __name__ == '__main__':
    unittest.main()

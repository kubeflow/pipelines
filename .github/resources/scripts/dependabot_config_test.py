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

import json
import re
import subprocess
import unittest
from fnmatch import fnmatchcase
from pathlib import Path, PurePosixPath

import yaml

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
DEPENDABOT_PATH = REPOSITORY_ROOT / '.github/dependabot.yml'
CI_SCRIPTS_WORKFLOW_PATH = (
    REPOSITORY_ROOT / '.github/workflows/ci-scripts-tests.yml'
)
GENERATED_PYTHON_CLIENTS = {
    '/backend/api/v2/python_http_client',
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
            encoding='utf-8'
        )

    def update_blocks(self) -> list[tuple[str, str]]:
        return re.findall(
            r'^  - package-ecosystem: (\S+)\n'
            r'(.*?)(?=^  - package-ecosystem:|\Z)',
            self.config,
            flags=re.MULTILINE | re.DOTALL,
        )

    def test_mysql_updates_stay_on_lts_without_blocking_patches(self):
        docker = next(
            update
            for update in yaml.safe_load(self.config)['updates']
            if update['package-ecosystem'] == 'docker'
        )
        rules = [
            rule
            for rule in docker['ignore']
            if fnmatchcase('mysql', rule['dependency-name'])
        ]
        self.assertEqual(
            rules,
            [
                {
                    'dependency-name': 'mysql',
                    'versions': ['<8.4', '>=8.5'],
                }
            ],
        )

    def test_go_generators_are_separate_from_runtime_batches(self):
        gomod = next(
            update
            for update in yaml.safe_load(self.config)['updates']
            if update['package-ecosystem'] == 'gomod'
        )
        for dependency in (
            'github.com/grpc-ecosystem/grpc-gateway/v2',
            'google.golang.org/protobuf',
            'google.golang.org/grpc/cmd/protoc-gen-go-grpc',
            'github.com/go-swagger/go-swagger',
        ):
            matching = [
                name
                for name, group in gomod['groups'].items()
                if any(
                    fnmatchcase(dependency, pattern)
                    for pattern in group.get('patterns', [])
                )
            ]
            self.assertEqual(matching[0], 'go-api-generators', dependency)

    def test_protobuf_updates_stay_compatible_with_gcpc(self):
        uv = next(
            update
            for update in yaml.safe_load(self.config)['updates']
            if update['package-ecosystem'] == 'uv'
        )
        rules = [
            rule
            for rule in uv['ignore']
            if fnmatchcase('protobuf', rule['dependency-name'])
        ]
        self.assertEqual(
            rules,
            [
                {
                    'dependency-name': 'protobuf',
                    'versions': ['>=7'],
                }
            ],
        )

    def update_block(self, ecosystem: str) -> str:
        matching_blocks = [
            block
            for configured_ecosystem, block in self.update_blocks()
            if configured_ecosystem == ecosystem
            and not yaml.safe_load(block).get('target-branch')
        ]
        self.assertEqual(
            len(matching_blocks),
            1,
            f'expected exactly one Dependabot ecosystem {ecosystem}',
        )
        return matching_blocks[0]

    def configured_directories(self, ecosystem: str) -> set[str]:
        block = self.update_block(ecosystem)
        singular_match = re.search(
            r'^    directory: "([^"]+)"$', block, re.MULTILINE
        )
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
            re.findall(
                r'^      - "([^"]+)"$', directories_match.group(1), re.MULTILINE
            )
        )

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
            ecosystem
            for ecosystem, block in self.update_blocks()
            if not yaml.safe_load(block).get('target-branch')
        ]

        self.assertCountEqual(
            configured_ecosystems,
            (
                'gomod',
                'docker',
                'npm',
                'pip',
                'uv',
                'github-actions',
                'pre-commit',
            ),
        )
        self.assertEqual(
            len(configured_ecosystems), len(set(configured_ecosystems))
        )

    def test_version_and_security_updates_start_unheld_with_defaults_preserved(
        self,
    ):
        ecosystem_labels = {
            'gomod': 'go',
            'docker': 'docker',
            'npm': 'javascript',
            'pip': 'python',
            'uv': 'python:uv',
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
                    ['dependencies', ecosystem_label],
                )

    def release_updates(self):
        return [
            update
            for update in yaml.safe_load(self.config)['updates']
            if update.get('target-branch') == 'release-2.18'
        ]

    def release_update(self, ecosystem):
        matches = [
            update
            for update in self.release_updates()
            if update['package-ecosystem'] == ecosystem
        ]
        self.assertEqual(len(matches), 1, ecosystem)
        return matches[0]

    def test_release_ecosystems_are_complete_and_held(self):
        release = self.release_updates()
        labels = {
            'gomod': 'go',
            'docker': 'docker',
            'npm': 'javascript',
            'pip': 'python',
            'github-actions': 'github_actions',
            'pre-commit': 'pre_commit',
        }
        self.assertEqual(len(release), len(labels))
        self.assertEqual(
            {entry['package-ecosystem'] for entry in release}, set(labels)
        )
        for update in release:
            with self.subTest(ecosystem=update['package-ecosystem']):
                self.assertEqual(update['schedule']['interval'], 'weekly')
                self.assertEqual(update['open-pull-requests-limit'], 10)
                self.assertEqual(
                    update['labels'],
                    [
                        'dependencies',
                        labels[update['package-ecosystem']],
                        'do-not-merge/hold',
                    ],
                )
                self.assertNotIn('allow', update)
                self.assertEqual(
                    update['commit-message']['prefix'], 'chore(deps)'
                )

    def test_release_scopes_cover_recursive_projects_and_actions(self):
        for ecosystem in ('gomod', 'docker', 'npm'):
            self.assertEqual(
                self.release_update(ecosystem)['directories'], ['**/*']
            )
        self.assertEqual(
            self.release_update('github-actions')['directories'],
            ['/', '/.github/actions/*'],
        )
        self.assertEqual(self.release_update('pre-commit')['directory'], '/')

    def test_release_python_keeps_legacy_pip_layout_and_excludes_generated_clients(
        self,
    ):
        expected = {
            '/.github/resources/scripts/kfp-readiness',
            '/api/v2alpha1/python',
            '/backend',
            '/backend/metadata_writer',
            '/backend/src/apiserver/visualization',
            '/backend/src/tensorboard',
            '/backend/src/v2/test',
            '/components/google-cloud',
            '/docs',
            '/docs/sdk',
            '/kubernetes_platform/python',
            '/kubernetes_platform/python/docs',
            '/manifests/kustomize/base/installs/multi-user/pipelines-profile-controller',
            '/proxy',
            '/release',
            '/samples/core/artifact_previews',
            '/sdk/python',
            '/test/kfp-kubernetes-native-migration-tests',
            '/tools/upgrade-readiness',
        }
        directories = self.release_update('pip')['directories']
        self.assertEqual(len(directories), len(expected))
        self.assertEqual(set(directories), expected)
        for generated in (
            '/backend/api/v1beta1/python_http_client',
            '/backend/api/v2beta1/python_http_client',
        ):
            self.assertNotIn(generated, directories)
        self.assertNotIn(
            'uv',
            {entry['package-ecosystem'] for entry in self.release_updates()},
        )

    def test_release_known_incompatible_python_and_formatter_updates_are_ignored(
        self,
    ):
        self.assertIn(
            {'dependency-name': 'protobuf', 'versions': ['>=7']},
            self.release_update('pip')['ignore'],
        )
        self.assertIn(
            {
                'dependency-name': 'https://github.com/pycqa/docformatter',
                'versions': ['1.7.8'],
            },
            self.release_update('pre-commit')['ignore'],
        )

    def test_release_argo_images_stay_paired_and_outside_bulk_go_updates(self):
        groups = self.release_update('docker')['groups']
        self.assertEqual(
            set(groups['argo-images']['patterns']),
            {'argoproj/workflow-controller', 'argoproj/argoexec'},
        )
        bulk = self.release_update('gomod')['groups']['go-minor-and-patch']
        self.assertTrue(
            any(
                fnmatchcase('github.com/argoproj/argo-workflows/v4', rule)
                for rule in bulk['exclude-patterns']
            )
        )

    def test_release_docker_does_not_inherit_master_only_mysql_policy(self):
        docker = self.release_update('docker')
        self.assertNotIn('exclude-paths', docker)
        self.assertFalse(
            any(
                fnmatchcase('mysql', rule['dependency-name'])
                for rule in docker.get('ignore', [])
            )
        )
        self.assertEqual(
            docker['groups']['docker-images'], {'group-by': 'dependency-name'}
        )

    def test_release_preserves_coupled_and_shared_dependency_groups(self):
        gomod = self.release_update('gomod')
        self.assertEqual(
            gomod['groups']['golang-x']['patterns'], ['golang.org/x/*']
        )
        npm = self.release_update('npm')['groups']
        self.assertLess(
            list(npm).index('mui'), list(npm).index('npm-dependencies')
        )
        self.assertEqual(npm['mui']['patterns'], ['@mui/*'])
        self.assertEqual(
            npm['npm-dependencies'], {'group-by': 'dependency-name'}
        )
        self.assertEqual(
            self.release_update('pip')['groups']['python-dependencies'],
            {'group-by': 'dependency-name'},
        )

    def test_argo_updates_remain_visible_outside_bulk_go_group(self):
        config = yaml.safe_load(self.config)
        gomod = next(
            update
            for update in config['updates']
            if update['package-ecosystem'] == 'gomod'
        )
        bulk = gomod['groups']['go-minor-and-patch']

        def grouped(dependency):
            return any(
                fnmatchcase(dependency, pattern) for pattern in bulk['patterns']
            ) and not any(
                fnmatchcase(dependency, pattern)
                for pattern in bulk.get('exclude-patterns', [])
            )

        for dependency in (
            'github.com/argoproj/argo-workflows/v3',
            'github.com/argoproj/argo-workflows/v4',
        ):
            with self.subTest(dependency=dependency):
                self.assertFalse(grouped(dependency))
                self.assertFalse(
                    any(
                        fnmatchcase(dependency, rule['dependency-name'])
                        for rule in gomod.get('ignore', [])
                    )
                )
        self.assertNotIn('allow', gomod)
        self.assertTrue(grouped('github.com/stretchr/testify'))

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
                    for pattern in configured_directories
                )
                for module_directory in module_directories
            )
        )

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
                    for pattern in configured_directories
                )
                for npm_directory in npm_directories
            )
        )

    def test_all_maintained_python_projects_are_covered(self):
        # Installed dependencies in .venv are not repository manifests.
        tracked_manifests = subprocess.check_output(
            [
                'git',
                'ls-files',
                '-z',
                '--',
                'setup.py',
                '**/setup.py',
                'pyproject.toml',
                '**/pyproject.toml',
                'requirements*.txt',
                '**/requirements*.txt',
            ],
            cwd=REPOSITORY_ROOT,
            text=True,
        )
        python_manifests = {
            REPOSITORY_ROOT / path
            for path in tracked_manifests.split('\0')
            if path
        }
        python_directories = {
            repository_directory(path)
            for path in python_manifests
            if repository_directory(path) not in GENERATED_PYTHON_CLIENTS
        }

        # Workspace manifests have one owner. Exported requirements are outputs,
        # not an independent pip dependency graph.
        workspace_directories = {'/', '/sdk/python'}
        self.assertEqual(
            self.configured_directories('pip'),
            python_directories - workspace_directories,
        )
        self.assertEqual(self.configured_directories('uv'), {'/'})

    def test_uv_has_no_obsolete_export_exclusions(self):
        config = yaml.safe_load(self.config)
        uv = next(
            update
            for update in config['updates']
            if update['package-ecosystem'] == 'uv'
        )
        self.assertNotIn('exclude-paths', uv)
        self.assertNotIn('allow', uv)

    def test_react_runtime_and_types_share_version_and_security_groups(self):
        config = yaml.safe_load(self.config)
        npm = next(
            update
            for update in config['updates']
            if update['package-ecosystem'] == 'npm'
        )
        dependencies = (
            'react',
            'react-dom',
            '@types/react',
            '@types/react-dom',
        )
        for applies_to in ('version-updates', 'security-updates'):
            groups = {
                name: group
                for name, group in npm['groups'].items()
                if group.get('applies-to', 'version-updates') == applies_to
            }
            matches = []
            for dependency in dependencies:
                matches.append(
                    next(
                        (
                            name
                            for name, group in groups.items()
                            if any(
                                fnmatchcase(dependency, pattern)
                                for pattern in group.get('patterns', [])
                            )
                        ),
                        None,
                    )
                )
            self.assertIsNotNone(matches[0])
            self.assertEqual(len(set(matches)), 1)

    def test_all_uv_lockfiles_are_covered(self):
        tracked_locks = subprocess.check_output(
            ['git', 'ls-files', '-z', '--', 'uv.lock', '**/uv.lock'],
            cwd=REPOSITORY_ROOT,
            text=True,
        )
        lock_directories = {
            repository_directory(REPOSITORY_ROOT / path)
            for path in tracked_locks.split('\0')
            if path
        }
        self.assertTrue(lock_directories)
        self.assertEqual(self.configured_directories('uv'), lock_directories)

    def test_vitest_runner_and_providers_share_update_groups(self):
        npm = next(
            update
            for update in yaml.safe_load(self.config)['updates']
            if update['package-ecosystem'] == 'npm'
        )
        group_names = list(npm['groups'])
        for applies_to, expected_group in (
            ('version-updates', 'vitest-version'),
            ('security-updates', 'vitest-security'),
        ):
            self.assertLess(
                group_names.index(expected_group),
                group_names.index('npm-dependencies'),
            )
            matches = []
            for dependency in (
                'vitest',
                '@vitest/coverage-v8',
                '@vitest/coverage-istanbul',
                '@vitest/ui',
            ):
                matches.append(
                    next(
                        (
                            name
                            for name, group in npm['groups'].items()
                            if group.get('applies-to', 'version-updates')
                            == applies_to
                            and any(
                                fnmatchcase(dependency, pattern)
                                for pattern in group.get('patterns', [])
                            )
                        ),
                        None,
                    )
                )
            self.assertEqual(matches, [expected_group] * 4)

    def test_typescript_eslint_packages_share_update_groups(self):
        npm = next(
            update
            for update in yaml.safe_load(self.config)['updates']
            if update['package-ecosystem'] == 'npm'
        )
        group_names = list(npm['groups'])
        for applies_to, expected_group in (
            ('version-updates', 'typescript-eslint-version'),
            ('security-updates', 'typescript-eslint-security'),
        ):
            self.assertLess(
                group_names.index(expected_group),
                group_names.index('npm-dependencies'),
            )
            for dependency in (
                '@typescript-eslint/parser',
                '@typescript-eslint/eslint-plugin',
                '@typescript-eslint/scope-manager',
                '@typescript-eslint/typescript-estree',
                '@typescript-eslint/utils',
            ):
                with self.subTest(applies_to=applies_to, dependency=dependency):
                    matching_group = next(
                        (
                            name
                            for name, group in npm['groups'].items()
                            if group.get('applies-to', 'version-updates')
                            == applies_to
                            and any(
                                fnmatchcase(dependency, pattern)
                                for pattern in group.get('patterns', ['*'])
                            )
                            and not any(
                                fnmatchcase(dependency, pattern)
                                for pattern in group.get('exclude-patterns', [])
                            )
                        ),
                        None,
                    )
                    self.assertEqual(matching_group, expected_group)

    def test_frontend_typescript_eslint_versions_match(self):
        frontend = REPOSITORY_ROOT / 'frontend'
        manifest = json.loads((frontend / 'package.json').read_text())
        packages = json.loads((frontend / 'package-lock.json').read_text())[
            'packages'
        ]
        expected_version = manifest['devDependencies'][
            '@typescript-eslint/eslint-plugin'
        ]
        for dependency in (
            '@typescript-eslint/eslint-plugin',
            '@typescript-eslint/parser',
        ):
            with self.subTest(dependency=dependency):
                self.assertEqual(
                    manifest['devDependencies'][dependency], expected_version
                )
                self.assertEqual(
                    packages['']['devDependencies'][dependency],
                    expected_version,
                )
                self.assertEqual(
                    packages[f'node_modules/{dependency}']['version'],
                    expected_version,
                )

    def test_frontend_vitest_coverage_matches_locked_runner(self):
        self.assert_vitest_coverage_matches_locked_runner('frontend')

    def test_frontend_node_pins_and_engine_enforcement_agree(self):
        frontend = REPOSITORY_ROOT / 'frontend'
        version = (frontend / '.nvmrc').read_text().strip().removeprefix('v')
        manifest = json.loads((frontend / 'package.json').read_text())
        lock = json.loads((frontend / 'package-lock.json').read_text())
        self.assertEqual(manifest['engines']['node'], version)
        self.assertEqual(lock['packages']['']['engines']['node'], version)
        self.assertIn('engine-strict=true', (frontend / '.npmrc').read_text())

    def test_frontend_node_consumers_use_shared_pin(self):
        for path in (
            '.github/workflows/frontend.yml',
            '.github/workflows/e2e-test-frontend.yml',
            '.github/workflows/image-builds.yml',
            '.github/workflows/build-and-push.yml',
            '.github/actions/build-tools-images/action.yml',
            'test/cloudbuild/frontend.yaml',
            'test/cloudbuild/batch_build.yaml',
        ):
            with self.subTest(path=path):
                self.assertIn(
                    'frontend/.nvmrc', (REPOSITORY_ROOT / path).read_text()
                )
        dockerfile = (REPOSITORY_ROOT / 'frontend/Dockerfile').read_text()
        self.assertIn('FROM node:${NODE_VERSION}-slim AS build', dockerfile)
        self.assertIn('FROM node:${NODE_VERSION}-${BASE_IMAGE}', dockerfile)

    def test_server_vitest_coverage_matches_locked_runner(self):
        self.assert_vitest_coverage_matches_locked_runner('frontend/server')

    def assert_vitest_coverage_matches_locked_runner(self, directory):
        frontend = REPOSITORY_ROOT / directory
        manifest = json.loads((frontend / 'package.json').read_text())
        self.assertEqual(
            manifest['devDependencies']['vitest'],
            manifest['devDependencies']['@vitest/coverage-v8'],
        )
        packages = json.loads((frontend / 'package-lock.json').read_text())[
            'packages'
        ]
        runner = packages['node_modules/vitest']['version']
        coverage = packages['node_modules/@vitest/coverage-v8']
        self.assertEqual(coverage['version'], runner)
        self.assertEqual(coverage['peerDependencies']['vitest'], runner)

    def test_workflows_and_reusable_actions_are_covered(self):
        configured_directories = self.configured_directories('github-actions')
        self.assertIn('/', configured_directories)
        action_directories = {
            repository_directory(path)
            for path in (REPOSITORY_ROOT / '.github/actions').rglob(
                'action.y*ml'
            )
        }

        self.assertTrue(action_directories)
        self.assertTrue(
            all(
                any(
                    matches_dependabot_directory(
                        action_directory, configured_directory
                    )
                    for configured_directory in configured_directories
                    if configured_directory != '/'
                )
                for action_directory in action_directories
            )
        )

    def test_pre_commit_configuration_is_covered(self):
        self.assertTrue((REPOSITORY_ROOT / '.pre-commit-config.yaml').is_file())
        self.assertEqual(self.configured_directories('pre-commit'), {'/'})

    def test_new_ecosystems_use_bounded_weekly_updates(self):
        for ecosystem in ('npm', 'pip', 'uv', 'github-actions', 'pre-commit'):
            with self.subTest(ecosystem=ecosystem):
                block = self.update_block(ecosystem)
                self.assertIn('      interval: weekly', block)
                self.assertIn('    open-pull-requests-limit: 10', block)
                self.assertIn('      prefix: "chore(deps)"', block)

    def test_config_changes_run_ci_script_tests(self):
        self.assertIn(
            "      - '.github/dependabot.yml'", self.ci_scripts_workflow
        )
        for manifest_pattern in (
            '**/go.mod',
            '**/package.json',
            '**/requirements*.txt',
            '**/setup.py',
            '**/pyproject.toml',
            '**/uv.lock',
            '**/action.yml',
            '**/action.yaml',
            '.pre-commit-config.yaml',
        ):
            with self.subTest(manifest_pattern=manifest_pattern):
                self.assertIn(
                    f"      - '{manifest_pattern}'", self.ci_scripts_workflow
                )


if __name__ == '__main__':
    unittest.main()

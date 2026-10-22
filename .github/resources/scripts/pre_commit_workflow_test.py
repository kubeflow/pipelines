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

import os
import re
import shlex
import subprocess
import tempfile
import tomllib
import unittest
from pathlib import Path

import yaml

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
PRE_COMMIT_WORKFLOW = (
    REPOSITORY_ROOT / '.github' / 'workflows' / 'pre-commit.yml'
)
PRE_COMMIT_CONFIG = REPOSITORY_ROOT / '.pre-commit-config.yaml'
CI_SCRIPTS_WORKFLOW = (
    REPOSITORY_ROOT / '.github' / 'workflows' / 'ci-scripts-tests.yml'
)


def golangci_lint_entry(config: str) -> str:
    hook = config.split('      - id: golangci-lint\n', maxsplit=1)[1]
    entry = hook.split('        entry: >-\n', maxsplit=1)[1]
    lines = []
    for line in entry.splitlines():
        if not line.startswith('          '):
            break
        lines.append(line.strip())
    return ' '.join(lines)


class PreCommitWorkflowTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workflow = PRE_COMMIT_WORKFLOW.read_text(encoding='utf-8')
        cls.config = PRE_COMMIT_CONFIG.read_text(encoding='utf-8')
        cls.ci_scripts_workflow = CI_SCRIPTS_WORKFLOW.read_text(
            encoding='utf-8'
        )

    def test_workflow_uses_the_pinned_pre_commit_hooks(self):
        self.assertIn('uses: pre-commit/action@', self.workflow)
        self.assertNotIn('golangci/golangci-lint-action', self.workflow)
        self.assertIn('persist-credentials: false', self.workflow)
        self.assertIn('permissions:\n  contents: read', self.workflow)
        self.assertIn(
            '--from-ref ${{ steps.pre-commit-range.outputs.base-sha }}',
            self.workflow,
        )
        self.assertIn('PRE_COMMIT_FROM_REF', self.config)
        self.assertIn('git diff --quiet "${base}" -- "*.go"', self.config)
        self.assertIn('golangci-lint config verify', self.config)
        self.assertIn(
            'echo "PRE_COMMIT_FROM_REF=${base_sha}" >> "${GITHUB_ENV}"',
            self.workflow,
        )

    def test_whitespace_hooks_preserve_generated_viewer_bundles(self):
        config = yaml.safe_load(self.config)
        hooks = {
            hook['id']: hook
            for repo in config['repos']
            for hook in repo['hooks']
        }
        for hook_id in ('end-of-file-fixer', 'trailing-whitespace'):
            exclude = re.compile(hooks[hook_id].get('exclude', r'^$'))
            for name in ('viewer.js', 'viewer.css'):
                path = f'sdk/python/kfp/cli/viewer_assets/{name}'
                with self.subTest(hook=hook_id, generated_file=path):
                    self.assertRegex(path, exclude)
                    self.assertNotRegex(path, config['exclude'])
            for path in (
                'frontend/src/standalone/Viewer.tsx',
                'frontend/src/standalone/viewer.css',
                'frontend/vite.viewer.config.mts',
                'sdk/python/kfp/cli/visualize.py',
                'sdk/python/kfp/cli/viewer_assets/LICENSES.txt',
            ):
                with self.subTest(hook=hook_id, checked_file=path):
                    self.assertNotRegex(path, exclude)

    def test_viewer_hook_rebuilds_shared_sources_and_bundled_assets(self):
        hooks = {
            hook['id']: hook
            for repo in yaml.safe_load(self.config)['repos']
            for hook in repo['hooks']
        }
        self.assertIn('sdk-viewer-assets', hooks)
        hook = hooks['sdk-viewer-assets']
        self.assertEqual(hook['language'], 'system')
        self.assertEqual(
            shlex.split(hook['entry']),
            ['npm', '--prefix', 'frontend', 'run', 'build:viewer'],
        )
        self.assertFalse(hook['pass_filenames'])
        self.assertTrue(hook['require_serial'])
        for path in (
            'frontend/src/components/graph/PipelineGraph.tsx',
            'frontend/src/standalone/Viewer.tsx',
            'frontend/src/tailwind.css',
            'frontend/package-lock.json',
            'frontend/.nvmrc',
            'frontend/vite.viewer.config.mts',
            'sdk/python/kfp/cli/viewer_assets/viewer.js',
            'sdk/python/kfp/cli/viewer_assets/viewer.css',
            'sdk/python/kfp/cli/viewer_assets/LICENSES.txt',
            '.pre-commit-config.yaml',
            '.github/workflows/pre-commit.yml',
        ):
            with self.subTest(path=path):
                self.assertRegex(path, hook['files'])
        for path in ('sdk/python/kfp/cli/visualize.py', 'backend/main.go'):
            with self.subTest(path=path):
                self.assertNotRegex(path, hook['files'])

    def test_viewer_ci_installs_pinned_toolchain_before_hooks(self):
        steps = yaml.safe_load(self.workflow)['jobs']['pre-commit']['steps']
        node_steps = [
            step
            for step in steps
            if step.get('uses', '').startswith('actions/setup-node@')
        ]
        self.assertEqual(len(node_steps), 1)
        node = node_steps[0]
        self.assertEqual(node['with']['node-version-file'], 'frontend/.nvmrc')
        install = next(
            step
            for step in steps
            if step.get('name') == 'Install viewer build dependencies'
        )
        self.assertEqual(install['working-directory'], 'frontend')
        self.assertIn(
            'require("./package.json").packageManager', install['run']
        )
        self.assertIn('npm ci', install['run'])
        condition = "steps.pre-commit-range.outputs.viewer-required == 'true'"
        self.assertEqual(node['if'], condition)
        self.assertEqual(install['if'], condition)
        self.assertLess(steps.index(node), steps.index(install))
        self.assertLess(
            steps.index(install),
            next(
                i
                for i, step in enumerate(steps)
                if step.get('uses', '').startswith('pre-commit/action@')
            ),
        )

    def test_viewer_ci_setup_covers_hook_inputs_and_config_smoke(self):
        steps = yaml.safe_load(self.workflow)['jobs']['pre-commit']['steps']
        command = next(
            step['run']
            for step in steps
            if step.get('id') == 'pre-commit-range'
        )
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)

            def git(*args):
                return subprocess.check_output(
                    ['git', *args],
                    cwd=root,
                    text=True,
                ).strip()

            git('init', '-q')
            git(
                '-c',
                'user.name=Test',
                '-c',
                'user.email=test@example.com',
                'commit',
                '-qm',
                'base',
                '--allow-empty',
            )
            for path, required in (
                ('backend/main.go', False),
                ('frontend/src/components/graph/PipelineGraph.tsx', True),
                ('frontend/package-lock.json', True),
                ('sdk/python/kfp/cli/viewer_assets/viewer.js', True),
                ('.github/workflows/pre-commit.yml', True),
                ('.pre-commit-config.yaml', True),
                ('uv.lock', True),
            ):
                with self.subTest(path=path):
                    base = git('rev-parse', 'HEAD')
                    file = root / path
                    file.parent.mkdir(parents=True, exist_ok=True)
                    file.write_text('changed\n')
                    git('add', path)
                    git(
                        '-c',
                        'user.name=Test',
                        '-c',
                        'user.email=test@example.com',
                        'commit',
                        '-qm',
                        'change',
                    )
                    output = root / 'output'
                    output.write_text('')
                    result = subprocess.run(
                        ['bash', '-c', command],
                        cwd=root,
                        env={
                            **os.environ,
                            'EVENT_BASE_SHA': base,
                            'GITHUB_OUTPUT': str(output),
                            'GITHUB_ENV': str(root / 'env'),
                        },
                        text=True,
                        capture_output=True,
                        check=False,
                    )
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertIn(
                        f'viewer-required={str(required).lower()}',
                        output.read_text().splitlines(),
                    )

    def test_workflow_fails_closed_when_the_event_base_is_unavailable(self):
        self.assertIn('Unable to resolve the event base commit', self.workflow)
        self.assertNotIn('git rev-parse HEAD^', self.workflow)

    def test_ruff_hooks_match_workspace_dependency(self) -> None:
        """Keep local and CI linting and formatting on the same pinned
        version."""
        project = tomllib.loads(
            (REPOSITORY_ROOT / 'pyproject.toml').read_text(encoding='utf-8')
        )
        dependencies = project['project']['optional-dependencies']['lint']
        versions = [
            dep.removeprefix('ruff==')
            for dep in dependencies
            if dep.startswith('ruff==')
        ]
        self.assertEqual(len(versions), 1)
        lock = tomllib.loads(
            (REPOSITORY_ROOT / 'uv.lock').read_text(encoding='utf-8')
        )
        self.assertEqual(
            [
                package['version']
                for package in lock['package']
                if package['name'] == 'ruff'
            ],
            versions,
        )
        repositories = yaml.safe_load(self.config)['repos']
        hooks = [
            (repo, hook)
            for repo in repositories
            for hook in repo['hooks']
            if hook['id'] in ('ruff-check', 'ruff-format')
        ]
        self.assertEqual(
            [hook['id'] for _, hook in hooks], ['ruff-check', 'ruff-format']
        )
        self.assertEqual(hooks[0][1]['args'], ['--fix'])
        for repo, hook in hooks:
            self.assertEqual(repo['repo'], 'local')
            self.assertEqual(hook['language'], 'system')
            self.assertNotIn('additional_dependencies', hook)
            self.assertEqual(
                shlex.split(hook['entry']),
                [
                    'uv',
                    'run',
                    '--frozen',
                    '--extra',
                    'lint',
                    'ruff',
                    hook['id'].removeprefix('ruff-'),
                ],
            )
            self.assertTrue(hook['require_serial'])
            self.assertEqual(hook['types_or'], ['python', 'pyi'])
        self.assertIn('uses: astral-sh/setup-uv@', self.workflow)
        hook_ids = [
            hook['id'] for repo in repositories for hook in repo['hooks']
        ]
        self.assertLess(
            hook_ids.index('docformatter'), hook_ids.index('ruff-format')
        )
        self.assertTrue(
            {'yapf', 'isort', 'flake8', 'pycln'}.isdisjoint(hook_ids)
        )

    def test_ruff_preserves_excluded_trees_and_public_exports(self) -> None:
        project = tomllib.loads(
            (REPOSITORY_ROOT / 'pyproject.toml').read_text(encoding='utf-8')
        )
        ruff = project['tool']['ruff']
        hook_exclude = re.compile(yaml.safe_load(self.config)['exclude'])
        for path in (
            'sdk/python/kfp/pipeline_spec/pipeline_spec_pb2.py',
            'sdk/python/kfp/kubernetes/kubernetes_executor_config_pb2.py',
            'components/google-cloud/google_cloud_pipeline_components/proto/gcp_resources_pb2.py',
            'components/google-cloud/setup.py',
            'components/google-cloud/google_cloud_pipeline_components/version.py',
            'sdk/python/kfp/server_api/api_client.py',
            'backend/api/v2/python_http_client/setup.py',
            'backend/api/v2beta1/python_http_client/setup.py',
        ):
            with self.subTest(excluded_file=path):
                self.assertRegex(path, hook_exclude)
        self.assertIsNone(
            hook_exclude.search('sdk/python/kfp/dsl/structures.py')
        )
        self.assertTrue(ruff['force-exclude'])
        self.assertEqual(ruff['include'], ['*.py', '*.pyi'])
        self.assertEqual(
            set(ruff['extend-exclude']),
            {
                'backend/api/v2/python_http_client',
                'backend/api/v2beta1/python_http_client',
                'sdk/python/kfp/server_api',
                '*_pb2.py',
                'components/google-cloud',
            },
        )
        self.assertEqual(set(ruff['lint']['select']), {'F401', 'I', 'W605'})
        self.assertIn('F401', ruff['lint']['per-file-ignores']['__init__.py'])
        self.assertEqual(ruff['target-version'], 'py311')

    def test_ruff_presubmit_checks_the_whole_repository_without_writing(self):
        script = (REPOSITORY_ROOT / 'test/presubmit-ruff.sh').read_text()
        self.assertIn('uv run --frozen --extra lint ruff check .', script)
        self.assertIn(
            'uv run --frozen --extra lint ruff format --check .', script
        )
        self.assertNotIn('--fix', script)
        self.assertNotIn('string_fixer', script)
        self.assertNotIn('|| true', script)

    def test_ruff_workflows_keep_gate_paths_and_cover_all_python(self):
        for filename, name, command in (
            (
                'sdk-isort.yml',
                'Python Ruff Lint',
                'uv run --frozen --extra lint ruff check .',
            ),
            (
                'sdk-yapf.yml',
                'Python Ruff Format',
                'uv run --frozen --extra lint ruff format --check .',
            ),
        ):
            with self.subTest(workflow=filename):
                workflow = yaml.safe_load(
                    (
                        REPOSITORY_ROOT / '.github/workflows' / filename
                    ).read_text()
                )
                self.assertEqual(workflow['name'], name)
                # PyYAML's YAML 1.1 loader reads the unquoted "on" key as True.
                paths = workflow[True]['pull_request']['paths']
                self.assertTrue(
                    {'**/*.py', '**/*.pyi', '**/pyproject.toml'}.issubset(paths)
                )
                commands = [
                    step.get('run')
                    for job in workflow['jobs'].values()
                    for step in job['steps']
                ]
                self.assertIn(command, commands)

    def test_config_changes_execute_each_applicable_hook_family(self):
        steps = yaml.safe_load(self.workflow)['jobs']['pre-commit']['steps']
        smoke_steps = [
            step
            for step in steps
            if step.get('if')
            == "steps.pre-commit-range.outputs.config-changed == 'true'"
        ]
        self.assertEqual(len(smoke_steps), 1)
        command = shlex.split(smoke_steps[0]['run'])
        self.assertEqual(command[:2], ['pre-commit', 'run'])
        self.assertIn('--files', command)
        files = command[command.index('--files') + 1 :]
        self.assertIn(
            'git diff --quiet "${base_sha}" HEAD -- '
            '.pre-commit-config.yaml .golangci.yaml pyproject.toml uv.lock',
            self.workflow,
        )
        for representative_file in (
            '.pre-commit-config.yaml',
            '.github/workflows/pre-commit.yml',
            '.golangci.yaml',
            'frontend/package.json',
            '.github/resources/scripts/update_go_version.py',
            'sdk/python/kfp/cli/__init__.py',
            'sdk/python/kfp/dsl/structures.py',
            'backend/src/common/types.go',
        ):
            with self.subTest(representative_file=representative_file):
                self.assertIn(representative_file, files)

        self.assertIn('id: golangci-lint-fmt', self.config)
        self.assertIn('id: golangci-lint-config-verify', self.config)

    def test_go_formatter_receives_only_pre_commit_selected_files(self):
        self.assertIn(
            '      - id: golangci-lint-fmt\n        pass_filenames: true',
            self.config,
        )

    def test_config_only_smoke_runs_bounded_golangci_package(self):
        entry = golangci_lint_entry(self.config)

        with tempfile.TemporaryDirectory() as temp_dir:
            bin_dir = Path(temp_dir)
            calls_path = bin_dir / 'golangci-calls'
            (bin_dir / 'git').write_text(
                '#!/bin/sh\nexit 0\n', encoding='utf-8'
            )
            (bin_dir / 'golangci-lint').write_text(
                '#!/bin/sh\n'
                'printf "%s\\n" "$*" >> "$GOLANGCI_CALLS"\n'
                'if [ "$1" = "config" ]; then\n'
                '  exit "${CONFIG_VERIFY_EXIT:-0}"\n'
                'fi\n'
                'exit "${GOLANGCI_RUN_EXIT:-0}"\n',
                encoding='utf-8',
            )
            (bin_dir / 'git').chmod(0o755)
            (bin_dir / 'golangci-lint').chmod(0o755)

            env = os.environ.copy()
            env.update(
                {
                    'GOLANGCI_CALLS': str(calls_path),
                    'PATH': f'{bin_dir}{os.pathsep}{env["PATH"]}',
                    'PRE_COMMIT_FROM_REF': 'base-sha',
                }
            )
            result = subprocess.run(
                shlex.split(entry),
                check=False,
                env=env,
                text=True,
                capture_output=True,
            )

            self.assertEqual(0, result.returncode, result.stderr)
            self.assertEqual(
                [
                    'config verify',
                    'run ./backend/src/common/plugins',
                ],
                calls_path.read_text(encoding='utf-8').splitlines(),
            )

            calls_path.unlink()
            env['CONFIG_VERIFY_EXIT'] = '17'
            result = subprocess.run(
                shlex.split(entry),
                check=False,
                env=env,
                text=True,
                capture_output=True,
            )

            self.assertEqual(17, result.returncode)
            self.assertEqual(
                ['config verify'],
                calls_path.read_text(encoding='utf-8').splitlines(),
            )

            calls_path.unlink()
            env['CONFIG_VERIFY_EXIT'] = '0'
            env['GOLANGCI_RUN_EXIT'] = '23'
            result = subprocess.run(
                shlex.split(entry),
                check=False,
                env=env,
                text=True,
                capture_output=True,
            )

            self.assertEqual(23, result.returncode)
            self.assertEqual(
                [
                    'config verify',
                    'run ./backend/src/common/plugins',
                ],
                calls_path.read_text(encoding='utf-8').splitlines(),
            )

    def test_workflow_changes_run_ci_script_tests(self):
        self.assertIn(
            "      - '.pre-commit-config.yaml'", self.ci_scripts_workflow
        )
        self.assertIn(
            "      - '.github/workflows/pre-commit.yml'",
            self.ci_scripts_workflow,
        )


if __name__ == '__main__':
    unittest.main()

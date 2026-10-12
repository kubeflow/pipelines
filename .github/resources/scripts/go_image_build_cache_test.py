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
"""Check Go image source coverage and the compiler-cache persistence
contract."""

import json
from pathlib import Path
import re
import shlex
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[3]
DOCKERFILES = (
    'backend/Dockerfile',
    'backend/Dockerfile.driver',
    'backend/Dockerfile.launcher',
    'backend/Dockerfile.persistenceagent',
    'backend/Dockerfile.scheduledworkflow',
    'backend/Dockerfile.viewercontroller',
)
MODULE_PREFIX = 'github.com/kubeflow/pipelines/'


def builder_instructions(path):
    source = (ROOT / path).read_text(encoding='utf-8').replace('\\\n', ' ')
    # These images all compile in their first stage. Later API-server stages
    # intentionally consume Python SDK sources and samples independently.
    return re.split(r'(?m)^FROM ', source)[1].splitlines()


def copied_sources(instructions):
    return [
        Path(source) for line in instructions if line.startswith('COPY ')
        for source in shlex.split(line)[1:-1]
    ]


def is_copied(path, sources):
    return any(source == path or source in path.parents for source in sources)


def local_dependencies(entrypoint):
    """Follow repository import declarations without downloading Go modules.

    This conservatively covers platform-specific source files, as well
    as non-Go package files used by embeds or cgo. Compiler-accurate
    Linux source closure is also checked with go list when changing the
    Dockerfile inputs.
    """
    remaining = [entrypoint]
    seen = set()
    while remaining:
        package = remaining.pop()
        if package in seen:
            continue
        seen.add(package)
        files = list((ROOT / package).glob('*.go'))
        if not files:
            raise AssertionError(f'Local import has no Go source: {package}')
        yield package
        for path in files:
            if path.name.endswith('_test.go'):
                continue
            source = path.read_text(encoding='utf-8')
            for declaration in re.findall(r'(?m)^import(?:\s*\([^)]*\)|[^\n]+)',
                                          source):
                for imported in re.findall(r'"([^"]+)"', declaration):
                    if imported.startswith(MODULE_PREFIX):
                        remaining.append(Path(imported[len(MODULE_PREFIX):]))


class GoImageBuildCacheTest(unittest.TestCase):

    def test_go_compilers_use_the_mount_restored_by_ci(self):
        for path in DOCKERFILES:
            with self.subTest(dockerfile=path):
                instructions = builder_instructions(path)
                commands = [
                    line for line in instructions
                    if line.startswith('RUN ') and 'go build ' in line
                ]
                self.assertEqual(len(commands), 1)
                command = shlex.split(commands[0])
                mounts = [
                    part for part in command if part.startswith('--mount=')
                ]
                self.assertEqual(len(mounts), 1)
                mount = dict(
                    option.split('=', 1)
                    for option in mounts[0][len('--mount='):].split(','))
                self.assertEqual(mount['type'], 'cache')
                self.assertEqual(mount['id'], '${GO_BUILD_CACHE_ID}')
                self.assertEqual(mount['sharing'], 'locked')
                environment = dict(
                    part.split('=', 1)
                    for part in command
                    if part.startswith('GOCACHE='))
                self.assertEqual(mount['target'], environment['GOCACHE'])
                self.assertEqual(mount['target'], '/root/.cache/go-build')
                self.assertIn('ARG GO_BUILD_CACHE_ID=kfp-go-build',
                              instructions)

    def test_copied_sources_cover_local_imports_and_package_data(self):
        for path in DOCKERFILES:
            with self.subTest(dockerfile=path):
                instructions = builder_instructions(path)
                command = next(
                    line for line in instructions
                    if line.startswith('RUN ') and 'go build ' in line)
                entrypoint = Path(shlex.split(command)[-1]).parent
                sources = copied_sources(instructions)
                for package in local_dependencies(entrypoint):
                    # Copy whole package directories so current and future
                    # embedded files are included, not only *.go files.
                    self.assertTrue(
                        is_copied(package, sources),
                        f'{path} omits package data for {package}')

    def test_local_module_metadata_precedes_dependency_download(self):
        replacements = re.findall(r'=>\s+\./([^\s)]+)',
                                  (ROOT / 'go.mod').read_text(encoding='utf-8'))
        required = [Path('go.mod'), Path('go.sum')]
        required.extend(
            Path(directory) / 'go.mod' for directory in replacements)
        for path in DOCKERFILES:
            with self.subTest(dockerfile=path):
                instructions = builder_instructions(path)
                download = next(
                    index for index, line in enumerate(instructions)
                    if line.startswith('RUN ') and 'go mod download' in line)
                sources = copied_sources(instructions[:download])
                for module_file in required:
                    self.assertTrue(
                        is_copied(module_file, sources),
                        f'{path} omits {module_file}')

    def test_source_and_dockerfile_changes_trigger_ci_scripts_tests(self):
        paths = list(DOCKERFILES) + [
            'backend/src/apiserver/main.go',
            'backend/api/v2beta1/go_client/run.pb.go',
            'api/v2alpha1/go/pipelinespec/pipeline_spec.pb.go',
            'kubernetes_platform/go/kubernetesplatform/kubernetes_executor_config.pb.go',
            'go.mod',
            'go.sum',
            'api/go.mod',
            'kubernetes_platform/go.mod',
        ]
        script = r"""
const gate = require('./.github/resources/scripts/ci_expected_workflows.js');
const {inventory} = gate.loadLocalInventory(process.cwd());
const workflow = inventory.workflows.find(workflow =>
  workflow.path === '.github/workflows/ci-scripts-tests.yml');
const paths = JSON.parse(process.argv[1]);
console.log(JSON.stringify(paths.filter(path =>
  !gate.applicable(workflow.pull_request, 'master', [path]))));
"""
        result = subprocess.run(
            ['node', '-e', script, json.dumps(paths)],
            cwd=ROOT,
            check=True,
            capture_output=True,
            text=True)
        self.assertEqual(json.loads(result.stdout), [])

    def test_unrelated_changes_do_not_invalidate_go_compilation(self):
        unrelated = (
            'README.md',
            'docs/agents/ci.md',
            '.github/workflows/image-builds.yml',
            'frontend/package.json',
            'sdk/python/kfp/__init__.py',
            'samples/core/pipeline_with_sidecars/pipeline_with_sidecars.py',
            'backend/test/v2/api/run_test.go',
        )
        for path in DOCKERFILES:
            with self.subTest(dockerfile=path):
                sources = copied_sources(builder_instructions(path))
                for changed in unrelated:
                    self.assertFalse(
                        is_copied(Path(changed), sources),
                        f'{path}: {changed} invalidates Go compilation')


if __name__ == '__main__':
    unittest.main()

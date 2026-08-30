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
"""Repository-level consistency checks for pinned Go versions."""

from pathlib import Path
import re
import subprocess
import unittest

from go_version_metadata import (has_go_runtime_reference, has_setup_go_use,
                                 top_level_module_matches)

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]

GO_DOCKERFILES = {
    Path('backend/Dockerfile'),
    Path('backend/Dockerfile.cacheserver'),
    Path('backend/Dockerfile.conformance'),
    Path('backend/Dockerfile.driver'),
    Path('backend/Dockerfile.launcher'),
    Path('backend/Dockerfile.persistenceagent'),
    Path('backend/Dockerfile.scheduledworkflow'),
    Path('backend/Dockerfile.viewercontroller'),
    Path('backend/api/Dockerfile'),
}

GO_SETUP_ACTIONS = (
    Path('.github/actions/setup-go/action.yml'),
    Path('.github/actions/test-and-report/action.yml'),
)

PRECOMMIT_WORKFLOW = Path('.github/workflows/pre-commit.yml')
CI_SCRIPTS_WORKFLOW = Path('.github/workflows/ci-scripts-tests.yml')
GO_IMAGE_DIGEST_WORKFLOW = Path('.github/workflows/go-image-digests.yml')
GO_VERSION_METADATA = Path(
    '.github/resources/scripts/go_version_metadata.py')

DECIMAL_PATTERN = r'(?:0|[1-9][0-9]*)'

GO_DIRECTIVE_PATTERN = re.compile(
    rf'^[ \t]*go[ \t]+(1\.{DECIMAL_PATTERN}(?:\.{DECIMAL_PATTERN})?)'
    r'(?:[ \t]*//[^\r\n]*)?[ \t]*$', re.MULTILINE)
GO_DIRECTIVE_LINE_PATTERN = re.compile(
    r'^[ \t]*go(?:[ \t]+(.*?))?[ \t]*$', re.MULTILINE)
TOOLCHAIN_PATTERN = re.compile(
    rf'^[ \t]*toolchain[ \t]+go(1\.{DECIMAL_PATTERN}\.{DECIMAL_PATTERN})'
    r'(?:[ \t]*//[^\r\n]*)?[ \t]*$', re.MULTILINE)
TOOLCHAIN_DIRECTIVE_PATTERN = re.compile(
    r'^[ \t]*toolchain(?:[ \t]+(.*?))?[ \t]*$', re.MULTILINE)
GO_IMAGE_PATTERN = re.compile(
    r'^FROM\s+(golang:(\d+\.\d+\.\d+)(-[^@\s]+)?'
    r'@sha256:[0-9a-f]{64})\s+AS\s+\w+',
    re.IGNORECASE | re.MULTILINE,
)
PRECOMMIT_CHECK_PATTERN = re.compile(
    r'(?m)^[ \t]*(?:-[ \t]+)?run:[ \t]+make[ \t]+'
    r'check-go-version[ \t]*$')
DIGEST_CHECK_PATTERN = re.compile(
    r'(?m)^[ \t]*(?:-[ \t]+)?run:[ \t]+python3[ \t]+'
    r'\.github/resources/scripts/update_go_version\.py[ \t]+'
    r'--check-image-digests[ \t]*$')


def _parse_version(version):
    parts = tuple(int(part) for part in version.split('.'))
    return parts + (0,) * (3 - len(parts))


def _read(relative_path):
    return (REPOSITORY_ROOT / relative_path).read_text(encoding='utf-8')


def _repository_paths():
    return {
        Path(path)
        for path in subprocess.run(
            ('git', 'ls-files', '-z', '--cached', '--others',
             '--exclude-standard'),
            cwd=REPOSITORY_ROOT,
            check=True,
            capture_output=True,
            text=True,
        ).stdout.split('\0') if path
    }


def _go_module_paths():
    return sorted(
        path for path in _repository_paths()
        if path.name == 'go.mod' and (REPOSITORY_ROOT / path).exists())


def _module_versions_from_contents(contents, relative_path):
    go_directive_matches = top_level_module_matches(
        contents, GO_DIRECTIVE_LINE_PATTERN)
    go_directive_lines = [match.group(1) for match in go_directive_matches]
    go_directives = [
        match.group(1)
        for match in top_level_module_matches(contents, GO_DIRECTIVE_PATTERN)
    ]
    if len(go_directive_lines) != 1:
        raise ValueError(
            f'{relative_path} must contain exactly one go directive, found '
            f'{go_directive_lines}')
    if len(go_directives) != len(go_directive_lines):
        raise ValueError(
            f'{relative_path} contains an invalid go directive: '
            f'{go_directive_lines}')
    toolchain_directive_matches = top_level_module_matches(
        contents, TOOLCHAIN_DIRECTIVE_PATTERN)
    toolchain_directives = [
        match.group(1) for match in toolchain_directive_matches
    ]
    toolchains = [
        match.group(1)
        for match in top_level_module_matches(contents, TOOLCHAIN_PATTERN)
    ]
    if len(toolchain_directives) > 1:
        raise ValueError(
            f'{relative_path} must contain at most one toolchain directive, '
            f'found {toolchain_directives}')
    if len(toolchains) != len(toolchain_directives):
        raise ValueError(
            f'{relative_path} contains an invalid toolchain directive: '
            f'{toolchain_directives}')
    return _parse_version(go_directives[0]), (
        _parse_version(toolchains[0]) if toolchains else None)


def _module_versions(relative_path):
    return _module_versions_from_contents(_read(relative_path), relative_path)


class GoVersionConsistencyTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        root_go_version, root_toolchain_version = _module_versions(
            Path('go.mod'))
        cls.effective_version = root_toolchain_version or root_go_version

    def test_all_go_modules_match_root_effective_go_version(self):
        expected_minor = self.effective_version[:2]
        for relative_path in _go_module_paths():
            with self.subTest(path=relative_path):
                go_version, toolchain_version = _module_versions(relative_path)
                self.assertEqual(
                    go_version[:2],
                    expected_minor,
                    f'{relative_path} go directive must match the root '
                    'effective Go major and minor version',
                )
                self.assertLessEqual(
                    go_version,
                    self.effective_version,
                    f'{relative_path} go directive cannot exceed the root '
                    'effective Go version',
                )
                if toolchain_version is None:
                    self.assertEqual(
                        go_version,
                        self.effective_version,
                        f'{relative_path} go directive must match the root '
                        'effective Go version when toolchain is omitted',
                    )
                else:
                    self.assertEqual(
                        toolchain_version,
                        self.effective_version,
                        f'{relative_path} toolchain directive must match '
                        'go.mod',
                    )

    def test_malformed_toolchain_directives_are_rejected(self):
        for directive in ('toolchain', 'toolchain go1.27',
                          '  toolchain default',
                          'toolchain go1.27.1 extra'):
            with self.subTest(directive=directive):
                contents = f'module example.com/test\n\ngo 1.27.0\n\n{directive}\n'
                with self.assertRaisesRegex(ValueError,
                                            'invalid toolchain directive'):
                    _module_versions_from_contents(contents,
                                                   Path('test/go.mod'))

    def test_malformed_go_directives_are_rejected(self):
        for contents in ('module example.com/test\n\ngo\n',
                         'module example.com/test\n\ngo 1.27.0 extra\n',
                         'module example.com/test\n\ngo 1.27.0\n\n  go\n',
                         'module example.com/test\n\ngo 1.027.000\n',
                         'module example.com/test\n\ngo 1.٢٧.٠\n',
                         'module example.com/test\n\ngo １.２７.０\n'):
            with self.subTest(contents=contents):
                with self.assertRaisesRegex(ValueError, 'go directive'):
                    _module_versions_from_contents(contents,
                                                   Path('test/go.mod'))

    def test_indented_module_directives_are_parsed(self):
        self.assertEqual(
            _module_versions_from_contents(
                'module example.com/test\n\n  go 1.27.0// language floor\n\n'
                '\ttoolchain go1.27.1// compiler\n', Path('test/go.mod')),
            ((1, 27, 0), (1, 27, 1)),
        )

    def test_module_block_entries_are_not_directives(self):
        contents = (
            'module example.com/test\n\n'
            'require (\n'
            '  go v1.0.0\n'
            '  toolchain v1.0.0\n'
            ')\n\n'
            'go 1.27.0\n\n'
            'toolchain go1.27.1\n')

        self.assertEqual(
            _module_versions_from_contents(contents, Path('test/go.mod')),
            ((1, 27, 0), (1, 27, 1)),
        )

    def test_all_go_builder_images_match_root_effective_go_version(self):
        discovered = set()
        image_references_by_flavor = {}
        for relative_path in _repository_paths():
            if not relative_path.name.startswith('Dockerfile'):
                continue
            dockerfile = REPOSITORY_ROOT / relative_path
            if not dockerfile.exists():
                continue
            matches = GO_IMAGE_PATTERN.findall(
                dockerfile.read_text(encoding='utf-8'))
            if not matches:
                continue
            discovered.add(relative_path)
            with self.subTest(path=relative_path):
                self.assertEqual(
                    len(matches),
                    1,
                    f'{relative_path} must contain exactly one Go builder image',
                )
                image_reference, version, flavor = matches[0]
                self.assertEqual(
                    _parse_version(version),
                    self.effective_version,
                    f'{relative_path} Go image must match the root effective '
                    'Go version',
                )
                image_references_by_flavor.setdefault(flavor, set()).add(
                    image_reference)

        self.assertEqual(
            discovered,
            GO_DOCKERFILES,
            'update GO_DOCKERFILES when adding or removing a Go builder image',
        )
        for flavor, image_references in image_references_by_flavor.items():
            with self.subTest(flavor=flavor or '<default>'):
                self.assertEqual(
                    len(image_references),
                    1,
                    'Go builder images with the same flavor must use the same '
                    'tag and digest',
                )

    def test_no_unmanaged_go_runtime_pins(self):
        discovered = set()
        for relative_path in _repository_paths():
            if not (relative_path.name.startswith('Dockerfile') or
                    relative_path.suffix in {'.sh', '.yaml', '.yml'}):
                continue
            path = REPOSITORY_ROOT / relative_path
            if not path.exists():
                continue
            if has_go_runtime_reference(
                    relative_path,
                    path.read_text(encoding='utf-8', errors='ignore')):
                discovered.add(relative_path)

        self.assertEqual(
            discovered,
            GO_DOCKERFILES,
            'all Go runtime pins must be managed builder images; remove stale '
            'pins or add an intentional image to GO_DOCKERFILES',
        )

    def test_go_runtime_reference_detection_is_tag_agnostic(self):
        for reference in ('FROM golang AS builder',
                          'FROM golang@sha256:' + ('a' * 64) + ' AS builder',
                          'container: golang',
                          'image: docker.io/library/golang',
                          '  - image: "golang" # builder',
                          'container: { image: golang }',
                          'container: {"image":"golang"}',
                          'container: {credentials: {user: test}, image: golang}',
                          'FROM golang:latest',
                          'FROM golang:${GO_VERSION}',
                          'https://dl.google.com/go/go${GO_VERSION}.tar.gz',
                          'https://go.dev/dl/go1.27.0.linux-amd64.tar.gz'):
            with self.subTest(reference=reference):
                relative_path = (Path('Dockerfile')
                                 if reference.startswith('FROM') else
                                 Path('test.yaml'))
                self.assertTrue(
                    has_go_runtime_reference(relative_path, reference))
        for non_runtime_reference in ('language: golang', 'golangci-lint'):
            with self.subTest(non_runtime_reference=non_runtime_reference):
                self.assertFalse(
                    has_go_runtime_reference(Path('test.yaml'),
                                             non_runtime_reference))
        for commented_reference in (
                '# golang:latest',
                '  # image: golang:latest',
                '# https://go.dev/dl/go1.27.0.linux-amd64.tar.gz',
                'run: echo \'{"image":"golang"}\'',
                'container: alpine:3.22 # not golang:latest',
        ):
            with self.subTest(commented_reference=commented_reference):
                self.assertFalse(
                    has_go_runtime_reference(Path('test.yaml'),
                                             commented_reference))

    def test_setup_go_detection_supports_workflow_list_items(self):
        for step in ('      uses: actions/setup-go@v7',
                     '    - uses: actions/setup-go@v7',
                     '    - uses: actions/setup-go@abc123 # v7',
                     "    - uses: 'actions/setup-go@abc123' # v7",
                     '    - uses: "actions/setup-go@abc123" # v7',
                     '    - {uses: actions/setup-go@abc123, name: Go}',
                     '    - {"uses":"actions/setup-go@abc123","name":"Go"}',
                     '    - {name: Go, "uses":"actions/setup-go@abc123"}',
                     '    steps: [{env: {A: B}, uses: actions/setup-go@abc123}]'):
            with self.subTest(step=step):
                self.assertTrue(has_setup_go_use(step))
        self.assertFalse(has_setup_go_use(
            '    # - uses: actions/setup-go@v7'))
        self.assertFalse(has_setup_go_use(
            '    # - {"uses":"actions/setup-go@v7"}'))
        self.assertFalse(has_setup_go_use(
            '    run: echo \'{"uses":"actions/setup-go@v7"}\''))
        self.assertFalse(has_setup_go_use(
            '    run: |\n'
            '      cat <<EOF\n'
            '      uses: actions/setup-go@v7\n'
            '      EOF\n'))

    def test_setup_go_actions_use_root_module_version(self):
        for relative_path in GO_SETUP_ACTIONS:
            contents = _read(relative_path)
            with self.subTest(path=relative_path):
                self.assertRegex(
                    contents,
                    r'(?m)^\s+go-version-file:\s+go\.mod\s*$',
                    f'{relative_path} must resolve Go from the root go.mod',
                )
                self.assertNotRegex(
                    contents,
                    r'(?m)^\s+go-version:\s+',
                    f'{relative_path} must not contain a separate Go version',
                )

    def test_all_direct_setup_go_uses_are_managed(self):
        discovered = set()
        for relative_path in _repository_paths():
            if relative_path.suffix not in {'.yaml', '.yml'}:
                continue
            path = REPOSITORY_ROOT / relative_path
            if path.exists() and has_setup_go_use(
                    path.read_text(encoding='utf-8', errors='ignore')):
                discovered.add(relative_path)
        self.assertEqual(
            discovered,
            set(GO_SETUP_ACTIONS),
            'route new setup-go callers through the managed composite action '
            'or add them to GO_SETUP_ACTIONS',
        )

    def test_presubmit_runs_go_version_consistency_check(self):
        self.assertRegex(_read(PRECOMMIT_WORKFLOW), PRECOMMIT_CHECK_PATTERN)
        self.assertNotRegex('# run: make check-go-version',
                            PRECOMMIT_CHECK_PATTERN)

    def test_precommit_wiring_changes_run_ci_script_tests(self):
        contents = _read(CI_SCRIPTS_WORKFLOW)
        self.assertRegex(
            contents,
            r"(?m)^[ \t]*-[ \t]+'\.github/workflows/pre-commit\.yml'[ \t]*$",
        )
        self.assertRegex(
            contents,
            r"(?m)^[ \t]*-[ \t]+'\.github/workflows/go-image-digests\.yml'"
            r'[ \t]*$',
        )
        self.assertNotRegex(contents, DIGEST_CHECK_PATTERN)
        self.assertRegex(_read(GO_IMAGE_DIGEST_WORKFLOW),
                         DIGEST_CHECK_PATTERN)
        self.assertIn(str(GO_VERSION_METADATA),
                      _read(GO_IMAGE_DIGEST_WORKFLOW))


if __name__ == '__main__':
    unittest.main()

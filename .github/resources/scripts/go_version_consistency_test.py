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
import json
import re
import shlex
import subprocess
import unittest

from go_version_metadata import docker_buildkit_metadata
from go_version_metadata import docker_runtime_classification
from go_version_metadata import has_go_runtime_reference
from go_version_metadata import has_setup_go_use
from go_version_metadata import is_container_recipe
from go_version_metadata import is_runtime_metadata_path
from go_version_metadata import is_yaml_metadata_path
from go_version_metadata import module_versions

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
GO_VERSION_METADATA = Path('.github/resources/scripts/go_version_metadata.py')
GO_VERSION_HELPER = Path('tools/go-version-metadata/main.go')

# `backend/api/tools/go.mod` carries no Go source: it exists so the API
# generator image can read a pinned tool version out of it. Because nothing
# imports the module, `go mod tidy` reports `"all" matched no packages` and
# prunes the requirement, which silently breaks the image build.
API_TOOLS_MODULE = Path('backend/api/tools/go.mod')
API_TOOLS_DOCKERFILE = Path('backend/api/Dockerfile')
API_TOOLS_PINS = ('github.com/go-swagger/go-swagger',)
API_TOOLS_CONTAINER_MODULE = '/tmp/api-generator-tools'
API_TOOL_BASE_STAGE = 'go-base'
API_TOOL_INSTALLER_STAGE = 'api-tool-installer'
API_TOOL_GENERATOR_STAGE = 'generator'
API_TOOLS_VERSION_VARIABLES = {
    'github.com/go-swagger/go-swagger': 'go_swagger_version',
}

DECIMAL_PATTERN = r'(?:0|[1-9][0-9]*)'

GO_DIRECTIVE_PATTERN = re.compile(
    rf'^[ \t]*go[ \t]+(1\.{DECIMAL_PATTERN}(?:\.{DECIMAL_PATTERN})?)'
    r'(?:[ \t]*//[^\r\n]*)?[ \t]*$', re.MULTILINE)
TOOLCHAIN_PATTERN = re.compile(
    rf'^[ \t]*toolchain[ \t]+go(1\.{DECIMAL_PATTERN}\.{DECIMAL_PATTERN})'
    r'(?:[ \t]*//[^\r\n]*)?[ \t]*$', re.MULTILINE)
ROOT_GO_DIRECTIVE_PATTERN = re.compile(
    rf'^go (1\.{DECIMAL_PATTERN}\.{DECIMAL_PATTERN})'
    r'(?:[ \t]*//[^\r\n]*)?[ \t]*$', re.MULTILINE)
ROOT_TOOLCHAIN_PATTERN = re.compile(
    rf'^toolchain go(1\.{DECIMAL_PATTERN}\.{DECIMAL_PATTERN})'
    r'(?:[ \t]*//[^\r\n]*)?[ \t]*$', re.MULTILINE)
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


def _api_tool_install_commands(module_path):
    """The frozen repository-specific API tool installation protocol."""
    variable = API_TOOLS_VERSION_VARIABLES[module_path]
    module_file = API_TOOLS_CONTAINER_MODULE + '/go.mod'
    return [
        'set -eu',
        f'{variable}="$({_api_tool_version_query(module_path, module_file)})"',
        "/usr/bin/curl -q -fL --proto '=https' --proto-redir '=https' "
        '-o /usr/bin/swagger '
        '"https://github.com/go-swagger/go-swagger/releases/download/'
        f'${{{variable}}}/swagger_linux_amd64"',
        '/bin/chmod +x /usr/bin/swagger',
        f'/usr/bin/swagger version | /usr/bin/grep -Fqx '
        f'"version: ${{{variable}}}"',
    ]


def _api_tool_install_argv(module_path):
    """Hermetic exec-form argv for manifest-to-artifact installation."""
    return [
        '/usr/bin/env',
        '-i',
        'HOME=/tmp',
        'PATH=/usr/local/go/bin:/usr/bin:/bin',
        'GOENV=off',
        'GOTOOLCHAIN=local',
        'GOFLAGS=-mod=readonly',
        'GOWORK=off',
        '/bin/sh',
        '-c',
        '; '.join(_api_tool_install_commands(module_path)),
    ]


def _api_tool_install_instruction(module_path, script=None):
    """Render the isolated exec-form instruction used by the Dockerfile."""
    argv = _api_tool_install_argv(module_path)
    if script is not None:
        argv[-1] = script
    return 'RUN ' + json.dumps(argv)


def _api_tool_version_query(module_path, module_file):
    """Read one selected module version with the pinned Go executable."""
    module_file = Path(module_file)
    return (
        f'cd {shlex.quote(str(module_file.parent))} && '
        '/usr/local/go/bin/go list -m -mod=readonly '
        f'-modfile={shlex.quote(str(module_file))} '
        f"-f '{{{{.Version}}}}' {shlex.quote(module_path)}")


def _docker_instruction_payload(instruction):
    return {key: value for key, value in instruction.items() if key != 'line'}


def _api_tool_pin_dataflow_errors(contents, module_path):
    """Validate the isolated BuildKit stage and final artifact transfer."""
    expected_installer_stage = {
        'command': 'from',
        'flags': [],
        'stage': {
            'baseName': API_TOOL_BASE_STAGE,
            'name': API_TOOL_INSTALLER_STAGE,
            'platform': '',
        },
    }
    expected_copy = {
        'command': 'copy',
        'flags': [],
        'copy': {
            'from': '',
            'sources': [str(API_TOOLS_MODULE)],
            'destination': API_TOOLS_CONTAINER_MODULE + '/go.mod',
            'inlineSources': 0,
        },
    }
    expected_run = {
        'command': 'run',
        'flags': [],
        'run': {
            'arguments': _api_tool_install_argv(module_path),
            'prependShell': False,
            'heredocFiles': 0,
        },
    }
    expected_generator_stage = {
        'command': 'from',
        'flags': [],
        'stage': {
            'baseName': API_TOOL_BASE_STAGE,
            'name': API_TOOL_GENERATOR_STAGE,
            'platform': '',
        },
    }
    expected_artifact_copy = {
        'command': 'copy',
        'flags': [f'--from={API_TOOL_INSTALLER_STAGE}'],
        'copy': {
            'from': API_TOOL_INSTALLER_STAGE,
            'sources': ['/usr/bin/swagger'],
            'destination': '/usr/bin/swagger',
            'inlineSources': 0,
        },
    }
    errors = []
    projection = docker_buildkit_metadata(contents)
    directives = projection['directives']
    syntax_directives = [
        directive for directive in directives
        if directive.get('name') == 'syntax'
    ]
    if syntax_directives:
        return [
            'API tools Dockerfile must not select a custom Docker frontend; '
            f'found active syntax directive {syntax_directives[0]!r}'
        ]
    instructions = projection['instructions']
    if len(instructions) < 6:
        errors.append(
            'API tools Dockerfile must contain the isolated installer and '
            'generator-stage artifact transfer protocol')
        return errors

    base = instructions[0]
    base_payload = _docker_instruction_payload(base)
    base_stage = base_payload.get('stage', {})
    if (base_payload.get('command') != 'from'
            or base_payload.get('flags') != []
            or base_stage.get('name') != API_TOOL_BASE_STAGE
            or base_stage.get('platform') != ''
            or not re.fullmatch(
                r'golang:[0-9]+\.[0-9]+\.[0-9]+'
                r'(?:-[a-z0-9][a-z0-9._-]*)?@sha256:[0-9a-f]{64}',
                base_stage.get('baseName', ''),
            )):
        errors.append(
            f'line {base.get("line", "?")}: first Docker instruction is '
            f'{base_payload!r}; want an otherwise-empty canonical Golang '
            f'stage named {API_TOOL_BASE_STAGE!r}')

    expected_installer = [
        expected_installer_stage,
        expected_copy,
        expected_run,
        expected_generator_stage,
    ]
    actual_installer = [
        _docker_instruction_payload(instruction)
        for instruction in instructions[1:5]
    ]
    if actual_installer != expected_installer:
        errors.append(
            'the instructions between the pinned base and generator are '
            f'{actual_installer!r}; want the complete hermetic installer '
            f'protocol {expected_installer!r}')

    stage_names = [
        instruction.get('stage', {}).get('name')
        for instruction in instructions if instruction.get('command') == 'from'
    ]
    expected_stage_names = [
        API_TOOL_BASE_STAGE,
        API_TOOL_INSTALLER_STAGE,
        API_TOOL_GENERATOR_STAGE,
    ]
    if stage_names != expected_stage_names:
        errors.append(
            f'Docker stages are {stage_names!r}; want exactly '
            f'{expected_stage_names!r}')

    artifact_copy = instructions[-1]
    artifact_payload = _docker_instruction_payload(artifact_copy)
    if artifact_payload != expected_artifact_copy:
        errors.append(
            f'line {artifact_copy.get("line", "?")}: final Docker '
            f'instruction is {artifact_payload!r}; want '
            f'{expected_artifact_copy!r}')
    return errors


def _repository_paths():
    return {
        Path(path) for path in subprocess.run(
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
    go_version, toolchain_version = module_versions(contents, relative_path)
    return _parse_version(go_version), (
        _parse_version(toolchain_version) if toolchain_version else None)


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

    def test_root_module_uses_setup_go_canonical_directives(self):
        contents = _read(Path('go.mod'))
        go_version, toolchain_version = module_versions(contents,
                                                        Path('go.mod'))
        go_matches = ROOT_GO_DIRECTIVE_PATTERN.findall(contents)
        self.assertEqual(
            go_matches,
            [go_version],
            'root go.mod must contain one column-zero, patch-qualified go '
            'directive so actions/setup-go resolves an exact version',
        )
        toolchain_matches = ROOT_TOOLCHAIN_PATTERN.findall(contents)
        self.assertEqual(
            toolchain_matches,
            [toolchain_version] if toolchain_version else [],
            'the root toolchain directive, when present, must be column-zero '
            'and patch-qualified so actions/setup-go can resolve it',
        )

    def test_root_directives_use_setup_go_literal_space_syntax(self):
        for directive in ('go\t1.27.0', 'go  1.27.0'):
            with self.subTest(directive=directive):
                self.assertEqual(ROOT_GO_DIRECTIVE_PATTERN.findall(directive),
                                 [])
        for directive in ('toolchain\tgo1.27.1',
                          'toolchain  go1.27.1'):
            with self.subTest(directive=directive):
                self.assertEqual(
                    ROOT_TOOLCHAIN_PATTERN.findall(directive), [])

    def test_malformed_toolchain_directives_are_rejected(self):
        for directive in ('toolchain', 'toolchain go1.27',
                          '  toolchain default', 'toolchain go1.27.1 extra'):
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
        contents = ('module example.com/test\n\n'
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

    def test_malformed_go_and_toolchain_blocks_are_rejected(self):
        for contents in ('module example.com/test\n\ngo (\n  1.27.0\n)\n',
                         'module example.com/test\n\ngo 1.27.0\n\n'
                         'toolchain (\n  go1.27.1\n)\n'):
            with self.subTest(contents=contents):
                with self.assertRaisesRegex(ValueError, 'invalid .* directive'):
                    _module_versions_from_contents(contents,
                                                   Path('test/go.mod'))

    def test_all_go_builder_images_match_root_effective_go_version(self):
        discovered = set()
        image_references_by_flavor = {}
        for relative_path in _repository_paths():
            if not is_container_recipe(relative_path):
                continue
            dockerfile = REPOSITORY_ROOT / relative_path
            if not dockerfile.exists():
                continue
            docker = docker_runtime_classification(
                dockerfile.read_text(encoding='utf-8'))
            if docker['classification'] == 'irrelevant':
                continue
            discovered.add(relative_path)
            with self.subTest(path=relative_path):
                self.assertEqual(
                    docker['classification'],
                    'managed',
                    f'{relative_path} must use the canonical literal '
                    'digest-pinned Golang FROM form',
                )
                self.assertEqual(
                    len(docker['candidates']),
                    1,
                    f'{relative_path} must contain exactly one Go builder image',
                )
                candidate = docker['candidates'][0]
                version = candidate['version']
                flavor = candidate['flavor']
                image_reference = (
                    f'golang:{version}{flavor}@{candidate["digest"]}')
                self.assertEqual(
                    _parse_version(version),
                    self.effective_version,
                    f'{relative_path} Go image must match the root effective '
                    'Go version',
                )
                image_references_by_flavor.setdefault(
                    flavor, set()).add(image_reference)

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

    def test_all_go_builder_images_reject_custom_frontends(self):
        for relative_path in sorted(GO_DOCKERFILES):
            contents = _read(relative_path)
            for prefix in (
                    '# syntax=example.com/frontend:latest\n',
                    '#\u00a0syntax=example.com/frontend:latest\n'):
                with self.subTest(path=relative_path, prefix=repr(prefix)):
                    docker = docker_runtime_classification(prefix + contents)
                    self.assertEqual(docker['classification'], 'unsupported')
                    self.assertIn(
                        'unsupported-frontend',
                        [candidate['kind']
                         for candidate in docker['candidates']],
                    )

    def test_no_unmanaged_go_runtime_pins(self):
        discovered = set()
        for relative_path in _repository_paths():
            if not is_runtime_metadata_path(relative_path):
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
        self.assertTrue(
            has_go_runtime_reference(
                Path('runtime.YAML'), 'image: golang:1.20\n'))
        for reference in (
                'FROM golang AS builder',
                'FROM golang@sha256:' + ('a' * 64) + ' AS builder',
                'container: golang', 'image: docker.io/library/golang',
                '  - image: "golang" # builder', 'container: { image: golang }',
                'container: {"image":"golang"}',
                'container: {credentials: {user: test}, image: golang}',
                'steps: [{uses: docker://golang:1.20}]', 'FROM golang:latest',
                'FROM golang:${GO_VERSION}',
                'https://dl.google.com/go/go${GO_VERSION}.tar.gz',
                'https://go.dev/dl/go1.27.0.linux-amd64.tar.gz'):
            with self.subTest(reference=reference):
                relative_path = (
                    Path('Dockerfile')
                    if reference.startswith('FROM') else Path('test.yaml'))
                self.assertTrue(
                    has_go_runtime_reference(relative_path, reference))
        for non_runtime_reference in ('language: golang', 'golangci-lint'):
            with self.subTest(non_runtime_reference=non_runtime_reference):
                self.assertFalse(
                    has_go_runtime_reference(
                        Path('test.yaml'), non_runtime_reference))
        for commented_reference in (
                '# golang:latest',
                '  # image: golang:latest',
                '# https://go.dev/dl/go1.27.0.linux-amd64.tar.gz',
                'run: echo \'{"image":"golang"}\'',
                'container: alpine:3.22 # not golang:latest',
        ):
            with self.subTest(commented_reference=commented_reference):
                self.assertFalse(
                    has_go_runtime_reference(
                        Path('test.yaml'), commented_reference))

    def test_setup_go_detection_supports_workflow_list_items(self):
        for step in (
                '      uses: actions/setup-go@v7',
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
        self.assertTrue(has_setup_go_use('    - uses: Actions/Setup-Go@v7'))
        self.assertFalse(has_setup_go_use('    # - uses: actions/setup-go@v7'))
        self.assertFalse(
            has_setup_go_use('    # - {"uses":"actions/setup-go@v7"}'))
        self.assertFalse(
            has_setup_go_use(
                '    run: echo \'{"uses":"actions/setup-go@v7"}\''))
        self.assertFalse(
            has_setup_go_use('    run: |\n'
                             '      cat <<EOF\n'
                             '      uses: actions/setup-go@v7\n'
                             '      EOF\n'))
        for action in ('./.github/actions/golang', 'owner/golang@v1'):
            with self.subTest(action=action):
                self.assertFalse(
                    has_go_runtime_reference(
                        Path('workflow.yaml'), f'- uses: {action}\n'))

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
            if not is_yaml_metadata_path(relative_path):
                continue
            is_workflow = relative_path.is_relative_to(
                Path('.github/workflows'))
            is_action = relative_path.name in {'action.yaml', 'action.yml'}
            if not (is_workflow or is_action):
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
        self.assertRegex(_read(GO_IMAGE_DIGEST_WORKFLOW), DIGEST_CHECK_PATTERN)
        self.assertIn(str(GO_VERSION_METADATA), _read(GO_IMAGE_DIGEST_WORKFLOW))
        self.assertIn('tools/go-version-metadata/**',
                      _read(GO_IMAGE_DIGEST_WORKFLOW))
        self.assertTrue((REPOSITORY_ROOT / GO_VERSION_HELPER).exists())

    def test_api_tools_module_retains_downloaded_tool_pins(self):
        """The API generator image reads tool versions out of this module."""
        contents = _read(API_TOOLS_MODULE)
        for module_path in API_TOOLS_PINS:
            with self.subTest(module_path=module_path):
                self.assertRegex(
                    contents,
                    rf'(?m)^[ \t]*(?:require[ \t]+)?'
                    rf'{re.escape(module_path)}[ \t]+v[^\s]+',
                    f'{API_TOOLS_MODULE} must keep a require for '
                    f'{module_path}; it is the version source for a tool the '
                    f'{API_TOOLS_DOCKERFILE} image downloads, and `go mod '
                    f'tidy` prunes it because no Go source imports it.',
                )

    def test_api_tools_pins_are_read_by_the_generator_image(self):
        """Guard manifest -> query -> download dataflow in one RUN."""
        contents = _read(API_TOOLS_DOCKERFILE)
        for module_path in API_TOOLS_PINS:
            with self.subTest(module_path=module_path):
                self.assertEqual(
                    _api_tool_pin_dataflow_errors(contents, module_path),
                    [],
                    f'{API_TOOLS_DOCKERFILE} must copy {API_TOOLS_MODULE}, '
                    f'query {module_path} from that copy, and pass the '
                    'resulting variable to the matching download',
                )

    def test_api_tool_protocol_tolerates_prior_go_environment_redirects(self):
        """Generator poison cannot enter the sibling installer stage."""
        contents = _read(API_TOOLS_DOCKERFILE)
        module_path = API_TOOLS_PINS[0]
        generator_stage = f'FROM {API_TOOL_BASE_STAGE} AS generator'
        redirected = contents.replace(
            generator_stage,
            generator_stage + '\n'
            'ENV PATH=/tmp/fake-bin:/usr/bin:/bin\n'
            'ENV GOENV=/tmp/evil-goenv\n'
            'RUN printf \'url = "file:///tmp/wrong"\\n\' > /root/.curlrc',
        )
        self.assertNotEqual(redirected, contents)
        self.assertEqual(
            _api_tool_pin_dataflow_errors(redirected, module_path), [])

    def test_api_tool_protocol_uses_buildkit_continuation_semantics(self):
        """The guard compares BuildKit-normalized instructions, not lines."""
        contents = _read(API_TOOLS_DOCKERFILE)
        continued = contents.replace('/usr/bin/curl', '/usr/bin/cu\\\nrl')
        self.assertNotEqual(continued, contents)
        for module_path in API_TOOLS_PINS:
            with self.subTest(module_path=module_path):
                self.assertEqual(
                    _api_tool_pin_dataflow_errors(continued, module_path), [])

    def test_api_tool_protocol_is_independent_of_inherited_shell(self):
        """Generator SHELL state cannot affect the sibling installer."""
        contents = _read(API_TOOLS_DOCKERFILE)
        module_path = API_TOOLS_PINS[0]
        generator_stage = f'FROM {API_TOOL_BASE_STAGE} AS generator'
        inherited_shell = contents.replace(
            generator_stage,
            generator_stage + '\nSHELL ["/bin/false", "-c"]',
        )
        self.assertNotEqual(inherited_shell, contents)
        self.assertEqual(
            _api_tool_pin_dataflow_errors(inherited_shell, module_path), [])

    def test_api_tools_pin_dataflow_rejects_decoupled_mutations(self):
        """Shared path strings must not satisfy the coupling invariant."""
        contents = _read(API_TOOLS_DOCKERFILE)
        module_path = API_TOOLS_PINS[0]
        canonical_copy = (
            f'COPY {API_TOOLS_MODULE} {API_TOOLS_CONTAINER_MODULE}/go.mod')
        canonical_run = _api_tool_install_instruction(module_path)
        copy_then_run = canonical_copy + '\n' + canonical_run
        canonical_script = _api_tool_install_argv(module_path)[-1]
        base_stage = re.search(
            r'(?m)^FROM golang:[^ ]+ AS go-base$', contents).group(0)
        installer_stage = (
            f'FROM {API_TOOL_BASE_STAGE} AS {API_TOOL_INSTALLER_STAGE}')
        generator_stage = (
            f'FROM {API_TOOL_BASE_STAGE} AS {API_TOOL_GENERATOR_STAGE}')
        artifact_copy = (
            f'COPY --from={API_TOOL_INSTALLER_STAGE} '
            '/usr/bin/swagger /usr/bin/swagger')

        def replace_script(old, new):
            mutated_script = canonical_script.replace(old, new)
            self.assertNotEqual(mutated_script, canonical_script)
            return contents.replace(
                canonical_run,
                _api_tool_install_instruction(module_path, mutated_script),
            )

        mutations = {
            'custom Docker frontend': '# syntax=docker/dockerfile:1.19\n' +
                contents,
            'BOM-prefixed custom Docker frontend':
                '\ufeff# syntax=docker/dockerfile:1.19\n' + contents,
            'shebang-prefixed custom Docker frontend':
                '#!/usr/bin/env dockerfile\n'
                '# syntax=docker/dockerfile:1.19\n' + contents,
            'BOM-and-shebang-prefixed custom Docker frontend':
                '\ufeff#!/usr/bin/env dockerfile\n'
                '# syntax=docker/dockerfile:1.19\n' + contents,
            'Unicode-whitespace custom Docker frontend':
                '#\u00a0syntax=docker/dockerfile:1.19\n' + contents,
            'hard-coded version': replace_script(
                'go_swagger_version="$(cd /tmp/api-generator-tools',
                'go_swagger_version="v0.31.0"; ignored="$(cd '
                '/tmp/api-generator-tools',
            ),
            'different manifest': replace_script(
                '-modfile=/tmp/api-generator-tools/go.mod',
                '-modfile=/tmp/kfp-module/go.mod',
            ),
            'different selected module': replace_script(
                'github.com/go-swagger/go-swagger)',
                'google.golang.org/protobuf)',
            ),
            'query is conditional': replace_script(
                'go_swagger_version="$(cd /tmp/api-generator-tools',
                'false && go_swagger_version="$(cd /tmp/api-generator-tools',
            ),
            'module directory change is omitted': replace_script(
                'cd /tmp/api-generator-tools && ',
                '',
            ),
            'GO environment file is enabled': contents.replace(
                '"GOENV=off", ', '',
            ),
            'toolchain download is enabled': contents.replace(
                '"GOTOOLCHAIN=local", ', '',
            ),
            'trusted GOFLAGS is empty': contents.replace(
                '"GOFLAGS=-mod=readonly"', '"GOFLAGS="',
            ),
            'workspace mode is enabled': contents.replace(
                '"GOWORK=off", ', '',
            ),
            'hard-coded download': replace_script(
                'releases/download/${go_swagger_version}/swagger_linux_amd64',
                'releases/download/v0.31.0/swagger_linux_amd64',
            ),
            'queried version overwritten': replace_script(
                '/usr/bin/curl -q',
                'go_swagger_version="v0.31.0"; '
                '/usr/bin/curl -q',
            ),
            'exit before install': replace_script(
                '/usr/bin/curl -q',
                'exit 0; /usr/bin/curl -q',
            ),
            'curl config is enabled': replace_script(
                '/usr/bin/curl -q', '/usr/bin/curl',
            ),
            'artifact version is not asserted': replace_script(
                '; /usr/bin/swagger version | /usr/bin/grep -Fqx '
                '"version: ${go_swagger_version}"', '',
            ),
            'later artifact overwrite':
                contents + '\nRUN cp /tmp/hard-coded-swagger '
                '/usr/bin/swagger\n',
            'WORKDIR inserted between manifest and query': contents.replace(
                copy_then_run,
                canonical_copy + '\nWORKDIR /tmp/api-generator-tools\n'
                + canonical_run,
            ),
            'ENV inserted between manifest and query': contents.replace(
                copy_then_run,
                canonical_copy + '\nENV GOFLAGS=-mod=mod\n' + canonical_run,
            ),
            'shell manifest writer inserted between copy and query':
                contents.replace(
                    copy_then_run,
                    canonical_copy + '\n'
                    'RUN sed -i \'s/v0.31.0/v0.32.0/\' '
                    '/tmp/api-generator-tools/go.mod\n' + canonical_run,
                ),
            'JSON manifest COPY inserted between copy and query':
                contents.replace(
                    copy_then_run,
                    canonical_copy + '\n'
                    'COPY ["decoy.mod", '
                    '"/tmp/api-generator-tools/go.mod"]\n' + canonical_run,
                ),
            'comment-hidden manifest COPY inserted between copy and query':
                contents.replace(
                    copy_then_run,
                    canonical_copy + '\n# ordinary comment \\\n'
                    'COPY ["decoy.mod", '
                    '"/tmp/api-generator-tools/go.mod"]\n' + canonical_run,
                ),
            'heredoc impersonates canonical install': contents.replace(
                canonical_run,
                "RUN <<'PIN_PROTOCOL'\n" + canonical_run +
                '\nPIN_PROTOCOL',
            ),
            'continuation alters canonical command': contents.replace(
                '/usr/bin/curl -q', '/usr/bin/curl\\\n-q',
            ),
            'shell-form install inherits SHELL': contents.replace(
                canonical_run, 'RUN ' + canonical_script,
            ),
            'nonempty base stage': contents.replace(
                base_stage,
                base_stage + '\nENV PATH=/tmp/fake-bin',
            ),
            'installer derives from generator': contents.replace(
                installer_stage,
                f'FROM {API_TOOL_GENERATOR_STAGE} AS '
                f'{API_TOOL_INSTALLER_STAGE}',
            ),
            'duplicate reserved installer stage': contents.replace(
                generator_stage,
                f'FROM scratch AS {API_TOOL_INSTALLER_STAGE}\n' +
                generator_stage,
            ),
            'wrong final artifact source': contents.replace(
                artifact_copy,
                'COPY --from=generator /usr/bin/swagger /usr/bin/swagger',
            ),
            'wrong final artifact destination': contents.replace(
                artifact_copy,
                f'COPY --from={API_TOOL_INSTALLER_STAGE} '
                '/usr/bin/swagger /tmp/swagger',
            ),
            'normalized artifact alias written after install':
                contents + '\nRUN cp /tmp/hard-coded-swagger '
                '/usr/bin/../bin/swagger\n',
            'JSON artifact COPY after install':
                contents + '\nCOPY ["swagger", "/usr/bin/swagger"]\n',
            'comment-hidden artifact COPY after install':
                contents + '\n# ordinary comment \\\n'
                'COPY ["swagger", "/usr/bin/swagger"]\n',
            'unrelated instruction after install':
                contents + '\nENV API_INSTALL_COMPLETE=true\n',
        }
        for name, mutated in mutations.items():
            with self.subTest(name=name):
                self.assertNotEqual(mutated, contents,
                                    f'{name} mutation did not alter fixture')
                self.assertTrue(
                    _api_tool_pin_dataflow_errors(mutated, module_path),
                    f'{name} must break the API tool pin dataflow guard',
                )

    def test_api_generator_invokes_the_frozen_swagger_path(self):
        contents = _read(Path('backend/api/hack/generator.sh'))
        self.assertNotRegex(contents, r'(?m)^[ \t]*swagger[ \t]+generate')
        self.assertEqual(
            len(re.findall(r'(?m)^[ \t]*/usr/bin/swagger[ \t]+generate',
                           contents)), 8)


if __name__ == '__main__':
    unittest.main()

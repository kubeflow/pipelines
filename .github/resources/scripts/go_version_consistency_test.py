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


def _docker_run_blocks(contents):
    """Return normalized logical RUN instructions without parsing shell."""
    lines = contents.splitlines()
    blocks = []
    index = 0
    while index < len(lines):
        line = lines[index]
        if not re.match(r'^[ \t]*RUN(?:[ \t]|$)', line, re.IGNORECASE):
            index += 1
            continue
        physical_lines = [line]
        while physical_lines[-1].rstrip().endswith('\\'):
            index += 1
            if index >= len(lines):
                break
            physical_lines.append(lines[index])
        blocks.append(' '.join(
            physical_line.rstrip().removesuffix('\\').strip()
            for physical_line in physical_lines))
        index += 1
    return blocks


def _shell_commands(run_block):
    """Split a normalized RUN and report whether its flow is linear."""
    command = re.sub(r'^[ \t]*RUN(?:[ \t]+|$)', '', run_block,
                     count=1, flags=re.IGNORECASE)
    commands = []
    start = 0
    quote = ''
    outer_quotes = []
    linear = True
    index = 0
    while index < len(command):
        character = command[index]
        if quote != "'" and character == '\\':
            index += 2
            continue
        if quote == "'":
            if character == quote:
                quote = ''
            index += 1
            continue
        if (character == '$' and index + 1 < len(command)
                and command[index + 1] == '('):
            outer_quotes.append(quote)
            quote = ''
            index += 2
            continue
        if outer_quotes and not quote and character == ')':
            quote = outer_quotes.pop()
            index += 1
            continue
        if character in {'"', "'"}:
            if not quote:
                quote = character
            elif quote == character:
                quote = ''
            index += 1
            continue
        separator_width = 0
        if not quote and not outer_quotes:
            if character == ';':
                separator_width = 1
            elif command[index:index + 2] in {'&&', '||'}:
                separator_width = 2
                linear = False
            elif character in {'&', '|'}:
                separator_width = 1
                linear = False
        if separator_width:
            candidate = command[start:index].strip()
            if candidate:
                commands.append(candidate)
            index += separator_width
            start = index
            continue
        index += 1
    candidate = command[start:].strip()
    if candidate:
        commands.append(candidate)
    return commands, linear


def _api_tool_install_commands(module_path):
    """The frozen repository-specific API tool installation protocol."""
    variable = API_TOOLS_VERSION_VARIABLES[module_path]
    return [
        'set -eu',
        f'{variable}="$(cd {API_TOOLS_CONTAINER_MODULE} && go mod edit '
        f'-json | jq -er \'.Require[] | select(.Path == "{module_path}") '
        '| .Version\')"',
        'curl -fL -o /usr/bin/swagger '
        '"https://github.com/go-swagger/go-swagger/releases/download/'
        f'${{{variable}}}/swagger_linux_amd64"',
        'chmod +x /usr/bin/swagger',
    ]


def _api_tool_pin_dataflow_errors(contents, module_path):
    """Validate manifest -> query -> version-variable consumer dataflow."""
    variable = API_TOOLS_VERSION_VARIABLES[module_path]
    copy_pattern = re.compile(
        rf'(?mi)^[ \t]*COPY[ \t]+{re.escape(str(API_TOOLS_MODULE))}'
        rf'[ \t]+{re.escape(API_TOOLS_CONTAINER_MODULE)}/go\.mod[ \t]*$')
    errors = []
    if not copy_pattern.search(contents):
        errors.append(
            f'{API_TOOLS_MODULE} is not copied to '
            f'{API_TOOLS_CONTAINER_MODULE}/go.mod')

    parsed_runs = [
        _shell_commands(block) for block in _docker_run_blocks(contents)
    ]
    markers = (variable, '/usr/bin/swagger', module_path)
    relevant_runs = [
        (commands, linear) for commands, linear in parsed_runs
        if any(marker in command for marker in markers
               for command in commands)
    ]
    if len(relevant_runs) != 1:
        errors.append(
            'expected exactly one RUN containing the API tool query/install '
            f'protocol, found {len(relevant_runs)}')
    else:
        commands, linear = relevant_runs[0]
        expected = _api_tool_install_commands(module_path)
        if not linear:
            errors.append(
                'the API tool query/install RUN must use linear semicolon '
                'sequencing')
        if commands != expected:
            mismatch = next((index for index, (actual, wanted) in enumerate(
                zip(commands, expected)) if actual != wanted),
                            min(len(commands), len(expected)))
            actual = commands[mismatch] if mismatch < len(commands) else '<missing>'
            wanted = expected[mismatch] if mismatch < len(expected) else '<no extra command>'
            errors.append(
                f'API tool query/install command {mismatch + 1} is {actual!r}; '
                f'want {wanted!r}')
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

    def test_api_tools_pin_dataflow_rejects_decoupled_mutations(self):
        """Shared path strings must not satisfy the coupling invariant."""
        contents = _read(API_TOOLS_DOCKERFILE)
        module_path = API_TOOLS_PINS[0]
        mutations = {
            'hard-coded version': contents.replace(
                'go_swagger_version="$(cd /tmp/api-generator-tools && go mod '
                'edit -json | jq -er \'.Require[] | select(.Path == '
                '"github.com/go-swagger/go-swagger") | .Version\')";',
                'go_swagger_version="v0.31.0";',
            ),
            'different manifest': contents.replace(
                'cd /tmp/api-generator-tools && go mod edit -json',
                'cd /tmp/kfp-module && go mod edit -json',
            ),
            'different selected module': contents.replace(
                'select(.Path == "github.com/go-swagger/go-swagger")',
                'select(.Path == "google.golang.org/protobuf")',
            ),
            'query is conditional': contents.replace(
                'go_swagger_version="$(cd /tmp/api-generator-tools',
                'false && go_swagger_version="$(cd /tmp/api-generator-tools',
            ),
            'hard-coded download': contents.replace(
                'releases/download/${go_swagger_version}/swagger_linux_amd64',
                'releases/download/v0.31.0/swagger_linux_amd64',
            ),
            'queried version overwritten': contents.replace(
                'curl -fL -o /usr/bin/swagger',
                'go_swagger_version="v0.31.0"; \\\n'
                '    curl -fL -o /usr/bin/swagger',
            ),
            'queried version exported over': contents.replace(
                'curl -fL -o /usr/bin/swagger',
                'export go_swagger_version="v0.31.0"; \\\n'
                '    curl -fL -o /usr/bin/swagger',
            ),
            'queried version overridden through eval': contents.replace(
                'curl -fL -o /usr/bin/swagger',
                'eval \'go_swagger_version="v0.31.0"\'; \\\n'
                '    curl -fL -o /usr/bin/swagger',
            ),
            'exit before install': contents.replace(
                'curl -fL -o /usr/bin/swagger',
                'exit 0; \\\n'
                '    curl -fL -o /usr/bin/swagger',
            ),
            'decoy variable download before hard-coded install':
                contents.replace(
                    'curl -fL -o /usr/bin/swagger '
                    '"https://github.com/go-swagger/go-swagger/releases/'
                    'download/${go_swagger_version}/swagger_linux_amd64";',
                    'curl -fL -o /tmp/swagger-decoy '
                    '"https://github.com/go-swagger/go-swagger/releases/'
                    'download/${go_swagger_version}/swagger_linux_amd64"; \\\n'
                    '    curl -fL -o /usr/bin/swagger '
                    '"https://github.com/go-swagger/go-swagger/releases/'
                    'download/v0.31.0/swagger_linux_amd64";',
                ),
            'later artifact overwrite':
                contents + '\nRUN cp /tmp/hard-coded-swagger '
                '/usr/bin/swagger\n',
        }
        for name, mutated in mutations.items():
            with self.subTest(name=name):
                self.assertNotEqual(mutated, contents,
                                    f'{name} mutation did not alter fixture')
                self.assertTrue(
                    _api_tool_pin_dataflow_errors(mutated, module_path),
                    f'{name} must break the API tool pin dataflow guard',
                )


if __name__ == '__main__':
    unittest.main()

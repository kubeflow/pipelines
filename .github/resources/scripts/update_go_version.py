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
"""Check and update the repository's explicitly managed Go version pins.

This is intentionally a repository policy tool, not a Docker, YAML, shell, or
Git interpreter. See docs/agents/go-version-policy.md before extending it.
"""

import argparse
from dataclasses import dataclass
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time
from typing import Callable, Dict, Iterable, List, Optional, Sequence, Set, Tuple


REPOSITORY_ROOT = Path(__file__).resolve().parents[3]


@dataclass(frozen=True)
class DockerPin:
    path: Path
    flavor: str
    stage: str


MANAGED_DOCKERFILES = (
    DockerPin(Path('backend/Dockerfile'), '-bookworm', 'builder'),
    DockerPin(Path('backend/Dockerfile.cacheserver'), '-alpine', 'builder'),
    DockerPin(Path('backend/Dockerfile.conformance'), '-alpine', 'builder'),
    DockerPin(Path('backend/Dockerfile.driver'), '-alpine', 'builder'),
    DockerPin(Path('backend/Dockerfile.launcher'), '-alpine', 'builder'),
    DockerPin(Path('backend/Dockerfile.persistenceagent'), '-alpine', 'builder'),
    DockerPin(Path('backend/Dockerfile.scheduledworkflow'), '-alpine', 'builder'),
    DockerPin(Path('backend/Dockerfile.viewercontroller'), '-alpine', 'builder'),
    DockerPin(Path('backend/api/Dockerfile'), '', 'generator'),
)

MANAGED_SETUP_GO_ACTIONS = (
    Path('.github/actions/setup-go/action.yml'),
    Path('.github/actions/test-and-report/action.yml'),
)

DECIMAL = r'(?:0|[1-9][0-9]*)'
EXACT_VERSION_PATTERN = re.compile(rf'^1\.{DECIMAL}\.{DECIMAL}$')
MODULE_VERSION_PATTERN = re.compile(rf'^1\.{DECIMAL}(?:\.{DECIMAL})?$')
DIGEST_PATTERN = re.compile(r'^sha256:[0-9a-f]{64}$')
ROOT_GO_LINE_PATTERN = re.compile(
    rf'^go (?P<version>1\.{DECIMAL}\.{DECIMAL})$', re.MULTILINE)
ROOT_TOOLCHAIN_LINE_PATTERN = re.compile(
    rf'^toolchain (?P<version>go1\.{DECIMAL}\.{DECIMAL})$', re.MULTILINE)
DOCKER_FROM_PATTERN = re.compile(
    rf'^FROM golang:(?P<version>1\.{DECIMAL}\.{DECIMAL})'
    r'(?P<flavor>-[a-z0-9][a-z0-9._-]*)?@'
    r'(?P<digest>sha256:[0-9a-f]{64}) AS '
    r'(?P<stage>[a-z0-9][a-z0-9_.-]*)$', re.MULTILINE)
GO_IMAGE_LITERAL_PATTERN = re.compile(
    r'(?i)(?:^|[^a-z0-9_.-])golang(?=[:@])')
GO_DOWNLOAD_LITERAL_PATTERN = re.compile(
    r'(?i)https://(?:go\.dev/dl/go|dl\.google\.com/go/go)')
SETUP_GO_USE_PATTERN = re.compile(
    r'^[ ]*uses: actions/setup-go@[^ \t\r\n#]+[ ]*$', re.MULTILINE)
GO_VERSION_FILE_INPUT_PATTERN = re.compile(
    r'^[ ]*go-version-file:', re.MULTILINE)
GO_VERSION_INPUT_PATTERN = re.compile(r'^[ ]*go-version:', re.MULTILINE)

DIGEST_LOOKUP_TIMEOUT_SECONDS = 30
DIGEST_LOOKUP_ATTEMPTS = 2

Version = Tuple[int, int, int]
DigestResolver = Callable[[str], str]


class PolicyError(RuntimeError):
    """The repository does not satisfy the supported Go version policy."""


@dataclass(frozen=True)
class ModuleMetadata:
    go_text: str
    go_version: Version
    toolchain_text: Optional[str]
    toolchain_version: Optional[Version]


@dataclass(frozen=True)
class DockerMetadata:
    version: str
    flavor: str
    digest: str
    stage: str
    start: int
    end: int


@dataclass(frozen=True)
class UpdatePlan:
    original: Dict[Path, str]
    expected: Dict[Path, str]

    @property
    def changed_paths(self) -> List[Path]:
        return sorted(path for path in self.expected
                      if self.original[path] != self.expected[path])


def _run(arguments: Sequence[str], repo_root: Path,
         timeout: Optional[int] = None) -> subprocess.CompletedProcess:
    try:
        return subprocess.run(
            arguments,
            cwd=repo_root,
            check=True,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
    except FileNotFoundError as error:
        raise PolicyError(f'{arguments[0]} is required to manage Go versions') \
            from error
    except subprocess.TimeoutExpired as error:
        raise PolicyError(
            f'{arguments[0]} timed out while managing Go versions') from error
    except subprocess.CalledProcessError as error:
        detail = error.stderr.strip() or error.stdout.strip() or str(error)
        raise PolicyError(
            f'{" ".join(arguments)} failed: {detail}') from error


def _parse_exact_version(value: str) -> Version:
    if EXACT_VERSION_PATTERN.fullmatch(value) is None:
        raise PolicyError(
            f'Go version must use the exact stable form 1.X.Y, found {value!r}')
    return tuple(int(part) for part in value.split('.'))


def _parse_module_version(value: str, field: str, path: Path) -> Version:
    if MODULE_VERSION_PATTERN.fullmatch(value) is None:
        raise PolicyError(
            f'{path} has unsupported {field} version {value!r}; use 1.X or 1.X.Y')
    parts = tuple(int(part) for part in value.split('.'))
    return parts + (0,) * (3 - len(parts))


def _version_text(version: Version) -> str:
    return '.'.join(str(part) for part in version)


def _tracked_paths(repo_root: Path) -> Set[Path]:
    output = _run(('git', 'ls-files', '-z'), repo_root).stdout
    return {
        Path(value) for value in output.split('\0')
        if value and (repo_root / value).exists()
    }


def _module_paths(tracked_paths: Iterable[Path]) -> List[Path]:
    paths = sorted(path for path in tracked_paths if path.name == 'go.mod')
    if Path('go.mod') not in paths:
        raise PolicyError('the tracked root go.mod is required')
    return paths


def _module_metadata(repo_root: Path, relative_path: Path) -> ModuleMetadata:
    result = _run(('go', 'mod', 'edit', '-json', str(relative_path)), repo_root)
    try:
        data = json.loads(result.stdout)
        go_text = data['Go']
        toolchain_text = data.get('Toolchain')
    except (json.JSONDecodeError, KeyError, TypeError) as error:
        raise PolicyError(f'{relative_path} is not a valid Go module') from error
    if not isinstance(go_text, str):
        raise PolicyError(f'{relative_path} must contain one go directive')
    go_version = _parse_module_version(go_text, 'go', relative_path)
    toolchain_version = None
    if toolchain_text is not None:
        if not isinstance(toolchain_text, str) or not toolchain_text.startswith('go'):
            raise PolicyError(
                f'{relative_path} has unsupported toolchain {toolchain_text!r}')
        toolchain_version = _parse_module_version(
            toolchain_text[2:], 'toolchain', relative_path)
        if EXACT_VERSION_PATTERN.fullmatch(toolchain_text[2:]) is None:
            raise PolicyError(
                f'{relative_path} toolchain must use the exact form go1.X.Y')
    return ModuleMetadata(go_text, go_version, toolchain_text,
                          toolchain_version)


def _root_compiler(repo_root: Path,
                   modules: Dict[Path, ModuleMetadata]) -> Version:
    root = modules[Path('go.mod')]
    contents = _read_text(repo_root, Path('go.mod'))
    go_lines = list(ROOT_GO_LINE_PATTERN.finditer(contents))
    toolchain_lines = list(ROOT_TOOLCHAIN_LINE_PATTERN.finditer(contents))
    if (len(go_lines) != 1 or
            go_lines[0].group('version') != root.go_text):
        raise PolicyError(
            'root go.mod must contain the exact line go 1.X.Y for setup-go')
    if root.toolchain_text is None:
        if toolchain_lines:
            raise PolicyError('root go.mod has an unrecognized toolchain line')
    elif (len(toolchain_lines) != 1 or
          toolchain_lines[0].group('version') != root.toolchain_text):
        raise PolicyError(
            'root go.mod must contain the exact line toolchain go1.X.Y')
    return root.toolchain_version or root.go_version


def _render_module(repo_root: Path, contents: str, go_version: Version,
                   toolchain: Optional[Version]) -> str:
    with tempfile.TemporaryDirectory(prefix='kfp-go-version-') as directory:
        temporary_mod = Path(directory) / 'go.mod'
        temporary_mod.write_text(contents, encoding='utf-8')
        toolchain_argument = ('none' if toolchain is None else
                              f'go{_version_text(toolchain)}')
        _run((
            'go',
            'mod',
            'edit',
            f'-go={_version_text(go_version)}',
            f'-toolchain={toolchain_argument}',
            str(temporary_mod),
        ), repo_root)
        return temporary_mod.read_text(encoding='utf-8')


def _is_container_recipe(path: Path) -> bool:
    return (path.name.startswith('Dockerfile') or
            path.name.startswith('Containerfile'))


def _read_text(repo_root: Path, relative_path: Path) -> str:
    path = repo_root / relative_path
    if path.is_symlink() or not path.is_file():
        raise PolicyError(f'{relative_path} must be a regular file')
    try:
        return path.read_text(encoding='utf-8')
    except UnicodeError as error:
        raise PolicyError(f'{relative_path} must be UTF-8 text') from error


def _docker_metadata(contents: str, pin: DockerPin) -> DockerMetadata:
    matches = list(DOCKER_FROM_PATTERN.finditer(contents))
    if len(matches) != 1:
        raise PolicyError(
            f'{pin.path} must contain exactly one canonical Go builder FROM line; '
            'see docs/agents/go-version-policy.md')
    match = matches[0]
    flavor = match.group('flavor') or ''
    stage = match.group('stage')
    if flavor != pin.flavor or stage != pin.stage:
        raise PolicyError(
            f'{pin.path} must use flavor {pin.flavor or "<none>"} and stage '
            f'{pin.stage}')
    literal_count = (len(GO_IMAGE_LITERAL_PATTERN.findall(contents)) +
                     len(GO_DOWNLOAD_LITERAL_PATTERN.findall(contents)))
    if literal_count != 1:
        raise PolicyError(
            f'{pin.path} must contain only its one registered literal Go source')
    return DockerMetadata(
        match.group('version'),
        flavor,
        match.group('digest'),
        stage,
        match.start(),
        match.end(),
    )


def _validate_setup_action(contents: str, relative_path: Path) -> None:
    matches = list(SETUP_GO_USE_PATTERN.finditer(contents))
    if len(matches) != 1:
        raise PolicyError(
            f'{relative_path} must contain one literal actions/setup-go step')
    lines = contents.splitlines()
    use_line = next(index for index, line in enumerate(lines)
                    if SETUP_GO_USE_PATTERN.fullmatch(line))
    indentation = lines[use_line][:len(lines[use_line]) -
                                  len(lines[use_line].lstrip(' '))]
    expected = (
        f'{indentation}with:',
        f'{indentation}  go-version-file: go.mod',
    )
    if tuple(lines[use_line + 1:use_line + 3]) != expected:
        raise PolicyError(
            f'{relative_path} setup-go step must read go-version-file: go.mod')
    if (len(GO_VERSION_FILE_INPUT_PATTERN.findall(contents)) != 1 or
            GO_VERSION_INPUT_PATTERN.search(contents)):
        raise PolicyError(
            f'{relative_path} must use only go-version-file: go.mod for setup-go')


def _validate_inventory(repo_root: Path, tracked_paths: Set[Path],
                        docker_pins: Sequence[DockerPin],
                        setup_actions: Sequence[Path]) \
        -> Tuple[Dict[Path, str], Dict[Path, DockerMetadata]]:
    docker_paths = {pin.path for pin in docker_pins}
    if len(docker_paths) != len(docker_pins):
        raise PolicyError('managed Dockerfile inventory contains duplicate paths')
    setup_paths = set(setup_actions)
    if len(setup_paths) != len(setup_actions):
        raise PolicyError('managed setup-go inventory contains duplicate paths')

    required = docker_paths | setup_paths
    missing = required - tracked_paths
    if missing:
        raise PolicyError(
            'managed paths are not tracked: ' +
            ', '.join(str(path) for path in sorted(missing)))

    docker_contents = {}
    docker_metadata = {}
    literal_source_paths = set()
    for relative_path in tracked_paths:
        if not _is_container_recipe(relative_path):
            continue
        contents = _read_text(repo_root, relative_path)
        if (GO_IMAGE_LITERAL_PATTERN.search(contents) or
                GO_DOWNLOAD_LITERAL_PATTERN.search(contents)):
            literal_source_paths.add(relative_path)
    if literal_source_paths != docker_paths:
        unregistered = literal_source_paths - docker_paths
        absent = docker_paths - literal_source_paths
        details = []
        if unregistered:
            details.append('register literal Go sources in ' +
                           ', '.join(str(path) for path in sorted(unregistered)))
        if absent:
            details.append('restore registered Go sources in ' +
                           ', '.join(str(path) for path in sorted(absent)))
        raise PolicyError('; '.join(details))

    for pin in docker_pins:
        contents = _read_text(repo_root, pin.path)
        docker_contents[pin.path] = contents
        docker_metadata[pin.path] = _docker_metadata(contents, pin)

    setup_use_paths = set()
    for relative_path in tracked_paths:
        if relative_path.suffix not in {'.yaml', '.yml'}:
            continue
        contents = _read_text(repo_root, relative_path)
        if 'actions/setup-go@' in contents:
            setup_use_paths.add(relative_path)
    if setup_use_paths != setup_paths:
        unregistered = setup_use_paths - setup_paths
        absent = setup_paths - setup_use_paths
        details = []
        if unregistered:
            details.append('route setup-go callers through a managed action: ' +
                           ', '.join(str(path) for path in sorted(unregistered)))
        if absent:
            details.append('restore setup-go in ' +
                           ', '.join(str(path) for path in sorted(absent)))
        raise PolicyError('; '.join(details))
    for relative_path in setup_actions:
        _validate_setup_action(_read_text(repo_root, relative_path),
                               relative_path)
    return docker_contents, docker_metadata


def check_repository(repo_root: Path,
                     docker_pins: Sequence[DockerPin] = MANAGED_DOCKERFILES,
                     setup_actions: Sequence[Path] =
                     MANAGED_SETUP_GO_ACTIONS) -> None:
    tracked_paths = _tracked_paths(repo_root)
    module_paths = _module_paths(tracked_paths)
    modules = {
        relative_path: _module_metadata(repo_root, relative_path)
        for relative_path in module_paths
    }
    compiler = _root_compiler(repo_root, modules)
    for relative_path, module in modules.items():
        if module.go_version[:2] != compiler[:2]:
            raise PolicyError(
                f'{relative_path} must use Go {compiler[0]}.{compiler[1]}')
        if module.go_version > compiler:
            raise PolicyError(
                f'{relative_path} requires {_version_text(module.go_version)}, '
                f'newer than the root compiler {_version_text(compiler)}')
        if module.toolchain_version is None:
            if module.go_version != compiler:
                raise PolicyError(
                    f'{relative_path} must use go {_version_text(compiler)} or '
                    'name the root toolchain')
        elif module.toolchain_version != compiler:
            raise PolicyError(
                f'{relative_path} toolchain must be go{_version_text(compiler)}')

    _, docker_metadata = _validate_inventory(
        repo_root, tracked_paths, docker_pins, setup_actions)
    expected_version = _version_text(compiler)
    digests_by_tag: Dict[str, Set[str]] = {}
    for metadata in docker_metadata.values():
        if metadata.version != expected_version:
            raise PolicyError(
                f'Go builder version {metadata.version} must match root compiler '
                f'{expected_version}')
        tag = metadata.version + metadata.flavor
        digests_by_tag.setdefault(tag, set()).add(metadata.digest)
    for tag, digests in digests_by_tag.items():
        if len(digests) != 1:
            raise PolicyError(
                f'Go builder tag {tag} must use one digest, found {sorted(digests)}')


def _inspect_image_digest(image: str) -> str:
    result = _run((
        'docker',
        'buildx',
        'imagetools',
        'inspect',
        image,
        '--format',
        '{{json .Manifest}}',
    ), REPOSITORY_ROOT, timeout=DIGEST_LOOKUP_TIMEOUT_SECONDS)
    try:
        digest = json.loads(result.stdout)['digest']
    except (json.JSONDecodeError, KeyError, TypeError) as error:
        raise PolicyError(f'docker returned no manifest digest for {image}') \
            from error
    if not isinstance(digest, str) or DIGEST_PATTERN.fullmatch(digest) is None:
        raise PolicyError(f'docker returned an invalid digest for {image}')
    return digest


def resolve_image_digest(tag: str) -> str:
    images = (f'golang:{tag}', f'mirror.gcr.io/library/golang:{tag}')
    failures = []
    for attempt in range(DIGEST_LOOKUP_ATTEMPTS):
        for image in images:
            try:
                return _inspect_image_digest(image)
            except PolicyError as error:
                failures.append(str(error))
        if attempt + 1 < DIGEST_LOOKUP_ATTEMPTS:
            time.sleep(attempt + 1)
    raise PolicyError(
        f'could not resolve Go builder tag {tag}: {"; ".join(failures)}')


def _updated_dockerfile(contents: str, metadata: DockerMetadata,
                        version: str, digest: str) -> str:
    replacement = (
        f'FROM golang:{version}{metadata.flavor}@{digest} AS {metadata.stage}')
    return contents[:metadata.start] + replacement + contents[metadata.end:]


def plan_update(repo_root: Path, target_text: str,
                digest_resolver: DigestResolver = resolve_image_digest,
                docker_pins: Sequence[DockerPin] = MANAGED_DOCKERFILES,
                setup_actions: Sequence[Path] =
                MANAGED_SETUP_GO_ACTIONS) -> UpdatePlan:
    target = _parse_exact_version(target_text)
    tracked_paths = _tracked_paths(repo_root)
    module_paths = _module_paths(tracked_paths)
    modules = {
        relative_path: _module_metadata(repo_root, relative_path)
        for relative_path in module_paths
    }
    current = _root_compiler(repo_root, modules)
    if target < current:
        raise PolicyError(
            f'refusing to downgrade Go from {_version_text(current)} to '
            f'{target_text}')

    docker_contents, docker_metadata = _validate_inventory(
        repo_root, tracked_paths, docker_pins, setup_actions)
    original = {
        relative_path: _read_text(repo_root, relative_path)
        for relative_path in module_paths
    }
    original.update(docker_contents)
    expected = {}
    for relative_path, module in modules.items():
        if module.go_version > target:
            raise PolicyError(
                f'{relative_path} requires {_version_text(module.go_version)}, '
                f'newer than target {target_text}')
        language_floor = (module.go_version
                          if module.go_version[:2] == target[:2] else
                          (target[0], target[1], 0))
        toolchain = target if target[2] else None
        expected[relative_path] = _render_module(
            repo_root, original[relative_path], language_floor, toolchain)

    digests = {}
    for flavor in sorted({pin.flavor for pin in docker_pins}):
        tag = target_text + flavor
        digest = digest_resolver(tag)
        if DIGEST_PATTERN.fullmatch(digest) is None:
            raise PolicyError(
                f'digest resolver returned an invalid digest for {tag}: {digest!r}')
        digests[flavor] = digest
    for relative_path, metadata in docker_metadata.items():
        expected[relative_path] = _updated_dockerfile(
            original[relative_path], metadata, target_text,
            digests[metadata.flavor])
    return UpdatePlan(original, expected)


def _require_clean_paths(repo_root: Path, relative_paths: Iterable[Path]) -> None:
    arguments = ['git', 'status', '--porcelain=v1', '--']
    arguments.extend(str(path) for path in sorted(relative_paths))
    output = _run(arguments, repo_root).stdout
    if output:
        raise PolicyError(
            'managed files must be clean before updating; commit or restore them:\n' +
            output.rstrip())


def update_repository(repo_root: Path, target_text: str,
                      digest_resolver: DigestResolver = resolve_image_digest,
                      docker_pins: Sequence[DockerPin] = MANAGED_DOCKERFILES,
                      setup_actions: Sequence[Path] =
                      MANAGED_SETUP_GO_ACTIONS) -> List[Path]:
    # Plan first so rerunning a completed update is a no-op even while that
    # update's managed-file diff is still uncommitted. Digest resolution is
    # part of deciding whether the plan is unchanged; writes still require a
    # clean-path check after resolution.
    plan = plan_update(repo_root, target_text, digest_resolver, docker_pins,
                       setup_actions)
    if not plan.changed_paths:
        return []
    managed_paths = set(plan.original) | set(setup_actions)
    _require_clean_paths(repo_root, managed_paths)
    for relative_path in plan.changed_paths:
        (repo_root / relative_path).write_text(
            plan.expected[relative_path], encoding='utf-8')
    return plan.changed_paths


def verify_image_digests(
        repo_root: Path,
        digest_resolver: DigestResolver = resolve_image_digest,
        docker_pins: Sequence[DockerPin] = MANAGED_DOCKERFILES,
        setup_actions: Sequence[Path] = MANAGED_SETUP_GO_ACTIONS) -> None:
    check_repository(repo_root, docker_pins, setup_actions)
    tracked_paths = _tracked_paths(repo_root)
    _, docker_metadata = _validate_inventory(
        repo_root, tracked_paths, docker_pins, setup_actions)
    pins_by_tag = {}
    for metadata in docker_metadata.values():
        pins_by_tag[metadata.version + metadata.flavor] = metadata.digest
    for tag, pinned_digest in sorted(pins_by_tag.items()):
        resolved_digest = digest_resolver(tag)
        if resolved_digest != pinned_digest:
            raise PolicyError(
                f'Go builder tag {tag} resolves to {resolved_digest}, not '
                f'pinned digest {pinned_digest}; rerun make update-go-version')


def main() -> int:
    parser = argparse.ArgumentParser(
        description='Check or update the repository Go compiler version.')
    operation = parser.add_mutually_exclusive_group(required=True)
    operation.add_argument('--check', action='store_true')
    operation.add_argument('--version')
    operation.add_argument('--check-image-digests', action='store_true')
    arguments = parser.parse_args()
    try:
        if arguments.check:
            check_repository(REPOSITORY_ROOT)
        elif arguments.check_image_digests:
            verify_image_digests(REPOSITORY_ROOT)
        else:
            changed_paths = update_repository(REPOSITORY_ROOT,
                                              arguments.version)
            for relative_path in changed_paths:
                print(relative_path)
            check_repository(REPOSITORY_ROOT)
    except PolicyError as error:
        print(f'Go version policy error: {error}', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    raise SystemExit(main())

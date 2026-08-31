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
"""Update the repository-wide Go compiler and pinned builder images."""

import argparse
from contextlib import contextmanager
from dataclasses import dataclass
import fcntl
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import time
from typing import Callable, Dict, Iterable, List, Optional, Set, Tuple
import warnings

from go_version_metadata import docker_runtime_classification
from go_version_metadata import has_go_runtime_reference
from go_version_metadata import is_container_recipe
from go_version_metadata import is_runtime_metadata_path
from go_version_metadata import METADATA_BUILD_TIMEOUT_SECONDS
from go_version_metadata import module_versions

DECIMAL_PATTERN = r'(?:0|[1-9][0-9]*)'
VERSION_PATTERN = re.compile(
    rf'^1\.({DECIMAL_PATTERN})(?:\.({DECIMAL_PATTERN}))?$')
EXACT_VERSION_PATTERN = re.compile(
    rf'^1\.{DECIMAL_PATTERN}\.{DECIMAL_PATTERN}$')
DIGEST_PATTERN = re.compile(r'^sha256:[0-9a-f]{64}$')
GO_DIRECTIVE_PATTERN = re.compile(
    rf'^[ \t]*go[ \t]+(?P<version>1\.{DECIMAL_PATTERN}'
    rf'(?:\.{DECIMAL_PATTERN})?)'
    r'(?P<comment>[ \t]*//[^\r\n]*)?[ \t]*(?=\r?$)', re.MULTILINE)
TOOLCHAIN_PATTERN = re.compile(
    rf'^[ \t]*toolchain[ \t]+go(?P<version>1\.{DECIMAL_PATTERN}'
    rf'\.{DECIMAL_PATTERN})'
    r'(?P<comment>[ \t]*//[^\r\n]*)?[ \t]*(?=\r?$)', re.MULTILINE)
TOOLCHAIN_LINE_PATTERN = re.compile(
    r'^[ \t]*toolchain[ \t]+go\d+\.\d+\.\d+'
    r'(?:[ \t]*//[^\r\n]*)?[ \t]*(?:\r?\n)?(?:\r?\n)?', re.MULTILINE)
DIGEST_LOOKUP_ATTEMPTS = 3
DIGEST_LOOKUP_BACKOFF_SECONDS = (1, 2)
DIGEST_LOOKUP_SOURCE_COUNT = 2
DIGEST_LOOKUP_TIMEOUT_SECONDS = 20
DIGEST_WORKFLOW_TIMEOUT_SECONDS = 600
DIGEST_WORKFLOW_RUNNER_HEADROOM_SECONDS = 60
DIGEST_VERIFICATION_BUDGET_SECONDS = (
    DIGEST_WORKFLOW_TIMEOUT_SECONDS - DIGEST_WORKFLOW_RUNNER_HEADROOM_SECONDS)


@dataclass(frozen=True)
class WorktreePathSnapshot:
    path: Path
    contents: bytes
    file_type: int
    mode: int


@dataclass(frozen=True)
class RepositorySnapshot:
    """Immutable repository transaction baseline."""

    head: str
    index_entries: Tuple[Tuple[Path, str, str], ...]
    worktree_paths: Tuple[WorktreePathSnapshot, ...]
    file_mode_enabled: bool


def _parse_version(version: str) -> Tuple[int, int, int]:
    match = VERSION_PATTERN.fullmatch(version)
    if match is None:
        raise ValueError(f'invalid Go version {version!r}')
    return 1, int(match.group(1)), int(match.group(2) or 0)


def _parse_target_version(version: str) -> Tuple[int, int, int]:
    if EXACT_VERSION_PATTERN.fullmatch(version) is None:
        raise ValueError(
            f'Go version must use the exact stable form 1.X.Y, found '
            f'{version!r}')
    return _parse_version(version)


def _repository_paths(repo_root: Path) -> Set[Path]:
    try:
        output = subprocess.run(
            ('git', 'ls-files', '-z', '--cached', '--others',
             '--exclude-standard'),
            cwd=repo_root,
            check=True,
            capture_output=True,
            text=True,
        ).stdout
    except (FileNotFoundError, subprocess.CalledProcessError) as error:
        raise RuntimeError(
            f'could not discover repository files with git: {error}') from error
    return {Path(path) for path in output.split('\0') if path}


def _head_blob_entries(repo_root: Path,
                       head: str) -> Dict[Path, Tuple[str, str]]:
    output = _git(repo_root, 'ls-tree', '-r', '-z', head).stdout
    entries = {}
    for entry in output.split('\0'):
        if not entry:
            continue
        metadata, path_text = entry.split('\t', 1)
        mode, object_type, object_id = metadata.split(' ')
        if object_type == 'blob':
            entries[Path(path_text)] = (mode, object_id)
    return entries


def _managed_identity_inventory(
    repo_root: Path,
    head: str,
    index_entries: Tuple[Tuple[Path, str, str], ...],
    repository_paths: Optional[Set[Path]],
) -> Tuple[Path, ...]:
    """Identify managed tracked paths without consulting the worktree."""
    sources: Dict[Path, Set[str]] = {}
    for relative_path, (_mode,
                        object_id) in _head_blob_entries(repo_root,
                                                         head).items():
        if repository_paths is None or relative_path in repository_paths:
            sources.setdefault(relative_path, set()).add(object_id)
    for relative_path, _mode, object_id in index_entries:
        if repository_paths is None or relative_path in repository_paths:
            sources.setdefault(relative_path, set()).add(object_id)

    managed = {
        relative_path for relative_path in sources
        if relative_path.name == 'go.mod'
    }
    for relative_path in sorted(managed):
        _ensure_regular_destination(repo_root / relative_path, relative_path)
    blob_contents = {}
    for relative_path, object_ids in sources.items():
        if relative_path.name == 'go.mod':
            continue
        if not is_runtime_metadata_path(relative_path):
            continue
        for object_id in object_ids:
            contents = blob_contents.get(object_id)
            if contents is None:
                contents = _git(repo_root, 'cat-file', 'blob', object_id).stdout
                blob_contents[object_id] = contents
            if has_go_runtime_reference(relative_path, contents):
                managed.add(relative_path)
                break
    return tuple(sorted(managed))


def _module_versions(
    contents: str,
    relative_path: Path,
) -> Tuple[Tuple[int, int, int], Optional[Tuple[int, int, int]]]:
    go_version_text, toolchain_version_text = module_versions(
        contents, relative_path)
    go_version = _parse_version(go_version_text)
    toolchain_version = (
        _parse_version(toolchain_version_text)
        if toolchain_version_text else None)
    return go_version, toolchain_version


def _updated_module_contents(contents: str, relative_path: Path,
                             target: Tuple[int, int, int]) -> str:
    go_version, _ = _module_versions(contents, relative_path)
    go_match = list(GO_DIRECTIVE_PATTERN.finditer(contents))[0]
    go_comment = go_match.group('comment') or ''
    toolchain_matches = list(TOOLCHAIN_PATTERN.finditer(contents))
    toolchain_match = toolchain_matches[0] if toolchain_matches else None
    line_ending = ('\r\n'
                   if contents[go_match.end():].startswith('\r\n') else '\n')
    toolchain_comment = (toolchain_match.group('comment')
                         if toolchain_match else '') or ''
    if go_version > target:
        floor = '.'.join(str(part) for part in go_version)
        compiler = '.'.join(str(part) for part in target)
        raise ValueError(
            f'{relative_path} requires Go {floor}, which exceeds the target '
            f'compiler {compiler}')
    language_floor = (
        go_version if go_version[:2] == target[:2] else
        (target[0], target[1], 0))
    language_version = '.'.join(str(part) for part in language_floor)
    updated = GO_DIRECTIVE_PATTERN.sub(
        lambda _match: f'go {language_version}{go_comment}',
        contents,
        count=1,
    )
    updated = TOOLCHAIN_LINE_PATTERN.sub('', updated)
    if target[2] != 0:
        target_version = '.'.join(str(part) for part in target)
        updated = GO_DIRECTIVE_PATTERN.sub(
            lambda match: (f'{match.group(0)}{line_ending}{line_ending}'
                           f'toolchain go{target_version}{toolchain_comment}'),
            updated,
            count=1,
        )
    return updated.rstrip('\r\n') + line_ending


def _managed_dockerfiles(repo_root: Path,
                         repository_paths: Iterable[Path]) \
        -> Dict[Path, Tuple[str, Dict]]:
    managed = {}
    unmanaged = []
    for relative_path in repository_paths:
        if not is_runtime_metadata_path(relative_path):
            continue
        path = repo_root / relative_path
        if not path.exists():
            continue
        contents = _read_regular_contents(path, relative_path, errors='ignore')
        if not has_go_runtime_reference(relative_path, contents):
            continue
        _ensure_regular_destination(path, relative_path)
        contents = _read_regular_contents(path, relative_path, errors='ignore')
        if is_container_recipe(relative_path):
            docker = docker_runtime_classification(contents)
        else:
            docker = {'classification': 'unsupported', 'candidates': []}
        candidates = docker['candidates']
        if (not is_container_recipe(relative_path) or
                docker['classification'] != 'managed' or len(candidates) != 1):
            unmanaged.append(f'{relative_path} ({docker["classification"]})')
            continue
        candidate = candidates[0]
        required = {'version', 'flavor', 'digest', 'alias', 'line', 'value'}
        if not required.issubset(candidate):
            raise RuntimeError(
                f'Go metadata helper returned incomplete managed Docker '
                f'metadata for {relative_path}')
        managed[relative_path] = (contents, candidate)
    if unmanaged:
        paths = ', '.join(sorted(unmanaged))
        raise ValueError(
            'unsupported Go runtime pins found; use exactly one digest-pinned '
            f'Golang builder image per container recipe: {paths}')
    if not managed:
        raise ValueError('no digest-pinned Golang builder images were found')
    return managed


def _current_managed_paths(repo_root: Path) -> Tuple[Path, ...]:
    repository_paths = _repository_paths(repo_root)
    module_paths = {
        path for path in repository_paths
        if path.name == 'go.mod' and (repo_root / path).exists()
    }
    docker_paths = set(_managed_dockerfiles(repo_root, repository_paths).keys())
    return tuple(sorted(module_paths | docker_paths))


def _verification_deadline_error() -> RuntimeError:
    return RuntimeError(
        'Go builder digest verification exceeded its '
        f'{DIGEST_VERIFICATION_BUDGET_SECONDS}-second end-to-end deadline')


def _remaining_verification_seconds(deadline: float) -> float:
    remaining = deadline - time.monotonic()
    if remaining <= 0:
        raise _verification_deadline_error()
    return remaining


def _inspect_manifest_digest(image: str,
                             timeout: float = DIGEST_LOOKUP_TIMEOUT_SECONDS) \
        -> str:
    result = subprocess.run(
        (
            'docker',
            'buildx',
            'imagetools',
            'inspect',
            image,
            '--format',
            '{{json .Manifest}}',
        ),
        check=True,
        capture_output=True,
        text=True,
        timeout=timeout,
    )
    try:
        digest = json.loads(result.stdout)['digest']
    except (json.JSONDecodeError, KeyError, TypeError) as error:
        raise RuntimeError(
            f'docker buildx returned no manifest digest for {image}') from error
    if not isinstance(digest, str) or DIGEST_PATTERN.fullmatch(digest) is None:
        raise RuntimeError(f'docker buildx returned an invalid digest for '
                           f'{image}: {digest!r}')
    return digest


def resolve_docker_hub_digest(tag: str,
                              *,
                              deadline: Optional[float] = None) -> str:
    images = (f'golang:{tag}', f'mirror.gcr.io/library/golang:{tag}')
    failures = {}
    for attempt in range(DIGEST_LOOKUP_ATTEMPTS):
        for image in images:
            try:
                timeout = (
                    DIGEST_LOOKUP_TIMEOUT_SECONDS if deadline is None else min(
                        DIGEST_LOOKUP_TIMEOUT_SECONDS,
                        _remaining_verification_seconds(deadline)))
                return _inspect_manifest_digest(image, timeout=timeout)
            except FileNotFoundError as error:
                raise RuntimeError(
                    'docker buildx is required to resolve Go builder image '
                    'digests') from error
            except subprocess.CalledProcessError as error:
                detail = (
                    error.stderr.strip() or error.stdout.strip() or str(error))
                failures[image] = detail
            except subprocess.TimeoutExpired as error:
                failures[image] = f'timed out after {error.timeout} seconds'
        if attempt < DIGEST_LOOKUP_ATTEMPTS - 1:
            backoff = DIGEST_LOOKUP_BACKOFF_SECONDS[attempt]
            if deadline is None:
                time.sleep(backoff)
            else:
                remaining = _remaining_verification_seconds(deadline)
                if remaining < backoff:
                    time.sleep(remaining)
                    raise _verification_deadline_error()
                time.sleep(backoff)
    raise RuntimeError('could not resolve the Go builder image from Docker Hub '
                       'or its configured mirror after '
                       f'{DIGEST_LOOKUP_ATTEMPTS} attempts: ' +
                       '; '.join(f'{image}: {detail}'
                                 for image, detail in failures.items()))


def verify_image_digests(
    repo_root: Path,
    digest_resolver: Callable[[str], str] = resolve_docker_hub_digest,
    repository_paths: Optional[Iterable[Path]] = None,
) -> None:
    deadline = time.monotonic() + DIGEST_VERIFICATION_BUDGET_SECONDS
    paths = set(repository_paths or _repository_paths(repo_root))
    dockerfiles = _managed_dockerfiles(repo_root, paths)
    _remaining_verification_seconds(deadline)
    pins_by_tag: Dict[str, Dict[str, List[Path]]] = {}
    for relative_path, (_contents, candidate) in dockerfiles.items():
        tag = candidate['version'] + candidate['flavor']
        digest = candidate['digest']
        pins_by_tag.setdefault(tag, {}).setdefault(digest,
                                                   []).append(relative_path)

    worst_case_seconds = _digest_verification_worst_case_seconds(
        len(pins_by_tag))
    configured_seconds = (METADATA_BUILD_TIMEOUT_SECONDS + worst_case_seconds)
    if configured_seconds > DIGEST_VERIFICATION_BUDGET_SECONDS:
        raise RuntimeError(
            'Go builder digest verification is configured for a worst-case '
            f'{configured_seconds} seconds including '
            f'{METADATA_BUILD_TIMEOUT_SECONDS} seconds of metadata helper '
            'build/setup headroom, exceeding its '
            f'{DIGEST_VERIFICATION_BUDGET_SECONDS}-second budget')

    errors = []
    for tag, pins_by_digest in sorted(pins_by_tag.items()):
        if len(pins_by_digest) != 1:
            errors.append(f'golang:{tag} has inconsistent pinned digests: '
                          f'{sorted(pins_by_digest)}')
            continue
        pinned_digest = next(iter(pins_by_digest))
        _remaining_verification_seconds(deadline)
        if digest_resolver is resolve_docker_hub_digest:
            resolved_digest = digest_resolver(tag, deadline=deadline)
        else:
            resolved_digest = digest_resolver(tag)
        _remaining_verification_seconds(deadline)
        if DIGEST_PATTERN.fullmatch(resolved_digest) is None:
            raise ValueError(f'invalid digest resolved for golang:{tag}: '
                             f'{resolved_digest!r}')
        if pinned_digest != resolved_digest:
            relative_paths = ', '.join(
                str(path) for path in sorted(pins_by_digest[pinned_digest]))
            errors.append(f'golang:{tag} resolves to {resolved_digest}, but '
                          f'{relative_paths} pin {pinned_digest}')
    if errors:
        raise ValueError('Go builder image digest verification failed:\n  ' +
                         '\n  '.join(errors))


def _digest_verification_worst_case_seconds(tag_count: int) -> int:
    return tag_count * (
        DIGEST_LOOKUP_ATTEMPTS * DIGEST_LOOKUP_SOURCE_COUNT *
        DIGEST_LOOKUP_TIMEOUT_SECONDS + sum(DIGEST_LOOKUP_BACKOFF_SECONDS))


def synchronized_contents(
    repo_root: Path,
    target_version: str,
    digest_resolver: Callable[[str], str] = resolve_docker_hub_digest,
    repository_paths: Optional[Iterable[Path]] = None,
) -> Dict[Path, str]:
    expected_contents, _ = _synchronized_contents_and_originals(
        repo_root,
        target_version,
        digest_resolver=digest_resolver,
        repository_paths=repository_paths,
    )
    return expected_contents


def _synchronized_contents_and_originals(
    repo_root: Path,
    target_version: str,
    digest_resolver: Callable[[str], str] = resolve_docker_hub_digest,
    repository_paths: Optional[Iterable[Path]] = None,
    originals_callback: Optional[Callable[[Dict[Path, str]], None]] = None,
) -> Tuple[Dict[Path, str], Dict[Path, str]]:
    target = _parse_target_version(target_version)
    paths = set(repository_paths or _repository_paths(repo_root))
    module_paths = sorted(
        path for path in paths
        if path.name == 'go.mod' and (repo_root / path).exists())
    if Path('go.mod') not in module_paths:
        raise ValueError('the repository root go.mod was not found')

    original_contents = {
        relative_path:
            _read_regular_contents(repo_root / relative_path, relative_path)
        for relative_path in module_paths
    }
    root_contents = original_contents[Path('go.mod')]
    root_go_version, root_toolchain_version = _module_versions(
        root_contents, Path('go.mod'))
    current = root_toolchain_version or root_go_version
    if target < current:
        current_version = '.'.join(str(part) for part in current)
        raise ValueError(f'refusing to downgrade Go from {current_version} to '
                         f'{target_version}')

    expected_contents = {}
    for relative_path in module_paths:
        contents = original_contents[relative_path]
        expected_contents[relative_path] = _updated_module_contents(
            contents, relative_path, target)

    dockerfiles = _managed_dockerfiles(repo_root, paths)
    original_contents.update({
        relative_path: contents
        for relative_path, (contents, _candidate) in dockerfiles.items()
    })
    if originals_callback is not None:
        originals_callback(original_contents)
    flavors = {
        candidate['flavor'] for _contents, candidate in dockerfiles.values()
    }
    digests = {}
    for flavor in sorted(flavors):
        tag = f'{target_version}{flavor}'
        digest = digest_resolver(tag)
        if DIGEST_PATTERN.fullmatch(digest) is None:
            raise ValueError(f'invalid digest resolved for golang:{tag}: '
                             f'{digest!r}')
        digests[flavor] = digest

    for relative_path, (contents, candidate) in dockerfiles.items():
        flavor = candidate['flavor']
        expected_contents[relative_path] = _updated_dockerfile_contents(
            contents,
            candidate,
            target_version,
            digests[flavor],
        )
    return expected_contents, original_contents


def _updated_dockerfile_contents(contents: str, candidate: Dict,
                                 target_version: str, digest: str) -> str:
    lines = contents.splitlines(keepends=True)
    line_index = candidate['line'] - 1
    if line_index < 0 or line_index >= len(lines):
        raise RuntimeError('managed Docker candidate has an invalid line')
    line = lines[line_index]
    if line.endswith('\r\n'):
        body, ending = line[:-2], '\r\n'
    elif line.endswith('\n') or line.endswith('\r'):
        body, ending = line[:-1], line[-1]
    else:
        body, ending = line, ''
    indentation = body[:len(body) - len(body.lstrip())]
    if body.strip() != candidate['value']:
        raise RuntimeError(
            'managed Docker candidate no longer matches its source line')
    lines[line_index] = (
        f'{indentation}FROM golang:{target_version}{candidate["flavor"]}@'
        f'{digest} AS {candidate["alias"]}{ending}')
    return ''.join(lines)


def sync(
    repo_root: Path,
    target_version: str,
    digest_resolver: Callable[[str], str] = resolve_docker_hub_digest,
    repository_paths: Optional[Iterable[Path]] = None,
) -> List[Path]:
    with _exclusive_transaction_lock(repo_root):
        return _sync_locked(repo_root, target_version, digest_resolver,
                            repository_paths)


def _sync_locked(
    repo_root: Path,
    target_version: str,
    digest_resolver: Callable[[str], str] = resolve_docker_hub_digest,
    repository_paths: Optional[Iterable[Path]] = None,
) -> List[Path]:
    initial_head = _git(repo_root, 'rev-parse', '--verify',
                        'HEAD').stdout.strip()
    initial_index = _complete_stage_zero_index(repo_root)
    scoped_repository_paths = (
        set(repository_paths) if repository_paths is not None else None)
    managed_identity = _managed_identity_inventory(
        repo_root,
        initial_head,
        initial_index,
        scoped_repository_paths,
    )
    for relative_path in managed_identity:
        _ensure_regular_destination(repo_root / relative_path, relative_path)
    identity_head = _require_clean_managed_paths(repo_root, managed_identity)
    if identity_head != initial_head:
        raise RuntimeError(
            f'HEAD changed during Go version update from {initial_head} '
            f'to {identity_head}')
    snapshot = None

    def capture_snapshot(originals: Dict[Path, str]) -> None:
        nonlocal snapshot
        if set(originals) != set(managed_identity):
            raise RuntimeError(
                'managed Go version identity changed while planning the '
                'update')
        snapshot = _capture_repository_snapshot(repo_root, originals,
                                                initial_head, initial_index)

    expected_contents, original_contents = _synchronized_contents_and_originals(
        repo_root,
        target_version,
        digest_resolver=digest_resolver,
        repository_paths=scoped_repository_paths,
        originals_callback=capture_snapshot,
    )
    if snapshot is None:
        raise RuntimeError('Go version update did not capture repository state')
    changed_paths = []
    for relative_path, expected in expected_contents.items():
        original = original_contents[relative_path]
        if original == expected:
            continue
        changed_paths.append(relative_path)

    _validate_all_managed_paths(repo_root, snapshot, original_contents)
    if not changed_paths:
        return []

    start_head = snapshot.head
    snapshot_index = {
        path: (mode, object_id)
        for path, mode, object_id in snapshot.index_entries
    }
    managed_index_entries = {
        path: snapshot_index[path] for path in original_contents
    }
    worktree_parent = tempfile.TemporaryDirectory(
        prefix='kfp-go-version-worktree-')
    worktree = Path(worktree_parent.name) / 'repository'
    recovery_bundle = None
    recovery_path = None
    original_restore_patches = {}
    original_restore_paths = {}
    owned_paths = []
    try:
        _git(repo_root, 'worktree', 'add', '--detach', str(worktree),
             start_head)
        for relative_path in changed_paths:
            path = worktree / relative_path
            _ensure_regular_destination(path, relative_path)
            path.write_text(expected_contents[relative_path], encoding='utf-8')
        _verify_worktree_plan(worktree, expected_contents,
                              managed_index_entries, changed_paths)
        _validate_all_managed_paths(repo_root, snapshot, original_contents)
        patch = _git(
            worktree,
            '--literal-pathspecs',
            'diff',
            '--binary',
            '--full-index',
            '--no-ext-diff',
            '--no-textconv',
            '--src-prefix=a/',
            '--dst-prefix=b/',
            'HEAD',
            '--',
            *(str(path) for path in changed_paths),
        ).stdout
        if not patch:
            raise RuntimeError('Git produced an empty Go version update patch')
        empty_tree = _empty_tree_oid(worktree)
        for relative_path in changed_paths:
            path_patch = _git(
                worktree,
                '--literal-pathspecs',
                'diff',
                '--binary',
                '--full-index',
                '--no-ext-diff',
                '--no-textconv',
                '--src-prefix=a/',
                '--dst-prefix=b/',
                'HEAD',
                '--',
                str(relative_path),
            ).stdout
            if not path_patch:
                raise RuntimeError(f'Git produced an empty recovery patch for '
                                   f'{relative_path}')
            restore_patch = _git(
                worktree,
                '--literal-pathspecs',
                'diff',
                '--binary',
                '--full-index',
                '--no-ext-diff',
                '--no-textconv',
                '--src-prefix=a/',
                '--dst-prefix=b/',
                empty_tree,
                start_head,
                '--',
                str(relative_path),
            ).stdout
            if not restore_patch:
                raise RuntimeError(
                    f'Git produced an empty original restore patch for '
                    f'{relative_path}')
            original_restore_patches[relative_path] = restore_patch
        (recovery_bundle, recovery_path,
         original_restore_paths) = _write_recovery_bundle(
             repo_root,
             start_head,
             target_version,
             patch,
             original_restore_patches,
             snapshot,
         )

        _validate_all_managed_paths(repo_root, snapshot, original_contents)
        _git(repo_root, 'apply', '--check', '--whitespace=nowarn',
             str(recovery_path))
        live_contents = dict(original_contents)
        for relative_path in changed_paths:
            _validate_all_managed_paths(repo_root, snapshot, live_contents)
            _apply_managed_contents(
                repo_root,
                relative_path,
                expected_contents[relative_path],
                snapshot,
            )
            owned_paths.append(relative_path)
            live_contents[relative_path] = expected_contents[relative_path]
        _validate_all_managed_paths(repo_root, snapshot, expected_contents)
        _verify_repository_consistency(repo_root)
        _validate_all_managed_paths(repo_root, snapshot, expected_contents)
    except BaseException as update_error:
        rollback_errors = []
        unresolved_paths = []
        recovery_interrupt = None
        if owned_paths:
            (rollback_errors, unresolved_paths,
             recovery_interrupt) = _recover_applied_paths(
                 repo_root,
                 owned_paths,
                 original_contents,
                 expected_contents,
                 snapshot,
                 original_restore_paths,
             )
        rollback_detail = ''
        if rollback_errors:
            rollback_detail += ('; automatic rollback failed: ' +
                                '; '.join(rollback_errors))
        if unresolved_paths:
            rollback_detail += (
                '; managed paths left unchanged because they no longer '
                'matched either the original or planned contents: ' +
                ', '.join(unresolved_paths))
        if recovery_interrupt is not None:
            recovery = (f'; recovery bundle retained at {recovery_bundle}'
                        if recovery_bundle else '')
            warnings.warn(
                f'Go update recovery interrupted after {update_error}'
                f'{rollback_detail}{recovery}', RuntimeWarning)
            raise recovery_interrupt from update_error
        if not isinstance(update_error, Exception):
            if recovery_bundle:
                message = ('Go update interrupted; recovery bundle retained '
                           f'at {recovery_bundle}{rollback_detail}')
            else:
                message = ('Go update interrupted before application; '
                           f'managed files were not changed{rollback_detail}')
            warnings.warn(message, RuntimeWarning)
            raise
        recovery = (f'; recovery bundle retained at {recovery_bundle}'
                    if recovery_bundle else '')
        raise RuntimeError(f'failed to apply Go version update: '
                           f'{update_error}{rollback_detail}{recovery}') \
            from update_error
    finally:
        try:
            if worktree.exists():
                _git(repo_root, 'worktree', 'remove', '--force', str(worktree))
        except (OSError, RuntimeError, subprocess.CalledProcessError) as error:
            warnings.warn(
                f'could not remove temporary Go update worktree {worktree}: '
                f'{error}', RuntimeWarning)
        try:
            worktree_parent.cleanup()
        except OSError as error:
            warnings.warn(
                f'could not remove temporary Go update directory '
                f'{worktree_parent.name}: {error}', RuntimeWarning)

    try:
        shutil.rmtree(recovery_bundle)
        _fsync_directory(recovery_bundle.parent)
    except OSError as error:
        warnings.warn(
            f'Go update committed and verified, but recovery bundle '
            f'{recovery_bundle} could not be removed: {error}', RuntimeWarning)
    return sorted(changed_paths)


@contextmanager
def _exclusive_transaction_lock(repo_root: Path):
    common_dir_text = _git(repo_root, 'rev-parse',
                           '--git-common-dir').stdout.strip()
    common_dir = Path(common_dir_text)
    if not common_dir.is_absolute():
        common_dir = repo_root / common_dir
    lock_path = common_dir.resolve() / 'go-version-update.lock'
    descriptor = os.open(lock_path, os.O_RDWR | os.O_CREAT, 0o600)
    try:
        fcntl.flock(descriptor, fcntl.LOCK_EX)
        yield
    finally:
        fcntl.flock(descriptor, fcntl.LOCK_UN)
        os.close(descriptor)


def _git(repo_root: Path, *arguments: str) -> subprocess.CompletedProcess:
    try:
        return subprocess.run(
            ('git', *arguments),
            cwd=repo_root,
            check=True,
            capture_output=True,
            text=True,
        )
    except (FileNotFoundError, subprocess.CalledProcessError) as error:
        detail = getattr(error, 'stderr', '') or str(error)
        raise RuntimeError(
            f'git {" ".join(arguments)} failed: {detail.strip()}') from error


def _empty_tree_oid(repo_root: Path) -> str:
    try:
        return subprocess.run(
            ('git', 'mktree'),
            cwd=repo_root,
            check=True,
            capture_output=True,
            text=True,
            input='',
        ).stdout.strip()
    except (FileNotFoundError, subprocess.CalledProcessError) as error:
        detail = getattr(error, 'stderr', '') or str(error)
        raise RuntimeError(
            f'could not create Git empty tree: {detail.strip()}') from error


def _require_clean_managed_paths(repo_root: Path,
                                 relative_paths: Iterable[Path]) -> str:
    relative_paths = tuple(sorted(relative_paths))
    _require_supported_managed_git_state(repo_root, relative_paths)
    paths = tuple(str(path) for path in relative_paths)
    head = _git(repo_root, 'rev-parse', '--verify', 'HEAD').stdout.strip()
    _git(repo_root, '--literal-pathspecs', 'ls-files', '--error-unmatch', '--',
         *paths)
    status = _git(repo_root, '--literal-pathspecs', 'status', '--porcelain=v1',
                  '-z', '--untracked-files=all', '--', *paths).stdout
    if status:
        raise RuntimeError(
            'managed Go version files must be tracked and clean before '
            'updating')
    return head


def _require_supported_managed_git_state(
    repo_root: Path,
    relative_paths: Iterable[Path],
) -> None:
    relative_paths = tuple(sorted(relative_paths))
    path_arguments = tuple(str(path) for path in relative_paths)
    flags = _git(repo_root, '--literal-pathspecs', 'ls-files', '-v', '-z', '--',
                 *path_arguments).stdout
    flagged = []
    seen = set()
    for entry in flags.split('\0'):
        if not entry:
            continue
        tag, path_text = entry.split(' ', 1)
        relative_path = Path(path_text)
        seen.add(relative_path)
        if tag != 'H':
            flagged.append(f'{relative_path} ({tag})')
    missing = set(relative_paths) - seen
    if missing:
        flagged.extend(f'{path} (missing)' for path in sorted(missing))
    if flagged:
        raise RuntimeError(
            'managed Go version files use unsupported Git index flags; '
            'clear assume-unchanged and skip-worktree before updating: ' +
            ', '.join(flagged))

    attributes = _git(repo_root, '--literal-pathspecs', 'check-attr', '-z',
                      'filter', '--', *path_arguments).stdout.split('\0')
    filtered = []
    for offset in range(0, len(attributes) - 2, 3):
        path_text, attribute, value = attributes[offset:offset + 3]
        if attribute == 'filter' and value != 'unspecified':
            filtered.append(f'{path_text} ({value})')
    if filtered:
        raise RuntimeError(
            'managed Go version files use unsupported Git clean/smudge '
            'filters: ' + ', '.join(filtered))


def _capture_repository_snapshot(
    repo_root: Path,
    expected_contents: Dict[Path, str],
    expected_head: str,
    expected_index: Tuple[Tuple[Path, str, str], ...],
) -> RepositorySnapshot:
    """Freeze the complete pre-update state of every managed path."""
    relative_paths = tuple(sorted(expected_contents))
    head = _git(repo_root, 'rev-parse', '--verify', 'HEAD').stdout.strip()
    if head != expected_head:
        raise RuntimeError(
            f'HEAD changed during Go version update from {expected_head} '
            f'to {head}')
    _git(repo_root, '--literal-pathspecs', 'ls-files', '--error-unmatch', '--',
         *(str(path) for path in relative_paths))
    current_index = _complete_stage_zero_index(repo_root)
    if current_index != expected_index:
        raise RuntimeError(
            'Git index changed while capturing the Go version update '
            'snapshot')
    file_mode_enabled = _core_file_mode_enabled(repo_root)
    worktree_paths = tuple(
        _snapshot_worktree_path(repo_root, path) for path in relative_paths)
    if any(state.contents != expected_contents[state.path].encode('utf-8')
           for state in worktree_paths):
        raise RuntimeError(
            'managed Go version files changed while capturing the update '
            'snapshot')
    snapshot = RepositorySnapshot(
        head=head,
        index_entries=current_index,
        worktree_paths=worktree_paths,
        file_mode_enabled=file_mode_enabled,
    )
    _validate_all_managed_paths(
        repo_root,
        snapshot,
        expected_contents,
        require_head_match=True,
    )
    return snapshot


def _snapshot_worktree_path(
    repo_root: Path,
    relative_path: Path,
) -> WorktreePathSnapshot:
    path = repo_root / relative_path
    try:
        path_stat = path.lstat()
    except FileNotFoundError as error:
        raise RuntimeError(f'{relative_path} disappeared during Go update') \
            from error
    if not stat.S_ISREG(path_stat.st_mode):
        raise ValueError(
            f'{relative_path} must be a regular file; symlinks and other '
            'special files are not supported')
    return WorktreePathSnapshot(
        path=relative_path,
        contents=path.read_bytes(),
        file_type=stat.S_IFMT(path_stat.st_mode),
        mode=stat.S_IMODE(path_stat.st_mode),
    )


def _ensure_snapshot_state(
    repo_root: Path,
    snapshot: RepositorySnapshot,
    expected_contents: Dict[Path, str],
    relative_paths: Iterable[Path],
) -> None:
    """Require HEAD, index identity, contents, and modes to match a snapshot."""
    relative_paths = tuple(relative_paths)
    _require_supported_managed_git_state(repo_root, relative_paths)
    head = _git(repo_root, 'rev-parse', '--verify', 'HEAD').stdout.strip()
    if head != snapshot.head:
        raise RuntimeError(
            f'HEAD changed during Go version update from {snapshot.head} '
            f'to {head}')
    actual_index = _complete_stage_zero_index(repo_root)
    if actual_index != snapshot.index_entries:
        raise RuntimeError(
            'Git index changed while committing Go version update')
    file_mode_enabled = _core_file_mode_enabled(repo_root)
    if file_mode_enabled != snapshot.file_mode_enabled:
        raise RuntimeError(
            'Git core.fileMode changed while committing Go version update')
    path_snapshots = {state.path: state for state in snapshot.worktree_paths}
    changed = []
    for relative_path in relative_paths:
        try:
            current = _snapshot_worktree_path(repo_root, relative_path)
        except (RuntimeError, ValueError) as error:
            changed.append(f'{relative_path} ({error})')
            continue
        if current.contents != expected_contents[relative_path].encode('utf-8'):
            changed.append(str(relative_path))
            continue
        expected_path = path_snapshots[relative_path]
        if current.file_type != expected_path.file_type:
            changed.append(f'{relative_path} (worktree file type changed)')
        elif current.mode != expected_path.mode:
            changed.append(f'{relative_path} (worktree mode changed)')
    if changed:
        raise RuntimeError(
            'files changed while committing Go version update: ' +
            ', '.join(sorted(changed)))


def _validate_all_managed_paths(
    repo_root: Path,
    snapshot: RepositorySnapshot,
    expected_contents: Dict[Path, str],
    *,
    require_head_match: bool = False,
) -> None:
    """Validate the complete managed-path transaction boundary."""
    managed_paths = tuple(state.path for state in snapshot.worktree_paths)
    try:
        current_managed_paths = _current_managed_paths(repo_root)
    except (RuntimeError, ValueError) as error:
        raise RuntimeError(
            f'managed Go version path membership changed: {error}') from error
    if current_managed_paths != managed_paths:
        added = set(current_managed_paths) - set(managed_paths)
        removed = set(managed_paths) - set(current_managed_paths)
        details = []
        if added:
            details.append('added ' +
                           ', '.join(str(path) for path in sorted(added)))
        if removed:
            details.append('removed ' +
                           ', '.join(str(path) for path in sorted(removed)))
        raise RuntimeError('managed Go version path membership changed: ' +
                           '; '.join(details))
    expected_paths = set(expected_contents)
    snapshot_paths = set(managed_paths)
    if expected_paths != snapshot_paths:
        missing = snapshot_paths - expected_paths
        extra = expected_paths - snapshot_paths
        details = []
        if missing:
            details.append('missing ' +
                           ', '.join(str(path) for path in sorted(missing)))
        if extra:
            details.append('unexpected ' +
                           ', '.join(str(path) for path in sorted(extra)))
        raise RuntimeError(
            'managed Go version validation received an incomplete path set: ' +
            '; '.join(details))
    if require_head_match:
        clean_head = _require_clean_managed_paths(repo_root, managed_paths)
        if clean_head != snapshot.head:
            raise RuntimeError(
                f'HEAD changed during Go version update from {snapshot.head} '
                f'to {clean_head}')
    _ensure_snapshot_state(repo_root, snapshot, expected_contents,
                           managed_paths)


def _indexed_stage_zero_entries(
        repo_root: Path,
        relative_paths: Iterable[Path]) -> Dict[Path, Tuple[str, str]]:
    relative_paths = tuple(sorted(relative_paths))
    output = _git(
        repo_root,
        '--literal-pathspecs',
        'ls-files',
        '--stage',
        '-z',
        '--',
        *(str(path) for path in relative_paths),
    ).stdout
    parsed_entries = []
    path_counts = {}
    for entry in output.split('\0'):
        if not entry:
            continue
        metadata, path_text = entry.split('\t', 1)
        mode, object_id, stage = metadata.split(' ')
        relative_path = Path(path_text)
        parsed_entries.append((relative_path, mode, object_id, stage))
        path_counts[relative_path] = path_counts.get(relative_path, 0) + 1
    duplicates = sorted(
        path for path, count in path_counts.items() if count > 1)
    if duplicates:
        raise RuntimeError(
            'managed Go version files have multiple Git index entries: ' +
            ', '.join(str(path) for path in duplicates))

    index_entries = {}
    for relative_path, mode, object_id, stage in parsed_entries:
        if stage != '0' or mode not in ('100644', '100755'):
            raise RuntimeError(
                f'{relative_path} has unsupported Git index mode {mode} '
                f'at stage {stage}')
        index_entries[relative_path] = (mode, object_id)
    missing = set(relative_paths) - set(index_entries)
    if missing:
        raise RuntimeError(
            'managed Go version files are missing from the Git index: ' +
            ', '.join(str(path) for path in sorted(missing)))
    return index_entries


def _complete_stage_zero_index(
        repo_root: Path) -> Tuple[Tuple[Path, str, str], ...]:
    """Return every complete stage-0 entry, rejecting an unmerged index."""
    output = _git(repo_root, 'ls-files', '--stage', '-z').stdout
    entries = []
    seen = set()
    for entry in output.split('\0'):
        if not entry:
            continue
        metadata, path_text = entry.split('\t', 1)
        mode, object_id, stage = metadata.split(' ')
        relative_path = Path(path_text)
        if relative_path in seen:
            raise RuntimeError(
                f'{relative_path} has multiple Git index entries')
        seen.add(relative_path)
        if stage != '0':
            raise RuntimeError(
                f'{relative_path} has unsupported Git index stage {stage}')
        entries.append((relative_path, mode, object_id))
    return tuple(sorted(entries, key=lambda item: str(item[0])))


def _core_file_mode_enabled(repo_root: Path) -> bool:
    result = subprocess.run(
        ('git', 'config', '--type=bool', '--get', 'core.fileMode'),
        cwd=repo_root,
        check=False,
        capture_output=True,
        text=True,
    )
    if result.returncode == 1:
        return True
    if result.returncode != 0:
        detail = result.stderr.strip() or f'exit status {result.returncode}'
        raise RuntimeError(f'could not read Git core.fileMode: {detail}')
    value = result.stdout.strip()
    if value not in ('true', 'false'):
        raise RuntimeError(f'Git returned invalid core.fileMode {value!r}')
    return value == 'true'


def _verify_worktree_plan(worktree: Path, expected_contents: Dict[Path, str],
                          expected_index_entries: Dict[Path, Tuple[str, str]],
                          changed_paths: Iterable[Path]) -> None:
    changed_paths = tuple(changed_paths)
    managed_paths = tuple(sorted(expected_contents))
    worktree_index_entries = _indexed_stage_zero_entries(
        worktree, managed_paths)
    if worktree_index_entries != expected_index_entries:
        raise RuntimeError(
            'temporary worktree Git index entries do not match the update '
            'plan')
    _ensure_expected_contents(worktree, expected_contents, managed_paths,
                              expected_index_entries)
    _verify_repository_consistency(worktree)
    actual = set(
        path for path in _git(
            worktree,
            '--literal-pathspecs',
            'diff',
            '--name-only',
            '-z',
            'HEAD',
        ).stdout.split('\0') if path)
    expected = {str(path) for path in changed_paths}
    if actual != expected:
        raise RuntimeError(
            f'temporary worktree changed {sorted(actual)}, expected '
            f'{sorted(expected)}')
    _git(worktree, 'diff', '--check')


def _verify_repository_consistency(repo_root: Path) -> None:
    checker = repo_root / '.github/resources/scripts/go_version_consistency_test.py'
    if not checker.exists():
        return
    try:
        subprocess.run(
            ('python3', str(checker)),
            cwd=repo_root,
            check=True,
            capture_output=True,
            text=True,
        )
    except (FileNotFoundError, subprocess.CalledProcessError) as error:
        detail = getattr(error, 'stderr', '') or getattr(error, 'stdout',
                                                         '') or str(error)
        raise RuntimeError(f'Go version consistency verification failed: '
                           f'{detail.strip()}') from error


def _write_recovery_bundle(
    repo_root: Path,
    start_head: str,
    target_version: str,
    combined_patch: str,
    original_restore_patches: Dict[Path, str],
    snapshot: RepositorySnapshot,
) -> Tuple[Path, Path, Dict[Path, Path]]:
    common_dir_text = _git(repo_root, 'rev-parse',
                           '--git-common-dir').stdout.strip()
    common_dir = Path(common_dir_text)
    if not common_dir.is_absolute():
        common_dir = repo_root / common_dir
    recovery_root = common_dir / 'go-version-update-recovery'
    recovery_root_existed = recovery_root.exists()
    recovery_root.mkdir(mode=0o700, parents=True, exist_ok=True)
    recovery_root.chmod(0o700)
    pending = Path(
        tempfile.mkdtemp(
            dir=recovery_root,
            prefix=f'.pending-{start_head[:12]}-to-go{target_version}-',
        ))
    pending.chmod(0o700)
    bundle = pending.with_name(
        pending.name.removeprefix('.pending-') + '.bundle')
    try:
        combined_name = 'combined-forward.patch'
        _write_durable_recovery_file(pending / combined_name, combined_patch)
        restore_names = {}
        manifest_paths = {}
        original_files = {}
        snapshot_paths = {
            state.path: state for state in snapshot.worktree_paths
        }
        for index, relative_path in enumerate(sorted(original_restore_patches)):
            restore_name = f'{index:04d}-restore-original.patch'
            _write_durable_recovery_file(
                pending / restore_name,
                original_restore_patches[relative_path],
            )
            restore_names[relative_path] = restore_name
            manifest_paths[str(relative_path)] = restore_name
            original_name = f'{index:04d}-original.bin'
            original = snapshot_paths[relative_path]
            _write_durable_recovery_bytes(pending / original_name,
                                          original.contents)
            original_files[str(relative_path)] = {
                'contents': original_name,
                'mode': f'{original.mode:04o}',
            }
        manifest = {
            'startHead': start_head,
            'targetVersion': target_version,
            'combinedForwardPatch': combined_name,
            'originalFiles': original_files,
            'originalRestorePatches': manifest_paths,
        }
        _write_durable_recovery_file(
            pending / 'manifest.json',
            json.dumps(manifest, indent=2, sort_keys=True) + '\n',
        )
        _fsync_directory(pending)
        os.rename(pending, bundle)
        _fsync_directory(recovery_root)
        if not recovery_root_existed:
            _fsync_directory(common_dir)
    except BaseException:
        incomplete = pending
        try:
            if not pending.exists():
                incomplete = bundle
            if incomplete.exists():
                shutil.rmtree(incomplete)
                _fsync_directory(recovery_root)
        except OSError as cleanup_error:
            warnings.warn(
                f'incomplete recovery bundle retained at {incomplete}: '
                f'{cleanup_error}', RuntimeWarning)
        raise
    return (
        bundle,
        bundle / combined_name,
        {
            relative_path: bundle / restore_name
            for relative_path, restore_name in restore_names.items()
        },
    )


def _write_durable_recovery_file(path: Path, contents: str) -> None:
    _write_durable_recovery_bytes(path, contents.encode('utf-8'))


def _write_durable_recovery_bytes(path: Path, contents: bytes) -> None:
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        with os.fdopen(descriptor, 'wb') as recovery_file:
            descriptor = -1
            recovery_file.write(contents)
            recovery_file.flush()
            os.fsync(recovery_file.fileno())
    except BaseException:
        if descriptor >= 0:
            os.close(descriptor)
        raise


def _fsync_directory(path: Path) -> None:
    descriptor = os.open(path, os.O_RDONLY)
    try:
        os.fsync(descriptor)
    finally:
        os.close(descriptor)


def _snapshot_state_matches(repo_root: Path, snapshot: RepositorySnapshot,
                            expected_contents: Dict[Path, str],
                            relative_paths: Iterable[Path]) -> bool:
    try:
        _ensure_snapshot_state(repo_root, snapshot, expected_contents,
                               relative_paths)
        return True
    except Exception:
        return False


def _write_managed_contents(
    repo_root: Path,
    relative_path: Path,
    contents: str,
    snapshot: RepositorySnapshot,
) -> None:
    path = repo_root / relative_path
    expected_path = next(state for state in snapshot.worktree_paths
                         if state.path == relative_path)
    _ensure_regular_destination(path, relative_path)
    with path.open('r+b') as destination:
        destination.seek(0)
        destination.write(contents.encode('utf-8'))
        destination.truncate()
        destination.flush()
        os.fsync(destination.fileno())
    current = _snapshot_worktree_path(repo_root, relative_path)
    if (current.contents != contents.encode('utf-8') or
            current.file_type != expected_path.file_type or
            current.mode != expected_path.mode):
        raise RuntimeError(
            f'{relative_path} changed while writing the Go version update')


def _apply_managed_contents(repo_root: Path, relative_path: Path, contents: str,
                            snapshot: RepositorySnapshot) -> None:
    _write_managed_contents(repo_root, relative_path, contents, snapshot)


def _restore_managed_contents(repo_root: Path, relative_path: Path,
                              contents: str,
                              snapshot: RepositorySnapshot) -> None:
    _write_managed_contents(repo_root, relative_path, contents, snapshot)


def _recover_applied_paths(
    repo_root: Path,
    relative_paths: Iterable[Path],
    original_contents: Dict[Path, str],
    expected_contents: Dict[Path, str],
    snapshot: RepositorySnapshot,
    original_restore_paths: Dict[Path, Path],
) -> Tuple[List[str], List[str], Optional[BaseException]]:
    errors = []
    unresolved = []
    interrupt = None
    for relative_path in relative_paths:
        if _snapshot_state_matches(repo_root, snapshot, original_contents,
                                   (relative_path,)):
            continue
        if not _snapshot_state_matches(repo_root, snapshot, expected_contents,
                                       (relative_path,)):
            unresolved.append(f'{relative_path} (original restore patch: '
                              f'{original_restore_paths[relative_path]})')
            continue
        try:
            _restore_managed_contents(repo_root, relative_path,
                                      original_contents[relative_path],
                                      snapshot)
            _ensure_snapshot_state(repo_root, snapshot, original_contents,
                                   (relative_path,))
        except BaseException as error:
            if not isinstance(error, Exception) and interrupt is None:
                interrupt = error
            errors.append(
                f'{relative_path}: {error}; original restore patch retained '
                f'at {original_restore_paths[relative_path]}')
    try:
        _validate_all_managed_paths(repo_root, snapshot, original_contents)
    except BaseException as error:
        if not isinstance(error, Exception) and interrupt is None:
            interrupt = error
        errors.append(f'all managed paths after recovery: {error}')
    return errors, unresolved, interrupt


def _ensure_expected_contents(
    repo_root: Path,
    expected_contents: Dict[Path, str],
    relative_paths: Iterable[Path],
    expected_index_entries: Optional[Dict[Path, Tuple[str,
                                                      str]]] = None) -> None:
    relative_paths = tuple(relative_paths)
    indexed_entries = (
        _indexed_stage_zero_entries(repo_root, relative_paths)
        if expected_index_entries is not None else {})
    file_mode_enabled = (
        _core_file_mode_enabled(repo_root)
        if expected_index_entries is not None else False)
    changed = []
    for relative_path in relative_paths:
        try:
            current = _read_regular_contents(repo_root / relative_path,
                                             relative_path)
        except (RuntimeError, ValueError) as error:
            changed.append(f'{relative_path} ({error})')
            continue
        if current != expected_contents[relative_path]:
            changed.append(str(relative_path))
            continue
        if expected_index_entries is not None:
            expected_entry = expected_index_entries[relative_path]
            current_entry = indexed_entries[relative_path]
            if current_entry != expected_entry:
                changed.append(f'{relative_path} (Git index entry changed)')
                continue
            expected_mode = expected_entry[0] == '100755'
            effective_mode = (
                _is_executable(repo_root / relative_path, relative_path)
                if file_mode_enabled else current_entry[0] == '100755')
            if effective_mode != expected_mode:
                changed.append(f'{relative_path} (executable mode changed)')
    if changed:
        raise RuntimeError(
            'files changed while committing Go version update: ' +
            ', '.join(sorted(changed)))


def _ensure_regular_destination(path: Path, relative_path: Path) -> None:
    try:
        mode = path.lstat().st_mode
    except FileNotFoundError as error:
        raise RuntimeError(f'{relative_path} disappeared during Go update') \
            from error
    if not stat.S_ISREG(mode):
        raise ValueError(
            f'{relative_path} must be a regular file; symlinks and other '
            'special files are not supported')


def _read_regular_contents(path: Path,
                           relative_path: Path,
                           *,
                           errors: str = 'strict') -> str:
    _ensure_regular_destination(path, relative_path)
    with path.open('r', encoding='utf-8', errors=errors, newline='') as source:
        return source.read()


def _is_executable(path: Path, relative_path: Path) -> bool:
    _ensure_regular_destination(path, relative_path)
    # Git records only 100755 versus 100644 and uses the owner-execute bit
    # when converting a filesystem mode to that binary distinction.
    return bool(path.stat().st_mode & stat.S_IXUSR)


def main() -> int:
    parser = argparse.ArgumentParser(
        description='Update or verify every pinned Go builder image.')
    operation = parser.add_mutually_exclusive_group(required=True)
    operation.add_argument('--version', help='Exact Go version X.Y.Z')
    operation.add_argument(
        '--check-image-digests',
        action='store_true',
        help='Verify that every pinned digest resolves from its declared tag',
    )
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parents[3]
    if args.check_image_digests:
        try:
            verify_image_digests(repo_root)
        except (RuntimeError, ValueError) as error:
            print(f'error: {error}', file=sys.stderr)
            return 2
        print('Go builder image digests match their declared tags.')
        return 0

    try:
        changed_paths = sync(repo_root, args.version)
    except (RuntimeError, ValueError) as error:
        print(f'error: {error}', file=sys.stderr)
        return 2

    if changed_paths:
        print('Updated Go version references:', flush=True)
        for path in changed_paths:
            print(f'  {path}', flush=True)
    else:
        print(
            f'Go version references are already current at {args.version}.',
            flush=True,
        )

    checker = repo_root / '.github/resources/scripts/go_version_consistency_test.py'
    return subprocess.run((sys.executable, str(checker)),
                          cwd=repo_root).returncode


if __name__ == '__main__':
    raise SystemExit(main())

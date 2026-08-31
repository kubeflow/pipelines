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

from go_version_metadata import (docker_runtime_classification,
                                 has_go_runtime_reference,
                                 is_container_recipe, module_versions)

DECIMAL_PATTERN = r'(?:0|[1-9][0-9]*)'
VERSION_PATTERN = re.compile(
    rf'^1\.({DECIMAL_PATTERN})(?:\.({DECIMAL_PATTERN}))?$')
EXACT_VERSION_PATTERN = re.compile(
    rf'^1\.{DECIMAL_PATTERN}\.{DECIMAL_PATTERN}$')
DIGEST_PATTERN = re.compile(r'^sha256:[0-9a-f]{64}$')
GO_DIRECTIVE_PATTERN = re.compile(
    rf'^[ \t]*go[ \t]+(?P<version>1\.{DECIMAL_PATTERN}'
    rf'(?:\.{DECIMAL_PATTERN})?)'
    r'(?P<comment>[ \t]*//[^\r\n]*)?[ \t]*$', re.MULTILINE)
TOOLCHAIN_PATTERN = re.compile(
    rf'^[ \t]*toolchain[ \t]+go(?P<version>1\.{DECIMAL_PATTERN}'
    rf'\.{DECIMAL_PATTERN})'
    r'(?P<comment>[ \t]*//[^\r\n]*)?[ \t]*$', re.MULTILINE)
TOOLCHAIN_LINE_PATTERN = re.compile(
    r'^[ \t]*toolchain[ \t]+go\d+\.\d+\.\d+'
    r'(?:[ \t]*//[^\r\n]*)?[ \t]*\n?(?:\n)?',
    re.MULTILINE)
SCANNED_RUNTIME_SUFFIXES = {'.sh', '.yaml', '.yml'}
DIGEST_LOOKUP_ATTEMPTS = 3
DIGEST_LOOKUP_BACKOFF_SECONDS = (1, 2)
DIGEST_LOOKUP_SOURCE_COUNT = 2
DIGEST_LOOKUP_TIMEOUT_SECONDS = 20
DIGEST_VERIFICATION_BUDGET_SECONDS = 540


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


def _module_versions(
    contents: str,
    relative_path: Path,
) -> Tuple[Tuple[int, int, int], Optional[Tuple[int, int, int]]]:
    go_version_text, toolchain_version_text = module_versions(
        contents, relative_path)
    go_version = _parse_version(go_version_text)
    toolchain_version = (_parse_version(toolchain_version_text)
                         if toolchain_version_text else None)
    return go_version, toolchain_version


def _updated_module_contents(contents: str, relative_path: Path,
                             target: Tuple[int, int, int]) -> str:
    go_version, _ = _module_versions(contents, relative_path)
    go_match = list(GO_DIRECTIVE_PATTERN.finditer(contents))[0]
    go_comment = go_match.group('comment') or ''
    toolchain_matches = list(TOOLCHAIN_PATTERN.finditer(contents))
    toolchain_match = toolchain_matches[0] if toolchain_matches else None
    toolchain_comment = (
        toolchain_match.group('comment') if toolchain_match else '') or ''
    if go_version > target:
        floor = '.'.join(str(part) for part in go_version)
        compiler = '.'.join(str(part) for part in target)
        raise ValueError(
            f'{relative_path} requires Go {floor}, which exceeds the target '
            f'compiler {compiler}')
    language_floor = (go_version if go_version[:2] == target[:2] else
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
            lambda match: (f'{match.group(0)}\n\n'
                           f'toolchain go{target_version}{toolchain_comment}'),
            updated,
            count=1,
        )
    return updated.rstrip('\n') + '\n'


def _managed_dockerfiles(repo_root: Path,
                         repository_paths: Iterable[Path]) \
        -> Dict[Path, Tuple[str, Dict]]:
    managed = {}
    unmanaged = []
    for relative_path in repository_paths:
        if not (is_container_recipe(relative_path) or
                relative_path.suffix in SCANNED_RUNTIME_SUFFIXES):
            continue
        path = repo_root / relative_path
        if not path.exists():
            continue
        contents = path.read_text(encoding='utf-8', errors='ignore')
        if not has_go_runtime_reference(relative_path, contents):
            continue
        _ensure_regular_destination(path, relative_path)
        contents = path.read_text(encoding='utf-8', errors='ignore')
        if is_container_recipe(relative_path):
            docker = docker_runtime_classification(contents)
        else:
            docker = {'classification': 'unsupported', 'candidates': []}
        candidates = docker['candidates']
        if (not is_container_recipe(relative_path) or
                docker['classification'] != 'managed' or
                len(candidates) != 1):
            unmanaged.append(
                f'{relative_path} ({docker["classification"]})')
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


def _inspect_manifest_digest(image: str) -> str:
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
        timeout=DIGEST_LOOKUP_TIMEOUT_SECONDS,
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


def resolve_docker_hub_digest(tag: str) -> str:
    images = (f'golang:{tag}', f'mirror.gcr.io/library/golang:{tag}')
    failures = {}
    for attempt in range(DIGEST_LOOKUP_ATTEMPTS):
        for image in images:
            try:
                return _inspect_manifest_digest(image)
            except FileNotFoundError as error:
                raise RuntimeError(
                    'docker buildx is required to resolve Go builder image '
                    'digests') from error
            except subprocess.CalledProcessError as error:
                detail = (error.stderr.strip() or error.stdout.strip() or
                          str(error))
                failures[image] = detail
            except subprocess.TimeoutExpired as error:
                failures[image] = f'timed out after {error.timeout} seconds'
        if attempt < DIGEST_LOOKUP_ATTEMPTS - 1:
            time.sleep(DIGEST_LOOKUP_BACKOFF_SECONDS[attempt])
    raise RuntimeError('could not resolve the Go builder image from Docker Hub '
                       'or its configured mirror after '
                       f'{DIGEST_LOOKUP_ATTEMPTS} attempts: ' + '; '.join(
                           f'{image}: {detail}'
                           for image, detail in failures.items()))


def verify_image_digests(
    repo_root: Path,
    digest_resolver: Callable[[str], str] = resolve_docker_hub_digest,
    repository_paths: Optional[Iterable[Path]] = None,
) -> None:
    paths = set(repository_paths or _repository_paths(repo_root))
    dockerfiles = _managed_dockerfiles(repo_root, paths)
    pins_by_tag: Dict[str, Dict[str, List[Path]]] = {}
    for relative_path, (_contents, candidate) in dockerfiles.items():
        tag = candidate['version'] + candidate['flavor']
        digest = candidate['digest']
        pins_by_tag.setdefault(tag, {}).setdefault(digest,
                                                   []).append(relative_path)

    worst_case_seconds = _digest_verification_worst_case_seconds(
        len(pins_by_tag))
    if worst_case_seconds > DIGEST_VERIFICATION_BUDGET_SECONDS:
        raise RuntimeError(
            'Go builder digest verification is configured for a worst-case '
            f'{worst_case_seconds} seconds, exceeding its '
            f'{DIGEST_VERIFICATION_BUDGET_SECONDS}-second budget')

    errors = []
    for tag, pins_by_digest in sorted(pins_by_tag.items()):
        if len(pins_by_digest) != 1:
            errors.append(
                f'golang:{tag} has inconsistent pinned digests: '
                f'{sorted(pins_by_digest)}')
            continue
        pinned_digest = next(iter(pins_by_digest))
        resolved_digest = digest_resolver(tag)
        if DIGEST_PATTERN.fullmatch(resolved_digest) is None:
            raise ValueError(f'invalid digest resolved for golang:{tag}: '
                             f'{resolved_digest!r}')
        if pinned_digest != resolved_digest:
            relative_paths = ', '.join(
                str(path) for path in sorted(pins_by_digest[pinned_digest]))
            errors.append(
                f'golang:{tag} resolves to {resolved_digest}, but '
                f'{relative_paths} pin {pinned_digest}')
    if errors:
        raise ValueError('Go builder image digest verification failed:\n  ' +
                         '\n  '.join(errors))


def _digest_verification_worst_case_seconds(tag_count: int) -> int:
    return tag_count * (
        DIGEST_LOOKUP_ATTEMPTS * DIGEST_LOOKUP_SOURCE_COUNT *
        DIGEST_LOOKUP_TIMEOUT_SECONDS +
        sum(DIGEST_LOOKUP_BACKOFF_SECONDS))


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
) -> Tuple[Dict[Path, str], Dict[Path, str]]:
    target = _parse_target_version(target_version)
    paths = set(repository_paths or _repository_paths(repo_root))
    module_paths = sorted(
        path for path in paths
        if path.name == 'go.mod' and (repo_root / path).exists())
    if Path('go.mod') not in module_paths:
        raise ValueError('the repository root go.mod was not found')

    original_contents = {
        relative_path: _read_regular_contents(repo_root / relative_path,
                                              relative_path)
        for relative_path in module_paths
    }
    root_contents = original_contents[Path('go.mod')]
    root_go_version, root_toolchain_version = _module_versions(
        root_contents, Path('go.mod'))
    current = root_toolchain_version or root_go_version
    if target < current:
        current_version = '.'.join(str(part) for part in current)
        raise ValueError(
            f'refusing to downgrade Go from {current_version} to '
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
    flavors = {
        candidate['flavor']
        for _contents, candidate in dockerfiles.values()
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
    initial_head = _git(repo_root, 'rev-parse', '--verify',
                        'HEAD').stdout.strip()
    expected_contents, original_contents = _synchronized_contents_and_originals(
        repo_root,
        target_version,
        digest_resolver=digest_resolver,
        repository_paths=repository_paths,
    )
    changed_paths = []
    for relative_path, expected in expected_contents.items():
        original = original_contents[relative_path]
        if original == expected:
            continue
        changed_paths.append(relative_path)

    if not changed_paths:
        return []

    start_head = _require_clean_managed_paths(repo_root,
                                              original_contents.keys())
    if start_head != initial_head:
        raise RuntimeError(
            f'HEAD changed during Go version update from {initial_head} '
            f'to {start_head}')
    expected_index_entries = _indexed_stage_zero_entries(
        repo_root, changed_paths)
    worktree_parent = tempfile.TemporaryDirectory(
        prefix='kfp-go-version-worktree-')
    worktree = Path(worktree_parent.name) / 'repository'
    recovery_bundle = None
    recovery_path = None
    path_patches = {}
    original_restore_patches = {}
    original_restore_paths = {}
    application_attempted = False
    try:
        _git(repo_root, 'worktree', 'add', '--detach', str(worktree),
             start_head)
        for relative_path in changed_paths:
            path = worktree / relative_path
            _ensure_regular_destination(path, relative_path)
            path.write_text(expected_contents[relative_path], encoding='utf-8')
        _verify_worktree_plan(worktree, expected_contents,
                              expected_index_entries, changed_paths)
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
                raise RuntimeError(
                    f'Git produced an empty recovery patch for '
                    f'{relative_path}')
            path_patches[relative_path] = path_patch
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
         )

        current_head = _require_clean_managed_paths(
            repo_root, original_contents.keys())
        if current_head != start_head:
            raise RuntimeError(
                f'HEAD changed during Go version update from {start_head} '
                f'to {current_head}')
        _git(repo_root, 'apply', '--check', '--whitespace=nowarn',
             str(recovery_path))
        application_attempted = True
        _git(repo_root, 'apply', '--whitespace=nowarn', str(recovery_path))
        _ensure_expected_contents(repo_root, expected_contents, changed_paths,
                                  expected_index_entries)
        _verify_repository_consistency(repo_root)
        _ensure_expected_contents(repo_root, expected_contents, changed_paths,
                                  expected_index_entries)
    except BaseException as update_error:
        rollback_errors = []
        unresolved_paths = []
        recovery_interrupt = None
        if application_attempted:
            (rollback_errors, unresolved_paths,
             recovery_interrupt) = _recover_applied_paths(
                repo_root,
                changed_paths,
                original_contents,
                expected_contents,
                expected_index_entries,
                path_patches,
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
    paths = tuple(str(path) for path in sorted(relative_paths))
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
    duplicates = sorted(path for path, count in path_counts.items()
                        if count > 1)
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


def _verify_worktree_plan(worktree: Path,
                          expected_contents: Dict[Path, str],
                          expected_index_entries: Dict[Path, Tuple[str, str]],
                          changed_paths: Iterable[Path]) -> None:
    changed_paths = tuple(changed_paths)
    worktree_index_entries = _indexed_stage_zero_entries(
        worktree, changed_paths)
    if worktree_index_entries != expected_index_entries:
        raise RuntimeError(
            'temporary worktree Git index entries do not match the update '
            'plan')
    _ensure_expected_contents(worktree, expected_contents, changed_paths,
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
        detail = getattr(error, 'stderr', '') or getattr(error, 'stdout', '') or str(
            error)
        raise RuntimeError(
            f'Go version consistency verification failed: '
            f'{detail.strip()}') from error


def _write_recovery_bundle(
    repo_root: Path,
    start_head: str,
    target_version: str,
    combined_patch: str,
    original_restore_patches: Dict[Path, str],
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
    bundle = pending.with_name(pending.name.removeprefix('.pending-') +
                               '.bundle')
    try:
        combined_name = 'combined-forward.patch'
        _write_durable_recovery_file(pending / combined_name, combined_patch)
        restore_names = {}
        manifest_paths = {}
        for index, relative_path in enumerate(sorted(original_restore_patches)):
            restore_name = f'{index:04d}-restore-original.patch'
            _write_durable_recovery_file(
                pending / restore_name,
                original_restore_patches[relative_path],
            )
            restore_names[relative_path] = restore_name
            manifest_paths[str(relative_path)] = restore_name
        manifest = {
            'startHead': start_head,
            'targetVersion': target_version,
            'combinedForwardPatch': combined_name,
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
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        with os.fdopen(descriptor, 'w', encoding='utf-8') as recovery_file:
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


def _contents_match(repo_root: Path, expected_contents: Dict[Path, str],
                    relative_paths: Iterable[Path],
                    expected_index_entries: Dict[Path, Tuple[str, str]]) -> bool:
    try:
        _ensure_expected_contents(repo_root, expected_contents, relative_paths,
                                  expected_index_entries)
        return True
    except Exception:
        return False


def _recover_applied_paths(
    repo_root: Path,
    relative_paths: Iterable[Path],
    original_contents: Dict[Path, str],
    expected_contents: Dict[Path, str],
    expected_index_entries: Dict[Path, Tuple[str, str]],
    path_patches: Dict[Path, str],
    original_restore_paths: Dict[Path, Path],
) -> Tuple[List[str], List[str], Optional[BaseException]]:
    errors = []
    unresolved = []
    interrupt = None
    for relative_path in relative_paths:
        if _contents_match(repo_root, original_contents, (relative_path,),
                           expected_index_entries):
            continue
        if not _contents_match(repo_root, expected_contents, (relative_path,),
                               expected_index_entries):
            unresolved.append(
                f'{relative_path} (original restore patch: '
                f'{original_restore_paths[relative_path]})')
            continue
        try:
            patch = path_patches[relative_path]
            _git_apply_contents(repo_root, patch, reverse=True, check=True)
            _git_apply_contents(repo_root, patch, reverse=True)
            _ensure_expected_contents(repo_root, original_contents,
                                      (relative_path,),
                                      expected_index_entries)
        except BaseException as error:
            if not isinstance(error, Exception) and interrupt is None:
                interrupt = error
            errors.append(
                f'{relative_path}: {error}; original restore patch retained '
                f'at {original_restore_paths[relative_path]}')
    return errors, unresolved, interrupt


def _git_apply_contents(repo_root: Path, patch: str, *, reverse: bool,
                        check: bool = False) -> None:
    arguments = ['git', 'apply']
    if reverse:
        arguments.append('--reverse')
    if check:
        arguments.append('--check')
    arguments.extend(('--whitespace=nowarn', '-'))
    try:
        subprocess.run(
            arguments,
            cwd=repo_root,
            check=True,
            capture_output=True,
            text=True,
            input=patch,
        )
    except (FileNotFoundError, subprocess.CalledProcessError) as error:
        detail = getattr(error, 'stderr', '') or str(error)
        raise RuntimeError(
            f'git apply recovery failed: {detail.strip()}') from error


def _ensure_expected_contents(repo_root: Path,
                              expected_contents: Dict[Path, str],
                              relative_paths: Iterable[Path],
                              expected_index_entries: Optional[Dict[
                                  Path, Tuple[str, str]]] = None) -> None:
    relative_paths = tuple(relative_paths)
    indexed_entries = (_indexed_stage_zero_entries(repo_root, relative_paths)
                       if expected_index_entries is not None else {})
    file_mode_enabled = (_core_file_mode_enabled(repo_root)
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


def _read_regular_contents(path: Path, relative_path: Path) -> str:
    _ensure_regular_destination(path, relative_path)
    return path.read_text(encoding='utf-8')


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
    return subprocess.run((sys.executable, str(checker)), cwd=repo_root).returncode


if __name__ == '__main__':
    raise SystemExit(main())

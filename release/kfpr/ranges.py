"""Resolve explicit release boundaries to immutable Git commits."""

import re
import subprocess
from dataclasses import asdict, dataclass
from pathlib import Path

from .notes import Commit


@dataclass(frozen=True)
class ReleaseRange:
    base_ref: str
    target_ref: str
    base_sha: str
    target_sha: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)

    @property
    def compare_url(self) -> str:
        return (
            'https://github.com/kubeflow/pipelines/compare/'
            f'{self.base_sha}...{self.target_sha}'
        )


def _git(root: Path, *args: str) -> str:
    try:
        result = subprocess.run(
            ['git', *args], cwd=root, check=True, capture_output=True, text=True
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        raise ValueError(
            'Could not read release history. Check the repository and fetch '
            'the required release refs and complete history.'
        ) from exc
    return result.stdout


def _commit(root: Path, ref: str) -> str:
    if not ref or not ref.strip() or ref.startswith('-') or '\x00' in ref:
        raise ValueError(
            'Release refs must be nonempty Git commit references; '
            'provide an explicit tag, branch, or commit SHA.'
        )
    try:
        return _git(
            root,
            'rev-parse',
            '--verify',
            '--end-of-options',
            f'{ref}^{{commit}}',
        ).strip()
    except ValueError as exc:
        raise ValueError(
            f'Cannot resolve release ref {ref!r} to a commit. '
            'Check the ref and fetch it before generating release notes.'
        ) from exc


def resolve_range(root: Path, base_ref: str, target_ref: str) -> ReleaseRange:
    """Resolve a base-exclusive, target-inclusive range on related
    histories."""
    base_sha = _commit(root, base_ref)
    target_sha = _commit(root, target_ref)
    try:
        common_ancestor = _git(root, 'merge-base', base_sha, target_sha).strip()
    except ValueError as exc:
        raise ValueError(
            f'Release base {base_ref!r} and target {target_ref!r} must share '
            'history. Choose related release refs and fetch complete history '
            'if the checkout is shallow.'
        ) from exc
    if base_sha != target_sha and common_ancestor == target_sha:
        raise ValueError(
            f'Release target {target_ref!r} precedes base {base_ref!r}. '
            'Choose the previous release as the base and the new release '
            'commit as the target.'
        )
    return ReleaseRange(base_ref, target_ref, base_sha, target_sha)


def _validate_sha(sha: str) -> None:
    if not re.fullmatch(r'(?:[0-9a-f]{40}|[0-9a-f]{64})', sha):
        raise ValueError(
            'Expected a resolved full commit SHA. Resolve the '
            'release range before reading release history.'
        )


def read_commits(root: Path, release_range: ReleaseRange) -> list[Commit]:
    """Read full messages from the resolved range, oldest first."""
    _validate_sha(release_range.base_sha)
    _validate_sha(release_range.target_sha)
    output = _git(
        root,
        'log',
        '--reverse',
        '-z',
        '--format=%H%x00%B',
        f'{release_range.base_sha}..{release_range.target_sha}',
        '--',
    )
    if not output:
        return []
    fields = output.split('\x00')
    if fields[-1] == '':
        fields.pop()
    if len(fields) % 2:
        raise ValueError(
            'Could not parse Git release history. Check commit '
            'messages for embedded NUL characters.'
        )
    return [
        Commit(sha=fields[i], message=fields[i + 1])
        for i in range(0, len(fields), 2)
    ]


def commit_date(root: Path, sha: str) -> str:
    """Return the commit's stable committer date for changelog headings."""
    _validate_sha(sha)
    return _git(root, 'show', '-s', '--format=%cs', sha, '--').strip()

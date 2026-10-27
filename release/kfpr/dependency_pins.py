"""Keep release version preparation separate from dependency upgrades."""

import re
import tomllib
from contextlib import contextmanager
from pathlib import Path

_RELEASE_PACKAGES = {
    'kfp',
    'kfp-pipeline-spec',
    'kfp-server-api',
    'kfp-kubernetes',
}
_REQUIREMENTS = (
    'requirements.txt',
    'api/v2alpha1/python/requirements.txt',
    'backend/api/v2beta1/python_http_client/requirements.txt',
    'sdk/python/requirements.txt',
    'kubernetes_platform/python/requirements.txt',
)


def _name(value: str) -> str:
    return re.sub(r'[-_.]+', '-', value).lower()


def _pins(path: Path) -> dict[str, set[str]]:
    pins: dict[str, set[str]] = {}
    if path.name == 'uv.lock':
        for package in tomllib.loads(path.read_text()).get('package', []):
            name = _name(package['name'])
            if name not in _RELEASE_PACKAGES:
                source = sorted(package.get('source', {}).items())
                pins.setdefault(name, set()).add(
                    f'{package.get("version", "")} {source}'
                )
        return pins
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith(('#', '--hash=')):
            continue
        match = re.fullmatch(
            r'([A-Za-z0-9_.-]+)(?:\[[^\]]+\])?==([^\s;\\]+)(?:\s*;[^\\]*)?\s*\\?',
            line,
        )
        if not match:
            raise ValueError(
                f'Cannot verify dependency pins in {path}: {line!r}. '
                'Regenerate a fully pinned requirements file before release preparation.'
            )
        name = _name(match[1])
        if name not in _RELEASE_PACKAGES:
            pins.setdefault(name, set()).add(match[2])
    return pins


@contextmanager
def preserve_dependency_pins(root: Path, *, dry_run: bool = False):
    """Restore locks and fail if preparation changes third-party dependencies.

    Resolvers prefer existing pins but may change them to satisfy new metadata.
    Such changes need their own review before release version preparation.
    """
    if dry_run:
        yield lambda: None
        return
    lock = root / 'uv.lock'
    paths = [root / relative for relative in _REQUIREMENTS]
    snapshots = {
        path: path.read_bytes() for path in [lock, *paths] if path.is_file()
    }
    checked = (
        [lock]
        if lock in snapshots
        else [
            path
            for path in snapshots
            if path.relative_to(root).as_posix()
            in {
                'api/v2alpha1/python/requirements.txt',
                'sdk/python/requirements.txt',
                'kubernetes_platform/python/requirements.txt',
            }
        ]
    )
    before = {path: _pins(path) for path in checked}

    def verify():
        changes = []
        for path, previous in before.items():
            current = _pins(path)
            for name in sorted(previous.keys() | current.keys()):
                old, new = previous.get(name, set()), current.get(name, set())
                if old != new:
                    changes.append(
                        f'{path.relative_to(root)}: {name}: '
                        f'{sorted(old)} -> {sorted(new)}'
                    )
        if changes:
            raise ValueError(
                'Release preparation changed third-party dependency pins:\n'
                + '\n'.join(changes)
                + '\nExisting lockfiles were restored. Make a separately reviewed '
                'dependency update, then rerun version preparation.'
            )

    try:
        yield verify
        verify()
    except BaseException:
        # A failed or interrupted resolver must not become the next run's baseline.
        for path, contents in snapshots.items():
            path.write_bytes(contents)
        raise

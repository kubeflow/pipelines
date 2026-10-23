"""Qualify the merged release source before any publication side effects."""

import hashlib
import json
import re
import subprocess
import sys
import tempfile
from datetime import datetime, timedelta, timezone
from pathlib import Path

import yaml

from .core import REPO, ReleaseContext
from .osv_image_scan import SCANNER
from .packages import clean_legacy_distributions, validate_legacy_wheels
from .security import evaluate_policy
from .security_exceptions import exception_digest, load_exceptions
from .smoke import supported_python_versions

MAX_AGE = timedelta(hours=24)


def _configured_exceptions(context: ReleaseContext) -> dict | None:
    path = context.state.answers.get('security_exceptions')
    policy = context.state.answers.get('security_policy', 'zero-fixable')
    if path:
        if policy != 'approved-exceptions':
            raise ValueError(
                'An exception file requires approved-exceptions policy'
            )
        return load_exceptions(Path(path))
    if policy == 'approved-exceptions':
        raise ValueError(
            'Approved-exceptions policy requires an exception file'
        )
    return None


def _identity(
    context: ReleaseContext, sha: str, exceptions: dict | None = None
) -> dict:
    mode = context.state.answers.get('image_qualification', 'ci')
    if mode != 'ci':
        raise ValueError(
            'Qualification scans require CI; select --image-qualification ci'
        )
    if context.state.answers.get('qualification_pythons'):
        raise ValueError(
            'CI selects Python versions from source; clear qualification_pythons'
        )
    if exceptions is None:
        exceptions = _configured_exceptions(context)
    identity = {
        'image_qualification': mode,
        'schema': 2,
        'security_scanner': SCANNER,
        'release_sha': sha,
        'version': context.metadata.tag,
        'include_backend': context.include_backend,
        'include_sdk': context.include_sdk,
        'security_policy': context.state.answers.get(
            'security_policy', 'zero-fixable'
        ),
    }
    if exceptions is not None:
        identity.update(
            exceptions=exceptions,
            exceptions_digest=exception_digest(exceptions),
        )
    return identity


def _run(command: list[str], root: Path) -> None:
    subprocess.run(command, cwd=root, check=True)


def build_wheels(root: Path, version: str) -> None:
    """Build from committed metadata without re-resolving release lockfiles."""
    clean_legacy_distributions(root)
    if (root / 'sdk/python/kfp/server_api/__init__.py').is_file():
        _run(['make', '-C', 'sdk', 'python'], root)
        return
    _run(['make', '-C', 'api', 'python'], root)
    _run(
        [
            'make',
            '-C',
            'backend/api',
            'API_VERSION=v2beta1',
            'generate-kfp-server-api-package',
        ],
        root,
    )
    for directory in (
        'api/v2alpha1/python',
        'backend/api/v2beta1/python_http_client',
        'sdk/python',
    ):
        _run([sys.executable, '-m', 'build', '.'], root / directory)
    _run(
        ['make', '-C', 'kubernetes_platform', 'python', 'USE_FIND_LINKS=true'],
        root,
    )
    _run(
        [sys.executable, '-m', 'build', '.'],
        root / 'kubernetes_platform/python',
    )
    validate_legacy_wheels(root, version)


def image_matrix(root: Path) -> list[dict]:
    """Read the release workflow's image/platform inventory, including
    exclusions."""
    workflow = yaml.safe_load(
        (root / '.github/workflows/image-builds-release.yml').read_text()
    )
    records = _matrix_records(workflow)
    for record in records:
        for field in ('dockerfile', 'context'):
            path = (root / record[field]).resolve()
            if not path.is_relative_to(root.resolve()) or not path.exists():
                raise ValueError(
                    f'Invalid release image {field}: {record[field]}'
                )
    return records


def _matrix_records(workflow: dict) -> list[dict]:
    matrix = workflow['jobs']['build-images-for-release']['strategy']['matrix']
    if 'include' in matrix or set(matrix) - {'image', 'arch', 'exclude'}:
        raise ValueError(
            'Unsupported release image matrix; update kfpr qualification'
        )
    records = []
    for image in matrix['image']:
        for arch in matrix['arch']:
            if any(
                all(
                    all(
                        (image if key == 'image' else arch).get(k) == v
                        for k, v in value.items()
                    )
                    for key, value in exclusion.items()
                )
                for exclusion in matrix.get('exclude', [])
            ):
                continue
            name, platform = image['name'], arch['platform']
            if not re.fullmatch(r'[a-z0-9-]+', name) or not re.fullmatch(
                r'linux/(amd64|arm64)', platform
            ):
                raise ValueError('Invalid release image name or platform')
            records.append({**image, 'platform': platform})
    if not records:
        raise ValueError('Release image inventory is empty')
    return records


def qualify_release(context: ReleaseContext) -> None:
    """Dispatch CI qualification of exact merged source and verify its
    evidence."""
    if context.runner.dry_run:
        print(
            '[dry-run] would qualify merged source: Python matrix, source dependencies and release images; no publication'
        )
        return
    from .steps import validate_release_pr

    sha = validate_release_pr(context)
    context.state.answers.pop('qualification_path', None)
    context.state.answers.pop('qualification_digest', None)
    context.state.save()
    exceptions = _configured_exceptions(context)
    output = Path(
        tempfile.mkdtemp(prefix=f'kfpr-{context.metadata.tag}-qualification-')
    )
    print(f'Qualification evidence: {output}')
    root = output / 'source'
    _run(
        [
            'git',
            'clone',
            '--no-checkout',
            '--no-hardlinks',
            str(context.root),
            str(root),
        ],
        output,
    )
    _run(['git', 'fetch', f'https://github.com/{REPO}.git', sha], root)
    _run(['git', 'checkout', '--detach', sha], root)
    if (
        context.include_backend
        and (root / 'VERSION').read_text().strip() != context.metadata.tag
    ):
        raise ValueError('Merged source VERSION does not match the release')
    report = _identity(context, sha, exceptions)
    from .ci_qualification_client import qualify_ci

    context.state.answers['release_sha'] = sha
    expected_images = image_matrix(root) if context.include_backend else []
    _, expected_minors = release_coverage(context, sha)
    result = qualify_ci(
        context, expected_images, expected_minors, output / 'ci'
    )
    report['images'] = result['images']
    report['wheels'] = result['wheels']
    report['security'] = result['security']
    report['security_ci'] = result['run']
    # Generation may create ignored files, but never change the approved source.
    if subprocess.check_output(
        ['git', 'diff', '--name-only', 'HEAD'], cwd=root, text=True
    ).strip():
        raise ValueError(
            'Qualification changed committed source; regenerate and review a new version PR'
        )
    if validate_release_pr(context) != sha:
        raise ValueError('Release source changed during qualification')
    if any(
        report.get(key) != value
        for key, value in _identity(context, sha).items()
    ):
        raise ValueError(
            'Exception approval changed during qualification; rerun qualify-release'
        )
    report['qualified_at'] = datetime.now(timezone.utc).isoformat()
    path = output / 'qualification.json'
    path.write_text(json.dumps(report, indent=2) + '\n')
    context.state.answers['qualification_path'] = str(path)
    context.state.answers['qualification_digest'] = hashlib.sha256(
        path.read_bytes()
    ).hexdigest()
    context.state.save()
    print(f'Qualification passed: {path}')


def release_coverage(context: ReleaseContext, sha: str) -> tuple[set, set]:
    """Read coverage requirements from the approved commit, not the
    checkout."""

    def read(path):
        return subprocess.check_output(
            ['git', 'show', f'{sha}:{path}'], cwd=context.root, text=True
        )

    images = set()
    if context.include_backend:
        records = _matrix_records(
            yaml.safe_load(read('.github/workflows/image-builds-release.yml'))
        )
        images = {(record['name'], record['platform']) for record in records}
    minors = set()
    if context.include_sdk:
        paths = subprocess.check_output(
            [
                'git',
                'ls-tree',
                '-r',
                '--name-only',
                sha,
                '--',
                'sdk/python/setup.py',
                'sdk/python/pyproject.toml',
            ],
            cwd=context.root,
            text=True,
        ).splitlines()
        minors = set(
            supported_python_versions('\n'.join(read(path) for path in paths))
        )
    return images, minors


def require_qualification(context: ReleaseContext) -> None:
    """Recheck live CI and reject missing, stale or differently scoped
    evidence."""
    if context.runner.dry_run:
        print(
            '[dry-run] would require fresh qualification for the merged release commit'
        )
        return
    from .steps import validate_release_pr

    sha = validate_release_pr(context)
    path = Path(context.state.answers.get('qualification_path', ''))
    if not path.is_file():
        raise ValueError('Run qualify-release before publication')
    payload = path.read_bytes()
    if hashlib.sha256(payload).hexdigest() != context.state.answers.get(
        'qualification_digest'
    ):
        raise ValueError(
            'Qualification evidence changed; rerun qualify-release'
        )
    report = json.loads(payload)
    exceptions = _configured_exceptions(context)
    if any(
        report.get(key) != value
        for key, value in _identity(context, sha, exceptions).items()
    ):
        raise ValueError(
            'Qualification belongs to a different commit/version/scope; rerun qualify-release'
        )
    security = report.get('security', {})
    digest = exception_digest(exceptions) if exceptions is not None else None
    if any(
        evidence.get('exceptions') != exceptions
        or evidence.get('exceptions_digest') != digest
        for evidence in (report, security)
    ):
        raise ValueError(
            'Exception evidence does not match the configured approval; rerun qualify-release'
        )
    policy = context.state.answers.get('security_policy', 'zero-fixable')
    if (
        security.get('passed') is not True
        or security.get('errors') != []
        or security.get('policy') != policy
        or security.get('release_version') != context.metadata.tag
        or security.get('scanner') != SCANNER
    ):
        raise ValueError(
            'Security qualification did not pass with the selected policy; rerun qualify-release'
        )
    evaluated = evaluate_policy(
        security.get('scans'),
        release_version=context.metadata.tag,
        policy=policy,
        exceptions=exceptions,
    )
    if evaluated['blocking_count'] or any(
        security.get(key) != value for key, value in evaluated.items()
    ):
        raise ValueError(
            'Security findings do not satisfy the selected policy; rerun qualify-release'
        )
    image_scans = [
        scan for scan in security['scans'] if scan['name'].startswith('image-')
    ]
    images = report.get('images', [])
    if context.include_backend and not images:
        raise ValueError(
            'Image qualification evidence is missing; rerun qualify-release'
        )
    if len(image_scans) != len(images) or any(
        scan.get('target') != record.get('image_id')
        or not re.fullmatch(r'sha256:[0-9a-f]{64}', record.get('image_id', ''))
        for scan, record in zip(image_scans, images)
    ):
        raise ValueError(
            'Image scan evidence does not match the built inventory; rerun qualify-release'
        )
    if (
        context.include_sdk
        and report.get('wheels', {}).get('passed') is not True
    ):
        raise ValueError(
            'Wheel qualification did not pass; rerun qualify-release'
        )
    scans = security['scans']
    if sum(scan['name'] == 'source' for scan in scans) != 1:
        raise ValueError(
            'Source scan coverage is incomplete; rerun qualify-release'
        )
    expected_images, expected_minors = release_coverage(context, sha)
    inventory = [
        (record.get('name'), record.get('platform')) for record in images
    ]
    if context.include_backend and (
        len(inventory) != len(set(inventory))
        or set(inventory) != expected_images
    ):
        raise ValueError(
            'Image scan coverage is incomplete; rerun qualify-release'
        )
    if context.include_sdk:
        environments = report.get('wheels', {}).get('python_environments', [])
        python_scans = [
            scan for scan in scans if scan['name'].startswith('python-')
        ]
        targets = [scan.get('target') for scan in python_scans]
        minors = {scan.get('python_version') for scan in python_scans}
        if (
            not environments
            or len(environments) != len(set(environments))
            or len(targets) != len(environments)
            or set(targets) != set(environments)
            or not expected_minors.issubset(minors)
        ):
            raise ValueError(
                'Python scan coverage is incomplete; rerun qualify-release'
            )
    from .ci_qualification_client import verify_run

    verify_run(context, report.get('security_ci', {}))
    age = datetime.now(timezone.utc) - datetime.fromisoformat(
        report['qualified_at']
    )
    if age < timedelta(0) or age > MAX_AGE:
        raise ValueError(
            'Qualification expired; rerun qualify-release to refresh security findings'
        )

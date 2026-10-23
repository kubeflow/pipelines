"""Validate full OSV JSON and derive package-scoped qualification findings."""

import re


def _object(value, label):
    if not isinstance(value, dict):
        raise ValueError(f'Invalid OSV {label}: expected an object')
    return value


def _list(value, label):
    if not isinstance(value, list):
        raise ValueError(f'Invalid OSV {label}: expected a list')
    return value


def _text(value, label):
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f'Invalid OSV {label}: expected nonempty text')
    return value


def _name(name, ecosystem):
    # PyPI distribution names normalize case and separators.
    return re.sub(r'[-_.]+', '-', name).lower() if ecosystem == 'PyPI' else name


def _packages(report, allow_unresolved=False):
    report = _object(report, 'report')
    if report.get('experimental_generic_findings'):
        raise ValueError('OSV generic findings require explicit policy support')
    results = _list(report.get('results'), 'results')
    if not results:
        raise ValueError('OSV reported no dependency targets; verify coverage')
    for result in results:
        result = _object(result, 'result')
        source = _object(result.get('source'), 'source')
        path = _text(source.get('path'), 'source path')
        _text(source.get('type'), 'source type')
        packages = _list(result.get('packages'), 'packages')
        if not packages:
            raise ValueError('OSV target has no packages; verify coverage')
        for item in packages:
            item = _object(item, 'package entry')
            package = _object(item.get('package'), 'package')
            name = _text(package.get('name'), 'package name')
            ecosystem = _text(package.get('ecosystem'), 'package ecosystem')
            version = package.get('version')
            if not (
                allow_unresolved
                and version == ''
                and source['type'] == 'lockfile'
            ):
                _text(version, 'installed version')
            elif _list(item.get('vulnerabilities', []), 'vulnerabilities'):
                raise ValueError(
                    'OSV unresolved dependency has vulnerability findings'
                )
            yield path, ecosystem, name, version, item


def inventory(
    report: dict, *, allow_unresolved=False
) -> list[tuple[str, str, str, str]]:
    """Return (source path, ecosystem, package name, installed version)."""
    return [
        (path, eco, name, version)
        for path, eco, name, version, _ in _packages(report, allow_unresolved)
    ]


def unresolved(report: dict) -> list[dict]:
    """Report unversioned source declarations; these are not audited
    packages."""
    return [
        {
            'target': path,
            'ecosystem': eco,
            'package': name,
            'installed_version': version,
        }
        for path, eco, name, version in inventory(report, allow_unresolved=True)
        if not version
    ]


def _fixed_versions(vulnerability, name, ecosystem):
    affected = _list(vulnerability.get('affected'), 'affected packages')
    matched = False
    fixed = set()
    for entry in affected:
        entry = _object(entry, 'affected entry')
        package = _object(entry.get('package'), 'affected package')
        affected_name = _text(package.get('name'), 'affected package name')
        affected_ecosystem = _text(
            package.get('ecosystem'), 'affected ecosystem'
        )
        if affected_ecosystem != ecosystem or _name(
            affected_name, ecosystem
        ) != _name(name, ecosystem):
            continue
        matched = True
        versions = _list(entry.get('versions', []), 'affected versions')
        for version in versions:
            _text(version, 'affected version')
        ranges = _list(entry.get('ranges', []), 'affected ranges')
        if not ranges and not versions:
            raise ValueError('OSV affected package has no ranges or versions')
        for affected_range in ranges:
            affected_range = _object(affected_range, 'range')
            if affected_range.get('type') not in ('SEMVER', 'ECOSYSTEM', 'GIT'):
                raise ValueError('Unsupported OSV affected range type')
            events = _list(affected_range.get('events'), 'range events')
            if not events:
                raise ValueError('OSV affected range has no events')
            introduced = False
            for event in events:
                event = _object(event, 'range event')
                keys = set(event)
                if len(keys) != 1 or not keys <= {
                    'introduced',
                    'fixed',
                    'last_affected',
                    'limit',
                }:
                    raise ValueError('Invalid OSV range event')
                kind = next(iter(keys))
                value = _text(event[kind], 'range event version')
                introduced |= kind == 'introduced'
                if kind == 'fixed':
                    fixed.add(value)
            if not introduced:
                raise ValueError('OSV range is missing its introduction event')
    if not matched:
        raise ValueError(
            'OSV advisory does not identify the scanned package and ecosystem'
        )
    return sorted(fixed)


def findings(report: dict, *, allow_unresolved=False) -> list[dict]:
    """Preserve all advisories; fixes belong only to the scanned package."""
    findings = []
    for path, ecosystem, name, version, item in _packages(
        report, allow_unresolved
    ):
        for vulnerability in _list(
            item.get('vulnerabilities', []), 'vulnerabilities'
        ):
            vulnerability = _object(vulnerability, 'vulnerability')
            identifier = _text(vulnerability.get('id'), 'advisory ID')
            aliases = _list(vulnerability.get('aliases', []), 'aliases')
            for alias in aliases:
                _text(alias, 'alias')
            fixes = _fixed_versions(vulnerability, name, ecosystem)
            findings.append(
                {
                    'target': path,
                    'id': identifier,
                    'aliases': sorted(set(aliases) - {identifier}),
                    'package': name,
                    'installed_version': version,
                    'fixed_versions': fixes,
                    'fixable': bool(fixes),
                }
            )
    return findings

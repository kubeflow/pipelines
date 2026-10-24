"""Qualify dependencies and images using explicit, bounded release policies."""

import json
import os
import subprocess
from collections.abc import Iterable
from datetime import datetime, timezone
from itertools import chain
from pathlib import Path

from .security_exceptions import exception_digest, validate_exceptions
from .security_policy_2173 import APPROVED_2173

# These distributions are built locally, so their unpublished release versions
# may not exist in PyPI. Their dependencies are still audited independently.
_LOCAL_PACKAGES = {
    'kfp',
    'kfp-kubernetes',
    'kfp-pipeline-spec',
    'kfp-server-api',
}


def _object(value, description):
    if not isinstance(value, dict):
        raise ValueError(f'Invalid {description}: expected an object')
    return value


def _list(value, description):
    if not isinstance(value, list):
        raise ValueError(f'Invalid {description}: expected a list')
    return value


def _text(value, description):
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f'Invalid {description}: expected nonempty text')
    return value


def _trivy_findings(report):
    report = _object(report, 'Trivy report')
    if report.get('SchemaVersion') != 2:
        raise ValueError('Unsupported Trivy schema; expected SchemaVersion 2')
    results = _list(report.get('Results'), 'Trivy Results')
    if not results:
        raise ValueError(
            'Trivy found no dependency targets; verify scanner coverage'
        )
    findings = []
    for result in results:
        result = _object(result, 'Trivy result')
        target = _text(result.get('Target'), 'Trivy Target')
        for vulnerability in _list(
            result.get('Vulnerabilities')
            if result.get('Vulnerabilities') is not None
            else [],
            'Trivy vulnerabilities',
        ):
            vulnerability = _object(vulnerability, 'Trivy vulnerability')
            fixed = vulnerability.get('FixedVersion', '')
            if not isinstance(fixed, str):
                raise ValueError('Invalid Trivy FixedVersion')
            findings.append(
                {
                    'target': target,
                    'id': _text(
                        vulnerability.get('VulnerabilityID'),
                        'Trivy vulnerability ID',
                    ),
                    'package': _text(
                        vulnerability.get('PkgName'), 'Trivy package'
                    ),
                    'installed_version': _text(
                        vulnerability.get('InstalledVersion'),
                        'Trivy installed version',
                    ),
                    'fixed_versions': [fixed] if fixed.strip() else [],
                    'fixable': bool(fixed.strip()),
                }
            )
    return findings


def _pip_findings(report):
    dependencies = _list(
        _object(report, 'pip-audit report').get('dependencies'),
        'pip-audit dependencies',
    )
    if not dependencies:
        raise ValueError('pip-audit found no installed dependencies')
    findings, skipped = [], []
    for dependency in dependencies:
        dependency = _object(dependency, 'pip-audit dependency')
        name = _text(dependency.get('name'), 'pip-audit name')
        if 'skip_reason' in dependency:
            reason = _text(dependency['skip_reason'], 'pip-audit skip reason')
            if name.lower().replace('_', '-') not in _LOCAL_PACKAGES:
                raise ValueError(
                    f'pip-audit skipped third-party dependency {name}: {reason}'
                )
            skipped.append({'package': name, 'reason': reason})
            continue
        version = _text(dependency.get('version'), 'pip-audit version')
        for vulnerability in _list(dependency.get('vulns'), 'pip-audit vulns'):
            vulnerability = _object(vulnerability, 'pip-audit vulnerability')
            fixes = _list(
                vulnerability.get('fix_versions'), 'pip-audit fix_versions'
            )
            for fix in fixes:
                _text(fix, 'pip-audit fixed version')
            findings.append(
                {
                    'id': _text(
                        vulnerability.get('id'), 'pip-audit vulnerability ID'
                    ),
                    'package': name,
                    'installed_version': version,
                    'fixed_versions': fixes,
                    'fixable': bool(fixes),
                }
            )
    return findings, skipped


def _scan(command, destination, allowed_codes=(0,)):
    # Do not inherit Trivy configuration that can hide findings through severity
    # or ignore filters. Explicit CLI policy applies to every qualification.
    environment = {
        key: value
        for key, value in os.environ.items()
        if not key.startswith('TRIVY_')
    }
    result = subprocess.run(
        command, capture_output=True, text=True, env=environment, check=False
    )
    destination.write_text(result.stdout)
    destination.with_suffix('.stderr.log').write_text(result.stderr)
    if result.returncode not in allowed_codes:
        raise RuntimeError(
            f'{command[0]} failed ({result.returncode}); see {destination.with_suffix(".stderr.log")}'
        )
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError as error:
        raise ValueError(f'Malformed scanner JSON in {destination}') from error


def evaluate_policy(
    scans,
    *,
    release_version: str = '',
    policy: str = 'zero-fixable',
    exceptions: dict | None = None,
    now: datetime | None = None,
) -> dict:
    """Recompute dispositions and revalidate all explicit exception expiry."""
    if policy not in (
        'zero-fixable',
        '2.17.3-approved-exceptions-v1',
        'approved-exceptions',
    ):
        raise ValueError(f'Unknown security policy: {policy}')
    approved = {}
    if policy == 'approved-exceptions':
        document = validate_exceptions(exceptions, now=now)
        approved = {
            (
                entry['scope'],
                entry['artifact'],
                entry['package'],
                entry['vulnerability'],
                entry['installed_version'],
            ): {'reason': entry['justification'], 'expires': entry['expires']}
            for entry in document['exceptions']
        }
    else:
        if exceptions is not None:
            raise ValueError(
                'An exception document requires the approved-exceptions policy'
            )
        if policy == '2.17.3-approved-exceptions-v1':
            if release_version != '2.17.3':
                raise ValueError(
                    'The approved exceptions apply only to release 2.17.3'
                )
            approved = {row[:5]: {'reason': row[5]} for row in APPROVED_2173}
    accepted, blocking = [], []
    residual_count = 0
    for scan in _list(scans, 'security scans'):
        scan = _object(scan, 'security scan')
        name = _text(scan.get('name'), 'security scan name')
        scope = (
            'source'
            if name == 'source'
            else 'python'
            if name.startswith('python-')
            else 'image'
        )
        for finding in _list(scan.get('findings'), 'security findings'):
            finding = _object(finding, 'security finding')
            fixes = _list(
                finding.get('fixed_versions'), 'security fixed versions'
            )
            for fix in fixes:
                _text(fix, 'security fixed version')
            if finding.get('fixable') is not bool(fixes):
                raise ValueError(
                    'Security fixable flag does not match fixed versions'
                )
            if not fixes:
                residual_count += 1
                continue
            target = (
                finding.get('target')
                if scope == 'source'
                else scan.get('python_version')
                if scope == 'python'
                else scan.get('target')
            )
            aliases = _list(finding.get('aliases', []), 'security aliases')
            for alias in aliases:
                _text(alias, 'security alias')
            identifiers = [
                _text(finding.get('id'), 'security vulnerability ID'),
                *aliases,
            ]
            approval = None
            if scope != 'image' or policy == 'approved-exceptions':
                for identifier in identifiers:
                    key = (
                        scope,
                        target,
                        finding.get('package'),
                        identifier,
                        finding.get('installed_version'),
                    )
                    if key in approved:
                        approval = approved[key]
                        break
            entry = {
                **finding,
                'scan': name,
                'scope': scope,
                'policy_target': target,
            }
            if approval:
                accepted.append({**entry, **approval})
            else:
                blocking.append(entry)
    return {
        'fixable_count': len(accepted) + len(blocking),
        'accepted_count': len(accepted),
        'blocking_count': len(blocking),
        'residual_count': residual_count,
        'accepted_findings': accepted,
        'blocking_findings': blocking,
    }


def qualify_security(
    root: Path,
    output_dir: Path,
    python_environments: list[Path],
    images: Iterable[str],
    *,
    release_version: str = '',
    policy: str = 'zero-fixable',
    image_reports: Iterable[tuple[str, Path]] = (),
    exceptions: dict | None = None,
) -> dict:
    """Save raw evidence and reject unaccepted fixable findings or incomplete
    scans.

    Python environments must already contain the installed release
    wheels. Images must identify locally built artifacts or immutable
    registry digests. Callers enforce which images and Python versions
    the release requires.
    """
    evaluate_policy(
        [],
        release_version=release_version,
        policy=policy,
        exceptions=exceptions,
    )
    root, output_dir = root.resolve(), output_dir.resolve()
    output_dir.mkdir(parents=True, exist_ok=True)
    report = {
        'policy': policy,
        'release_version': release_version,
        'created_at': datetime.now(timezone.utc).isoformat(),
        'scans': [],
        'errors': [],
        'passed': False,
    }
    if exceptions is not None:
        report['exceptions'] = validate_exceptions(exceptions)
        report['exceptions_digest'] = exception_digest(report['exceptions'])
    # Empty, explicit config files override repository configuration and ignores.
    config = output_dir / 'trivy-config.json'
    config.write_text('{}\n')
    ignore = output_dir / 'trivy-ignore.txt'
    ignore.write_text('')
    common = [
        '--config',
        str(config),
        '--ignorefile',
        str(ignore),
        '--scanners',
        'vuln',
        '--format',
        'json',
        '--list-all-pkgs',
        '--severity',
        'UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL',
        '--ignore-unfixed=false',
        '--exit-code',
        '0',
    ]
    scans = chain(
        [('source', ['trivy', 'fs', *common, str(root)])],
        (
            (
                f'image-{index}',
                [
                    'trivy',
                    'image',
                    '--detection-priority',
                    'comprehensive',
                    *common,
                    image,
                ],
            )
            for index, image in enumerate(images)
        ),
    )
    for name, command in scans:
        try:
            raw = output_dir / f'{name}.json'
            data = _scan(command, raw)
            findings = _trivy_findings(data)
            if name == 'source':
                targets = {
                    result['Target'].replace('\\', '/')
                    for result in data['Results']
                }
                for manifest in ('go.mod', 'frontend/package-lock.json'):
                    if (root / manifest).exists() and not any(
                        target == manifest or target.endswith('/' + manifest)
                        for target in targets
                    ):
                        raise ValueError(
                            f'Trivy omitted required dependency target {manifest}'
                        )
            report['scans'].append(
                {
                    'name': name,
                    'target': command[-1],
                    'raw_report': str(raw),
                    'findings': findings,
                }
            )
        except (OSError, ValueError, RuntimeError) as error:
            report['errors'].append(f'{name}: {error}')
    image_offset = sum(
        scan['name'].startswith('image-') for scan in report['scans']
    )
    for index, (image_id, source_report) in enumerate(
        image_reports, start=image_offset
    ):
        name = f'image-{index}'
        try:
            raw = output_dir / f'{name}.json'
            raw.write_bytes(source_report.read_bytes())
            data = json.loads(raw.read_text())
            if (
                not isinstance(data, dict)
                or data.get('ArtifactType') != 'container_image'
                or not isinstance(data.get('Metadata'), dict)
                or data['Metadata'].get('ImageID') != image_id
            ):
                raise ValueError(
                    'CI image report does not identify the expected immutable image'
                )
            report['scans'].append(
                {
                    'name': name,
                    'target': image_id,
                    'raw_report': str(raw),
                    'findings': _trivy_findings(data),
                }
            )
        except (OSError, ValueError, RuntimeError) as error:
            report['errors'].append(f'{name}: {error}')
    for index, environment in enumerate(python_environments):
        name = f'python-{index}'
        try:
            python = environment / 'bin' / 'python'
            paths = subprocess.run(
                [
                    str(python),
                    '-c',
                    'import json, sys, sysconfig; print(json.dumps({"version": "%s.%s" % sys.version_info[:2], "paths": sorted({sysconfig.get_path("purelib"), sysconfig.get_path("platlib")})}))',
                ],
                check=True,
                capture_output=True,
                text=True,
            )
            metadata = _object(json.loads(paths.stdout), 'Python metadata')
            python_version = _text(metadata.get('version'), 'Python version')
            paths = _list(metadata.get('paths'), 'Python site paths')
            if not paths:
                raise ValueError('No Python site paths were found')
            command = [
                'pip-audit',
                '--format',
                'json',
                '--progress-spinner',
                'off',
            ]
            for path in paths:
                command.extend(['--path', _text(path, 'Python site path')])
            raw = output_dir / f'{name}.json'
            findings, skipped = _pip_findings(_scan(command, raw, (0, 1)))
            report['scans'].append(
                {
                    'name': name,
                    'target': str(environment),
                    'python_version': python_version,
                    'raw_report': str(raw),
                    'findings': findings,
                    'skipped_local_packages': skipped,
                }
            )
        except (
            OSError,
            ValueError,
            RuntimeError,
            subprocess.CalledProcessError,
        ) as error:
            report['errors'].append(f'{name}: {error}')
    report.update(
        evaluate_policy(
            report['scans'],
            release_version=release_version,
            policy=policy,
            exceptions=exceptions,
        )
    )
    report['passed'] = not report['errors'] and report['blocking_count'] == 0
    summary = output_dir / 'security.json'
    summary.write_text(json.dumps(report, indent=2) + '\n')
    if not report['passed']:
        raise RuntimeError(
            f'Security qualification failed: {report["blocking_count"]} fixable blocking findings, {report["accepted_count"]} accepted findings, {len(report["errors"])} scanner errors; see {summary}'
        )
    return report

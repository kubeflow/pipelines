"""Collect complete source and installed-wheel OSV evidence in trusted CI."""

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import tempfile
from contextlib import contextmanager
from pathlib import Path, PurePosixPath

from . import ci_images_runner, osv_reports, qualification, smoke
from .osv_image_scan import SCANNER

_LOCAL = {'kfp', 'kfp-kubernetes', 'kfp-pipeline-spec', 'kfp-server-api'}
_INSTALLED = """import importlib.metadata, json, sys
print(json.dumps({'python_version': f'{sys.version_info.major}.{sys.version_info.minor}',
    'packages': [{'name': d.metadata['Name'], 'version': d.version}
                 for d in importlib.metadata.distributions()]}))
"""


def _digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _write(path: Path, value) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + '\n')


def _name(name: str) -> str:
    return re.sub(r'[-_.]+', '-', name).lower()


def _packages(packages: list) -> set[tuple[str, str]]:
    if not isinstance(packages, list) or not packages:
        raise ValueError('Installed package inventory is empty or malformed')
    result = set()
    names = set()
    for package in packages:
        if not isinstance(package, dict) or any(
            not isinstance(package.get(key), str) or not package[key].strip()
            for key in ('name', 'version')
        ):
            raise ValueError('Invalid installed distribution')
        name = _name(package['name'])
        if name in names:
            raise ValueError('Duplicate installed distribution: ' + name)
        names.add(name)
        result.add((name, package['version']))
    return result


def _source_coverage(report: dict, source_root: str) -> None:
    paths = {
        entry[0].replace('\\', '/')
        for entry in osv_reports.inventory(report, allow_unresolved=True)
        if entry[3]
    }
    for required in ('go.mod', 'frontend/package-lock.json'):
        if source_root.rstrip('/') + '/' + required not in paths:
            raise ValueError('Missing source dependency coverage: ' + required)


def _python_coverage(report: dict, packages: list) -> None:
    expected = _packages(packages)
    reported = set()
    for _, ecosystem, name, version in osv_reports.inventory(report):
        if ecosystem != 'PyPI':
            raise ValueError('Unexpected ecosystem in installed Python scan')
        reported.add((_name(name), version))
    if reported != expected:
        raise ValueError(
            'OSV installed Python inventory is incomplete or mismatched'
        )


def _source_identity(source: Path, identity: dict) -> None:
    actual = subprocess.check_output(
        ['git', 'rev-parse', 'HEAD'], cwd=source, text=True
    ).strip()
    if actual != identity['source_sha']:
        raise ValueError('Source checkout does not match the requested SHA')


def _scan(source: Path, output: Path, name: str, arguments: list[str]) -> dict:
    raw = output / 'scans' / (name + '.json')
    raw.parent.mkdir(parents=True, exist_ok=True)
    config = output / 'empty-osv.toml'
    config.write_text('')
    environment = {
        key: value
        for key, value in os.environ.items()
        if not key.startswith('OSV_')
    }
    command = [
        'osv-scanner',
        'scan',
        'source',
        '--config',
        str(config),
        '--all-packages',
        '--all-vulns',
        '--no-call-analysis=all',
        '--no-resolve',
        '--format',
        'json',
        '--output-file',
        str(raw),
        *arguments,
    ]
    raw.unlink(missing_ok=True)
    with raw.with_suffix('.stderr.log').open('w') as errors:
        result = subprocess.run(
            command, cwd=output, env=environment, stderr=errors, check=False
        )
    diagnostics = raw.with_suffix('.stderr.log').read_text()
    if any(
        marker in diagnostics
        for marker in (
            'Error during extraction:',
            'Failed to run code analysis',
        )
    ):
        raise ValueError(
            'OSV reported incomplete source extraction; inspect scanner diagnostics'
        )
    if result.returncode not in (0, 1):
        raise subprocess.CalledProcessError(result.returncode, command)
    if not raw.is_file() or raw.is_symlink():
        raise ValueError('OSV produced no regular JSON report')
    report = json.loads(raw.read_text())
    findings = osv_reports.findings(report, allow_unresolved=name == 'source')
    if bool(findings) != (result.returncode == 1):
        raise ValueError('OSV exit code does not match vulnerability findings')
    return {
        'name': name,
        'target': name,
        'python_version': None,
        'raw_report': raw.relative_to(output).as_posix(),
        'sha256': _digest(raw),
        'exit_code': result.returncode,
        **(
            {'unresolved_inventory': osv_reports.unresolved(report)}
            if name == 'source'
            else {}
        ),
    }


@contextmanager
def _smoke_workspace(output: Path):
    # Never upload executable environments or installer caches as evidence.
    with tempfile.TemporaryDirectory(
        prefix='kfpr-security-smoke-'
    ) as temporary:
        directory = Path(temporary)
        try:
            yield directory
        finally:
            for relative in ('qualification.json',):
                path = directory / relative
                if path.is_file():
                    destination = output / 'smoke' / relative
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(path, destination)
            for lane in directory.glob('python-*'):
                if not lane.is_dir():
                    continue
                for filename in (
                    'commands.log',
                    'requirements.txt',
                    'pipeline.yaml',
                ):
                    path = lane / filename
                    if path.is_file():
                        destination = output / 'smoke' / lane.name / filename
                        destination.parent.mkdir(parents=True, exist_ok=True)
                        shutil.copyfile(path, destination)


def collect(
    source: Path, output: Path, identity: dict, include_sdk: bool
) -> dict:
    """Build once, qualify every declared Python minor, and retain raw
    scans."""
    source, output = source.resolve(), output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    evidence = {
        **identity,
        'scanner': dict(SCANNER),
        'include_sdk': include_sdk,
        'passed': False,
        'source': None,
        'wheels': None,
        'scans': [],
    }
    try:
        _source_identity(source, identity)
        for relative in ('go.mod', 'frontend/package-lock.json'):
            if not (source / relative).is_file():
                raise ValueError(
                    'Required source dependency file is absent: ' + relative
                )
        source_scan = _scan(
            source,
            output,
            'source',
            ['--recursive', '--no-ignore', str(source)],
        )
        source_scan['source_root'] = str(source)
        evidence['scans'].append(source_scan)
        _source_coverage(
            json.loads((output / source_scan['raw_report']).read_text()),
            str(source),
        )
        evidence['source'] = {
            key: source_scan[key] for key in ('raw_report', 'sha256')
        }
        if include_sdk:
            with _smoke_workspace(output) as smoke_output:
                interpreters = smoke.default_interpreters(source)
                qualification.build_wheels(source, identity['version'])
                qualified = smoke.qualify_wheels(
                    source, identity['version'], interpreters, smoke_output
                )
                if qualified.get('passed') is not True:
                    raise ValueError('Wheel qualification did not pass')
                environments = qualified.get('environments', [])
                paths = qualified.get('python_environments', [])
                if len(environments) != len(interpreters) or len(paths) != len(
                    interpreters
                ):
                    raise ValueError(
                        'Incomplete wheel qualification environments'
                    )
                wheels = {
                    'version': identity['version'],
                    'passed': True,
                    'wheels': {},
                    'environments': [],
                    'python_environments': [],
                }
                evidence['wheels'] = wheels
                for filename, checksum in qualified['wheels'].items():
                    artifact = Path(filename)
                    if artifact.is_symlink() or _digest(artifact) != checksum:
                        raise ValueError('Qualified wheel checksum mismatch')
                    relative = 'wheels/files/' + artifact.name
                    if relative in wheels['wheels']:
                        raise ValueError('Duplicate wheel artifact')
                    destination = output / relative
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(artifact, destination)
                    wheels['wheels'][relative] = checksum
                for interpreter, path, qualified_environment in zip(
                    interpreters, paths, environments
                ):
                    minor = interpreter.removeprefix('python')
                    name = 'python-' + minor
                    installed = json.loads(
                        subprocess.check_output(
                            [
                                str(Path(path) / 'bin/python'),
                                '-I',
                                '-c',
                                _INSTALLED,
                            ],
                            cwd=output,
                            text=True,
                        )
                    )
                    if installed.get('python_version') != minor:
                        raise ValueError(
                            'Installed environment Python minor mismatch'
                        )
                    packages = installed.get('packages')
                    _packages(packages)
                    distributions = qualified_environment.get('distributions')
                    if qualified_environment.get(
                        'passed'
                    ) is not True or not isinstance(distributions, dict):
                        raise ValueError('Installed wheel smoke did not pass')
                    if any(
                        (_name(key), value) not in _packages(packages)
                        for key, value in distributions.items()
                    ):
                        raise ValueError(
                            'Qualified distribution is missing from installed environment'
                        )
                    input_path = output / 'inventories' / (name + '.json')
                    _write(
                        input_path,
                        {
                            'results': [
                                {
                                    'packages': [
                                        {
                                            'package': {
                                                **package,
                                                'ecosystem': 'PyPI',
                                            }
                                        }
                                        for package in packages
                                    ]
                                }
                            ]
                        },
                    )
                    scan = _scan(
                        source,
                        output,
                        name,
                        ['--lockfile', 'osv-scanner:' + str(input_path)],
                    )
                    scan.update(
                        python_version=minor,
                        installed_inventory=packages,
                        excluded=[],
                    )
                    evidence['scans'].append(scan)
                    _python_coverage(
                        json.loads((output / scan['raw_report']).read_text()),
                        packages,
                    )
                    wheels['python_environments'].append(name)
                    wheels['environments'].append(
                        {
                            'python_version': minor,
                            'version': identity['version'],
                            'distributions': distributions,
                            'passed': True,
                        }
                    )
        changed = subprocess.check_output(
            ['git', 'diff', '--name-only', 'HEAD'], cwd=source, text=True
        ).strip()
        if changed:
            raise ValueError(
                'Qualification changed tracked source; prepare a new version PR: '
                + changed
            )
        evidence['passed'] = True
        _write(output / 'security.json', evidence)
        verify_security(
            output,
            identity,
            [
                name.removeprefix('python')
                for name in smoke.default_interpreters(source)
            ]
            if include_sdk
            else [],
        )
        return evidence
    except Exception as error:
        evidence['passed'] = False
        evidence['error'] = str(error)
        raise
    finally:
        _write(output / 'security.json', evidence)


def _verified_file(directory: Path, relative, checksum) -> Path:
    if not isinstance(relative, str) or not relative or '\\' in relative:
        raise ValueError('Invalid evidence path')
    path = PurePosixPath(relative)
    if path.is_absolute() or '..' in path.parts or path.as_posix() != relative:
        raise ValueError('Evidence path must be canonical and relative')
    result = directory
    for part in path.parts:
        result = result / part
        if result.is_symlink():
            raise ValueError('Symlink evidence is not allowed')
    if not result.is_file() or _digest(result) != checksum:
        raise ValueError('Evidence checksum mismatch: ' + relative)
    return result


def verify_security(
    directory: Path, identity: dict, expected_minors: list[str]
) -> dict:
    """Verify downloaded evidence without depending on CI filesystem paths."""
    manifest = directory / 'security.json'
    if manifest.is_symlink():
        raise ValueError('Symlink security envelope is not allowed')
    report = json.loads(manifest.read_text())
    if not isinstance(report, dict) or any(
        report.get(key) != value for key, value in identity.items()
    ):
        raise ValueError('Security evidence identity mismatch')
    if report.get('scanner') != SCANNER or report.get('passed') is not True:
        raise ValueError(
            'Security evidence is incomplete or has the wrong scanner'
        )
    if type(report.get('include_sdk')) is not bool or report[
        'include_sdk'
    ] != bool(expected_minors):
        raise ValueError('Security SDK coverage does not match the request')
    if len(expected_minors) != len(set(expected_minors)) or any(
        not re.fullmatch(r'3\.\d+', minor) for minor in expected_minors
    ):
        raise ValueError('Invalid expected Python minors')
    expected = {'source', *('python-' + minor for minor in expected_minors)}
    scans = report.get('scans')
    if not isinstance(scans, list):
        raise ValueError('Missing security scans')
    normalized, seen = [], set()
    for scan in scans:
        if not isinstance(scan, dict):
            raise ValueError('Invalid security scan')
        name = scan.get('name')
        if name not in expected or name in seen or scan.get('target') != name:
            raise ValueError('Unexpected or duplicate security scan')
        seen.add(name)
        if scan.get('raw_report') != 'scans/' + name + '.json':
            raise ValueError('Security scan path does not match its target')
        raw = _verified_file(directory, scan['raw_report'], scan.get('sha256'))
        data = json.loads(raw.read_text())
        findings = osv_reports.findings(data, allow_unresolved=name == 'source')
        code = scan.get('exit_code')
        if (
            type(code) is not int
            or code not in (0, 1)
            or bool(findings) != (code == 1)
        ):
            raise ValueError('Security scan did not complete consistently')
        if name == 'source':
            if scan.get('unresolved_inventory') != osv_reports.unresolved(data):
                raise ValueError(
                    'Unresolved source inventory does not match the raw report'
                )
            source_root = scan.get('source_root')
            if not isinstance(source_root, str) or not source_root.startswith(
                '/'
            ):
                raise ValueError('Invalid source root identity')
            _source_coverage(data, source_root)
            for finding in findings:
                target = finding['target']
                prefix = source_root.rstrip('/') + '/'
                if target.startswith(prefix):
                    finding['target'] = target[len(prefix) :]
            if scan.get('python_version') is not None or report.get(
                'source'
            ) != {key: scan[key] for key in ('raw_report', 'sha256')}:
                raise ValueError('Source scan evidence mismatch')
        else:
            if (
                scan.get('python_version') != name.removeprefix('python-')
                or scan.get('excluded') != []
            ):
                raise ValueError(
                    'Invalid Python coverage or package exclusions'
                )
            _python_coverage(data, scan.get('installed_inventory'))
        normalized.append({**scan, 'findings': findings})
    if seen != expected:
        raise ValueError('Missing required security scan coverage')
    wheels = report.get('wheels')
    if expected_minors:
        if (
            not isinstance(wheels, dict)
            or wheels.get('passed') is not True
            or wheels.get('version') != identity['version']
        ):
            raise ValueError('Invalid wheel qualification report')
        wheel_files = wheels.get('wheels')
        if not isinstance(wheel_files, dict) or not wheel_files:
            raise ValueError('Missing qualified wheel artifacts')
        for filename, checksum in wheel_files.items():
            if (
                not filename.startswith('wheels/files/')
                or len(PurePosixPath(filename).parts) != 3
                or not filename.endswith('.whl')
            ):
                raise ValueError('Invalid qualified wheel path')
            _verified_file(directory, filename, checksum)
        targets = wheels.get('python_environments')
        if (
            not isinstance(targets, list)
            or len(targets) != len(expected_minors)
            or set(targets) != expected - {'source'}
        ):
            raise ValueError(
                'Missing or duplicate qualified Python environment'
            )
        environments = wheels.get('environments')
        if not isinstance(environments, list) or len(environments) != len(
            expected_minors
        ):
            raise ValueError('Incomplete wheel smoke evidence')
        seen_minors = set()
        for environment in environments:
            if not isinstance(environment, dict):
                raise ValueError('Invalid wheel smoke environment')
            minor = environment.get('python_version')
            if (
                minor not in expected_minors
                or minor in seen_minors
                or environment.get('passed') is not True
                or environment.get('version') != identity['version']
            ):
                raise ValueError('Invalid or duplicate wheel smoke environment')
            seen_minors.add(minor)
            distributions = environment.get('distributions')
            if (
                not isinstance(distributions, dict)
                or set(distributions) not in ({'kfp'}, _LOCAL)
                or any(
                    version != identity['version']
                    for version in distributions.values()
                )
            ):
                raise ValueError('Unverified local release distributions')
            packages = _packages(
                next(
                    scan['installed_inventory']
                    for scan in scans
                    if scan['name'] == 'python-' + minor
                )
            )
            if any(
                (_name(key), value) not in packages
                for key, value in distributions.items()
            ):
                raise ValueError(
                    'Qualified local distribution missing from scan'
                )
            for name in distributions:
                if (
                    sum(
                        Path(path).name.startswith(
                            name.replace('-', '_')
                            + '-'
                            + identity['version']
                            + '-'
                        )
                        for path in wheel_files
                    )
                    != 1
                ):
                    raise ValueError('Missing qualified release wheel')
        if len(wheel_files) != len(environments[0]['distributions']):
            raise ValueError('Unexpected qualified wheel artifact')
    elif wheels is not None:
        raise ValueError('Unexpected SDK evidence')
    return {'scans': normalized, 'wheels': wheels}


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--include-sdk', action='store_true')
    args = parser.parse_args()
    identity = ci_images_runner.identity()
    tool_sha = subprocess.check_output(
        ['git', 'rev-parse', 'HEAD'],
        cwd=Path(__file__).resolve().parents[2],
        text=True,
    ).strip()
    if tool_sha != identity['workflow_sha']:
        raise ValueError('Tool checkout does not match the workflow SHA')
    collect(args.source, args.output, identity, args.include_sdk)


if __name__ == '__main__':
    main()

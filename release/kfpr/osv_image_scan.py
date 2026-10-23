"""Scan one verified immutable Docker archive with pinned OSV-Scanner."""

import hashlib
import json
import os
import re
import subprocess
import tarfile
import tempfile
from pathlib import Path, PurePosixPath

SCANNER = {'name': 'osv-scanner', 'version': '2.5.0'}


def file_sha256(path: Path) -> str:
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def verify_archive(path: Path, image_id: str, platform: str) -> str:
    """Require one Docker image whose config identifies the built image."""
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', image_id):
        raise ValueError('Invalid immutable image ID')
    if platform not in ('linux/amd64', 'linux/arm64'):
        raise ValueError(
            'OSV archive requires a supported native Linux platform'
        )
    with tarfile.open(path, 'r:*') as archive:
        members = archive.getmembers()

        def read_member(name, limit):
            matches = [member for member in members if member.name == name]
            if (
                len(matches) != 1
                or not matches[0].isfile()
                or matches[0].size > limit
            ):
                raise ValueError('Docker archive requires one regular ' + name)
            with archive.extractfile(matches[0]) as stream:
                return stream.read()

        manifest = json.loads(read_member('manifest.json', 1024 * 1024))
        if (
            not isinstance(manifest, list)
            or len(manifest) != 1
            or not isinstance(manifest[0], dict)
        ):
            raise ValueError('Docker archive must contain exactly one image')
        config = manifest[0].get('Config')
        if (
            not isinstance(config, str)
            or not config
            or PurePosixPath(config).is_absolute()
            or '..' in PurePosixPath(config).parts
        ):
            raise ValueError('Docker archive has an invalid config path')
        config_bytes = read_member(config, 16 * 1024 * 1024)
        if 'sha256:' + hashlib.sha256(config_bytes).hexdigest() != image_id:
            raise ValueError(
                'Docker archive config does not match built image ID'
            )
        metadata = json.loads(config_bytes)
        if not isinstance(metadata, dict):
            raise ValueError('Docker archive config must be an object')
        if (metadata.get('os'), metadata.get('architecture')) != tuple(
            platform.split('/')
        ):
            raise ValueError(
                'Docker archive platform does not match the requested lane'
            )
    return file_sha256(path)


def validate_sarif(sarif: dict) -> list[dict]:
    """Validate complete OSV SARIF without filtering findings or severities."""
    if not isinstance(sarif, dict) or sarif.get('version') != '2.1.0':
        raise ValueError('OSV report must use SARIF 2.1.0')
    runs = sarif.get('runs')
    if (
        not isinstance(runs, list)
        or len(runs) != 1
        or not isinstance(runs[0], dict)
    ):
        raise ValueError('OSV report must contain exactly one SARIF run')
    run = runs[0]
    tool = run.get('tool')
    driver = tool.get('driver') if isinstance(tool, dict) else None
    if not isinstance(driver, dict) or any(
        driver.get(k) != v for k, v in SCANNER.items()
    ):
        raise ValueError('SARIF must identify OSV-Scanner 2.5.0')
    rules = driver.get('rules', [])
    results = run.get('results', [])
    if not isinstance(rules, list) or not isinstance(results, list):
        raise ValueError('OSV SARIF rules and results must be lists')
    ids = []
    for rule in rules:
        if (
            not isinstance(rule, dict)
            or not isinstance(rule.get('id'), str)
            or not rule['id']
        ):
            raise ValueError('OSV SARIF rule must identify an advisory')
        ids.append(rule['id'])
    if len(set(ids)) != len(ids):
        raise ValueError('OSV SARIF rules must have distinct IDs')
    for result in results:
        if not isinstance(result, dict) or result.get('ruleId') not in ids:
            raise ValueError('OSV SARIF result must reference a reported rule')
        index = result.get('ruleIndex')
        if index is not None and (
            type(index) is not int
            or not 0 <= index < len(ids)
            or ids[index] != result['ruleId']
        ):
            raise ValueError('OSV SARIF result has an invalid rule index')
        message = result.get('message')
        if (
            not isinstance(message, dict)
            or not isinstance(message.get('text'), str)
            or not message['text']
        ):
            raise ValueError('OSV SARIF result requires a text message')
        if 'locations' in result and not isinstance(result['locations'], list):
            raise ValueError('OSV SARIF result locations must be a list')
    return results


def validate_report(
    report: dict, image_id: str, report_format: str = 'sarif'
) -> None:
    if report_format not in ('sarif', 'json'):
        raise ValueError('Unsupported OSV report format')
    if (
        not isinstance(report, dict)
        or report.get('schema') != 1
        or report.get('scanner') != SCANNER
    ):
        raise ValueError('Invalid OSV image scan envelope or scanner identity')
    if report.get('image_id') != image_id:
        raise ValueError('OSV scan subject does not match the built image ID')
    if not isinstance(report.get('archive_sha256'), str) or not re.fullmatch(
        r'[0-9a-f]{64}', report['archive_sha256']
    ):
        raise ValueError('OSV scan requires a verified archive checksum')
    code = report.get('exit_code')
    if type(code) is not int or code not in (0, 1):
        raise ValueError('OSV scan did not complete successfully')
    actual_format = report.get('report_format', 'sarif')
    if actual_format != report_format:
        raise ValueError(
            'OSV envelope report format does not match requested format'
        )
    if report_format == 'json':
        from .osv_reports import findings

        results = findings(report.get('report'))
    else:
        results = validate_sarif(report.get('sarif'))
    if bool(results) != (code == 1):
        raise ValueError(
            'OSV exit code does not match reported vulnerability findings'
        )


def scan_image(
    image_id: str, platform: str, output: Path, report_format: str = 'sarif'
) -> None:
    """Bind an OSV report to one verified Docker archive and image ID."""
    if report_format not in ('sarif', 'json'):
        raise ValueError('Unsupported OSV report format')
    output = output.resolve()
    environment = {
        key: value
        for key, value in os.environ.items()
        if not key.startswith('OSV_')
    }
    with tempfile.TemporaryDirectory(prefix='kfpr-osv-') as temporary:
        directory = Path(temporary)
        archive = directory / 'image.tar'
        subprocess.run(
            ['docker', 'save', '--output', str(archive), image_id], check=True
        )
        archive_sha256 = verify_archive(archive, image_id, platform)
        config = directory / 'empty.toml'
        config.write_text('')
        report_path = output / (
            'scan.sarif' if report_format == 'sarif' else 'osv-report.json'
        )
        report_path.unlink(missing_ok=True)
        with (output / 'scan.stderr.log').open('w') as errors:
            result = subprocess.run(
                [
                    'osv-scanner',
                    'scan',
                    'image',
                    '--archive',
                    '--config',
                    str(config),
                    '--all-vulns',
                    *(['--all-packages'] if report_format == 'json' else []),
                    '--format',
                    report_format,
                    '--output-file',
                    str(report_path),
                    str(archive),
                ],
                cwd=directory,
                env=environment,
                stderr=errors,
                check=False,
            )
        if result.returncode not in (0, 1):
            raise subprocess.CalledProcessError(result.returncode, result.args)
        if (
            not report_path.is_file()
            or report_path.is_symlink()
            or not report_path.stat().st_size
        ):
            raise ValueError(
                'OSV scan produced no ' + report_format.upper() + ' report'
            )
        report = {
            'schema': 1,
            'scanner': SCANNER,
            'image_id': image_id,
            'archive_sha256': archive_sha256,
            'exit_code': result.returncode,
            'report_format': report_format,
            ('sarif' if report_format == 'sarif' else 'report'): json.loads(
                report_path.read_text()
            ),
        }
        validate_report(report, image_id, report_format)
        (output / 'scan.json').write_text(json.dumps(report, indent=2) + '\n')

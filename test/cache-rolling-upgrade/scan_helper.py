#!/usr/bin/env python3
"""Retain exact cache helper package inventory and OSV scanner evidence."""
import argparse
import json
from pathlib import Path
import re
import subprocess


def packages(database: str) -> list[dict[str, str]]:
    """Parse installed APK records and require the helper runtime packages."""
    result = []
    for stanza in database.strip().split('\n\n'):
        fields = dict(
            line.split(':', 1) for line in stanza.splitlines() if ':' in line)
        if 'P' not in fields or 'V' not in fields:
            raise ValueError(
                'Incomplete APK record; inspect image package metadata')
        result.append({'name': fields['P'], 'version': fields['V']})
    if not {'busybox', 'musl'} <= {p['name'] for p in result}:
        raise ValueError(
            'Missing runtime packages; inspect the helper image before accepting coverage'
        )
    return result


def load_cache_image() -> str:
    """Read the deployed helper reference from API configuration."""
    return json.loads(
        Path('backend/src/apiserver/config/config.json').read_text()
    )['CACHE_IMAGE']


def read_image_file(image_id: str, path: str) -> str:
    """Read package metadata from the inspected image without network access."""
    return subprocess.check_output([
        'docker', 'run', '--rm', '--network=none', '--entrypoint=cat', image_id,
        path
    ],
                                   text=True)


def alpine_ecosystem(release: str) -> str:
    """Derive the OSV ecosystem from the image's Alpine release version."""
    match = re.fullmatch(r'(\d+)\.(\d+)\.\d+(?:_\w+)?', release.strip())
    if not match:
        raise ValueError(
            f'Invalid Alpine release {release!r}; inspect /etc/alpine-release')
    return f'Alpine:v{match[1]}.{match[2]}'


def osv_alpine_package_names(report: dict, ecosystem: str) -> set[str]:
    """Require scanner coverage of BusyBox and musl in the image ecosystem."""
    names = {
        item['package']['name']
        for result in report.get('results', [])
        for item in result.get('packages', [])
        if item['package'].get('ecosystem') == ecosystem
    }
    if not {'busybox', 'musl'} <= names:
        raise ValueError(
            f'OSV did not identify BusyBox and musl for {ecosystem}; inspect scanner extraction'
        )
    return names


def smoke_test_helper(image_id: str) -> None:
    """Exercise the cache-hit command with the workload's restricted identity."""
    expected = 'This step output is taken from cache.'
    output = subprocess.check_output([
        'docker', 'run', '--rm', '--network=none', '--user=1000:1000',
        '--read-only', '--cap-drop=ALL', image_id, 'echo', expected
    ],
                                     text=True)
    if output.strip() != expected:
        raise RuntimeError(
            f'Unexpected helper output: {output!r}; inspect the image entrypoint'
        )


def run_osv_scan(archive: Path, output: Path) -> dict:
    """Retain complete scanner evidence and reject extraction or execution errors."""
    config = output / 'empty.toml'
    config.write_text('')
    for report_format in ('json', 'sarif'):
        result = subprocess.run([
            './osv-scanner', 'scan', 'image', '--archive', '--config',
            str(config), '--all-vulns',
            *(['--all-packages'] if report_format == 'json' else []),
            '--format', report_format, '--output-file',
            str(output / ('osv.' + report_format)),
            str(archive)
        ],
                                check=False,
                                capture_output=True,
                                text=True)
        (output / (report_format + '.stderr.log')).write_text(result.stderr)
        if 'Error during extraction:' in result.stderr:
            raise ValueError(
                'Incomplete package extraction; inspect scanner diagnostics')
        if result.returncode not in (0, 1):
            raise RuntimeError(
                'OSV image scan failed; retry before accepting package coverage'
            )
    return json.loads((output / 'osv.json').read_text())


def main() -> None:
    """Scan the inspected helper image and retain its package coverage evidence."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    image = load_cache_image()
    inspected = json.loads(
        subprocess.check_output(['docker', 'image', 'inspect', image],
                                text=True))[0]
    image_id = inspected['Id']
    database = read_image_file(image_id, '/lib/apk/db/installed')
    inventory = packages(database)
    ecosystem = alpine_ecosystem(
        read_image_file(image_id, '/etc/alpine-release'))
    subject = {
        'image': image,
        'image_id': image_id,
        'repo_digests': inspected['RepoDigests'],
        'platform': inspected['Os'] + '/' + inspected['Architecture'],
        'ecosystem': ecosystem,
        'packages': inventory
    }
    (args.output /
     'subject.json').write_text(json.dumps(subject, indent=2) + '\n')
    (args.output / 'apk-installed.txt').write_text(database)
    smoke_test_helper(image_id)
    archive = args.output / 'helper.tar'
    subprocess.run(['docker', 'save', '--output',
                    str(archive), image_id],
                   check=True)
    report = run_osv_scan(archive, args.output)
    osv_alpine_package_names(report, ecosystem)
    scanned = [
        item for result in report.get('results', [])
        for item in result.get('packages', [])
    ]
    summary = {
        'apk_package_count':
            len(inventory),
        'scanned_package_count':
            len(scanned),
        'advisory_occurrences':
            sum(len(item.get('vulnerabilities', [])) for item in scanned)
    }
    (args.output /
     'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    archive.unlink()
    # Retain installed metadata as well as findings: zero findings do not prove coverage.
    print(json.dumps(subject, indent=2))


if __name__ == '__main__':
    main()

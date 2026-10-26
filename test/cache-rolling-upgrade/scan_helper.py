#!/usr/bin/env python3
"""Retain exact cache helper package inventory and OSV scanner evidence."""
import argparse
import json
from pathlib import Path
import subprocess


def packages(database):
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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    image = json.loads(
        Path('backend/src/apiserver/config/config.json').read_text()
    )['CACHE_IMAGE']
    inspected = json.loads(
        subprocess.check_output(['docker', 'image', 'inspect', image],
                                text=True))[0]
    image_id = inspected['Id']
    database = subprocess.check_output([
        'docker', 'run', '--rm', '--network=none', '--entrypoint=cat', image_id,
        '/lib/apk/db/installed'
    ],
                                       text=True)
    inventory = packages(database)
    subject = {
        'image': image,
        'image_id': image_id,
        'repo_digests': inspected['RepoDigests'],
        'platform': inspected['Os'] + '/' + inspected['Architecture'],
        'packages': inventory
    }
    (args.output /
     'subject.json').write_text(json.dumps(subject, indent=2) + '\n')
    (args.output / 'apk-installed.txt').write_text(database)
    output = subprocess.check_output([
        'docker', 'run', '--rm', '--network=none', '--user=1000:1000',
        '--read-only', '--cap-drop=ALL', image_id, 'echo',
        'This step output is taken from cache.'
    ],
                                     text=True)
    assert output.strip() == 'This step output is taken from cache.'
    archive = args.output / 'helper.tar'
    subprocess.run(['docker', 'save', '--output',
                    str(archive), image_id],
                   check=True)
    config = args.output / 'empty.toml'
    config.write_text('')
    for report_format in ('json', 'sarif'):
        result = subprocess.run([
            './osv-scanner', 'scan', 'image', '--archive', '--config',
            str(config), '--all-vulns',
            *(['--all-packages'] if report_format == 'json' else []),
            '--format', report_format, '--output-file',
            str(args.output / ('osv.' + report_format)),
            str(archive)
        ],
                                check=False)
        if result.returncode not in (0, 1):
            raise RuntimeError(
                'OSV image scan failed; retry before accepting package coverage'
            )
    report = json.loads((args.output / 'osv.json').read_text())
    scanned = [
        item for result in report.get('results', [])
        for item in result.get('packages', [])
    ]
    names = {
        item['package']['name']
        for item in scanned
        if item['package'].get('ecosystem') == 'Alpine:v3.23'
    }
    if not {'busybox', 'musl'} <= names:
        raise ValueError(
            'OSV did not identify Alpine runtime packages; inspect scanner extraction'
        )
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
    # An empty finding list alone is not proof of package coverage: retain the
    # installed database and complete inventory alongside the scanner reports.
    print(json.dumps(subject, indent=2))


if __name__ == '__main__':
    main()

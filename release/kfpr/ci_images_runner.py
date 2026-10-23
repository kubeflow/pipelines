"""Build images and collect complete raw evidence in read-only CI jobs.

Successful collection validates identity and scanner coverage, not
vulnerability acceptance. The qualification client applies the selected
security policy.
"""

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
from pathlib import Path

from .security import _trivy_findings


def identity() -> dict:
    """Validate dispatch identity before using it in paths or artifacts."""
    fields = {
        'version': ('VERSION', r'[0-9]+\.[0-9]+\.[0-9]+'),
        'source_sha': ('SOURCE_SHA', r'[0-9a-f]{40}'),
        'workflow_sha': ('WORKFLOW_SHA', r'[0-9a-f]{40}'),
        'request_id': ('REQUEST_ID', r'[0-9a-f]{32}'),
        'run_id': ('GITHUB_RUN_ID', r'[1-9][0-9]*'),
    }
    result = {'schema': 1}
    for key, (variable, pattern) in fields.items():
        value = os.environ.get(variable, '')
        if not re.fullmatch(pattern, value):
            raise ValueError(f'Invalid {variable}')
        result[key] = value
    attempt = os.environ.get('GITHUB_RUN_ATTEMPT', '')
    if not re.fullmatch(r'[1-9][0-9]*', attempt):
        raise ValueError('Invalid GITHUB_RUN_ATTEMPT')
    result['run_attempt'] = int(attempt)
    return result


def inventory(source: Path, release: dict) -> list[dict]:
    """Use only the checked-out release commit's image/platform matrix."""
    # Lazy import avoids coupling the dispatcher to this CI entrypoint.
    from .qualification import image_matrix

    actual = subprocess.check_output(
        ['git', 'rev-parse', 'HEAD'], cwd=source, text=True
    ).strip()
    if actual != release['source_sha']:
        raise ValueError('Source checkout does not match the requested SHA')
    if (source / 'VERSION').read_text().strip() != release['version']:
        raise ValueError('Source VERSION does not match the requested release')
    records = image_matrix(source)
    pairs = [(record['name'], record['platform']) for record in records]
    if len(set(pairs)) != len(pairs):
        raise ValueError('Duplicate release image/platform in source matrix')
    return records


def matrix(source: Path, release: dict) -> dict:
    return {
        'include': [
            {
                'name': record['name'],
                'platform': record['platform'],
                'arch': record['platform'].split('/')[1],
                'runner': 'ubuntu-24.04'
                if record['platform'] == 'linux/amd64'
                else 'ubuntu-24.04-arm',
            }
            for record in inventory(source, release)
        ]
    }


def _logged(command: list[str], destination: Path, source: Path) -> None:
    with destination.open('w') as output:
        subprocess.run(
            command,
            cwd=source,
            stdout=output,
            stderr=subprocess.STDOUT,
            check=True,
        )


def _verify_scan(path: Path, image_id: str) -> str:
    data = json.loads(path.read_text())
    # The immutable Docker ID must be the actual subject of the raw scan.
    if (
        data.get('ArtifactName') != image_id
        or data.get('ArtifactType') != 'container_image'
        or data.get('Metadata', {}).get('ImageID') != image_id
    ):
        raise ValueError('Trivy scan subject does not match the built image ID')
    # Preserve every finding for the client's centrally evaluated policy.
    _trivy_findings(data)
    return hashlib.sha256(path.read_bytes()).hexdigest()


def lane(
    source: Path, output: Path, release: dict, name: str, platform: str
) -> None:
    output.mkdir(parents=True, exist_ok=True)
    records = [
        record
        for record in inventory(source, release)
        if (record['name'], record['platform']) == (name, platform)
    ]
    if len(records) != 1:
        raise ValueError(
            'Lane does not identify exactly one source release image'
        )
    record = records[0]
    tag = f'kfpr-qualification/{name}:{release["request_id"]}-{platform.split("/")[1]}'
    command = [
        'docker',
        'buildx',
        'build',
        '--load',
        '--pull',
        '--platform',
        platform,
        '--file',
        record['dockerfile'],
        '--tag',
        tag,
    ]
    node = (source / 'frontend/.nvmrc').read_text().strip().removeprefix('v')
    for key, value in {
        'NODE_VERSION': node,
        'COMMIT_SHA': release['source_sha'],
        'COMMIT_HASH': release['source_sha'],
        'TAG_NAME': release['version'],
    }.items():
        command.extend(['--build-arg', f'{key}={value}'])
    evidence = {**release, 'name': name, 'platform': platform, 'passed': False}
    try:
        _logged(command + [record['context']], output / 'build.log', source)
        image_id = subprocess.check_output(
            ['docker', 'image', 'inspect', '--format', '{{.Id}}', tag],
            cwd=source,
            text=True,
        ).strip()
        if not re.fullmatch(r'sha256:[0-9a-f]{64}', image_id):
            raise ValueError('Build did not produce an immutable image ID')
        evidence['image_id'] = image_id
        # Explicit empty configs prevent environment/repository ignore policies.
        (output / 'trivy-config.json').write_text('{}\n')
        (output / 'trivy-ignore.txt').write_text('')
        environment = {
            key: value
            for key, value in os.environ.items()
            if not key.startswith('TRIVY_')
        }
        scan = output / 'scan.json'
        with (output / 'scan.stderr.log').open('w') as errors:
            subprocess.run(
                [
                    'trivy',
                    'image',
                    '--config',
                    str(output / 'trivy-config.json'),
                    '--ignorefile',
                    str(output / 'trivy-ignore.txt'),
                    '--detection-priority',
                    'comprehensive',
                    '--scanners',
                    'vuln',
                    '--severity',
                    'UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL',
                    '--ignore-unfixed=false',
                    '--list-all-pkgs',
                    '--format',
                    'json',
                    '--output',
                    str(scan),
                    '--exit-code',
                    '0',
                    image_id,
                ],
                cwd=source,
                env=environment,
                stderr=errors,
                check=True,
            )
        evidence.update(
            scan_file='scan.json',
            scan_sha256=_verify_scan(scan, image_id),
            passed=True,
        )
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        evidence['error'] = str(error)
        raise
    finally:
        (output / 'lane.json').write_text(json.dumps(evidence, indent=2) + '\n')


def aggregate(source: Path, lanes: Path, output: Path, release: dict) -> dict:
    """Require complete successful evidence from this exact workflow
    attempt."""
    expected = {
        (record['name'], record['platform'])
        for record in inventory(source, release)
    }
    seen = set()
    images = []
    output.mkdir(parents=True, exist_ok=True)
    for path in sorted(lanes.glob('*/lane.json')):
        lane_report = json.loads(path.read_text())
        if any(lane_report.get(key) != value for key, value in release.items()):
            raise ValueError(
                'Lane identity does not match this workflow attempt'
            )
        pair = (lane_report.get('name'), lane_report.get('platform'))
        if (
            pair not in expected
            or pair in seen
            or lane_report.get('passed') is not True
        ):
            raise ValueError(
                'Unexpected, duplicate, or unsuccessful image lane'
            )
        image_id = lane_report.get('image_id', '')
        if not re.fullmatch(r'sha256:[0-9a-f]{64}', image_id):
            raise ValueError('Invalid image ID in lane evidence')
        if lane_report.get('scan_file') != 'scan.json':
            raise ValueError('Invalid lane scan path')
        scan = path.parent / 'scan.json'
        if scan.is_symlink():
            raise ValueError('Symlink scan evidence is not allowed')
        digest = _verify_scan(scan, image_id)
        if digest != lane_report.get('scan_sha256'):
            raise ValueError('Raw image scan checksum mismatch')
        filename = f'scans/{pair[0]}-{pair[1].split("/")[1]}.json'
        destination = output / filename
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(scan, destination)
        images.append(
            {
                'name': pair[0],
                'platform': pair[1],
                'image_id': image_id,
                'scan_file': filename,
                'scan_sha256': digest,
            }
        )
        seen.add(pair)
    if seen != expected:
        raise ValueError(
            f'Incomplete image inventory: missing {sorted(expected - seen)}'
        )
    manifest = {**release, 'images': images}
    (output / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    return manifest


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('matrix', 'lane', 'aggregate'))
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--output', type=Path)
    parser.add_argument('--lanes', type=Path)
    parser.add_argument('--name')
    parser.add_argument('--platform')
    args = parser.parse_args()
    release = identity()
    tool_sha = subprocess.check_output(
        ['git', 'rev-parse', 'HEAD'],
        cwd=Path(__file__).resolve().parents[2],
        text=True,
    ).strip()
    if tool_sha != release['workflow_sha']:
        raise ValueError('Tool checkout does not match the workflow SHA')
    source = args.source.resolve()
    if args.command == 'matrix':
        print(json.dumps(matrix(source, release), separators=(',', ':')))
    elif args.command == 'lane':
        if not args.output or not args.name or not args.platform:
            parser.error('lane requires --output, --name, and --platform')
        lane(source, args.output.resolve(), release, args.name, args.platform)
    else:
        if not args.output or not args.lanes:
            parser.error('aggregate requires --output and --lanes')
        aggregate(source, args.lanes.resolve(), args.output.resolve(), release)


if __name__ == '__main__':
    main()

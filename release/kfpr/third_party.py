"""Collect and verify third-party release image evidence without SARIF
uploads."""

import argparse
import hashlib
import json
import re
import subprocess
from pathlib import Path

from . import ci_images_runner, deployed_images
from .deployed_images import extract_images as extract_images
from .osv_image_scan import SCANNER, scan_image, validate_report
from .osv_reports import findings

PLATFORM = 'linux/amd64'
RAW_FILES = ('osv-report.json', 'subject.json', 'scan.stderr.log')
IDENTITY_FIELDS = (
    'schema',
    'version',
    'source_sha',
    'workflow_sha',
    'request_id',
    'run_id',
    'run_attempt',
)
# Historical publication aliases differ from current native image names.
LEGACY_ALIASES = {
    'kfp-api-server': 'apiserver',
    'kfp-persistence-agent': 'persistenceagent',
    'kfp-scheduled-workflow-controller': 'scheduledworkflow',
}


def repository_name(image):
    repository = image.split('@', 1)[0]
    if ':' in repository.rsplit('/', 1)[-1]:
        repository = repository.rsplit(':', 1)[0]
    return repository


def external_images(images, native_names):
    native = {f'ghcr.io/kubeflow/{name}' for name in native_names}
    native.update(
        f'gcr.io/ml-pipeline/{name.removeprefix("kfp-")}'
        for name in native_names
    )
    native.update(
        f'gcr.io/ml-pipeline/{LEGACY_ALIASES[name]}'
        for name in native_names
        if name in LEGACY_ALIASES
    )
    result = set()
    for image in images:
        repository = repository_name(image)
        if repository in native:
            continue
        if repository.startswith('ghcr.io/kubeflow/kfp-'):
            raise ValueError('Unclassified native image: ' + image)
        result.add(image)
    if not result:
        raise ValueError('No third-party images in release manifests')
    return result


def _sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _read(path):
    if path.is_symlink() or not path.is_file():
        raise ValueError('Missing regular third-party evidence: ' + str(path))
    return json.loads(path.read_text())


def verify(directory, identity):
    """Require raw evidence for every source-derived deployment target."""
    directory = Path(directory)
    if directory.is_symlink() or any(
        path.is_symlink() for path in directory.rglob('*')
    ):
        raise ValueError('Symlink third-party evidence is not allowed')
    manifest = _read(directory / 'third-party.json')
    if any(manifest.get(key) != identity[key] for key in IDENTITY_FIELDS):
        raise ValueError('Third-party provenance does not match qualification')
    if (
        manifest.get('scanner') != SCANNER
        or manifest.get('platform') != PLATFORM
        or manifest.get('report_format') != 'json'
    ):
        raise ValueError('Invalid third-party scanner or platform')
    if manifest.get('coverage_failures'):
        raise ValueError(
            'Incomplete third-party scan coverage; resolve failed image scans'
        )
    native_names = manifest.get('first_party_names')
    if (
        not isinstance(native_names, list)
        or not native_names
        or any(
            not isinstance(name, str)
            or not re.fullmatch(r'kfp-[a-z0-9-]+', name)
            for name in native_names
        )
        or native_names != sorted(set(native_names))
    ):
        raise ValueError('Invalid native image inventory')
    if 'inventory' not in manifest:
        raise ValueError(
            'Missing complete deployment inventory; collect fresh evidence'
        )
    inventory = deployed_images.discover(directory / 'inventory')
    if manifest['inventory'] != inventory:
        raise ValueError(
            'Deployment inventory checksum or target mismatch; recollect evidence'
        )
    expected = external_images(
        {target['image'] for target in inventory['targets']}, native_names
    )
    records = manifest.get('images')
    if (
        not isinstance(records, list)
        or len(records) != len(expected)
        or any(not isinstance(record, dict) for record in records)
        or {record.get('image') for record in records} != expected
    ):
        raise ValueError('Incomplete or duplicated third-party image evidence')
    expected_keys = {
        hashlib.sha256(image.encode()).hexdigest()[:16] for image in expected
    }
    lanes = directory / 'scans'
    if (
        not lanes.is_dir()
        or {path.name for path in lanes.iterdir()} != expected_keys
        or any(not path.is_dir() for path in lanes.iterdir())
    ):
        raise ValueError('Incomplete or unexpected third-party scan lanes')
    scans = []
    for record in records:
        image = record['image']
        key = hashlib.sha256(image.encode()).hexdigest()[:16]
        filename = f'scans/{key}/scan.json'
        image_id = record.get('image_id', '')
        if (
            record.get('key') != key
            or record.get('scan_file') != filename
            or not isinstance(image_id, str)
            or not re.fullmatch(r'sha256:[0-9a-f]{64}', image_id)
            or record.get('scanner') != SCANNER
            or record.get('report_format') != 'json'
        ):
            raise ValueError('Invalid third-party image scan identity')
        path = directory / filename
        envelope = _read(path)
        if _sha(path) != record.get('scan_sha256'):
            raise ValueError('Third-party scan checksum mismatch')
        validate_report(envelope, image_id, report_format='json')
        raw_checksums = record.get('raw_sha256')
        if not isinstance(raw_checksums, dict) or set(raw_checksums) != set(
            RAW_FILES
        ):
            raise ValueError('Missing third-party raw evidence checksums')
        for raw_filename in RAW_FILES:
            raw_path = path.parent / raw_filename
            if (
                not raw_path.is_file()
                or _sha(raw_path) != raw_checksums[raw_filename]
            ):
                raise ValueError(
                    'Third-party raw evidence checksum mismatch: '
                    + raw_filename
                )
        if _read(path.parent / 'osv-report.json') != envelope['report']:
            raise ValueError(
                'Third-party raw OSV report does not match envelope'
            )
        subject = _read(path.parent / 'subject.json')
        if (
            subject.get('image') != image
            or subject.get('image_id') != image_id
            or subject.get('platform') != PLATFORM
        ):
            raise ValueError('Third-party scan subject mismatch')
        scans.append(
            {
                'name': 'third-party-' + key,
                'target': image_id,
                'raw_report': 'third-party/' + filename,
                'findings': findings(envelope['report']),
            }
        )
    return {
        **{key: manifest[key] for key in IDENTITY_FIELDS},
        'scanner': SCANNER,
        'report_format': 'json',
        'platform': PLATFORM,
        'images': records,
        'scans': scans,
        'inventory': {**inventory, 'first_party_names': native_names},
    }


def collect(source, kustomize, output, identity):
    from .qualification import image_matrix

    actual = subprocess.check_output(
        ['git', 'rev-parse', 'HEAD'], cwd=source, text=True
    ).strip()
    if actual != identity['source_sha']:
        raise ValueError('Third-party source does not match requested SHA')
    if (source / 'VERSION').read_text().strip() != identity['version']:
        raise ValueError('Third-party source VERSION does not match request')
    names = sorted({record['name'] for record in image_matrix(source)})
    output.mkdir(parents=True, exist_ok=False)
    inventory = deployed_images.capture(source, kustomize, output / 'inventory')
    manifest = {
        **identity,
        'scanner': SCANNER,
        'report_format': 'json',
        'platform': PLATFORM,
        'first_party_names': names,
        'inventory': inventory,
        'images': [],
    }
    images = external_images(
        {target['image'] for target in inventory['targets']}, names
    )
    (output / 'third-party.json').write_text(
        json.dumps(manifest, indent=2) + '\n'
    )
    tooling = Path(__file__).resolve().parents[2]
    failures = []
    for image in sorted(images):
        key = hashlib.sha256(image.encode()).hexdigest()[:16]
        lane = output / 'scans' / key
        lane.mkdir(parents=True, exist_ok=False)
        subprocess.run(
            [
                'bash',
                str(
                    tooling
                    / '.github/resources/scripts/docker-pull-with-retry.sh'
                ),
                image,
            ],
            check=True,
        )
        metadata = json.loads(
            subprocess.check_output(
                ['docker', 'image', 'inspect', image], text=True
            )
        )
        if not isinstance(metadata, list) or len(metadata) != 1:
            raise ValueError('Expected exactly one pulled third-party image')
        item = metadata[0]
        if item.get('Os') != 'linux' or item.get('Architecture') != 'amd64':
            raise ValueError('Third-party qualification requires linux/amd64')
        subject = {
            'image': image,
            'image_id': item['Id'],
            'platform': PLATFORM,
            'repo_digests': item.get('RepoDigests', []),
        }
        (lane / 'subject.json').write_text(json.dumps(subject, indent=2) + '\n')
        try:
            scan_image(item['Id'], PLATFORM, lane, report_format='json')
        except (subprocess.CalledProcessError, ValueError) as error:
            failures.append(
                {'image': image, 'image_id': item['Id'], 'error': str(error)}
            )
            manifest['coverage_failures'] = failures
            (output / 'third-party.json').write_text(
                json.dumps(manifest, indent=2) + '\n'
            )
            continue
        manifest['images'].append(
            {
                'key': key,
                'image': image,
                'image_id': item['Id'],
                'scan_file': f'scans/{key}/scan.json',
                'scan_sha256': _sha(lane / 'scan.json'),
                'raw_sha256': {name: _sha(lane / name) for name in RAW_FILES},
                'scanner': SCANNER,
                'report_format': 'json',
            }
        )
        # Preserve progress as evidence even if a subsequent image fails.
        (output / 'third-party.json').write_text(
            json.dumps(manifest, indent=2) + '\n'
        )
    if failures:
        raise ValueError(
            'Incomplete third-party scan coverage; inspect retained diagnostics and resolve failed image scans'
        )
    verify(output, identity)
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('collect',))
    parser.add_argument('--source', required=True, type=Path)
    parser.add_argument('--kustomize', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    identity = ci_images_runner.identity()
    actual = subprocess.check_output(
        ['git', 'rev-parse', 'HEAD'],
        cwd=Path(__file__).resolve().parents[2],
        text=True,
    ).strip()
    if actual != identity['workflow_sha']:
        raise ValueError('Third-party tooling does not match workflow SHA')
    collect(args.source, args.kustomize, args.output, identity)


if __name__ == '__main__':
    main()

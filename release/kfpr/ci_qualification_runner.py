"""Collect complete source, SDK, and image qualification in read-only CI."""

import argparse
import hashlib
import json
import os
import shutil
import subprocess
from pathlib import Path

from . import ci_images_runner as images
from .ci_security_runner import verify_security
from .smoke import supported_python_versions


def inventory(
    source: Path, identity: dict, include_sdk: bool, include_backend: bool
) -> dict:
    if (
        type(include_sdk) is not bool
        or type(include_backend) is not bool
        or not (include_sdk or include_backend)
    ):
        raise ValueError(
            'Qualification requires a nonempty boolean SDK/backend scope'
        )
    actual = subprocess.check_output(
        ['git', 'rev-parse', 'HEAD'], cwd=source, text=True
    ).strip()
    if actual != identity['source_sha']:
        raise ValueError('Source checkout does not match the requested SHA')
    if (
        include_backend
        and (source / 'VERSION').read_text().strip() != identity['version']
    ):
        raise ValueError('Source VERSION does not match the requested release')
    minors = []
    if include_sdk:
        package = source / 'sdk/python'
        metadata = '\n'.join(
            path.read_text()
            for path in (package / 'pyproject.toml', package / 'setup.py')
            if path.is_file()
        )
        minors = supported_python_versions(metadata)
    return {
        'matrix': images.matrix(source, identity)
        if include_backend
        else {'include': []},
        'python_versions': minors,
    }


def aggregate(
    source: Path,
    lanes: Path,
    security: Path,
    output: Path,
    identity: dict,
    include_sdk: bool,
    include_backend: bool,
) -> dict:
    expected = inventory(source, identity, include_sdk, include_backend)
    verify_security(security, identity, expected['python_versions'])
    if include_backend:
        manifest = images.aggregate(
            source, lanes, output, identity, scanner='osv-json'
        )
    else:
        output.mkdir(parents=True, exist_ok=True)
        manifest = {
            **identity,
            'scanner': images.scanner_identity('osv-json'),
            'report_format': 'json',
            'images': [],
        }
    # Reject extra symlink evidence before copying a downloaded artifact tree.
    if security.is_symlink() or any(
        path.is_symlink() for path in security.rglob('*')
    ):
        raise ValueError('Symlink security evidence is not allowed')
    shutil.copytree(security, output / 'security')
    manifest.update(
        include_sdk=include_sdk,
        include_backend=include_backend,
        security_manifest_sha256=hashlib.sha256(
            (output / 'security/security.json').read_bytes()
        ).hexdigest(),
    )
    (output / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    return manifest


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('inventory', 'aggregate'))
    parser.add_argument('--source', required=True, type=Path)
    parser.add_argument('--lanes', type=Path, default=Path('lanes'))
    parser.add_argument('--security', type=Path, default=Path('security'))
    parser.add_argument('--output', type=Path, default=Path('qualified'))
    args = parser.parse_args()
    identity = images.identity()
    tool_sha = subprocess.check_output(
        ['git', 'rev-parse', 'HEAD'],
        cwd=Path(__file__).resolve().parents[2],
        text=True,
    ).strip()
    if tool_sha != identity['workflow_sha']:
        raise ValueError('Tool checkout does not match the workflow SHA')
    flags = []
    for name in ('INCLUDE_SDK', 'INCLUDE_BACKEND'):
        value = os.environ.get(name)
        if value not in ('true', 'false'):
            raise ValueError('Invalid boolean scope: ' + name)
        flags.append(value == 'true')
    if args.command == 'inventory':
        result = inventory(args.source, identity, *flags)
        with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
            print(
                'matrix=' + json.dumps(result['matrix'], separators=(',', ':')),
                file=output,
            )
            print('python-versions<<EOF', file=output)
            print('\n'.join(result['python_versions']), file=output)
            print('EOF', file=output)
    else:
        aggregate(
            args.source,
            args.lanes,
            args.security,
            args.output,
            identity,
            *flags,
        )


if __name__ == '__main__':
    main()

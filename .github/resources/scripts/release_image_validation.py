#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Qualify 3.x+ shared release tags on native Linux runners."""

import argparse
import hashlib
import json
from pathlib import Path
import platform
import re
import subprocess

from arm64_smoke import image_refs
from publish_image_index import digest
from publish_image_index import inspect
from publish_image_index import validate_index

VERSION = re.compile(
    r'^(\d+)\.(\d+)\.(\d+)(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$')


def validate_source(target_tag, source_version):
    """Reject cross-line dispatch before any build or registry mutation."""
    target = VERSION.fullmatch(target_tag)
    source = VERSION.fullmatch(source_version.strip())
    if not target or not source:
        raise ValueError(
            'Expected MAJOR.MINOR.PATCH versions with an optional prerelease suffix'
        )
    if int(target[1]) < 3:
        raise ValueError(
            'Publish 2.x using the workflow from its release-2.x branch; '
            'the post-MLMD workflow is for 3.0 and later')
    if target.groups() != source.groups():
        raise ValueError(
            'Target tag and source VERSION must have the same major.minor.patch'
        )


def validate_native_images(records, source_sha, target_tag, architecture):
    """Pull shared tags naturally and verify their native image identity."""
    if not VERSION.fullmatch(target_tag) or int(target_tag.split('.')[0]) < 3:
        raise ValueError(
            'Native release qualification requires a 3.0+ release tag')
    host_arch = {'x86_64': 'amd64', 'aarch64': 'arm64'}.get(platform.machine())
    if platform.system() != 'Linux' or architecture != host_arch:
        raise ValueError(f'Requires a native Linux {architecture} runner')
    refs = image_refs(records, source_sha, 'published')
    results = []
    for name, reference in sorted(refs.items()):
        repository, index_digest = reference.split('@')
        tag = f'{repository}:{target_tag}'
        index = inspect(tag)
        if index.get('digest') != index_digest:
            raise ValueError(
                f'{tag} does not resolve to this publication\'s verified index')
        inventory = validate_index(index, {'linux/amd64', 'linux/arm64'})
        child_digest = next(entry[2]
                            for entry in inventory
                            if entry[:2] == ('platform',
                                             f'linux/{architecture}'))
        # .Manifest contains only a descriptor for a single image, not its config.
        raw_child = subprocess.check_output([
            'docker', 'buildx', 'imagetools', 'inspect',
            f'{repository}@{child_digest}', '--raw'
        ])
        if f'sha256:{hashlib.sha256(raw_child).hexdigest()}' != child_digest:
            raise ValueError(
                f'{name}: registry returned a different child manifest')
        child = json.loads(raw_child)
        expected_config = digest(child['config']['digest'])
        # No --platform: prove the native daemon selects the right platform.
        subprocess.run(['docker', 'pull', tag], check=True)
        local = json.loads(
            subprocess.check_output(['docker', 'image', 'inspect', tag],
                                    text=True))[0]
        # Classic Docker uses the config ID; containerd uses an index/manifest ID.
        if (local.get('Os') != 'linux' or
                local.get('Architecture') != architecture or
                reference not in local.get('RepoDigests', []) or local.get('Id')
                not in {expected_config, index_digest, child_digest}):
            raise ValueError(
                f'{tag}: native pull did not select the verified {architecture} image'
            )
        if inspect(tag).get('digest') != index_digest:
            raise ValueError(f'{tag} changed during native verification')
        results.append({
            'image': name,
            'tag': tag,
            'reference': reference,
            'platform': f'linux/{architecture}',
            'child_digest': child_digest,
            'config_digest': expected_config,
            'local_image_id': local['Id']
        })
    return {'source_sha': source_sha, 'images': results}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='command', required=True)
    source = commands.add_parser('source')
    source.add_argument('--target-tag', required=True)
    source.add_argument('--version-file', type=Path, required=True)
    native = commands.add_parser('native')
    native.add_argument('--target-tag', required=True)
    native.add_argument('--source-sha', required=True)
    native.add_argument('--records', type=Path, required=True)
    native.add_argument(
        '--architecture', choices=('amd64', 'arm64'), required=True)
    native.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if args.command == 'source':
        validate_source(args.target_tag, args.version_file.read_text())
    else:
        result = validate_native_images(args.records, args.source_sha,
                                        args.target_tag, args.architecture)
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(result, indent=2) + '\n')


if __name__ == '__main__':
    main()

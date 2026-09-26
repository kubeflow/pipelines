#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Publish an index only after checking this run's complete platform
inventory."""

import argparse
import json
from pathlib import Path
import re
import subprocess
import tempfile

INDEX_TYPES = {
    'application/vnd.oci.image.index.v1+json',
    'application/vnd.docker.distribution.manifest.list.v2+json',
}
MANIFEST_TYPES = {
    'application/vnd.oci.image.manifest.v1+json',
    'application/vnd.docker.distribution.manifest.v2+json',
}


def digest(value):
    if not isinstance(value, str) or not re.fullmatch(r'sha256:[a-f0-9]{64}',
                                                      value):
        raise ValueError(f'Invalid image digest: {value!r}')
    return value


def load_sources(directory, source_sha, platforms):
    """Reject missing platforms and records from a different commit."""
    if not re.fullmatch(r'[a-f0-9]{40}', source_sha):
        raise ValueError('Source commit must be a full Git SHA')
    records = {}
    for path in sorted(directory.iterdir()):
        record = json.loads(path.read_text())
        platform = record.get('platform')
        if platform not in platforms or platform in records:
            raise ValueError(
                f'Unexpected or duplicate build platform: {platform}')
        if record.get('source_sha') != source_sha:
            raise ValueError(f'Build source commit does not match: {path}')
        records[platform] = digest(record.get('digest'))
    if set(records) != platforms:
        raise ValueError(f'Missing build platforms: {platforms - set(records)}')
    return records


def validate_index(index, platforms):
    """Check runnable platforms and linked BuildKit attestations."""
    if index.get('schemaVersion') != 2 or index.get(
            'mediaType') not in INDEX_TYPES:
        raise ValueError('Expected an OCI index or Docker manifest list')
    found = {}
    descriptors = set()
    attestations = []
    for manifest in index.get('manifests', []):
        child_digest = digest(manifest.get('digest'))
        if child_digest in descriptors or manifest.get(
                'mediaType') not in MANIFEST_TYPES:
            raise ValueError(
                'Duplicate digest or unexpected child manifest type')
        descriptors.add(child_digest)
        platform = manifest.get('platform', {})
        name = f"{platform.get('os')}/{platform.get('architecture')}"
        annotations = manifest.get('annotations', {})
        if name == 'unknown/unknown':
            if annotations.get(
                    'vnd.docker.reference.type') != 'attestation-manifest':
                raise ValueError(
                    'Unknown platform without a BuildKit attestation')
            attestations.append(annotations.get('vnd.docker.reference.digest'))
            continue
        if name not in platforms or name in found:
            raise ValueError(f'Unexpected or duplicate index platform: {name}')
        allowed_variants = ('', 'v8') if name == 'linux/arm64' else ('',)
        if platform.get('variant', '') not in allowed_variants:
            raise ValueError(f'Unexpected platform variant: {platform}')
        found[name] = child_digest
    if set(found) != platforms:
        raise ValueError(f'Missing index platforms: {platforms - set(found)}')
    if any(value not in found.values() for value in attestations):
        raise ValueError('Attestation does not reference a runnable image')
    return descriptors


def imagetools(*args):
    return subprocess.check_output(['docker', 'buildx', 'imagetools', *args],
                                   text=True)


def inspect(reference):
    return json.loads(
        imagetools('inspect', '--format', '{{json .Manifest}}', reference))


def publish(args):
    platforms = set(args.platforms.split(','))
    if not platforms or not platforms <= {'linux/amd64', 'linux/arm64'}:
        raise ValueError(
            'Expected platforms must be linux/amd64 and/or linux/arm64')
    records = load_sources(args.digests, args.source_sha, platforms)
    sources = []
    source_descriptors = set()
    for platform, source_digest in records.items():
        reference = f'{args.image}@{source_digest}'
        index = inspect(reference)
        if index.get('digest') != source_digest:
            raise ValueError('Registry returned a different build digest')
        source_descriptors.update(validate_index(index, {platform}))
        sources.append(reference)

    with tempfile.TemporaryDirectory() as temporary:
        metadata = Path(temporary) / 'metadata.json'
        run_reference = f'{args.image}:{args.run_tag}'
        imagetools('create', '--metadata-file', str(metadata), '-t',
                   run_reference, *sources)
        created_digest = digest(
            json.loads(
                metadata.read_text())['containerimage.descriptor']['digest'])
        reference = f'{args.image}@{created_digest}'
        index = inspect(reference)
        if index.get('digest') != created_digest:
            raise ValueError('Registry returned a different index digest')
        if validate_index(index, platforms) != source_descriptors:
            raise ValueError(
                'Published index does not contain exactly this run\'s build manifests'
            )
        if inspect(run_reference).get('digest') != created_digest:
            raise ValueError('Run tag does not resolve to the verified index')

        # Promote the checked digest, never read a mutable tag back as a source.
        tags = ['-t', f'{args.image}:{args.target_tag}']
        if args.set_latest == 'true':
            tags.extend(['-t', f'{args.image}:latest'])
        promotion = Path(temporary) / 'promotion.json'
        imagetools('create', '--metadata-file', str(promotion), *tags,
                   reference)
        promoted_digest = json.loads(
            promotion.read_text())['containerimage.descriptor']['digest']
        if promoted_digest != created_digest:
            raise ValueError(
                'Shared tag publication changed the verified index')

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(
        json.dumps(
            {
                'image': args.image,
                'source_sha': args.source_sha,
                'reference': reference,
                'platforms': sorted(platforms),
            },
            indent=2) + '\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('image', 'source-sha', 'platforms', 'run-tag', 'target-tag'):
        parser.add_argument(f'--{name}', required=True)
    parser.add_argument('--digests', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument(
        '--set-latest', choices=('true', 'false'), required=True)
    publish(parser.parse_args())


if __name__ == '__main__':
    main()

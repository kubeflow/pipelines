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
"""Fail-closed publication checks without registry writes."""

import argparse
import copy
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import publish_image_index as publication

SHA = '1' * 40
IMAGE = 'ghcr.io/kubeflow/kfp-driver'
PLATFORMS = {'linux/amd64', 'linux/arm64'}


def digest(number):
    return f'sha256:{number:064x}'


def index(arches):
    manifests = []
    for arch in arches:
        number = 1 if arch == 'amd64' else 2
        manifests.append({
            'mediaType': 'application/vnd.oci.image.manifest.v1+json',
            'digest': digest(number),
            'platform': {
                'os': 'linux',
                'architecture': arch
            },
        })
        manifests.append({
            'mediaType': 'application/vnd.oci.image.manifest.v1+json',
            'digest': digest(number + 2),
            'platform': {
                'os': 'unknown',
                'architecture': 'unknown'
            },
            'annotations': {
                'vnd.docker.reference.type': 'attestation-manifest',
                'vnd.docker.reference.digest': digest(number),
            },
        })
    return {
        'schemaVersion': 2,
        'mediaType': 'application/vnd.oci.image.index.v1+json',
        'manifests': manifests,
    }


class PublicationTest(unittest.TestCase):

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        for arch in ('amd64', 'arm64'):
            self.write_record(arch)

    def write_record(self, arch, **changes):
        record = {
            'digest': digest(10 if arch == 'amd64' else 20),
            'source_sha': SHA,
            'platform': f'linux/{arch}',
        }
        record.update(changes)
        (self.directory / f'{arch}.json').write_text(json.dumps(record))

    def test_complete_build_inventory(self):
        self.assertEqual(
            publication.load_sources(self.directory, SHA, PLATFORMS), {
                'linux/amd64': digest(10),
                'linux/arm64': digest(20),
            })

    def test_missing_architecture(self):
        (self.directory / 'arm64.json').unlink()
        with self.assertRaisesRegex(ValueError, 'Missing build'):
            publication.load_sources(self.directory, SHA, PLATFORMS)

    def test_stale_source_commit(self):
        self.write_record('arm64', source_sha='2' * 40)
        with self.assertRaisesRegex(ValueError, 'source commit'):
            publication.load_sources(self.directory, SHA, PLATFORMS)

    def test_invalid_digest(self):
        self.write_record('arm64', digest='')
        with self.assertRaisesRegex(ValueError, 'Invalid image digest'):
            publication.load_sources(self.directory, SHA, PLATFORMS)

    def test_duplicate_build_platform(self):
        self.write_record('arm64', platform='linux/amd64')
        with self.assertRaisesRegex(ValueError, 'duplicate build'):
            publication.load_sources(self.directory, SHA, PLATFORMS)

    def test_exact_platforms_with_attestations(self):
        result = publication.validate_index(
            index(['amd64', 'arm64']), PLATFORMS)
        self.assertEqual(result, {digest(number) for number in range(1, 5)})

    def test_inverse_proxy_single_platform(self):
        publication.validate_index(index(['amd64']), {'linux/amd64'})

    def test_missing_index_platform(self):
        with self.assertRaisesRegex(ValueError, 'Missing index'):
            publication.validate_index(index(['amd64']), PLATFORMS)

    def test_extra_index_platform(self):
        with self.assertRaisesRegex(ValueError,
                                    'Unexpected or duplicate index'):
            publication.validate_index(
                index(['amd64', 'arm64']), {'linux/amd64'})

    def test_unrecognized_unknown_platform(self):
        candidate = index(['amd64', 'arm64'])
        candidate['manifests'][1].pop('annotations')
        with self.assertRaisesRegex(ValueError,
                                    'without a BuildKit attestation'):
            publication.validate_index(candidate, PLATFORMS)

    def test_orphan_attestation(self):
        candidate = index(['amd64', 'arm64'])
        candidate['manifests'][1]['annotations'][
            'vnd.docker.reference.digest'] = digest(99)
        with self.assertRaisesRegex(ValueError, 'does not reference'):
            publication.validate_index(candidate, PLATFORMS)

    def test_duplicate_platform_and_unexpected_variant(self):
        for mutation in ('duplicate', 'variant'):
            with self.subTest(mutation=mutation):
                candidate = index(['amd64', 'arm64'])
                if mutation == 'duplicate':
                    duplicate = copy.deepcopy(candidate['manifests'][0])
                    duplicate['digest'] = digest(99)
                    candidate['manifests'].append(duplicate)
                else:
                    candidate['manifests'][2]['platform']['variant'] = 'v9'
                with self.assertRaisesRegex(ValueError, 'Unexpected'):
                    publication.validate_index(candidate, PLATFORMS)

    def publish(self, corrupt=None, set_latest='true'):
        calls = []

        def registry(*args):
            calls.append(args)
            if args[0] == 'create':
                if '--metadata-file' in args:
                    metadata = Path(args[args.index('--metadata-file') + 1])
                    metadata.write_text(
                        json.dumps({
                            'containerimage.descriptor': {
                                'digest':
                                    digest(99 if corrupt == 'promotion' and
                                           metadata.name == 'promotion.json'
                                           else 30)
                            },
                        }))
                return ''
            reference = args[-1]
            if reference.endswith(digest(10)):
                result = index(['amd64'])
                result['digest'] = digest(10)
            elif reference.endswith(digest(20)):
                result = index(['arm64'])
                result['digest'] = digest(20)
            else:
                result = index(['amd64', 'arm64'])
                result['digest'] = digest(30)
                if corrupt == 'substitution':
                    result['manifests'][0]['digest'] = digest(99)
                    result['manifests'][1]['annotations'][
                        'vnd.docker.reference.digest'] = digest(99)
                if corrupt == 'tag' and ':run-' in reference:
                    result['digest'] = digest(99)
            return json.dumps(result)

        args = argparse.Namespace(
            image=IMAGE,
            source_sha=SHA,
            platforms=','.join(sorted(PLATFORMS)),
            digests=self.directory,
            run_tag='run-123-1',
            target_tag='master',
            set_latest=set_latest,
            output=self.directory / 'output' / 'kfp-driver.json')
        with mock.patch.object(publication, 'imagetools', side_effect=registry):
            if corrupt:
                with self.assertRaises(ValueError):
                    publication.publish(args)
                self.assertEqual(
                    len([call for call in calls if call[0] == 'create']),
                    2 if corrupt == 'promotion' else 1)
                self.assertFalse(args.output.exists())
            else:
                publication.publish(args)
        return calls, args.output

    def test_verified_digest_promoted_and_exported(self):
        calls, output = self.publish()
        self.assertEqual(calls[-1][3:],
                         ('-t', f'{IMAGE}:master', '-t', f'{IMAGE}:latest',
                          f'{IMAGE}@{digest(30)}'))
        self.assertEqual(
            json.loads(output.read_text()), {
                'image': IMAGE,
                'source_sha': SHA,
                'reference': f'{IMAGE}@{digest(30)}',
                'platforms': sorted(PLATFORMS),
            })

    def test_latest_remains_optional(self):
        calls, _ = self.publish(set_latest='false')
        self.assertEqual(calls[-1][3:],
                         ('-t', f'{IMAGE}:master', f'{IMAGE}@{digest(30)}'))

    def test_substituted_manifest_not_promoted(self):
        self.publish(corrupt='substitution')

    def test_changed_staging_tag_not_promoted(self):
        self.publish(corrupt='tag')

    def test_changed_promotion_digest_not_exported(self):
        self.publish(corrupt='promotion')


if __name__ == '__main__':
    unittest.main()

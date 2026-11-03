"""Promotion copies complete released indexes and fails before partial
preflight."""

import base64
import copy
import json
import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import promotion as p


class PromotionTest(unittest.TestCase):
    def setUp(self):
        tmp = TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.output = Path(tmp.name) / 'evidence.json'
        self.inventory = {
            'version': '2.17.3',
            'source_sha': 'a' * 40,
            'images': {
                'first': ['linux/amd64', 'linux/arm64'],
                'second': ['linux/amd64'],
            },
        }
        self.registry = {}
        self.commands = []
        for i, (name, platforms) in enumerate(self.inventory['images'].items()):
            index = {
                'schemaVersion': 2,
                'mediaType': 'application/vnd.oci.image.index.v1+json',
                'digest': 'sha256:' + str(i + 1) * 64,
                'manifests': [
                    {
                        'digest': 'sha256:' + str(j + 5) * 64,
                        'mediaType': 'application/vnd.oci.image.manifest.v1+json',
                        'platform': dict(
                            zip(('os', 'architecture'), platform.split('/'))
                        ),
                    }
                    for j, platform in enumerate(platforms)
                ],
            }
            index['manifests'].append(
                {
                    'digest': 'sha256:' + '8' * 64,
                    'mediaType': 'application/vnd.oci.image.manifest.v1+json',
                    'platform': {'os': 'unknown', 'architecture': 'unknown'},
                    'annotations': {
                        'vnd.docker.reference.type': 'attestation-manifest',
                        'vnd.docker.reference.digest': index['manifests'][0][
                            'digest'
                        ],
                    },
                }
            )
            image = 'ghcr.io/kubeflow/' + name
            self.registry[image + ':2.17.3'] = copy.deepcopy(index)
            self.registry[image + '@' + index['digest']] = copy.deepcopy(index)
            old = copy.deepcopy(index)
            old['digest'] = 'sha256:' + '9' * 64
            self.registry[image + ':latest'] = old

    def docker(self, command, **kwargs):
        self.commands.append(command)
        self.assertEqual(command[:3], ['docker', 'buildx', 'imagetools'])
        if command[3] == 'inspect':
            return json.dumps(self.registry[command[-1]])
        self.assertEqual(
            command[3:6], ['create', '--prefer-index=false', '--tag']
        )
        target, source = command[-2:]
        self.assertTrue(
            target.startswith('ghcr.io/kubeflow/')
            and target.endswith(':latest')
        )
        self.assertIn('@sha256:', source)
        self.registry[target] = copy.deepcopy(self.registry[source])
        return ''

    def run_promotion(self):
        with (
            mock.patch.object(p, 'discover', return_value=self.inventory),
            mock.patch.object(
                p.subprocess, 'check_output', side_effect=self.docker
            ),
        ):
            return p.promote(self.inventory, self.output)

    def test_copy_keeps_complete_indexes_and_versioned_tags_unchanged(self):
        original = copy.deepcopy(self.registry)
        result = self.run_promotion()
        self.assertTrue(result['passed'])
        for record in result['images']:
            image = record['image']
            self.assertEqual(
                self.registry[image + ':latest'], original[image + ':2.17.3']
            )
            self.assertEqual(
                self.registry[image + ':2.17.3'], original[image + ':2.17.3']
            )
            self.assertEqual(record['new_digest'], record['source_digest'])
            self.assertNotEqual(record['new_digest'], record['previous_digest'])
        self.assertEqual(
            sum(command[3] == 'create' for command in self.commands), 2
        )

    def test_invalid_late_index_blocks_all_writes(self):
        self.registry['ghcr.io/kubeflow/second:2.17.3']['manifests'] = []
        with self.assertRaisesRegex(ValueError, 'Missing index platforms'):
            self.run_promotion()
        self.assertFalse(
            any(command[3] == 'create' for command in self.commands)
        )
        self.assertFalse(json.loads(self.output.read_text())['passed'])

    def test_partial_inventory_and_newer_release_block_all_writes(self):
        partial = copy.deepcopy(self.inventory)
        partial['images'].pop('second')
        with mock.patch.object(p, 'discover', return_value=self.inventory):
            with self.assertRaisesRegex(ValueError, 'inventory changed'):
                p.promote(partial, self.output)
        with (
            mock.patch.object(
                p,
                'discover',
                side_effect=[self.inventory, ValueError('newer stable')],
            ),
            mock.patch.object(
                p.subprocess, 'check_output', side_effect=self.docker
            ),
        ):
            with self.assertRaisesRegex(ValueError, 'newer stable'):
                p.promote(self.inventory, self.output)
        self.assertFalse(
            any(command[3] == 'create' for command in self.commands)
        )

    def test_partial_push_failure_retains_successful_image_evidence(self):
        def docker(command, **kwargs):
            if command[3] == 'create' and '/second:' in command[-2]:
                raise subprocess.CalledProcessError(1, command)
            return self.docker(command, **kwargs)

        with (
            mock.patch.object(p, 'discover', return_value=self.inventory),
            mock.patch.object(p.subprocess, 'check_output', side_effect=docker),
        ):
            with self.assertRaises(subprocess.CalledProcessError):
                p.promote(self.inventory, self.output)
        report = json.loads(self.output.read_text())
        self.assertFalse(report['passed'])
        self.assertTrue(report['images'][0]['promoted'])
        self.assertFalse(report['images'][1]['promoted'])

    def test_discovery_validates_stable_release_and_exact_source_matrix(self):
        workflow = {
            'jobs': {
                'build-images-for-release': {
                    'strategy': {
                        'matrix': {
                            'image': [
                                {
                                    'name': 'first',
                                    'dockerfile': 'Dockerfile',
                                    'context': '.',
                                }
                            ],
                            'arch': [
                                {
                                    'platform': 'linux/amd64',
                                    'arch_name': 'amd64',
                                }
                            ],
                        }
                    }
                }
            }
        }
        release = {
            'tag_name': '2.17.3',
            'draft': False,
            'prerelease': False,
            'published_at': '2026-10-22T00:00:00Z',
        }
        with (
            mock.patch.object(
                p, '_should_set_backend_latest', return_value=True
            ),
            mock.patch.object(
                p,
                'api',
                side_effect=[
                    release,
                    {'sha': 'a' * 40},
                    {
                        'content': base64.b64encode(
                            json.dumps(workflow).encode()
                        ).decode()
                    },
                ],
            ) as api,
        ):
            result = p.discover('2.17.3')
        self.assertEqual(result['images'], {'first': ['linux/amd64']})
        self.assertIn('ref=' + 'a' * 40, api.call_args.args[0])
        for version in (
            'sdk-2.17.3',
            '2.17.3;touch /tmp/no',
            '2.17.3-rc.1',
            '../master',
        ):
            with self.assertRaises(ValueError):
                p.discover(version)
        with mock.patch.object(
            p, 'api', return_value={**release, 'draft': True}
        ):
            with self.assertRaises(ValueError):
                p.discover('2.17.3')

    def test_dispatch_dry_run_never_queries_or_runs_docker(self):
        with (
            mock.patch.object(p, 'api') as api,
            mock.patch.object(p.subprocess, 'run') as run,
        ):
            p.dispatch('2.17.3', dry_run=True)
        api.assert_not_called()
        run.assert_not_called()

    def test_changed_release_mid_batch_stops_remaining_writes(self):
        with (
            mock.patch.object(
                p,
                'discover',
                side_effect=[self.inventory] * 3 + [ValueError('newer stable')],
            ),
            mock.patch.object(
                p.subprocess, 'check_output', side_effect=self.docker
            ),
        ):
            with self.assertRaisesRegex(ValueError, 'newer stable'):
                p.promote(self.inventory, self.output)
        self.assertEqual(
            sum(command[3] == 'create' for command in self.commands), 1
        )
        self.assertFalse(json.loads(self.output.read_text())['passed'])

    def test_preflight_failure_also_retains_evidence(self):
        with mock.patch.object(
            p, 'discover', side_effect=ValueError('unavailable')
        ):
            with self.assertRaisesRegex(ValueError, 'unavailable'):
                p.promote(self.inventory, self.output)
        self.assertEqual(
            json.loads(self.output.read_text())['error'], 'unavailable'
        )

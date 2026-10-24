"""Third-party qualification requires complete immutable release evidence."""

import hashlib
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import yaml

from . import third_party as third
from .osv_image_scan import SCANNER
from .osv_image_scan_test import json_envelope

IMAGE = 'docker.io/example/service:1'
IMAGE_ID = 'sha256:' + 'd' * 64


def evidence(directory, identity):
    directory.mkdir(parents=True)
    rendered = 'kind: Pod\nspec:\n  containers:\n  - image: ' + IMAGE + '\n'
    key = hashlib.sha256(IMAGE.encode()).hexdigest()[:16]
    lane = directory / 'scans' / key
    lane.mkdir(parents=True)
    report = lane / 'scan.json'
    envelope = json_envelope(IMAGE_ID, True)
    report.write_text(json.dumps(envelope))
    (lane / 'osv-report.json').write_text(json.dumps(envelope['report']))
    (lane / 'scan.stderr.log').write_text('')
    (lane / 'subject.json').write_text(
        json.dumps(
            {'image': IMAGE, 'image_id': IMAGE_ID, 'platform': third.PLATFORM}
        )
    )
    manifest = {
        **identity,
        'scanner': SCANNER,
        'report_format': 'json',
        'platform': third.PLATFORM,
        'first_party_names': ['kfp-api-server'],
        'rendered_files': {},
        'images': [
            {
                'key': key,
                'image': IMAGE,
                'image_id': IMAGE_ID,
                'scan_file': f'scans/{key}/scan.json',
                'scan_sha256': third._sha(report),
                'raw_sha256': {
                    name: third._sha(lane / name) for name in third.RAW_FILES
                },
                'scanner': SCANNER,
                'report_format': 'json',
            }
        ],
    }
    (directory / 'rendered').mkdir()
    for profile in third.PROFILES:
        path = directory / 'rendered' / f'{profile}.yaml'
        path.write_text(rendered)
        manifest['rendered_files'][profile] = {
            'file': f'rendered/{profile}.yaml',
            'sha256': third._sha(path),
        }
    (directory / 'third-party.json').write_text(json.dumps(manifest))
    return manifest


class ThirdPartyTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.directory = self.root / 'evidence'
        self.identity = {
            'schema': 1,
            'version': '2.18.0',
            'source_sha': 'a' * 40,
            'workflow_sha': 'b' * 40,
            'request_id': 'c' * 32,
            'run_id': '123',
            'run_attempt': 1,
        }

    def test_extracts_executor_and_real_containers_not_embedded_templates(self):
        document = {
            'kind': 'Pod',
            'spec': {
                'containers': [
                    {
                        'image': 'quay.io/argoproj/workflow-controller:v4',
                        'args': [
                            '--executor-image',
                            'quay.io/argoproj/argoexec:v4',
                        ],
                    }
                ]
            },
        }
        embedded = {
            'kind': 'ConfigMap',
            'data': {
                'template': yaml.safe_dump(
                    {'containers': [{'image': 'ignore:1'}]}
                )
            },
        }
        result = third.extract_images(yaml.safe_dump_all([document, embedded]))
        self.assertEqual(
            result,
            {
                'quay.io/argoproj/workflow-controller:v4',
                'quay.io/argoproj/argoexec:v4',
            },
        )
        document['spec']['containers'][0]['args'] = [
            '--executor-image=quay.io/argoproj/argoexec:v5'
        ]
        self.assertIn(
            'quay.io/argoproj/argoexec:v5',
            third.extract_images(yaml.safe_dump(document)),
        )
        for args in (
            ['--executor-image'],
            ['--executor-image='],
            ['--executor-image', '--other'],
        ):
            document['spec']['containers'][0]['args'] = args
            with self.assertRaisesRegex(ValueError, 'executor image'):
                third.extract_images(yaml.safe_dump(document))

    def test_native_names_control_classification_without_hiding_unknown_components(
        self,
    ):
        images = {
            IMAGE,
            'ghcr.io/kubeflow/kfp-api-server:1',
            'gcr.io/ml-pipeline/apiserver:1',
            'gcr.io/ml-pipeline/application-crd-controller:1',
        }
        self.assertEqual(
            third.external_images(images, ['kfp-api-server']),
            {IMAGE, 'gcr.io/ml-pipeline/application-crd-controller:1'},
        )
        with self.assertRaisesRegex(ValueError, 'Unclassified'):
            third.external_images(
                {'ghcr.io/kubeflow/kfp-new:1'}, ['kfp-api-server']
            )

    def test_verify_preserves_every_raw_finding_and_provenance(self):
        manifest = evidence(self.directory, self.identity)
        result = third.verify(self.directory, self.identity)
        self.assertEqual(result['images'], manifest['images'])
        self.assertEqual(len(result['scans']), 1)
        self.assertEqual(len(result['scans'][0]['findings']), 1)
        self.assertEqual(result['scans'][0]['target'], IMAGE_ID)
        self.assertEqual(result['source_sha'], self.identity['source_sha'])
        self.assertEqual(
            set(result['inventory']['profiles']), set(third.PROFILES)
        )

    def test_missing_duplicate_or_replaced_images_fail(self):
        manifest = evidence(self.directory, self.identity)
        for records in (
            [],
            manifest['images'] * 2,
            [{**manifest['images'][0], 'image': 'other:1'}],
        ):
            (self.directory / 'third-party.json').write_text(
                json.dumps({**manifest, 'images': records})
            )
            with self.assertRaisesRegex(
                ValueError, 'third-party image evidence'
            ):
                third.verify(self.directory, self.identity)

    def test_provenance_render_and_scan_tampering_fail(self):
        manifest = evidence(self.directory, self.identity)
        with self.assertRaisesRegex(ValueError, 'provenance'):
            third.verify(self.directory, {**self.identity, 'run_attempt': 2})
        path = self.directory / manifest['images'][0]['scan_file']
        original = path.read_text()
        path.write_text(original + '\n')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            third.verify(self.directory, self.identity)
        path.write_text(original)
        rendered = (
            self.directory
            / manifest['rendered_files'][third.PROFILES[0]]['file']
        )
        rendered.write_text(rendered.read_text() + '# changed\n')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            third.verify(self.directory, self.identity)

    def test_missing_profile_escaping_scan_or_symlink_fail(self):
        manifest = evidence(self.directory, self.identity)
        original = json.dumps(manifest)
        manifest['rendered_files'].pop(third.PROFILES[0])
        (self.directory / 'third-party.json').write_text(json.dumps(manifest))
        with self.assertRaisesRegex(ValueError, 'profile coverage'):
            third.verify(self.directory, self.identity)
        manifest = json.loads(original)
        manifest['images'][0]['scan_file'] = '../scan.json'
        (self.directory / 'third-party.json').write_text(json.dumps(manifest))
        with self.assertRaisesRegex(ValueError, 'scan identity'):
            third.verify(self.directory, self.identity)
        (self.directory / 'third-party.json').write_text(original)
        (self.directory / 'alias').symlink_to(
            self.directory / 'third-party.json'
        )
        with self.assertRaisesRegex(ValueError, 'Symlink'):
            third.verify(self.directory, self.identity)

    def test_collect_renders_exact_source_and_scans_immutable_subject(self):
        source = self.root / 'source'
        source.mkdir()
        (source / 'VERSION').write_text('2.18.0')
        calls = []

        def command(args, **kwargs):
            calls.append(args)
            if args[:2] == ['git', 'rev-parse']:
                return self.identity['source_sha']
            if args[:2] == ['docker', 'image']:
                return json.dumps(
                    [
                        {
                            'Id': IMAGE_ID,
                            'Os': 'linux',
                            'Architecture': 'amd64',
                            'RepoDigests': [],
                        }
                    ]
                )
            return 'kind: Pod\nspec:\n  containers:\n  - image: ' + IMAGE + '\n'

        def scan(image_id, platform, output, report_format):
            self.assertEqual(
                (image_id, platform, report_format),
                (IMAGE_ID, 'linux/amd64', 'json'),
            )
            envelope = json_envelope(image_id)
            (output / 'scan.json').write_text(json.dumps(envelope))
            (output / 'osv-report.json').write_text(
                json.dumps(envelope['report'])
            )
            (output / 'scan.stderr.log').write_text('')

        with (
            mock.patch.object(
                third.subprocess, 'check_output', side_effect=command
            ),
            mock.patch.object(third.subprocess, 'run'),
            mock.patch(
                'kfpr.qualification.image_matrix',
                return_value=[{'name': 'kfp-api-server'}],
            ),
            mock.patch.object(third, 'scan_image', side_effect=scan),
        ):
            result = third.collect(
                source, Path('/bin/kustomize'), self.directory, self.identity
            )
        self.assertEqual(len(result['images']), 1)
        self.assertEqual(
            len(third.verify(self.directory, self.identity)['scans']), 1
        )
        renders = [call for call in calls if call[0] == '/bin/kustomize']
        self.assertEqual(
            {call[-1] for call in renders},
            {
                str(source / 'manifests/kustomize/env' / profile)
                for profile in third.PROFILES
            },
        )

    def test_raw_report_missing_or_tampered_fails(self):
        manifest = evidence(self.directory, self.identity)
        record = manifest['images'][0]
        path = self.directory / record['scan_file']
        raw = path.parent / 'osv-report.json'
        original = raw.read_text()
        raw.unlink()
        with self.assertRaisesRegex(ValueError, 'raw evidence checksum'):
            third.verify(self.directory, self.identity)
        raw.write_text('{"results": []}')
        with self.assertRaisesRegex(ValueError, 'raw evidence checksum'):
            third.verify(self.directory, self.identity)
        record['raw_sha256']['osv-report.json'] = third._sha(raw)
        (self.directory / 'third-party.json').write_text(json.dumps(manifest))
        with self.assertRaisesRegex(ValueError, 'does not match envelope'):
            third.verify(self.directory, self.identity)
        raw.write_text(original)

    def test_extra_lane_and_missing_diagnostics_fail(self):
        manifest = evidence(self.directory, self.identity)
        extra = self.directory / 'scans/unexpected'
        extra.mkdir()
        with self.assertRaisesRegex(
            ValueError, 'unexpected third-party scan lanes'
        ):
            third.verify(self.directory, self.identity)
        extra.rmdir()
        path = self.directory / manifest['images'][0]['scan_file']
        (path.parent / 'scan.stderr.log').unlink()
        with self.assertRaisesRegex(ValueError, 'raw evidence checksum'):
            third.verify(self.directory, self.identity)

    def test_source_mismatch_prevents_collection(self):
        with mock.patch.object(
            third.subprocess, 'check_output', return_value='f' * 40
        ):
            with self.assertRaisesRegex(ValueError, 'requested SHA'):
                third.collect(
                    self.root, Path('kustomize'), self.directory, self.identity
                )
        self.assertFalse(self.directory.exists())


if __name__ == '__main__':
    unittest.main()

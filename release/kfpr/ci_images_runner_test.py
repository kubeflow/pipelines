"""Exercise CI image qualification without Docker builds or publication."""

import hashlib
import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import yaml

from kfpr import ci_images_runner as runner

IMAGE_ID = 'sha256:' + 'a' * 64


class CiImagesRunnerTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.source = self.root / 'source'
        self.source.mkdir()
        (self.source / 'VERSION').write_text('2.17.3\n')
        (self.source / 'frontend').mkdir()
        (self.source / 'frontend/.nvmrc').write_text('v24.21.0\n')
        (self.source / 'Dockerfile').write_text('FROM scratch\n')
        workflow = self.source / '.github/workflows/image-builds-release.yml'
        workflow.parent.mkdir(parents=True)
        workflow.write_text(
            yaml.safe_dump(
                {
                    'jobs': {
                        'build-images-for-release': {
                            'strategy': {
                                'matrix': {
                                    'image': [
                                        {
                                            'name': 'image-a',
                                            'dockerfile': 'Dockerfile',
                                            'context': '.',
                                        }
                                    ],
                                    'arch': [
                                        {'platform': 'linux/amd64'},
                                        {'platform': 'linux/arm64'},
                                    ],
                                }
                            },
                        }
                    }
                }
            )
        )
        subprocess.run(['git', 'init', '--quiet', str(self.source)], check=True)
        subprocess.run(['git', 'add', '.'], cwd=self.source, check=True)
        subprocess.run(
            [
                'git',
                '-c',
                'user.name=Test',
                '-c',
                'user.email=test@example.invalid',
                'commit',
                '--quiet',
                '-m',
                'Fixture',
            ],
            cwd=self.source,
            check=True,
        )
        sha = subprocess.check_output(
            ['git', 'rev-parse', 'HEAD'], cwd=self.source, text=True
        ).strip()
        self.release = {
            'schema': 1,
            'version': '2.17.3',
            'source_sha': sha,
            'workflow_sha': 'b' * 40,
            'request_id': 'c' * 32,
            'run_id': '123',
            'run_attempt': 1,
        }

    def scan(self, fixed=False):
        return {
            'SchemaVersion': 2,
            'ArtifactName': IMAGE_ID,
            'ArtifactType': 'container_image',
            'Metadata': {'ImageID': IMAGE_ID},
            'Results': [
                {
                    'Target': 'bin/app',
                    'Vulnerabilities': [
                        {
                            'VulnerabilityID': 'CVE-test',
                            'PkgName': 'stdlib',
                            'InstalledVersion': '1',
                            'FixedVersion': '2' if fixed else '',
                        }
                    ],
                }
            ],
        }

    def write_lane(self, arch='amd64', fixed=False):
        directory = self.root / 'lanes' / arch
        directory.mkdir(parents=True, exist_ok=True)
        scan = directory / 'scan.json'
        scan.write_text(json.dumps(self.scan(fixed)))
        report = {
            **self.release,
            'name': 'image-a',
            'platform': f'linux/{arch}',
            'image_id': IMAGE_ID,
            'scan_file': 'scan.json',
            'scan_sha256': hashlib.sha256(scan.read_bytes()).hexdigest(),
            'passed': True,
        }
        (directory / 'lane.json').write_text(json.dumps(report))
        return directory

    def test_matrix_uses_source_inventory_and_native_runners(self):
        entries = runner.matrix(self.source, self.release)['include']
        self.assertEqual(
            [entry['runner'] for entry in entries],
            ['ubuntu-24.04', 'ubuntu-24.04-arm'],
        )
        self.assertEqual(len(entries), 2)
        with self.assertRaisesRegex(ValueError, 'SHA'):
            runner.matrix(self.source, {**self.release, 'source_sha': '0' * 40})
        with self.assertRaisesRegex(ValueError, 'VERSION'):
            runner.matrix(self.source, {**self.release, 'version': '2.17.4'})

    def test_scan_rejects_mismatched_immutable_metadata(self):
        data = self.scan()
        data['Metadata'] = {'ImageID': 'sha256:' + 'f' * 64}
        scan = self.root / 'scan.json'
        scan.write_text(json.dumps(data))
        with self.assertRaisesRegex(ValueError, 'subject'):
            runner._verify_scan(scan, IMAGE_ID)

    def test_complete_aggregate_preserves_raw_scans_and_identity(self):
        self.write_lane(fixed=True)
        self.write_lane('arm64')
        result = runner.aggregate(
            self.source, self.root / 'lanes', self.root / 'final', self.release
        )
        for key, value in self.release.items():
            self.assertEqual(result[key], value)
        self.assertEqual(len(result['images']), 2)
        for record in result['images']:
            scan = self.root / 'final' / record['scan_file']
            self.assertEqual(
                hashlib.sha256(scan.read_bytes()).hexdigest(),
                record['scan_sha256'],
            )
            self.assertEqual(
                json.loads(scan.read_text()),
                self.scan(fixed=record['platform'] == 'linux/amd64'),
            )

    def test_aggregate_rejects_missing_duplicate_mismatched_or_invalid_lanes(
        self,
    ):
        first = self.write_lane()
        with self.assertRaisesRegex(ValueError, 'Incomplete'):
            runner.aggregate(
                self.source,
                self.root / 'lanes',
                self.root / 'final',
                self.release,
            )
        second = self.write_lane('arm64')
        original = json.loads((first / 'lane.json').read_text())
        for updates in (
            {'run_attempt': 2},
            {'source_sha': 'f' * 40},
            {'workflow_sha': 'e' * 40},
            {'request_id': 'd' * 32},
            {'passed': False},
            {'platform': 'linux/arm64'},
            {'scan_sha256': '0' * 64},
            {'scan_file': '../outside.json'},
        ):
            with self.subTest(updates=updates):
                (first / 'lane.json').write_text(
                    json.dumps({**original, **updates})
                )
                with self.assertRaises(ValueError):
                    runner.aggregate(
                        self.source,
                        self.root / 'lanes',
                        self.root / 'final',
                        self.release,
                    )
        (first / 'lane.json').write_text(json.dumps(original))
        scan_path = second / 'scan.json'
        data = self.scan()
        data['Results'] = []
        scan_path.write_text(json.dumps(data))
        lane_path = second / 'lane.json'
        lane_data = json.loads(lane_path.read_text())
        lane_data['scan_sha256'] = hashlib.sha256(
            scan_path.read_bytes()
        ).hexdigest()
        lane_path.write_text(json.dumps(lane_data))
        with self.assertRaisesRegex(ValueError, 'no dependency targets'):
            runner.aggregate(
                self.source,
                self.root / 'lanes',
                self.root / 'final',
                self.release,
            )
        self.assertFalse((self.root / 'final/manifest.json').exists())
        self.assertTrue((second / 'scan.json').exists())

    def test_lane_executes_build_and_comprehensive_scan_and_keeps_failures(
        self,
    ):
        binaries = self.root / 'bin'
        binaries.mkdir()
        script = """#!/usr/bin/env python3
import json, os, pathlib, sys
args = sys.argv[1:]
with open(os.environ['COMMAND_LOG'], 'a') as log:
    log.write(json.dumps([pathlib.Path(sys.argv[0]).name, *args]) + '\\n')
if pathlib.Path(sys.argv[0]).name == 'docker':
    if args[:2] == ['image', 'inspect']:
        print(os.environ['IMAGE_ID'])
else:
    if os.environ.get('SCANNER_ERROR'):
        sys.exit(2)
    pathlib.Path(args[args.index('--output') + 1]).write_text(os.environ['FAKE_SCAN'])
"""
        for name in ('docker', 'trivy'):
            path = binaries / name
            path.write_text(script)
            path.chmod(0o755)
        environment = {
            **os.environ,
            'PATH': str(binaries) + os.pathsep + os.environ['PATH'],
            'COMMAND_LOG': str(self.root / 'commands.jsonl'),
            'IMAGE_ID': IMAGE_ID,
            'FAKE_SCAN': json.dumps(self.scan(fixed=True)),
            'TRIVY_IGNORE_UNFIXED': 'true',
        }
        with mock.patch.dict(os.environ, environment, clear=True):
            runner.lane(
                self.source,
                self.root / 'good',
                self.release,
                'image-a',
                'linux/amd64',
            )
            invalid_scan = self.scan()
            invalid_scan['Results'] = []
            os.environ['FAKE_SCAN'] = json.dumps(invalid_scan)
            with self.assertRaisesRegex(ValueError, 'no dependency targets'):
                runner.lane(
                    self.source,
                    self.root / 'bad',
                    self.release,
                    'image-a',
                    'linux/amd64',
                )
            os.environ['SCANNER_ERROR'] = '1'
            with self.assertRaises(subprocess.CalledProcessError):
                runner.lane(
                    self.source,
                    self.root / 'scanner-error',
                    self.release,
                    'image-a',
                    'linux/amd64',
                )
        self.assertFalse(
            json.loads((self.root / 'scanner-error/lane.json').read_text())[
                'passed'
            ]
        )
        self.assertTrue(
            json.loads((self.root / 'good/lane.json').read_text())['passed']
        )
        self.assertEqual(
            json.loads((self.root / 'good/scan.json').read_text()),
            self.scan(fixed=True),
        )
        self.assertFalse(
            json.loads((self.root / 'bad/lane.json').read_text())['passed']
        )
        self.assertTrue((self.root / 'bad/scan.json').exists())
        commands = [
            json.loads(line)
            for line in (self.root / 'commands.jsonl').read_text().splitlines()
        ]
        self.assertIn('--load', commands[0])
        self.assertNotIn('--push', commands[0])
        for argument in (
            'NODE_VERSION=24.21.0',
            'COMMIT_SHA=' + self.release['source_sha'],
            'COMMIT_HASH=' + self.release['source_sha'],
            'TAG_NAME=2.17.3',
        ):
            self.assertIn(argument, commands[0])
        self.assertEqual(commands[2][-1], IMAGE_ID)
        self.assertIn('comprehensive', commands[2])
        self.assertIn('--ignore-unfixed=false', commands[2])

    def test_identity_rejects_unsafe_dispatch_inputs(self):
        environment = {
            'VERSION': '2.17.3',
            'SOURCE_SHA': 'a' * 40,
            'WORKFLOW_SHA': 'b' * 40,
            'REQUEST_ID': 'c' * 32,
            'GITHUB_RUN_ID': '123',
            'GITHUB_RUN_ATTEMPT': '1',
        }
        with mock.patch.dict(os.environ, environment, clear=True):
            self.assertEqual(runner.identity()['run_attempt'], 1)
            for field in environment:
                with (
                    self.subTest(field=field),
                    mock.patch.dict(os.environ, {field: '../untrusted'}),
                ):
                    with self.assertRaises(ValueError):
                        runner.identity()

    def test_workflow_is_dispatch_only_read_only_and_pins_actions(self):
        path = (
            Path(__file__).resolve().parents[2]
            / '.github/workflows/release-qualification.yml'
        )
        workflow = yaml.safe_load(path.read_text())
        self.assertEqual(
            set(workflow.get('on', workflow.get(True))), {'workflow_dispatch'}
        )
        self.assertEqual(workflow['permissions'], {})
        self.assertEqual(
            workflow['jobs']['images']['strategy']['matrix'],
            '${{ fromJson(needs.inventory.outputs.matrix) }}',
        )
        for job in workflow['jobs'].values():
            self.assertEqual(job['permissions'], {'contents': 'read'})
            for step in job['steps']:
                if (
                    step.get('uses')
                    == './tooling/.github/actions/download-artifact-with-retry'
                ):
                    self.assertIn('required-files', step['with'])
                    continue
                if step.get('uses') == './tooling/.github/actions/setup-trivy':
                    self.assertEqual(step['with']['arch'], '${{ matrix.arch }}')
                    continue
                if 'uses' in step:
                    self.assertRegex(step['uses'], r'@[0-9a-f]{40}$')
                    if step['uses'].startswith('actions/checkout@'):
                        self.assertIs(
                            step['with']['persist-credentials'], False
                        )
        self.assertNotIn('secrets.', path.read_text())
        self.assertNotIn('docker/login-action', path.read_text())


if __name__ == '__main__':
    unittest.main()

from kfpr import workflow_runs

"""CI image evidence must match the dispatched run, source and full
inventory."""

import hashlib
import json
import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from types import SimpleNamespace
from unittest.mock import Mock, patch

from kfpr import ci_images_client as client

SHA = 'a' * 40
WORKFLOW_SHA = 'b' * 40
REQUEST = 'c' * 32
IMAGE = 'sha256:' + 'd' * 64
INVENTORY = [{'name': 'kfp-api-server', 'platform': 'linux/amd64'}]


class CIImagesClientTest(unittest.TestCase):
    def identity(self):
        return {
            'repository': client.REPO,
            'workflow': client.WORKFLOW,
            'workflow_sha': WORKFLOW_SHA,
            'source_sha': SHA,
            'version': '2.17.3',
            'request_id': REQUEST,
            'title': f'KFP image qualification 2.17.3 {SHA} {REQUEST}',
            'run_id': '123',
            'run_attempt': 1,
        }

    def run_data(self):
        identity = self.identity()
        return {
            'id': 123,
            'repository': {'full_name': client.REPO},
            'path': client.WORKFLOW,
            'event': 'workflow_dispatch',
            'head_sha': WORKFLOW_SHA,
            'display_title': identity['title'],
            'run_attempt': 1,
            'status': 'completed',
            'conclusion': 'success',
        }

    def context(self, root):
        return SimpleNamespace(
            root=root,
            state=SimpleNamespace(answers={'release_sha': SHA}),
            metadata=SimpleNamespace(tag='2.17.3'),
            runner=Mock(),
        )

    def artifact(self, directory, mutate_scan=None, mutate_manifest=None):
        directory.mkdir(parents=True, exist_ok=True)
        scan = {
            'SchemaVersion': 2,
            'ArtifactType': 'container_image',
            'ArtifactName': IMAGE,
            'Metadata': {'ImageID': IMAGE},
            'Results': [{'Target': 'debian', 'Vulnerabilities': []}],
        }
        if mutate_scan:
            mutate_scan(scan)
        path = directory / 'scans/kfp-api-server-amd64.json'
        path.parent.mkdir(exist_ok=True)
        path.write_text(json.dumps(scan))
        manifest = {
            key: value
            for key, value in self.identity().items()
            if key
            in (
                'version',
                'source_sha',
                'workflow_sha',
                'request_id',
                'run_id',
                'run_attempt',
            )
        }
        manifest.update(
            schema=1,
            images=[
                {
                    **INVENTORY[0],
                    'image_id': IMAGE,
                    'scan_file': 'scans/kfp-api-server-amd64.json',
                    'scan_sha256': hashlib.sha256(
                        path.read_bytes()
                    ).hexdigest(),
                }
            ],
        )
        if mutate_manifest:
            mutate_manifest(manifest)
        (directory / 'manifest.json').write_text(json.dumps(manifest))
        return path

    def image_exception(self):
        return {
            'schema': 1,
            'exceptions': [
                {
                    'scope': 'image',
                    'artifact': IMAGE,
                    'vulnerability': 'CVE-2026-1234',
                    'package': 'example',
                    'installed_version': '1',
                    'justification': 'Approved until the upstream package fix is validated',
                    'expires': '2099-01-01',
                }
            ],
        }

    def add_vulnerability(self, scan):
        scan['Results'][0]['Vulnerabilities'] = [
            {
                'VulnerabilityID': 'CVE-2026-1234',
                'PkgName': 'example',
                'InstalledVersion': '1',
                'FixedVersion': '2',
            }
        ]

    def test_exact_approved_image_exception_passes(self):
        with TemporaryDirectory() as temporary:
            directory = Path(temporary)
            self.artifact(directory, mutate_scan=self.add_vulnerability)
            result = client._read_artifact(
                directory,
                INVENTORY,
                self.identity(),
                policy='approved-exceptions',
                exceptions=self.image_exception(),
            )
            self.assertEqual(result['reports'][0]['image_id'], IMAGE)

    def test_python_named_image_uses_image_exception_scope(self):
        with TemporaryDirectory() as temporary:
            directory = Path(temporary)
            scan_file = 'scans/python-runtime-amd64.json'
            path = self.artifact(
                directory,
                mutate_scan=self.add_vulnerability,
                mutate_manifest=lambda manifest: manifest['images'][0].update(
                    name='python-runtime', scan_file=scan_file
                ),
            )
            path.rename(directory / scan_file)
            inventory = [{'name': 'python-runtime', 'platform': 'linux/amd64'}]
            result = client._read_artifact(
                directory,
                inventory,
                self.identity(),
                policy='approved-exceptions',
                exceptions=self.image_exception(),
            )
            self.assertEqual(result['reports'][0]['image_id'], IMAGE)

    def test_image_exception_must_match_every_field(self):
        for field, value in (
            ('artifact', 'sha256:' + 'e' * 64),
            ('vulnerability', 'CVE-2026-5678'),
            ('package', 'different'),
            ('installed_version', '0'),
        ):
            with self.subTest(field=field), TemporaryDirectory() as temporary:
                directory = Path(temporary)
                self.artifact(directory, mutate_scan=self.add_vulnerability)
                exceptions = self.image_exception()
                exceptions['exceptions'][0][field] = value
                with self.assertRaisesRegex(
                    ValueError, 'fixable vulnerabilities'
                ):
                    client._read_artifact(
                        directory,
                        INVENTORY,
                        self.identity(),
                        policy='approved-exceptions',
                        exceptions=exceptions,
                    )

    def test_exception_document_does_not_override_strict_default(self):
        with TemporaryDirectory() as temporary:
            directory = Path(temporary)
            self.artifact(directory, mutate_scan=self.add_vulnerability)
            with self.assertRaises(ValueError):
                client._read_artifact(
                    directory,
                    INVENTORY,
                    self.identity(),
                    exceptions=self.image_exception(),
                )

    def test_exceptions_do_not_accept_invalid_scanner_reports(self):
        for mutate in (
            lambda scan: scan.update(SchemaVersion=3),
            lambda scan: scan.update(Results=None),
            lambda scan: scan.update(Results=[]),
            lambda scan: scan['Metadata'].update(ImageID='sha256:' + 'e' * 64),
        ):
            with self.subTest(mutate=mutate), TemporaryDirectory() as temporary:
                directory = Path(temporary)
                self.artifact(directory, mutate_scan=mutate)
                with self.assertRaises(ValueError):
                    client._read_artifact(
                        directory,
                        INVENTORY,
                        self.identity(),
                        policy='approved-exceptions',
                        exceptions=self.image_exception(),
                    )

    def test_complete_artifact_passes(self):
        with TemporaryDirectory() as temporary:
            directory = Path(temporary)
            path = self.artifact(directory)
            result = client._read_artifact(
                directory, INVENTORY, self.identity()
            )
            self.assertEqual(
                result['reports'],
                [{'image_id': IMAGE, 'path': str(path.resolve())}],
            )

    def test_rejects_provenance_missing_inventory_checksum_and_traversal(self):
        mutations = [
            lambda m: m.update(source_sha='e' * 40),
            lambda m: m.update(run_attempt=2),
            lambda m: m.update(images=[]),
            lambda m: m['images'].append(m['images'][0]),
            lambda m: m['images'][0].update(scan_sha256='bad'),
            lambda m: m['images'][0].update(scan_file='../outside.json'),
        ]
        for mutate in mutations:
            with self.subTest(mutate=mutate), TemporaryDirectory() as temporary:
                directory = Path(temporary)
                self.artifact(directory, mutate_manifest=mutate)
                with self.assertRaises(ValueError):
                    client._read_artifact(directory, INVENTORY, self.identity())

    def test_rejects_symlink(self):
        with TemporaryDirectory() as temporary:
            directory = Path(temporary)
            path = self.artifact(directory)
            original = path.read_text()
            path.unlink()
            target = directory / 'original.json'
            target.write_text(original)
            path.symlink_to(target)
            with self.assertRaisesRegex(ValueError, 'symlink'):
                client._read_artifact(directory, INVENTORY, self.identity())

    def test_rejects_wrong_image_identity_or_fixable_finding(self):
        def vulnerability(scan):
            scan['Results'][0]['Vulnerabilities'] = [
                {
                    'VulnerabilityID': 'CVE-2026-1234',
                    'PkgName': 'example',
                    'InstalledVersion': '1',
                    'FixedVersion': '2',
                }
            ]

        for mutate in (
            lambda scan: scan.update(ArtifactType='filesystem'),
            lambda scan: scan['Metadata'].update(ImageID='sha256:' + 'e' * 64),
            vulnerability,
        ):
            with self.subTest(mutate=mutate), TemporaryDirectory() as temporary:
                directory = Path(temporary)
                self.artifact(directory, mutate_scan=mutate)
                with self.assertRaises(ValueError):
                    client._read_artifact(directory, INVENTORY, self.identity())

    def test_verify_run_rejects_changed_attempt_event_workflow_or_result(self):
        for key, value in (
            ('run_attempt', 2),
            ('event', 'push'),
            ('head_sha', 'e' * 40),
            ('conclusion', 'failure'),
            ('path', '.github/workflows/other.yml'),
            ('display_title', 'other'),
            ('repository', {'full_name': 'fork/pipelines'}),
        ):
            with self.subTest(key=key), TemporaryDirectory() as temporary:
                context = self.context(Path(temporary))
                data = self.run_data()
                data[key] = value
                context.runner.capture.return_value = json.dumps(data)
                with self.assertRaises(ValueError):
                    client.verify_run(context, self.identity())

    def simulate(self, context, failure=False, mutate_scan=None):
        calls = []

        def capture(command, **kwargs):
            endpoint = command[-1]
            calls.append(endpoint)
            if endpoint.endswith('commits/master'):
                return json.dumps({'sha': WORKFLOW_SHA})
            if '/contents/' in endpoint:
                return json.dumps({'type': 'file', 'path': client.WORKFLOW})
            if 'workflow_dispatch&' in endpoint:
                return json.dumps({'workflow_runs': [self.run_data()]})
            data = self.run_data()
            if failure:
                data['conclusion'] = 'failure'
            return json.dumps(data)

        def run(command, **kwargs):
            if 'download' in command and '--name' in command:
                self.artifact(
                    Path(command[command.index('--dir') + 1]),
                    mutate_scan=mutate_scan,
                )
            return SimpleNamespace(returncode=0)

        context.runner.capture.side_effect = capture
        context.runner.run.side_effect = run
        return calls

    def test_dispatch_download_and_post_download_recheck(self):
        with (
            TemporaryDirectory() as temporary,
            patch.object(
                client.uuid, 'uuid4', return_value=SimpleNamespace(hex=REQUEST)
            ),
        ):
            context = self.context(Path(temporary))
            calls = self.simulate(context)
            result = client.qualify_images(
                context, INVENTORY, Path(temporary) / 'evidence'
            )
            self.assertEqual(result['run'], self.identity())
            self.assertEqual(
                sum(
                    endpoint.endswith('actions/runs/123') for endpoint in calls
                ),
                2,
            )
            dispatch = context.runner.run.call_args_list[0].args[0]
            self.assertIn(f'source_sha={SHA}', dispatch)
            self.assertIn(f'request_id={REQUEST}', dispatch)
            downloads = [
                call.args[0]
                for call in context.runner.run.call_args_list
                if 'download' in call.args[0]
            ]
            self.assertIn(f'kfpr-images-{REQUEST}-1', downloads[0])

    def test_dispatch_uses_validated_exception_file(self):
        with (
            TemporaryDirectory() as temporary,
            patch.object(
                client.uuid, 'uuid4', return_value=SimpleNamespace(hex=REQUEST)
            ),
        ):
            root = Path(temporary)
            exception_path = root / 'approved.json'
            exception_path.write_text(json.dumps(self.image_exception()))
            context = self.context(root)
            context.state.answers.update(
                security_policy='approved-exceptions',
                security_exceptions=str(exception_path),
            )
            self.simulate(context, mutate_scan=self.add_vulnerability)
            result = client.qualify_images(
                context, INVENTORY, root / 'evidence'
            )
            self.assertEqual(result['reports'][0]['image_id'], IMAGE)

    def test_missing_exception_file_fails_before_dispatch(self):
        with TemporaryDirectory() as temporary:
            context = self.context(Path(temporary))
            context.state.answers['security_policy'] = 'approved-exceptions'
            with self.assertRaisesRegex(ValueError, 'security_exceptions'):
                client.qualify_images(
                    context, INVENTORY, Path(temporary) / 'evidence'
                )
            context.runner.capture.assert_not_called()
            context.runner.run.assert_not_called()

    def test_failed_run_downloads_diagnostics_without_qualifying(self):
        with (
            TemporaryDirectory() as temporary,
            patch.object(
                client.uuid, 'uuid4', return_value=SimpleNamespace(hex=REQUEST)
            ),
        ):
            context = self.context(Path(temporary))
            self.simulate(context, failure=True)
            with self.assertRaisesRegex(RuntimeError, 'CI failed'):
                client.qualify_images(
                    context, INVENTORY, Path(temporary) / 'evidence'
                )
            commands = [
                call.args[0] for call in context.runner.run.call_args_list
            ]
            self.assertTrue(
                any(
                    f'kfpr-lane-{REQUEST}-1-*' in command
                    for command in commands
                )
            )
            self.assertFalse(any('--name' in command for command in commands))

    def test_missing_upstream_workflow_explains_merge_prerequisite(self):
        with TemporaryDirectory() as temporary:
            context = self.context(Path(temporary))
            context.runner.capture.side_effect = [
                json.dumps({'sha': WORKFLOW_SHA}),
                subprocess.CalledProcessError(
                    1,
                    ['gh', 'api'],
                    output='{"message":"Not Found","status":"404"}',
                ),
            ]
            with self.assertRaisesRegex(
                RuntimeError, 'merge the workflow before qualification'
            ):
                client.qualify_images(
                    context, INVENTORY, Path(temporary) / 'evidence'
                )
            context.runner.run.assert_not_called()

    def test_run_id_must_be_positive_ascii_integer(self):
        for run_id in ('0', '-1', '１２３'):
            with self.subTest(run_id=run_id), TemporaryDirectory() as temporary:
                context = self.context(Path(temporary))
                identity = self.identity()
                identity['run_id'] = run_id
                with self.assertRaisesRegex(ValueError, 'run identity'):
                    client.verify_run(context, identity)
                context.runner.capture.assert_not_called()

    def test_missing_or_duplicate_run_does_not_select_latest(self):
        for runs in ([], [self.run_data(), self.run_data()]):
            with (
                self.subTest(runs=len(runs)),
                TemporaryDirectory() as temporary,
                patch.object(
                    client.uuid,
                    'uuid4',
                    return_value=SimpleNamespace(hex=REQUEST),
                ),
                patch.object(workflow_runs.time, 'sleep'),
            ):
                context = self.context(Path(temporary))
                context.runner.capture.side_effect = [
                    json.dumps({'sha': WORKFLOW_SHA}),
                    json.dumps({'type': 'file', 'path': client.WORKFLOW}),
                ] + [json.dumps({'workflow_runs': runs})] * 24
                with self.assertRaises((RuntimeError, ValueError)):
                    client.qualify_images(
                        context, INVENTORY, Path(temporary) / 'evidence'
                    )
                self.assertEqual(context.runner.run.call_count, 1)


if __name__ == '__main__':
    unittest.main()

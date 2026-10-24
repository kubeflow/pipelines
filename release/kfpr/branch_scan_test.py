"""Exercise release branch scanning without Docker or privileged uploads."""

import copy
import hashlib
import json
import unittest
from unittest import mock

from . import branch_scan as scan
from . import ci_images_runner as runner
from . import ci_images_runner_test as fixtures


def event(conclusion='failure'):
    return {
        'workflow_run': {
            'path': '.github/workflows/pre-commit.yml',
            'event': 'push',
            'head_branch': 'release-2.18',
            'head_sha': 'a' * 40,
            'status': 'completed',
            'conclusion': conclusion,
            'repository': {'full_name': 'kubeflow/pipelines'},
            'head_repository': {'full_name': 'kubeflow/pipelines'},
        }
    }


class BranchScanTest(unittest.TestCase):
    setUp = fixtures.CiImagesRunnerTest.setUp
    scan = fixtures.CiImagesRunnerTest.scan
    write_lane = fixtures.CiImagesRunnerTest.write_lane

    def test_accepts_push_wakeups_regardless_of_lint_conclusion(self):
        for conclusion in ('success', 'failure', 'cancelled', 'timed_out'):
            scan.validate_trigger('workflow_run', event(conclusion))
        scan.validate_trigger('schedule', {})
        scan.validate_trigger('workflow_dispatch', {})

    def test_rejects_other_repository_workflow_event_or_branch(self):
        for field, value in (
            ('path', '.github/workflows/other.yml'),
            ('event', 'pull_request'),
            ('head_branch', 'master'),
            ('head_sha', 'short'),
            ('status', 'in_progress'),
            ('repository', {'full_name': 'fork/pipelines'}),
            ('head_repository', {'full_name': 'fork/pipelines'}),
        ):
            with self.subTest(field=field):
                trigger = event()
                trigger['workflow_run'][field] = value
                with self.assertRaises(ValueError):
                    scan.validate_trigger('workflow_run', trigger)

    @mock.patch('kfpr.branch_scan.subprocess.run')
    def test_late_event_resolves_current_branch_head(self, run):
        run.return_value.stdout = json.dumps(
            {
                'ref': 'refs/heads/release-2.18',
                'object': {'type': 'commit', 'sha': 'b' * 40},
            }
        )
        self.assertEqual(scan.resolve_source('workflow_run', event()), 'b' * 40)
        self.assertIn(
            'repos/kubeflow/pipelines/git/ref/heads/release-2.18',
            run.call_args.args[0],
        )

    @mock.patch('kfpr.branch_scan.subprocess.run')
    def test_branch_resolution_rejects_wrong_ref_or_object(self, run):
        for ref in (
            {
                'ref': 'refs/heads/master',
                'object': {'type': 'commit', 'sha': 'b' * 40},
            },
            {
                'ref': 'refs/heads/release-2.18',
                'object': {'type': 'tag', 'sha': 'b' * 40},
            },
            {
                'ref': 'refs/heads/release-2.18',
                'object': {'type': 'commit', 'sha': 'short'},
            },
        ):
            run.return_value.stdout = json.dumps(ref)
            with self.assertRaises(ValueError):
                scan.resolve_source('schedule', {})

    def test_matrix_uses_source_version_without_inferred_branch_version(self):
        matrix = scan.stable_matrix(self.source, self.release)
        self.assertEqual(
            matrix,
            {
                'include': [
                    {'name': 'image-a', 'platform': 'linux/amd64'},
                    {'name': 'image-a', 'platform': 'linux/arm64'},
                ]
            },
        )
        self.assertEqual(self.release['version'], '2.17.3')
        for key, bad in (('source_sha', 'd' * 40), ('version', '2.18.0')):
            with self.subTest(key=key), self.assertRaises(ValueError):
                scan.stable_matrix(self.source, {**self.release, key: bad})

    def test_expected_lane_download_is_bound_to_request_and_attempt(self):
        files = scan.required_lanes(
            scan.stable_matrix(self.source, self.release), 'a' * 32, 2
        )
        self.assertEqual(len(files), 4)
        self.assertIn(f'kfpr-lane-{"a" * 32}-2-image-a-arm64/scan.json', files)

    def qualified(self, clean=False):
        for arch in ('amd64', 'arm64'):
            directory = self.write_lane(arch)
            sarif = {
                'version': '2.1.0',
                'runs': [
                    {
                        'tool': {
                            'driver': {
                                'name': 'osv-scanner',
                                'version': '2.5.0',
                                'rules': [] if clean else [{'id': 'CVE-test'}],
                            }
                        },
                        'results': []
                        if clean
                        else [
                            {
                                'ruleId': 'CVE-test',
                                'message': {'text': 'Vulnerable package'},
                                'locations': [
                                    {
                                        'physicalLocation': {
                                            'artifactLocation': {
                                                'uri': 'file:///usr/bin/app'
                                            }
                                        }
                                    }
                                ],
                            }
                        ],
                    }
                ],
            }
            report = {
                'schema': 1,
                'scanner': scan.SCANNER,
                'image_id': fixtures.IMAGE_ID,
                'archive_sha256': 'd' * 64,
                'exit_code': 0 if clean else 1,
                'sarif': sarif,
            }
            path = directory / 'scan.json'
            path.write_text(json.dumps(report))
            lane_path = directory / 'lane.json'
            lane = json.loads(lane_path.read_text())
            lane.update(
                scanner=scan.SCANNER,
                scan_sha256=hashlib.sha256(path.read_bytes()).hexdigest(),
            )
            lane_path.write_text(json.dumps(lane))
        directory = self.root / 'qualified'
        runner.aggregate(
            self.source,
            self.root / 'lanes',
            directory,
            self.release,
            scanner='osv',
        )
        return directory

    def test_preserves_all_osv_findings_locations_and_raw_evidence(self):
        directory = self.qualified()
        before = {
            path.name: path.read_bytes()
            for path in (directory / 'scans').iterdir()
        }
        scan.convert_scans(self.source, directory, self.release)
        self.assertEqual(len(list((directory / 'sarif').glob('*.sarif'))), 2)
        for name, content in before.items():
            self.assertEqual((directory / 'scans' / name).read_bytes(), content)
            expected = json.loads(content)['sarif']
            actual = json.loads(
                (
                    directory / 'sarif' / name.replace('.json', '.sarif')
                ).read_text()
            )
            self.assertEqual(actual, expected)

    def test_clean_reports_produce_valid_empty_sarif(self):
        directory = self.qualified(clean=True)
        scan.convert_scans(self.source, directory, self.release)
        for path in (directory / 'sarif').iterdir():
            self.assertEqual(
                json.loads(path.read_text())['runs'][0]['results'], []
            )

    def test_rejects_incomplete_wrong_attempt_or_scanner_before_output(self):
        directory = self.qualified()
        manifest_path = directory / 'manifest.json'
        manifest = json.loads(manifest_path.read_text())
        for mutation in ('missing', 'attempt', 'duplicate', 'scanner'):
            changed = copy.deepcopy(manifest)
            if mutation == 'missing':
                changed['images'].pop()
            if mutation == 'attempt':
                changed['run_attempt'] += 1
            if mutation == 'duplicate':
                changed['images'][1] = changed['images'][0]
            if mutation == 'scanner':
                changed['scanner'] = {'name': 'trivy', 'version': '0.76.0'}
            manifest_path.write_text(json.dumps(changed))
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                scan.convert_scans(self.source, directory, self.release)
            self.assertFalse((directory / 'sarif').exists())

    def test_rejects_raw_scan_tampering_before_output(self):
        directory = self.qualified()
        path = directory / 'scans/image-a-amd64.json'
        path.write_text(path.read_text() + ' ')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            scan.convert_scans(self.source, directory, self.release)
        self.assertFalse((directory / 'sarif').exists())

    def test_rejects_trivy_report_even_with_matching_checksum(self):
        directory = self.qualified()
        path = directory / 'scans/image-a-amd64.json'
        path.write_text(json.dumps(self.scan()))
        manifest_path = directory / 'manifest.json'
        manifest = json.loads(manifest_path.read_text())
        manifest['images'][0]['scan_sha256'] = hashlib.sha256(
            path.read_bytes()
        ).hexdigest()
        manifest_path.write_text(json.dumps(manifest))
        with self.assertRaises(ValueError):
            scan.convert_scans(self.source, directory, self.release)
        self.assertFalse((directory / 'sarif').exists())


if __name__ == '__main__':
    unittest.main()

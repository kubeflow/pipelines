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

    def qualified(self):
        self.write_lane('amd64')
        self.write_lane('arm64')
        directory = self.root / 'qualified'
        runner.aggregate(
            self.source, self.root / 'lanes', directory, self.release
        )
        return directory

    def fake_convert(self, report, destination):
        results = [
            {
                'ruleId': vuln['VulnerabilityID'],
                'locations': [{'uri': result['Target']}],
            }
            for result in report['Results']
            for vuln in (result.get('Vulnerabilities') or [])
        ]
        destination.write_text(
            json.dumps({'version': '2.1.0', 'runs': [{'results': results}]})
        )

    @mock.patch('kfpr.branch_scan.convert_report')
    def test_conversion_preserves_findings_and_raw_scan_bytes(self, run):
        directory = self.qualified()
        before = {
            path.name: path.read_bytes()
            for path in (directory / 'scans').iterdir()
        }
        run.side_effect = self.fake_convert
        scan.convert_scans(self.source, directory, self.release)
        self.assertEqual(run.call_count, 2)
        self.assertEqual(len(list((directory / 'sarif').glob('*.sarif'))), 2)
        self.assertEqual(
            before,
            {
                path.name: path.read_bytes()
                for path in (directory / 'scans').iterdir()
            },
        )

    @mock.patch('kfpr.branch_scan.convert_report')
    def test_clean_null_vulnerability_reports_produce_empty_results(self, run):
        directory = self.qualified()
        manifest_path = directory / 'manifest.json'
        manifest = json.loads(manifest_path.read_text())
        for record in manifest['images']:
            path = directory / record['scan_file']
            report = json.loads(path.read_text())
            report['Results'][0]['Vulnerabilities'] = None
            path.write_text(json.dumps(report))
            record['scan_sha256'] = hashlib.sha256(
                path.read_bytes()
            ).hexdigest()
        manifest_path.write_text(json.dumps(manifest))
        run.side_effect = self.fake_convert
        scan.convert_scans(self.source, directory, self.release)
        for path in (directory / 'sarif').iterdir():
            self.assertEqual(
                json.loads(path.read_text())['runs'][0]['results'], []
            )

    @mock.patch('kfpr.branch_scan.convert_report')
    def test_conversion_rejects_incomplete_or_wrong_attempt_before_running_trivy(
        self, run
    ):
        directory = self.qualified()
        manifest_path = directory / 'manifest.json'
        manifest = json.loads(manifest_path.read_text())
        for mutation in ('missing', 'attempt', 'duplicate'):
            changed = copy.deepcopy(manifest)
            if mutation == 'missing':
                changed['images'].pop()
            if mutation == 'attempt':
                changed['run_attempt'] += 1
            if mutation == 'duplicate':
                changed['images'][1] = changed['images'][0]
            manifest_path.write_text(json.dumps(changed))
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                scan.convert_scans(self.source, directory, self.release)
        run.assert_not_called()

    @mock.patch('kfpr.branch_scan.convert_report')
    def test_conversion_rejects_raw_scan_tampering(self, run):
        directory = self.qualified()
        path = directory / 'scans/image-a-amd64.json'
        path.write_text(path.read_text() + ' ')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            scan.convert_scans(self.source, directory, self.release)
        run.assert_not_called()

    @mock.patch('kfpr.branch_scan.convert_report')
    def test_conversion_rejects_dropped_findings(self, run):
        directory = self.qualified()

        def drop_findings(report, destination):
            destination.write_text(
                json.dumps({'version': '2.1.0', 'runs': [{'results': []}]})
            )

        run.side_effect = drop_findings
        with self.assertRaisesRegex(ValueError, 'every vulnerability'):
            scan.convert_scans(self.source, directory, self.release)

    @mock.patch('kfpr.branch_scan.convert_report')
    def test_os_locations_stay_stable_across_image_ids_without_changing_binary_paths(
        self, run
    ):
        directory = self.qualified()
        manifest_path = directory / 'manifest.json'
        manifest = json.loads(manifest_path.read_text())
        for index, record in enumerate(manifest['images']):
            image_id = 'sha256:' + str(index + 1) * 64
            record['image_id'] = image_id
            report = self.scan()
            report['ArtifactName'] = image_id
            report['Metadata']['ImageID'] = image_id
            report['Results'][0]['Class'] = 'os-pkgs'
            report['Results'][0]['Target'] = image_id + ' (debian 12)'
            report['Results'].append(
                {
                    'Class': 'lang-pkgs',
                    'Target': 'usr/bin/app',
                    'Vulnerabilities': [
                        {
                            'VulnerabilityID': 'CVE-binary',
                            'PkgName': 'binary',
                            'InstalledVersion': '1',
                        }
                    ],
                }
            )
            path = directory / record['scan_file']
            path.write_text(json.dumps(report))
            record['scan_sha256'] = hashlib.sha256(
                path.read_bytes()
            ).hexdigest()
        manifest_path.write_text(json.dumps(manifest))
        run.side_effect = self.fake_convert
        scan.convert_scans(self.source, directory, self.release)
        locations = []
        for path in sorted((directory / 'sarif').iterdir()):
            locations.append(
                [
                    item['locations']
                    for item in json.loads(path.read_text())['runs'][0][
                        'results'
                    ]
                ]
            )
        self.assertEqual(locations[0], locations[1])
        self.assertEqual(
            locations[0], [[{'uri': 'image-a'}], [{'uri': 'usr/bin/app'}]]
        )


if __name__ == '__main__':
    unittest.main()

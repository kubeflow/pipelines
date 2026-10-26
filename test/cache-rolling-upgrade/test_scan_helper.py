"""Unknown package inventory must not become a clean vulnerability result."""
import json
from pathlib import Path
import re
import unittest
from unittest import mock

import scan_helper
import yaml


class CoverageTest(unittest.TestCase):

    def test_parses_runtime_versions(self):
        self.assertEqual(
            scan_helper.packages(
                'P:busybox\nV:1.37.0-r1\n\nP:musl\nV:1.2.5-r0\n'), [{
                    'name': 'busybox',
                    'version': '1.37.0-r1'
                }, {
                    'name': 'musl',
                    'version': '1.2.5-r0'
                }])

    def test_missing_inventory_fails(self):
        for data in ('', 'P:busybox\nV:1.37.0-r1', 'P:busybox\n\nP:musl\nV:1'):
            with self.subTest(data=data), self.assertRaises(ValueError):
                scan_helper.packages(data)

    def test_ecosystem_comes_from_image_release(self):
        self.assertEqual(
            scan_helper.alpine_ecosystem('3.24.1\n'), 'Alpine:v3.24')
        with self.assertRaises(ValueError):
            scan_helper.alpine_ecosystem('unknown')

    def test_scanner_requires_both_runtime_packages_in_correct_ecosystem(self):

        def report(names, ecosystem):
            return {
                'results': [{
                    'packages': [{
                        'package': {
                            'name': name,
                            'ecosystem': ecosystem
                        }
                    } for name in names]
                }]
            }

        self.assertEqual(
            scan_helper.osv_alpine_package_names(
                report(['busybox', 'musl'], 'Alpine:v3.24'), 'Alpine:v3.24'),
            {'busybox', 'musl'})
        for names, ecosystem in ((['busybox'], 'Alpine:v3.24'),
                                 (['busybox', 'musl'], 'Alpine:v3.23')):
            with self.subTest(
                    names=names,
                    ecosystem=ecosystem), self.assertRaises(ValueError):
                scan_helper.osv_alpine_package_names(
                    report(names, ecosystem), 'Alpine:v3.24')

    @mock.patch(
        'scan_helper.subprocess.check_output', return_value='unexpected\n')
    def test_helper_output_mismatch_reports_actual_output(self, output):
        with self.assertRaisesRegex(RuntimeError, 'unexpected'):
            scan_helper.smoke_test_helper('sha256:test')

    def test_deployed_and_fallback_helper_references_agree(self):
        root = Path(__file__).resolve().parents[2]
        image = json.loads((root / 'backend/src/apiserver/config/config.json'
                           ).read_text())['CACHE_IMAGE']
        manifest = yaml.safe_load((
            root /
            'manifests/kustomize/base/installs/generic/pipeline-install-config.yaml'
        ).read_text())
        self.assertEqual(manifest['data']['cacheImage'], image)
        source = (root / 'backend/src/cache/server/mutation.go').read_text()
        fallback = re.search(r'const defaultCacheImage = "([^"]+)"',
                             source).group(1)
        self.assertEqual(fallback, image)
        self.assertTrue(
            image.startswith('mirror.gcr.io/library/alpine:'))

    def test_scan_outage_does_not_gate_functional_upgrade(self):
        root = Path(__file__).resolve().parents[2]
        workflow = yaml.safe_load(
            (root / '.github/workflows/cache-rolling-upgrade.yml').read_text())
        functional = workflow['jobs']['cache-rolling-upgrade']
        self.assertNotIn('needs', functional)
        self.assertFalse(
            any('scan_helper.py' in step.get('run', '')
                for step in functional['steps']))
        self.assertTrue(
            any('scan_helper.py' in step.get('run', '')
                for step in workflow['jobs']['cache-helper-scan']['steps']))


if __name__ == '__main__':
    unittest.main()

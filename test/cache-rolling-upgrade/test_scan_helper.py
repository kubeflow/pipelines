"""Unknown package inventory must not become a clean vulnerability result."""
import unittest

import scan_helper


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


if __name__ == '__main__':
    unittest.main()

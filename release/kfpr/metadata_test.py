#!/usr/bin/env python3
"""Tests for release version metadata."""

import unittest

from kfpr.core import ReleaseMetadata


class TestReleaseMetadata(unittest.TestCase):
    def test_from_version_major_release(self):
        metadata = ReleaseMetadata.from_version('major', '2.0.0')
        self.assertEqual(metadata.release_type, 'major')
        self.assertEqual(metadata.version, '2.0.0')
        self.assertEqual(metadata.major, 2)
        self.assertEqual(metadata.minor, 0)
        self.assertEqual(metadata.patch, 0)
        self.assertEqual(metadata.tag, '2.0.0')
        self.assertEqual(metadata.sdk_tag, 'sdk-2.0.0')
        self.assertEqual(metadata.release_branch, 'release-2.0')
        self.assertEqual(metadata.patch_branch, None)

    def test_from_version_minor_release(self):
        metadata = ReleaseMetadata.from_version('minor', '1.3.0')
        self.assertEqual(metadata.release_type, 'minor')
        self.assertEqual(metadata.version, '1.3.0')
        self.assertEqual(metadata.major, 1)
        self.assertEqual(metadata.minor, 3)
        self.assertEqual(metadata.patch, 0)
        self.assertEqual(metadata.tag, '1.3.0')
        self.assertEqual(metadata.sdk_tag, 'sdk-1.3.0')
        self.assertEqual(metadata.release_branch, 'release-1.3')
        self.assertEqual(metadata.patch_branch, None)

    def test_from_version_patch_release(self):
        metadata = ReleaseMetadata.from_version('patch', '1.3.2')
        self.assertEqual(metadata.release_type, 'patch')
        self.assertEqual(metadata.version, '1.3.2')
        self.assertEqual(metadata.major, 1)
        self.assertEqual(metadata.minor, 3)
        self.assertEqual(metadata.patch, 2)
        self.assertEqual(metadata.tag, '1.3.2')
        self.assertEqual(metadata.sdk_tag, 'sdk-1.3.2')
        self.assertEqual(metadata.release_branch, 'release-1.3')
        self.assertEqual(metadata.patch_branch, 'release-1.3.2')

    def test_from_version_invalid_release_type(self):
        with self.assertRaises(ValueError):
            ReleaseMetadata.from_version('invalid', '1.0.0')

    def test_from_version_invalid_semver_format(self):
        with self.assertRaises(ValueError):
            ReleaseMetadata.from_version('major', '1.0')

    def test_prerelease_versions_remain_outside_cli_contract(self):
        with self.assertRaises(ValueError):
            ReleaseMetadata.from_version('major', '3.0.0-rc.1')

    def test_from_version_major_not_x_0_0(self):
        with self.assertRaises(ValueError):
            ReleaseMetadata.from_version('major', '1.1.0')

    def test_from_version_minor_not_x_y_0(self):
        with self.assertRaises(ValueError):
            ReleaseMetadata.from_version('minor', '1.3.1')


if __name__ == '__main__':
    unittest.main()

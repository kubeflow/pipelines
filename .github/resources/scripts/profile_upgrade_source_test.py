# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0

import json
import subprocess
import unittest
from pathlib import Path

SELECTOR = Path(__file__).with_name('profile-upgrade-source.jq')


class ProfileUpgradeSourceTest(unittest.TestCase):
    def select(self, target, *pages):
        return subprocess.run(
            ['jq', '-ser', '--arg', 'target', target, '-f', str(SELECTOR)],
            input='\n'.join(json.dumps(page) for page in pages),
            text=True,
            capture_output=True,
        )

    def test_selects_latest_published_patch_across_pages(self):
        stable = {'draft': False, 'prerelease': False}
        first = [
            dict(stable, tag_name='2.18.12', draft=True),
            dict(stable, tag_name='2.18.11', prerelease=True),
            dict(stable, tag_name='2.18.13-rc.1'),
            dict(stable, tag_name='sdk-2.18.14'),
            dict(stable, tag_name='2.19.0'),
            dict(stable, tag_name='2.18.9'),
        ]
        second = [dict(stable, tag_name='2.18.10')]
        for target, expected in [
            ('', '2.18.10'),
            ('candidate', '2.18.10'),
            ('2.18.10', '2.18.9'),
            ('2.18.10-rc.1', '2.18.9'),
        ]:
            with self.subTest(target=target):
                result = self.select(target, first, second)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.strip(), expected)

    def test_requires_an_older_published_patch(self):
        candidate = {'tag_name': '2.18.2', 'draft': False, 'prerelease': False}
        result = self.select('2.18.2', [candidate])
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('No eligible published 2.18 patch', result.stderr)

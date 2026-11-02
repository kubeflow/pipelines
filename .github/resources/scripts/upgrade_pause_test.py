"""Keep migration-dependent upgrade jobs paused until #14029 is delivered."""

import unittest
from pathlib import Path

import yaml


class UpgradePauseTest(unittest.TestCase):
    def test_all_upgrade_jobs_have_a_checked_in_pause(self):
        root = Path(__file__).resolve().parents[3]
        workflow = (root / '.github/workflows/upgrade-test.yml').read_text()
        jobs = yaml.safe_load(workflow)['jobs']
        self.assertEqual(set(jobs), {'upgrade-test'})
        for job in jobs.values():
            self.assertEqual(job['if'], '${{ false }}')
        self.assertIn('TODO(#14029)', workflow)

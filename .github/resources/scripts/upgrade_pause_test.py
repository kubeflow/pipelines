"""Keep migration-dependent upgrade jobs paused until #14029 is delivered."""

from pathlib import Path
import unittest


class UpgradePauseTest(unittest.TestCase):

    def test_both_upgrade_jobs_have_a_checked_in_pause(self):
        root = Path(__file__).resolve().parents[3]
        workflow = (root / '.github/workflows/upgrade-test.yml').read_text()
        gate = '    if: ${{ false }}'
        for name in ('build', 'upgrade-test'):
            job = workflow.split(f'  {name}:\n', 1)[1]
            self.assertEqual(job.splitlines()[1], gate)
        self.assertEqual(workflow.count(gate), 2)
        self.assertIn('TODO(#14029)', workflow)

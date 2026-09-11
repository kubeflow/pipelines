"""Keep migration-dependent upgrade jobs opt-in until #14029 is delivered."""

from pathlib import Path
import unittest


class UpgradePauseTest(unittest.TestCase):

    def test_both_upgrade_jobs_require_explicit_opt_in(self):
        root = Path(__file__).resolve().parents[3]
        workflow = (root / '.github/workflows/upgrade-test.yml').read_text()
        gate = "    if: ${{ vars.KFP_ENABLE_MLMD_UPGRADE_TESTS == 'true' }}"
        for name in ('build', 'upgrade-test'):
            job = workflow.split(f'  {name}:\n', 1)[1]
            self.assertEqual(job.splitlines()[1], gate)
        self.assertEqual(workflow.count(gate), 2)
        self.assertIn('TODO(#14029)', workflow)

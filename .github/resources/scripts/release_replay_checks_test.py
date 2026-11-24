"""Exercise the landed release gate against exact-commit external checks."""

import json
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SHA = 'a' * 40


def assess(conclusion, status='success'):
    program = r"""
const {checkLandedCommit} = require('./.github/resources/scripts/release_replay_checks');
const {sha, conclusion, status} = JSON.parse(require('node:fs').readFileSync(0, 'utf8'));
const check = {id: 30, app: {id: 99, slug: 'quality-gate'}, name: 'External review',
  check_suite: {id: 50}, head_sha: sha, status: conclusion ? 'completed' : 'in_progress', conclusion};
const requested = [];
async function read(path) {
  requested.push(path);
  if (path === `commits/${sha}/check-runs?filter=all&per_page=100&page=1`) {
    return {total_count: 1, check_runs: [check]};
  }
  if (path === `actions/runs?head_sha=${sha}&per_page=100&page=1`) {
    return {total_count: 0, workflow_runs: []};
  }
  if (path === `commits/${sha}/status?per_page=100&page=1`) {
    return {sha, total_count: 2, statuses: [
      {context: 'tide', state: 'pending', target_url: 'https://oss.gprow.dev/tide'},
      {context: 'ci-passed', state: status}]};
  }
  throw Error('Unexpected check route: ' + path);
}
checkLandedCommit(sha, 12, read).then(() => console.log(JSON.stringify({passed: true, requested})),
  e => console.log(JSON.stringify({passed: false, requested, reason: e.message})));
"""
    return json.loads(
        subprocess.check_output(
            ['node', '-e', program],
            cwd=ROOT,
            input=json.dumps(
                {'sha': SHA, 'conclusion': conclusion, 'status': status}
            ),
            text=True,
        )
    )


class ReleaseReplayChecksTest(unittest.TestCase):
    def test_failed_or_pending_external_check_blocks_landed_ci(self):
        for conclusion in ('failure', None):
            with self.subTest(conclusion=conclusion):
                result = assess(conclusion)
                self.assertFalse(result['passed'])
                self.assertIn('Check External review', result['reason'])
        self.assertTrue(assess('success')['passed'])

    def test_failed_or_pending_aggregate_blocks_landed_ci(self):
        for status in ('failure', 'pending'):
            with self.subTest(status=status):
                result = assess('success', status)
                self.assertFalse(result['passed'])
                self.assertIn('ci-passed', result['reason'])

"""Keep master migration opt-in while permitting MLMD-retaining release
tests."""

import ast
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/upgrade-test.yml'


def job_enabled(name,
                base='',
                ref='',
                migration=False,
                readiness=False,
                dispatch=False):
    text = WORKFLOW.read_text()
    job = re.search(r'^  ' + re.escape(name) + r':\n(.*?)(?=^  [a-zA-Z]|\Z)',
                    text, re.M | re.S).group(1)
    expression = re.search(r'\$\{\{(.*?)\}\}', job, re.S).group(1)
    values = {
        'github.base_ref': base,
        'github.ref': ref,
        'vars.KFP_ENABLE_MLMD_UPGRADE_TESTS': 'true' if migration else '',
        'vars.KFP_ENABLE_READINESS_SCHEDULE_TESTS': 'true' if readiness else '',
        'inputs.readiness_schedules': dispatch,
    }
    for key, value in values.items():
        expression = expression.replace(key, repr(value))
    expression = expression.replace('||', ' or ').replace('&&', ' and ')
    expression = re.sub(r'\btrue\b(?![\'"])', 'True', expression)
    tree = ast.parse(' '.join(expression.split()), mode='eval')
    # These gates intentionally use only boolean operators and equality. Reject
    # unsupported syntax instead of silently modeling GitHub expressions poorly.
    allowed = (ast.Expression, ast.BoolOp, ast.And, ast.Or, ast.Compare, ast.Eq,
               ast.Constant)
    if any(not isinstance(node, allowed) for node in ast.walk(tree)):
        raise ValueError('Unsupported upgrade job condition')
    return eval(compile(tree, str(WORKFLOW), 'eval'), {'__builtins__': {}})


class UpgradePauseTest(unittest.TestCase):

    def test_persistence_jobs_preserve_master_gate_and_allow_release(self):
        for name in ('build', 'upgrade-test'):
            for context in ({
                    'ref': 'refs/heads/master'
            }, {
                    'base': 'master'
            }, {
                    'ref': 'refs/heads/feature'
            }, {}):
                with self.subTest(job=name, context=context):
                    self.assertFalse(job_enabled(name, **context))
                    self.assertTrue(
                        job_enabled(name, migration=True, **context))
            self.assertTrue(job_enabled(name, ref='refs/heads/release-2.18'))
            self.assertTrue(job_enabled(name, base='release-2.18'))

    def test_readiness_requires_release_context_and_separate_opt_in(self):
        for context in ({
                'ref': 'refs/heads/release-2.18'
        }, {
                'base': 'release-2.18'
        }):
            self.assertFalse(job_enabled('readiness-schedules', **context))
            self.assertFalse(
                job_enabled('readiness-schedules', migration=True, **context))
            self.assertTrue(
                job_enabled('readiness-schedules', readiness=True, **context))
            self.assertTrue(
                job_enabled('readiness-schedules', dispatch=True, **context))
        for context in ({'ref': 'refs/heads/master'}, {'base': 'master'}, {}):
            self.assertFalse(
                job_enabled(
                    'readiness-schedules',
                    readiness=True,
                    dispatch=True,
                    migration=True,
                    **context))


if __name__ == '__main__':
    unittest.main()

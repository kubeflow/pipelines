"""Keep every kfpr test module in a CI unittest command."""

import re
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
KFPR_DIR = ROOT / 'release/kfpr'
WORKFLOWS_DIR = ROOT / '.github/workflows'
# Matches kfpr.<module>_test and release.kfpr.<module>_test.
MODULE_RE = re.compile(r'(?<![\w.])(?:release\.)?kfpr\.(\w+_test)\b')


def kfpr_test_modules():
    return {path.stem for path in KFPR_DIR.glob('*_test.py')}


def workflow_run_commands():
    for path in sorted(WORKFLOWS_DIR.glob('*.y*ml')):
        workflow = yaml.safe_load(path.read_text(encoding='utf-8')) or {}
        for job in (workflow.get('jobs') or {}).values():
            for step in job.get('steps') or []:
                command = step.get('run')
                if command:
                    yield path.name, command


def modules_run_in_ci():
    modules = {}
    for workflow, command in workflow_run_commands():
        if 'unittest' not in command:
            continue
        for name in MODULE_RE.findall(command):
            modules.setdefault(name, set()).add(workflow)
    return modules


class KfprTestCoverageTest(unittest.TestCase):
    def test_every_kfpr_test_module_runs_in_ci(self):
        missing = sorted(kfpr_test_modules() - set(modules_run_in_ci()))

        self.assertEqual(
            missing,
            [],
            'Add these release/kfpr test modules to a unittest command in '
            '.github/workflows/ci-scripts-tests.yml or '
            'readthedocs-builds.yml so CI runs them.',
        )

    def test_ci_runs_only_existing_kfpr_test_modules(self):
        stale = {
            name: sorted(workflows)
            for name, workflows in modules_run_in_ci().items()
            if name not in kfpr_test_modules()
        }

        self.assertEqual(
            stale,
            {},
            'These kfpr test modules no longer exist under release/kfpr.',
        )


if __name__ == '__main__':
    unittest.main()

"""Keep parallel frontend validation complete and fail closed."""

import json
import os
import subprocess
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]


class FrontendCITest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workflow = yaml.safe_load(
            (ROOT / '.github/workflows/frontend.yml').read_text()
        )
        cls.scripts = json.loads((ROOT / 'frontend/package.json').read_text())[
            'scripts'
        ]
        cls.job = cls.workflow['jobs']['frontend-checks']

    def test_all_validation_lanes_run_without_fail_fast(self):
        self.assertFalse(self.job['strategy']['fail-fast'])
        self.assertNotIn('needs', self.job)
        lanes = set(self.job['strategy']['matrix']['check'])
        self.assertEqual(lanes, {'static', 'ui', 'server', 'bundle'})
        commands = {
            lane: '\n'.join(
                step.get('run', '')
                for step in self.job['steps']
                if step.get('if') == f"matrix.check == '{lane}'"
            )
            for lane in lanes
        }
        for command in (
            'npm run test:static',
            'npm run apis:all',
            'git diff --exit-code',
            'bash scripts/check-spec-generation.sh',
        ):
            self.assertIn(command, commands['static'])
        self.assertIn('npm run test:ui:coverage', commands['ui'])
        self.assertIn('npm run test:server:coverage', commands['server'])
        self.assertIn('npm run test:bundle:ci', commands['bundle'])
        self.assertIn(
            'playwright install --with-deps chromium', commands['bundle']
        )
        self.assertEqual(
            self.scripts['test:static'].split(' && '),
            [
                f'npm run {name}'
                for name in (
                    'format:check',
                    'lint',
                    'typecheck',
                    'typecheck:mock-backend',
                    'check:react-peers',
                )
            ],
        )
        python_setup = next(
            step
            for step in self.job['steps']
            if step.get('uses', '').startswith('actions/setup-python@')
        )
        self.assertEqual(python_setup['if'], "matrix.check == 'server'")

    def test_bundle_smoke_keeps_build_without_duplicate_static_checks(self):
        script = self.scripts['test:bundle:ci']
        self.assertIn('npm run build:tailwind', script)
        self.assertIn('vite build', script)
        self.assertIn('node --test scripts/production-bundle.smoke.mjs', script)
        self.assertNotIn('npm run lint', script)
        self.assertNotIn('npm run typecheck', script)
        # Developer builds retain validation and their existing smoke entrypoint.
        self.assertIn(
            'npm run lint && npm run typecheck', self.scripts['build']
        )
        self.assertTrue(
            self.scripts['test:bundle'].startswith('npm run build &&')
        )
        self.assertIn('npm run test:static', self.scripts['test:ci'])
        self.assertIn('npm run test:coverage', self.scripts['test:ci'])

    def test_npm_cache_tracks_all_installed_trees(self):
        setup = next(
            step
            for step in self.job['steps']
            if step.get('uses', '').startswith('actions/setup-node@')
        )['with']
        self.assertEqual(setup['cache'], 'npm')
        self.assertEqual(
            set(setup['cache-dependency-path'].split()),
            {
                'frontend/package-lock.json',
                'frontend/server/package-lock.json',
                'frontend/mock-backend/package-lock.json',
            },
        )
        install = next(
            step
            for step in self.job['steps']
            if step.get('name') == 'Install dependencies'
        )['run']
        self.assertTrue(install.startswith('npm ci ||'))
        self.assertIn('npm cache clean --force && npm ci', install)
        self.assertFalse(
            any(
                step.get('run', '').strip() == 'npm cache clean --force'
                for step in self.job['steps']
            )
        )

    def test_aggregate_rejects_failed_cancelled_and_skipped_lanes(self):
        gate = self.workflow['jobs']['frontend-tests']
        self.assertEqual(gate['needs'], 'frontend-checks')
        self.assertEqual(gate['if'], 'always()')
        step = gate['steps'][0]
        self.assertEqual(
            step['env']['CHECK_RESULT'], '${{ needs.frontend-checks.result }}'
        )
        for result in ('success', 'failure', 'cancelled', 'skipped', ''):
            with self.subTest(result=result):
                completed = subprocess.run(
                    ['bash', '-c', step['run']],
                    env={**os.environ, 'CHECK_RESULT': result},
                    check=False,
                )
                self.assertEqual(completed.returncode == 0, result == 'success')


if __name__ == '__main__':
    unittest.main()

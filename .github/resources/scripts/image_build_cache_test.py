"""Contracts for reusable image caches and source-independent npm layers."""

import json
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]


class ImageBuildCacheTest(unittest.TestCase):
    def test_backend_cache_is_scoped_and_recovery_does_not_use_it(self):
        jobs = yaml.safe_load(
            (ROOT / '.github/workflows/image-builds.yml').read_text()
        )['jobs']
        steps = {step.get('id'): step for step in jobs['image-build']['steps']}
        cached = steps['save-image']['with']
        for option in ('cache-from', 'cache-to'):
            self.assertIn('version=2', cached[option])
            self.assertIn('${{ matrix.image }}', cached[option])
            self.assertIn('${{ inputs.architecture }}', cached[option])
            self.assertIn('timeout=', cached[option])
        self.assertIn('ignore-error=true', cached['cache-to'])
        retry = steps['rebuild']['with']
        self.assertTrue(retry['no-cache'])
        self.assertNotIn('cache-from', retry)
        self.assertNotIn('cache-to', retry)

    def test_frontend_cache_credentials_are_exposed_before_shell_builder(self):
        jobs = yaml.safe_load(
            (ROOT / '.github/workflows/image-builds.yml').read_text()
        )['jobs']
        steps = jobs['image-build']['steps']
        build_index = next(
            i
            for i, step in enumerate(steps)
            if step.get('id') == 'frontend-image'
        )
        credentials = steps[build_index - 1]
        script = credentials['with']['script']
        self.assertIn('ACTIONS_RUNTIME_TOKEN', script)
        self.assertIn('ACTIONS_RESULTS_URL', script)
        self.assertIn('core.setSecret(value)', script)
        self.assertIn('core.exportVariable(name, value)', script)
        command = steps[build_index]['run']
        self.assertIn('--cache-from', command)
        self.assertIn('--cache-to', command)
        self.assertIn(
            'scope=kfp-image-frontend-${{ inputs.architecture }}', command
        )
        self.assertIn('ignore-error=true', command)

    def test_npm_install_inputs_precede_source_copy(self):
        dockerfile = (ROOT / 'frontend/Dockerfile').read_text()
        dependencies, source = dockerfile.split('COPY . .', maxsplit=1)
        self.assertIn('RUN (npm ci ||', dependencies)
        for package in ('frontend', 'frontend/server', 'frontend/mock-backend'):
            for name in ('package.json', 'package-lock.json'):
                path = f'{package}/{name}'
                self.assertIn(path, dependencies)
                self.assertTrue((ROOT / path).is_file(), path)
        self.assertIn('frontend/.npmrc', dependencies)
        self.assertIn('RUN npm run build', source)
        package = json.loads((ROOT / 'frontend/package.json').read_text())
        self.assertEqual(
            package['scripts']['postinstall'],
            'npm ci --prefix ./server && npm ci --prefix ./mock-backend',
        )


if __name__ == '__main__':
    unittest.main()

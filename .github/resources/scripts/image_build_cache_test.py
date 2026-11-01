"""Contracts for reusable image caches and source-independent npm layers."""

import json
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]


class ImageBuildCacheTest(unittest.TestCase):
    def test_exports_default_to_read_only_with_one_default_branch_writer(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/image-builds.yml').read_text()
        )
        self.assertIs(
            workflow[True]['workflow_call']['inputs']['cache-write']['default'],
            False,
        )
        writers = []
        for path in (ROOT / '.github/workflows').iterdir():
            if path.suffix not in {'.yml', '.yaml'}:
                continue
            for name, job in yaml.safe_load(path.read_text())['jobs'].items():
                if job.get('uses') != './.github/workflows/image-builds.yml':
                    continue
                if 'cache-write' in job.get('with', {}):
                    writers.append(
                        (path.name, name, job['with']['cache-write'])
                    )
        self.assertEqual(
            writers,
            [
                (
                    'ci-images.yml',
                    'amd64',
                    "${{ github.event_name == 'push' && github.ref == "
                    "format('refs/heads/{0}', github.event.repository.default_branch) }}",
                )
            ],
        )

    def test_backend_cache_is_scoped_and_recovery_does_not_use_it(self):
        jobs = yaml.safe_load(
            (ROOT / '.github/workflows/image-builds.yml').read_text()
        )['jobs']
        job = jobs['image-build']
        steps = {step.get('id'): step for step in job['steps']}
        cached = steps['save-image']['with']
        cache_to = job['env']['IMAGE_CACHE_TO']
        self.assertTrue(
            cache_to.startswith('${{ inputs.cache-write && format(')
        )
        self.assertTrue(cache_to.endswith(" || '' }}"))
        self.assertIn('scope=kfp-image-{0}-{1}', cache_to)
        self.assertIn('matrix.image, inputs.architecture', cache_to)
        self.assertIn('ignore-error=true', cache_to)
        self.assertEqual(cached['cache-to'], '${{ env.IMAGE_CACHE_TO }}')
        self.assertIn('version=2', cached['cache-from'])
        self.assertIn('${{ matrix.image }}', cached['cache-from'])
        self.assertIn('${{ inputs.architecture }}', cached['cache-from'])
        self.assertIn('timeout=', cached['cache-from'])
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
        self.assertIn('${IMAGE_CACHE_TO:+--cache-to', command)

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

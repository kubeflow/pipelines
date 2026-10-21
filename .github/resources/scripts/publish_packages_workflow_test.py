#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Keep package builds outside the PyPI publisher's OIDC identity."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from artifact_workflow_test_support import evaluate
import yaml

ROOT = Path(__file__).resolve().parents[3]


class PublishPackagesWorkflowTest(unittest.TestCase):

    def setUp(self):
        self.workflow = yaml.safe_load(
            (ROOT / '.github/workflows/publish-packages.yml').read_text())
        self.jobs = self.workflow['jobs']
        self.build = self.jobs['build']
        self.publish = self.jobs['publish']
        self.prepare = next(
            step for step in self.build['steps']
            if step.get('name') == 'Prepare distributions for publication')
        self.verify = next(
            step for step in self.publish['steps']
            if step.get('name') == 'Verify distributions before publishing')

    def step_using(self, job, action):
        return next(step for step in job['steps']
                    if step.get('uses', '').split('@')[0] == action)

    def test_source_execution_never_receives_publishing_permissions(self):
        for job_id in ('select-packages', 'build'):
            with self.subTest(job=job_id):
                job = self.jobs[job_id]
                self.assertEqual(job['permissions'], {'contents': 'read'})
                checkout = self.step_using(job, 'actions/checkout')
                self.assertIs(checkout['with']['persist-credentials'], False)
                self.assertEqual(checkout['with']['ref'],
                                 '${{ github.event.inputs.tag }}')
        self.assertEqual(self.publish['permissions'], {'id-token': 'write'})
        self.assertEqual([
            step['uses'].split('@')[0]
            for step in self.publish['steps']
            if 'uses' in step
        ], ['actions/download-artifact', 'pypa/gh-action-pypi-publish'])
        self.assertEqual(
            [step for step in self.publish['steps'] if 'run' in step],
            [self.verify])
        self.assertIn('python3 -I -', self.verify['run'])

    def test_dry_runs_cannot_start_oidc_job_and_keep_distribution_artifacts(
            self):
        upload_steps = [
            step for step in self.build['steps']
            if step.get('uses', '').startswith('actions/upload-artifact@')
        ]
        self.assertEqual(len(upload_steps), 2)
        for dry_run in ('true', 'false'):
            with self.subTest(dry_run=dry_run):
                context = {
                    'github': {
                        'event': {
                            'inputs': {
                                'dry_run': dry_run
                            }
                        }
                    }
                }
                self.assertEqual(
                    evaluate(self.publish['if'], context), dry_run == 'false')
                self.assertEqual(
                    evaluate(self.prepare['if'], context), dry_run == 'false')
                selected = [
                    step for step in upload_steps
                    if evaluate(step['if'], context)
                ]
                self.assertEqual(len(selected), 1)
                upload = selected[0]['with']
                expected = '${{ matrix.package.name }}-dist'
                if dry_run == 'false':
                    expected = '${{ matrix.package.name }}-publish-dist'
                    self.assertIs(upload['overwrite'], True)
                else:
                    self.assertNotIn('overwrite', upload)
                self.assertEqual(upload['name'], expected)
                self.assertEqual(
                    upload['path'],
                    '${{ matrix.package.path }}/dist/' if dry_run == 'true' else
                    '${{ runner.temp }}/package-artifact/')
                self.assertEqual(upload['if-no-files-found'], 'error')

    def test_publisher_requires_all_selected_builds_and_exact_same_run_artifact(
            self):
        self.assertEqual(
            set(self.publish['needs']), {'select-packages', 'build'})
        self.assertEqual(self.build['needs'], 'select-packages')
        self.assertEqual(self.build['strategy']['matrix'],
                         self.publish['strategy']['matrix'])
        self.assertEqual(
            self.publish['strategy']['matrix']['package'],
            '${{ fromJSON(needs.select-packages.outputs.packages) }}')
        download = self.step_using(self.publish,
                                   'actions/download-artifact')['with']
        self.assertEqual(download['name'],
                         '${{ matrix.package.name }}-publish-dist')
        for cross_run_input in ('github-token', 'run-id', 'repository',
                                'pattern'):
            self.assertNotIn(cross_run_input, download)
        publisher = self.step_using(self.publish,
                                    'pypa/gh-action-pypi-publish')['with']
        self.assertEqual(download['path'], 'package-artifact/')
        self.assertEqual(publisher['packages-dir'], 'package-artifact/dist/')
        self.assertIs(publisher['print-hash'], True)
        self.assertNotIn('password', publisher)
        self.assertNotIn('skip-existing', publisher)
        steps = self.publish['steps']
        self.assertLess(
            steps.index(
                self.step_using(self.publish, 'actions/download-artifact')),
            steps.index(self.verify))
        self.assertLess(
            steps.index(self.verify),
            steps.index(
                self.step_using(self.publish, 'pypa/gh-action-pypi-publish')))
        for step in steps:
            self.assertNotIn('continue-on-error', step)
            self.assertNotIn('if', step)

    def run_step(self, step, root):
        return subprocess.run(
            ['bash', '-e', '-o', 'pipefail', '-c', step['run']],
            cwd=root,
            env={
                'PATH':
                    f'{Path(sys.executable).parent}{os.pathsep}{os.environ["PATH"]}',
                'PACKAGE_PATH':
                    'source',
                'RUNNER_TEMP':
                    str(root),
            },
            capture_output=True,
            text=True,
            check=False)

    def distributions(self, root, name='kfp'):
        distributions = root / 'source/dist'
        distributions.mkdir(parents=True)
        (distributions /
         f'{name}-1.0-py3-none-any.whl').write_bytes(b'wheel fixture')
        (distributions / f'{name}-1.0.tar.gz').write_bytes(b'sdist fixture')
        return distributions

    def test_package_artifact_preserves_all_distribution_bytes(self):
        for name in ('kfp', 'kfp_pipeline_spec', 'kfp_server_api',
                     'kfp_kubernetes'):
            with self.subTest(
                    package=name), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                source = self.distributions(root, name)
                prepared = self.run_step(self.prepare, root)
                self.assertEqual(prepared.returncode, 0, prepared.stderr)
                verified = self.run_step(self.verify, root)
                self.assertEqual(verified.returncode, 0, verified.stderr)
                artifact = root / 'package-artifact'
                self.assertEqual({path.name for path in artifact.iterdir()},
                                 {'dist', 'manifest.json'})
                self.assertEqual(
                    {path.name for path in (artifact / 'dist').iterdir()},
                    {path.name for path in source.iterdir()})
                for path in source.iterdir():
                    self.assertEqual(
                        (artifact / 'dist' / path.name).read_bytes(),
                        path.read_bytes())

    def test_uv_gitignore_stays_outside_publication_artifact(self):
        for name in ('kfp', 'kfp_pipeline_spec', 'kfp_server_api',
                     'kfp_kubernetes'):
            with self.subTest(
                    package=name), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                source = self.distributions(root, name)
                expected = {
                    path.name: path.read_bytes() for path in source.iterdir()
                }
                (source / '.gitignore').write_text('*')
                prepared = self.run_step(self.prepare, root)
                self.assertEqual(prepared.returncode, 0, prepared.stderr)
                artifact = root / 'package-artifact'
                self.assertEqual(
                    {
                        path.name: path.read_bytes()
                        for path in (artifact / 'dist').iterdir()
                    }, expected)
                self.assertEqual(
                    set(json.loads((artifact / 'manifest.json').read_text())),
                    set(expected))
                verified = self.run_step(self.verify, root)
                self.assertEqual(verified.returncode, 0, verified.stderr)

    def test_build_rejects_missing_empty_or_irrelevant_distributions(self):
        for invalid in ('missing-wheel', 'missing-sdist', 'empty', 'extra',
                        'symlink', 'directory', 'gitignore-symlink',
                        'gitignore-directory', 'extra-dotfile'):
            with self.subTest(invalid=invalid), tempfile.TemporaryDirectory(
            ) as directory:
                root = Path(directory)
                source = self.distributions(root)
                wheel = source / 'kfp-1.0-py3-none-any.whl'
                if invalid == 'missing-wheel':
                    wheel.unlink()
                elif invalid == 'missing-sdist':
                    (source / 'kfp-1.0.tar.gz').unlink()
                elif invalid == 'empty':
                    wheel.write_bytes(b'')
                elif invalid == 'extra':
                    (source / 'unexpected.txt').write_text('unexpected')
                elif invalid == 'symlink':
                    wheel.unlink()
                    wheel.symlink_to('kfp-1.0.tar.gz')
                elif invalid == 'gitignore-symlink':
                    (source / '.gitignore').symlink_to(wheel.name)
                elif invalid == 'gitignore-directory':
                    (source / '.gitignore').mkdir()
                elif invalid == 'extra-dotfile':
                    (source / '.unexpected').write_text('unexpected')
                else:
                    (source / 'unexpected.whl').mkdir()
                result = self.run_step(self.prepare, root)
                self.assertNotEqual(result.returncode, 0)

    def test_publisher_rejects_incomplete_or_changed_artifacts(self):
        for invalid in ('missing-wheel', 'missing-sdist',
                        'missing-wheel-with-manifest',
                        'missing-sdist-with-manifest', 'changed', 'extra',
                        'extra-with-manifest', 'empty', 'missing-manifest',
                        'malformed-manifest', 'symlink', 'extra-top-level'):
            with self.subTest(invalid=invalid), tempfile.TemporaryDirectory(
            ) as directory:
                root = Path(directory)
                self.distributions(root)
                prepared = self.run_step(self.prepare, root)
                self.assertEqual(prepared.returncode, 0, prepared.stderr)
                artifact = root / 'package-artifact'
                wheel = artifact / 'dist/kfp-1.0-py3-none-any.whl'
                manifest = artifact / 'manifest.json'
                if invalid.startswith('missing-wheel'):
                    wheel.unlink()
                elif invalid.startswith('missing-sdist'):
                    (artifact / 'dist/kfp-1.0.tar.gz').unlink()
                elif invalid in ('changed', 'empty'):
                    wheel.write_bytes(b'changed' if invalid ==
                                      'changed' else b'')
                elif invalid in ('extra', 'extra-with-manifest'):
                    (artifact / 'dist/unexpected.txt').write_text('unexpected')
                    if invalid == 'extra-with-manifest':
                        data = json.loads(manifest.read_text())
                        data['unexpected.txt'] = '0' * 64
                        manifest.write_text(json.dumps(data))
                elif invalid == 'missing-manifest':
                    manifest.unlink()
                elif invalid == 'malformed-manifest':
                    manifest.write_text('[]')
                elif invalid == 'symlink':
                    wheel.unlink()
                    wheel.symlink_to(
                        '../../source/dist/kfp-1.0-py3-none-any.whl')
                else:
                    (artifact / 'unexpected').write_text('unexpected')
                if invalid in ('missing-wheel-with-manifest',
                               'missing-sdist-with-manifest'):
                    data = json.loads(manifest.read_text())
                    filename = ('kfp-1.0-py3-none-any.whl'
                                if invalid.startswith('missing-wheel') else
                                'kfp-1.0.tar.gz')
                    data.pop(filename)
                    manifest.write_text(json.dumps(data))
                result = self.run_step(self.verify, root)
                self.assertNotEqual(result.returncode, 0)


if __name__ == '__main__':
    unittest.main()

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

import fnmatch
from pathlib import Path
import shlex
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]


class PythonPackagingTest(unittest.TestCase):

    def test_docs_resolve_unpublished_server_api_with_source_sdk(self) -> None:
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/docs.yml').read_text())
        for name in ('build', 'linkcheck'):
            with self.subTest(job=name):
                steps = workflow['jobs'][name]['steps']
                install = next(
                    i for i, step in enumerate(steps)
                    if 'docs/requirements.txt' in step.get('run', ''))
                self.assertTrue(
                    any(
                        step.get('uses') == './.github/actions/protobuf'
                        for step in steps[:install]))
                args = shlex.split(steps[install]['run'])
                self.assertIn('./backend/api/v2beta1/python_http_client', args)
                self.assertNotIn('--no-deps', args)

    def test_sdk_client_installs_companion_before_pinned_requirements(
            self) -> None:
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/kfp-sdk-client-tests.yml').read_text())
        steps = workflow['jobs']['sdk-client-tests']['steps']
        resolve = next(
            i for i, step in enumerate(steps)
            if '-r sdk/python/requirements.txt' in step.get('run', ''))
        local_api_installs = [
            i for i, step in enumerate(steps) if step.get('working-directory')
            == 'backend/api/v2beta1/python_http_client' and
            'pip install dist/*.whl' in step.get('run', '')
        ]
        self.assertTrue(local_api_installs, 'No local server API installation')
        self.assertLess(min(local_api_installs), resolve)

    def test_dra_host_and_component_image_receive_unpublished_companions(
            self) -> None:
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/e2e-test.yml').read_text())
        steps = [
            step for job in workflow['jobs'].values()
            for step in job.get('steps', [])
        ]
        build = next(step['run']
                     for step in steps
                     if 'dra-component-build' in step.get('run', ''))
        commands = [
            shlex.split(line, comments=True)
            for line in build.replace('\\\n', ' ').splitlines()
            if line.strip()
        ]
        sdk_install = next(args for args in commands
                           if args[:4] == ['python', '-m', 'pip', 'install'] and
                           'sdk/python' in args)
        api = 'backend/api/v2beta1/python_http_client'
        self.assertIn(api, sdk_install)
        component_build = next(
            args for args in commands if 'kfp' in args and 'component' in args)
        extras = [
            component_build[i + 1]
            for i, arg in enumerate(component_build)
            if arg == '--additional-package-path'
        ]
        for path in ('api/v2alpha1/python', api, 'kubernetes_platform/python'):
            self.assertIn('${GITHUB_WORKSPACE}/' + path, extras)

    def test_package_setup_workflows_trigger_regression_checks(self) -> None:
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/ci-scripts-tests.yml').read_text())
        paths = workflow[True]['pull_request']['paths']
        for changed in ('VERSION', '.github/workflows/docs.yml',
                        '.github/workflows/kfp-sdk-client-tests.yml',
                        '.github/workflows/e2e-test.yml'):
            with self.subTest(changed=changed):
                self.assertTrue(
                    any(
                        fnmatch.fnmatchcase(changed, pattern)
                        for pattern in paths))

    def test_readthedocs_bounds_font_download_concurrency(self) -> None:
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/readthedocs-builds.yml').read_text())
        job = workflow['jobs']['test-readthedocs-builds']
        variable = 'SPHINX_IMMATERIAL_FONT_FETCH_MAX_WORKERS'
        environment = job.get('env', {})
        self.assertEqual(int(environment.get(variable, 128)), 4)
        self.assertFalse(job.get('continue-on-error', False))
        builds = [
            step for step in job['steps']
            if step['name'] in ('Build KFP SDK Docs',
                                'Build KFP Kubernetes SDK Docs')
        ]
        self.assertEqual(len(builds), 2)
        for step in builds:
            with self.subTest(step=step['name']):
                effective = {**environment, **step.get('env', {})}
                self.assertEqual(int(effective[variable]), 4)
                self.assertEqual(step['run'].strip(),
                                 'sphinx-build -b html . _build/html')
                self.assertFalse(step.get('continue-on-error', False))


if __name__ == '__main__':
    unittest.main()

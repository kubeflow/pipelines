# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Ensure nested SDK installs test queue candidates rather than master."""

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml
from artifact_workflow_test_support import render

ROOT = Path(__file__).resolve().parents[3]


class QueueTestSourcesTest(unittest.TestCase):
    def test_workflow_sources_preserve_pr_fallback_and_queue_sha(self):
        sha = 'a' * 40
        cases = [
            ('pull_request', 'refs/pull/7/merge', {}, ''),
            ('push', 'refs/heads/master', {}, ''),
            ('workflow_dispatch', 'refs/heads/master', {}, ''),
            (
                'merge_group',
                'refs/heads/gh-readonly-queue/master/pr-7-test',
                {'merge_group': {'head_sha': sha}},
                sha,
            ),
            (
                'workflow_dispatch',
                'refs/heads/gh-readonly-queue/master/pr-7-test',
                {},
                sha,
            ),
            (
                'workflow_dispatch',
                'refs/heads/gh-readonly-queue/main/pr-7-test',
                {},
                sha,
            ),
        ]
        count = 0
        for filename in [
            'kfp-sdk-tests.yml',
            'kfp-sdk-unit-tests.yml',
            'kfp-kubernetes-native-migration-tests.yaml',
        ]:
            workflow = yaml.safe_load(
                (ROOT / '.github/workflows' / filename).read_text()
            )
            for job in workflow['jobs'].values():
                for step in job.get('steps', []):
                    expression = step.get('env', {}).get('TEST_SOURCE_REF')
                    if expression is None:
                        continue
                    count += 1
                    for event, ref, payload, expected in cases:
                        with self.subTest(
                            workflow=filename,
                            step=step.get('name'),
                            event=event,
                            ref=ref,
                        ):
                            actual = render(
                                expression,
                                {
                                    'github': {
                                        'event_name': event,
                                        'ref': ref,
                                        'sha': sha,
                                        'event': payload,
                                    }
                                },
                            )
                            self.assertEqual(actual, expected)
        self.assertEqual(count, 4)

    def test_sdk_install_sources(self):
        sha = 'a' * 40
        for script in ['presubmit-tests-sdk-unit.sh']:
            for source, pull, expected in [
                (sha, '', '@' + sha),
                (sha, '7', '@' + sha),
                ('', '7', '@refs/pull/7/merge'),
                ('', '', ''),
            ]:
                with self.subTest(script=script, source=source, pull=pull):
                    with tempfile.TemporaryDirectory() as directory:
                        uv = Path(directory) / 'uv'
                        uv.write_text(
                            '#!/bin/sh\n'
                            'if [ "$1" = build ]; then\n'
                            '  while [ "$1" != --out-dir ]; do shift; done\n'
                            '  touch "$2/kfp-test.whl"\n'
                            'else\n'
                            '  printf "%s\\n" "$KFP_PACKAGE_PATH"\n'
                            'fi\n'
                        )
                        uv.chmod(0o755)
                        result = subprocess.run(
                            ['bash', str(ROOT / 'test' / script)],
                            env={
                                **os.environ,
                                'PATH': directory + ':' + os.environ['PATH'],
                                'SETUP_ENV': 'false',
                                'TEST_SOURCE_REF': source,
                                'PULL_NUMBER': pull,
                                'REPO_NAME': 'kubeflow/pipelines',
                            },
                            cwd=ROOT,
                            capture_output=True,
                            text=True,
                            check=True,
                        )
                        self.assertEqual(
                            result.stdout.splitlines()[0],
                            'git+https://github.com/kubeflow/pipelines'
                            + expected
                            + '#egg=kfp&subdirectory=sdk/python',
                        )

    def test_go_suite_receives_queue_source(self):
        action = yaml.safe_load(
            (ROOT / '.github/actions/test-and-report/action.yml').read_text()
        )
        step = next(
            step
            for step in action['runs']['steps']
            if step.get('id') == 'run-tests'
        )
        self.assertEqual(
            step['env']['TEST_SOURCE_REF'],
            '${{ github.event.merge_group.head_sha || '
            "(github.event_name == 'workflow_dispatch' && "
            "startsWith(github.ref, 'refs/heads/gh-readonly-queue/') && "
            "github.sha) || 'master' }}",
        )
        self.assertIn('-branchName="$TEST_SOURCE_REF"', step['run'])


if __name__ == '__main__':
    unittest.main()

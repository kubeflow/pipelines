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

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import textwrap
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
SCRIPT = ROOT / '.github/resources/scripts/download-go-modules.sh'
WORKFLOW = ROOT / '.github/workflows/legacy-v2-api-integration-tests.yml'
ACTION = ROOT / '.github/actions/test-and-report/action.yml'
BACKEND_WORKFLOW = ROOT / '.github/workflows/presubmit-backend.yml'


class GoModuleDownloadRetryTest(unittest.TestCase):

    def setUp(self):
        directory = self.enterContext(tempfile.TemporaryDirectory())
        self.directory = Path(directory)
        self.bin_directory = self.directory / 'bin'
        self.bin_directory.mkdir()
        self.events_path = self.directory / 'events.jsonl'
        self.count_path = self.directory / 'go-count'
        program = f'#!{sys.executable}\n' + textwrap.dedent('''\
            import json
            import os
            from pathlib import Path
            import sys

            command = Path(sys.argv[0]).name
            args = sys.argv[1:]
            event = {
                'command': command,
                'args': args,
                'cwd': str(Path.cwd()),
                'deadline_depth': os.environ.get('FAKE_DEADLINE_DEPTH', '0'),
                'environment': {
                    key: os.environ.get(key)
                    for key in ('GOPROXY', 'GOSUMDB', 'GONOSUMDB', 'GOPRIVATE')
                },
            }
            with open(os.environ['FAKE_EVENTS'], 'a') as output:
                output.write(json.dumps(event) + '\\n')

            if command == 'timeout':
                status = int(os.environ.get('FAKE_TIMEOUT_EXIT', '0'))
                if status:
                    sys.exit(status)
                if args[:2] != ['--kill-after=10s', '300s']:
                    sys.exit('Unexpected timeout options: ' + repr(args))
                environment = dict(os.environ)
                environment['FAKE_DEADLINE_DEPTH'] = str(
                    int(environment.get('FAKE_DEADLINE_DEPTH', '0')) + 1)
                os.execvpe(args[2], args[2:], environment)
            elif command == 'go':
                if args != ['mod', 'download']:
                    if os.environ.get('FAKE_BACKEND_MODE'):
                        if args == ['env', 'GOPATH']:
                            print(os.environ['FAKE_GOPATH'])
                            sys.exit(0)
                        if args == ['mod', 'tidy']:
                            sys.exit(int(os.environ.get('FAKE_TIDY_EXIT', '0')))
                        if args == ['list', './backend/...']:
                            print('github.com/kubeflow/pipelines/backend/example')
                            sys.exit(0)
                        if args[:3] == ['test', '-v', '-cover']:
                            sys.exit(int(os.environ.get('FAKE_TEST_EXIT', '0')))
                    sys.exit('Unexpected Go command: ' + repr(args))
                counter = Path(os.environ['FAKE_GO_COUNT'])
                count = int(counter.read_text()) + 1 if counter.exists() else 1
                counter.write_text(str(count))
                failures = int(os.environ.get('FAKE_GO_FAILURES', '0'))
                sys.exit(17 if count <= failures else 0)
            elif command == 'git':
                if args != ['diff', '--exit-code', '--', 'go.mod', 'go.sum']:
                    sys.exit('Unexpected Git command: ' + repr(args))
                sys.exit(0)
            elif command == 'sleep':
                sys.exit(0)
            else:
                sys.exit('Unexpected fake command: ' + command)
            ''')
        for name in ('go', 'git', 'sleep', 'timeout'):
            executable = self.bin_directory / name
            executable.write_text(program, encoding='utf-8')
            executable.chmod(0o755)

    def run_download(self,
                     failures=0,
                     environment=None,
                     command=None,
                     cwd=None):
        self.events_path.unlink(missing_ok=True)
        self.count_path.unlink(missing_ok=True)
        effective_environment = dict(os.environ)
        for key in ('GOPROXY', 'GOSUMDB', 'GONOSUMDB', 'GOPRIVATE',
                    'FAKE_DEADLINE_DEPTH', 'FAKE_TIMEOUT_EXIT'):
            effective_environment.pop(key, None)
        effective_environment.update({
            'PATH': f'{self.bin_directory}{os.pathsep}{os.environ["PATH"]}',
            'FAKE_EVENTS': str(self.events_path),
            'FAKE_GO_COUNT': str(self.count_path),
            'FAKE_GO_FAILURES': str(failures),
        })
        effective_environment.update(environment or {})
        return subprocess.run(
            command or ['bash', str(SCRIPT)],
            cwd=cwd or self.directory,
            env=effective_environment,
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )

    def events(self, command=None):
        events = [
            json.loads(line) for line in self.events_path.read_text(
                encoding='utf-8').splitlines()
        ] if self.events_path.exists() else []
        return [
            event for event in events
            if command is None or event['command'] == command
        ]

    def assert_one_outer_deadline(self):
        deadlines = self.events('timeout')
        self.assertEqual(len(deadlines), 1)
        self.assertEqual(deadlines[0]['args'][:3],
                         ['--kill-after=10s', '300s', 'bash'])
        for event in self.events():
            if event['command'] != 'timeout':
                self.assertEqual(event['deadline_depth'], '1', event)

    def test_first_success_downloads_from_repository_root(self):
        result = self.run_download()

        self.assertEqual(result.returncode, 0, result.stderr)
        downloads = self.events('go')
        self.assertEqual(len(downloads), 1)
        self.assertEqual(downloads[0]['args'], ['mod', 'download'])
        self.assertEqual(downloads[0]['cwd'], str(ROOT))
        self.assertEqual(downloads[0]['environment']['GOPROXY'],
                         'https://proxy.golang.org|direct')
        self.assertIsNone(downloads[0]['environment']['GOSUMDB'])
        self.assertEqual(self.events('sleep'), [])
        self.assert_one_outer_deadline()

    def test_transient_failures_recover_within_one_deadline(self):
        result = self.run_download(failures=2)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual([event['command'] for event in self.events()],
                         ['timeout', 'go', 'sleep', 'go', 'sleep', 'go'])
        self.assertEqual([event['args'] for event in self.events('sleep')],
                         [['10'], ['10']])
        self.assert_one_outer_deadline()

    def test_retry_exhaustion_fails_after_three_downloads(self):
        result = self.run_download(failures=3)

        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertEqual(len(self.events('go')), 3)
        self.assertEqual([event['args'] for event in self.events('sleep')],
                         [['10'], ['10']])
        self.assert_one_outer_deadline()

    def test_custom_proxy_and_checksum_configuration_are_preserved(self):
        for proxy in ('https://modules.example.test,direct', 'off'):
            with self.subTest(proxy=proxy):
                expected = {
                    'GOPROXY': proxy,
                    'GOSUMDB': 'sum.golang.org',
                    'GONOSUMDB': 'private.example.test/*',
                    'GOPRIVATE': 'private.example.test/*',
                }
                result = self.run_download(environment=expected)

                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(self.events('go')[0]['environment'], expected)

    def test_outer_timeout_status_is_preserved_without_retrying_deadline(self):
        for status in (124, 137):
            with self.subTest(status=status):
                result = self.run_download(
                    environment={'FAKE_TIMEOUT_EXIT': str(status)})

                self.assertEqual(result.returncode, status, result.stderr)
                self.assertEqual(len(self.events('timeout')), 1)
                self.assertEqual(self.events('go'), [])
                self.assertEqual(self.events('sleep'), [])

    def test_callers_download_before_setup_and_keep_tests_outside_retries(self):
        workflow = yaml.safe_load(WORKFLOW.read_text(encoding='utf-8'))
        action = yaml.safe_load(ACTION.read_text(encoding='utf-8'))
        backend = yaml.safe_load(BACKEND_WORKFLOW.read_text(encoding='utf-8'))
        callers = (
            (WORKFLOW, workflow['jobs']['api-integration-tests-v2']['steps'],
             'Set up Go', 'Create KFP cluster', 'API integration tests v2'),
            (BACKEND_WORKFLOW, backend['jobs']['backend-tests']['steps'],
             'Set up Go', 'Run Backend Tests', 'Run Backend Tests'),
            (ACTION, action['runs']['steps'],
             'Restore Go build and module caches', 'Configure API Access',
             'Run Tests'),
        )
        for path, steps, before, after, test_step_name in callers:
            with self.subTest(path=path):
                names = [step.get('name') for step in steps]
                self.assertEqual(names.count('Download Go modules'), 1)
                download_index = names.index('Download Go modules')
                self.assertLess(names.index(before), download_index)
                self.assertLess(download_index, names.index(after))
                download = steps[download_index]
                self.assertNotIn('continue-on-error', download)
                self.assertNotIn('if', download)
                self.assertEqual(
                    download['run'].strip(),
                    'bash .github/resources/scripts/download-go-modules.sh')
                test_step = steps[names.index(test_step_name)]
                self.assertLess(download_index, names.index(test_step_name))
                self.assertNotIn('go mod download', test_step['run'])
                self.assertNotIn('download-go-modules.sh', test_step['run'])
                self.assertNotRegex(test_step['run'], r'\bretry\s')
                self.assertNotIn('always()', test_step.get('if', ''))
                self.assertNotIn('failure()', test_step.get('if', ''))

                result = self.run_download(
                    failures=3,
                    command=[
                        'bash', '-e', '-o', 'pipefail', '-c', download['run']
                    ],
                    cwd=ROOT,
                )

                self.assertEqual(result.returncode, 1, result.stderr)
                self.assertEqual(len(self.events('go')), 3)
                self.assert_one_outer_deadline()

    def test_backend_tidy_and_tests_run_once_after_downloads(self):
        workflow = yaml.safe_load(BACKEND_WORKFLOW.read_text(encoding='utf-8'))
        steps = workflow['jobs']['backend-tests']['steps']
        test_step = next(
            step for step in steps if step.get('name') == 'Run Backend Tests')
        for tidy_status, test_status, expected in ((0, 0, 0), (23, 0, 23),
                                                   (0, 29, 29)):
            with self.subTest(tidy=tidy_status, tests=test_status):
                result = self.run_download(
                    environment={
                        'FAKE_BACKEND_MODE': '1',
                        'FAKE_GOPATH': str(self.directory),
                        'FAKE_TIDY_EXIT': str(tidy_status),
                        'FAKE_TEST_EXIT': str(test_status),
                    },
                    command=['bash', '-e', '-c', test_step['run']],
                    cwd=ROOT,
                )

                self.assertEqual(result.returncode, expected, result.stderr)
                go_commands = [event['args'] for event in self.events('go')]
                expected_commands = [['env', 'GOPATH'], ['mod', 'tidy']]
                if not tidy_status:
                    expected_commands.extend([
                        ['list', './backend/...'],
                        [
                            'test', '-v', '-cover',
                            'github.com/kubeflow/pipelines/backend/example'
                        ],
                    ])
                self.assertEqual(go_commands, expected_commands)
                self.assertEqual(self.events('timeout'), [])
                self.assertEqual(self.events('sleep'), [])
                for event in self.events('go'):
                    self.assertEqual(event['environment']['GOPROXY'],
                                     'https://proxy.golang.org|direct')

    def test_workflows_watch_shared_downloader_dependencies(self):
        for name in ('compiler-tests.yml', 'presubmit-backend.yml'):
            with self.subTest(workflow=name):
                workflow = yaml.load(
                    (ROOT / '.github/workflows' /
                     name).read_text(encoding='utf-8'),
                    Loader=yaml.BaseLoader)
                paths = workflow['on']['pull_request']['paths']
                self.assertIn(
                    '.github/resources/scripts/download-go-modules.sh', paths)
                self.assertIn('.github/resources/scripts/helper-functions.sh',
                              paths)
        workflow = yaml.load(
            (ROOT / '.github/workflows/ci-scripts-tests.yml').read_text(
                encoding='utf-8'),
            Loader=yaml.BaseLoader)
        self.assertIn('test/presubmit-backend-test.sh',
                      workflow['on']['pull_request']['paths'])


if __name__ == '__main__':
    unittest.main()

# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import contextlib
import io
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock

import conformance
import schedule_policy


class ConformanceTest(unittest.TestCase):

    def test_workflow_uses_the_same_immutable_reference(self):
        workflow = (Path(__file__).resolve().parents[2] /
                    '.github/workflows/upgrade-readiness.yml').read_text()
        self.assertIn('ref: ' + schedule_policy.POLICY_SOURCE, workflow)

    def test_fixtures_cover_sar_and_pre_sar_decisions(self):
        cases = {case['name']: case for case in conformance.cases()}
        for name in ('default-exemption', 'allowlist-denied',
                     'literal-allowlist-star'):
            self.assertEqual(cases[name]['expected_sar_requests'], 0)
        for name in ('custom-scoped-grant', 'custom-denied', 'audit-denied',
                     'audit-allowlist-denied', 'wrong-named-grant',
                     'trimmed-allowlist', 'group-only-grant'):
            self.assertEqual(cases[name]['expected_sar_requests'], 1)
        self.assertEqual(cases['audit-denied']['prediction'],
                         'operational_impact')
        self.assertEqual(cases['custom-denied']['prediction'],
                         'policy_rejection')

    def test_wrong_policy_revision_cannot_run_backend(self):
        with mock.patch.object(
                conformance.subprocess, 'check_output',
                return_value='a' * 40), mock.patch.object(
                    conformance, '_run_backend') as run:
            with self.assertRaisesRegex(ValueError,
                                        'requested target revision'):
                conformance.run(Path('.'))
            run.assert_not_called()

    def test_dirty_policy_checkout_cannot_run_backend(self):
        with mock.patch.object(
                conformance.subprocess,
                'check_output',
                side_effect=[
                    schedule_policy.POLICY_SOURCE, ' M backend/source.go'
                ]), mock.patch.object(conformance, '_run_backend') as run:
            with self.assertRaisesRegex(ValueError, 'clean backend checkout'):
                conformance.run(Path('.'))
            run.assert_not_called()

    def test_candidate_must_match_and_have_no_untracked_files(self):
        candidate = 'b' * 40
        for revision, dirty, message in ((schedule_policy.POLICY_SOURCE, '',
                                          'requested target revision'),
                                         (candidate, '?? unexpected.go',
                                          'clean backend checkout')):
            with self.subTest(
                    revision=revision, dirty=dirty), mock.patch.object(
                        conformance.subprocess,
                        'check_output',
                        side_effect=[revision, dirty]), mock.patch.object(
                            conformance, '_run_backend') as backend:
                with self.assertRaisesRegex(ValueError, message):
                    conformance.run(Path('.'), candidate)
                backend.assert_not_called()

    def test_invalid_target_revision_or_timeout_cannot_run_commands(self):
        with mock.patch.object(conformance.subprocess, 'check_output') as git:
            for revision in ('master', 'b' * 39, 'g' * 40, None):
                with self.subTest(revision=revision), self.assertRaisesRegex(
                        ValueError, 'full 40-character target revision'):
                    conformance.run(Path('.'), revision)
            for timeout in (0, -1, 3601, float('nan'), float('inf')):
                with self.subTest(timeout=timeout), self.assertRaisesRegex(
                        ValueError, 'Timeout must be'):
                    conformance.run(Path('.'), timeout_seconds=timeout)
            git.assert_not_called()

    def test_candidate_command_overlay_and_provenance(self):
        candidate = 'b' * 40
        source = Path('.').resolve()

        def backend(command, actual_source, environment, timeout):
            self.assertEqual(source, actual_source)
            self.assertEqual(45, timeout)
            self.assertEqual(['go', 'test', '-overlay'], command[:3])
            self.assertEqual([
                './backend/src/apiserver/resource', '-run',
                '^TestReadinessPolicyConformance$', '-count=1', '-v'
            ], command[4:])
            overlay = json.loads(Path(command[3]).read_text())
            template = Path(
                conformance.__file__
            ).parent / 'conformance' / 'backend_policy_test.go.tmpl'
            self.assertEqual(
                {
                    'Replace': {
                        str(source /
                            'backend/src/apiserver/resource/readiness_conformance_test.go'
                           ):
                            str(template.resolve())
                    }
                }, overlay)
            fixtures = json.loads(
                Path(environment['KFP_READINESS_CASES']).read_text())
            self.assertEqual(conformance.cases(candidate), fixtures)
            self.assertEqual(10, len(fixtures))

        output = io.StringIO()
        with mock.patch.object(conformance.subprocess, 'check_output',
                               side_effect=[candidate, '']) as git, \
                mock.patch.object(conformance, '_run_backend', side_effect=backend) as run, \
                mock.patch.object(schedule_policy, 'validate', wraps=schedule_policy.validate) as validate, \
                contextlib.redirect_stdout(output):
            conformance.run(source, candidate.upper(), 45)
        run.assert_called_once()
        for call in git.call_args_list:
            self.assertEqual(conformance.GIT_TIMEOUT_SECONDS,
                             call.kwargs['timeout'])
        for call in validate.call_args_list:
            self.assertEqual(candidate, call.args[0]['target_revision'])
        self.assertIn('Modeled policy contract: ' + schedule_policy.CONTRACT,
                      output.getvalue())
        self.assertIn('Modeled policy source: ' + schedule_policy.POLICY_SOURCE,
                      output.getvalue())
        self.assertIn('Tested backend revision: ' + candidate,
                      output.getvalue())
        self.assertIn('Live upgrade validation remains unassessed.',
                      output.getvalue())

    def test_backend_failure_never_prints_a_pass(self):
        output = io.StringIO()
        with mock.patch.object(conformance.subprocess, 'check_output',
                               side_effect=[schedule_policy.POLICY_SOURCE, '']), \
                mock.patch.object(conformance, '_run_backend',
                                  side_effect=subprocess.CalledProcessError(1, ['go', 'test'])), \
                contextlib.redirect_stdout(output):
            with self.assertRaises(subprocess.CalledProcessError):
                conformance.run(Path('.'))
        self.assertEqual('', output.getvalue())

    def test_cli_defaults_and_explicit_candidate(self):
        for options, revision, timeout in (
            ([], schedule_policy.POLICY_SOURCE,
             conformance.CONFORMANCE_TIMEOUT_SECONDS),
            (['--target-revision', 'b' * 40, '--timeout-seconds',
              '45'], 'b' * 40, 45)):
            with self.subTest(options=options), mock.patch.object(
                    sys, 'argv', ['conformance.py', '--backend-source', 'backend'] + options), \
                    mock.patch.object(conformance, 'run') as run:
                conformance.main()
                run.assert_called_once_with(Path('backend'), revision, timeout)

    def test_timeout_kills_and_reaps_process_group(self):
        process = mock.MagicMock()
        process.pid = 12345
        process.wait.side_effect = [subprocess.TimeoutExpired(['go'], 1), 0]
        with mock.patch.object(conformance.subprocess, 'Popen') as popen, \
                mock.patch.object(conformance.os, 'killpg') as kill:
            popen.return_value.__enter__.return_value = process
            with self.assertRaises(subprocess.TimeoutExpired):
                conformance._run_backend(['go'], Path('.'), {}, 1)
            self.assertTrue(popen.call_args.kwargs['start_new_session'])
            kill.assert_called_once_with(process.pid, signal.SIGKILL)
            self.assertEqual(
                [mock.call(timeout=1), mock.call()],
                process.wait.call_args_list)

    def test_timeout_closes_descendant_socket(self):
        with tempfile.TemporaryDirectory() as directory:
            marker = Path(directory) / 'listener.json'
            child = (
                'import json, os, socket, time; from pathlib import Path; '
                'listener = socket.socket(); '
                'listener.bind(("127.0.0.1", 0)); listener.listen(); '
                f'Path({str(marker)!r}).write_text(json.dumps('
                '{"group": os.getpgrp(), "port": listener.getsockname()[1]})); '
                'time.sleep(60)')
            parent = ('import subprocess, sys, time; '
                      f'subprocess.Popen([sys.executable, "-c", {child!r}]); '
                      'time.sleep(60)')
            try:
                with self.assertRaises(subprocess.TimeoutExpired):
                    conformance._run_backend([sys.executable, '-c', parent],
                                             Path(directory), dict(os.environ),
                                             1)
                details = json.loads(marker.read_text())
                deadline = time.monotonic() + 1
                while True:
                    with socket.socket() as connection:
                        connection.settimeout(1)
                        result = connection.connect_ex(
                            ('127.0.0.1', details['port']))
                    if result:
                        break
                    if time.monotonic() >= deadline:
                        self.fail(
                            'Timed-out backend descendant still holds its socket.'
                        )
                    time.sleep(0.01)
            finally:
                if marker.exists():
                    try:
                        os.killpg(
                            json.loads(marker.read_text())['group'],
                            signal.SIGKILL)
                    except ProcessLookupError:
                        pass

    def test_cli_timeout_is_a_failed_conformance(self):
        output = io.StringIO()
        with mock.patch.object(sys, 'argv', [
                'conformance.py', '--backend-source', 'backend']), \
                mock.patch.object(conformance, 'run',
                                  side_effect=subprocess.TimeoutExpired(['go'], 1)), \
                contextlib.redirect_stderr(output):
            with self.assertRaises(SystemExit) as error:
                conformance.main()
        self.assertEqual(1, error.exception.code)
        self.assertEqual(
            'Policy conformance failed: command exceeded its deadline.\n',
            output.getvalue())


if __name__ == '__main__':
    unittest.main()

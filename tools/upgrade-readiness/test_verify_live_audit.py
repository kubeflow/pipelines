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
"""Audit evidence is bounded and limited to the exclusive fixture window."""

import contextlib
import copy
from datetime import datetime
from datetime import timedelta
from datetime import timezone
import io
import json
import subprocess
import sys
import time
import unittest
from unittest import mock

from kfp_http import CollectionError
import verify_live_audit as audit


def completion(start):
    return dict(
        scope='fixture_run_completion',
        outcome='passed',
        mode='audit',
        all_expected_runs_succeeded=True,
        namespace=audit.NAMESPACE,
        observation_start=start.isoformat(),
        cases=[
            dict(
                scenario=scenario,
                schedule_uid=scenario + '-uid',
                service_account=account,
                runs=[dict(run_id=scenario + '-run', state='SUCCEEDED')])
            for scenario, account in audit.SCENARIOS.items()
        ])


def line(start, message=audit.MESSAGE):
    return start.isoformat(
    ) + ' W1005 01:02:03.123456 1 resource_manager.go:3976] ' + message + '\n'


class LiveAuditTest(unittest.TestCase):

    def setUp(self):
        self.start = datetime.now(timezone.utc) - timedelta(seconds=60)

    def test_precise_record_and_window_are_required(self):
        self.assertEqual(audit.count_records(line(self.start), self.start), 1)
        self.assertEqual(
            audit.count_records(
                line(self.start - timedelta(seconds=1)), self.start), 0)
        self.assertEqual(
            audit.count_records(
                line(self.start + timedelta(hours=1)), self.start), 0)
        changes = [('namespace="kfp-readiness-test"', 'namespace="other"'),
                   ('service_account="readiness-denied"',
                    'service_account="readiness-denied-extra"'),
                   ('mode=audit', 'mode=enforce'),
                   ('reason=account_denied', 'reason=account_not_allowed'),
                   ('control=service_account', 'control=workflow_identity'),
                   ('operation=authorize_service_account',
                    'operation=create_run'),
                   ('disposition=allow_policy_violation', 'disposition=deny')]
        for before, after in changes:
            with self.subTest(before=before):
                self.assertEqual(
                    audit.count_records(
                        line(self.start, audit.MESSAGE.replace(before, after)),
                        self.start), 0)
        self.assertEqual(
            audit.count_records(
                line(self.start, 'private payload: ' + audit.MESSAGE),
                self.start), 0)

    def test_matching_record_with_invalid_timestamp_is_not_evidence(self):
        with self.assertRaises(CollectionError):
            audit.count_records(
                line(self.start).replace(self.start.isoformat(),
                                         'invalid-time'), self.start)

    def test_complete_successful_audit_phase_is_required(self):
        report = completion(self.start)
        audit.validate_completion(report, self.start)
        for key, value in [('scope', 'run_creation_only'), ('mode', 'enforce'),
                           ('outcome', 'inconclusive'), ('namespace', 'other'),
                           ('all_expected_runs_succeeded', False),
                           ('observation_start',
                            (self.start - timedelta(seconds=1)).isoformat())]:
            with self.subTest(key=key), self.assertRaises(ValueError):
                audit.validate_completion(
                    dict(report, **{key: value}), self.start)
        for state in ('RUNNING', 'FAILED', 'CANCELED', 'SKIPPED'):
            changed = copy.deepcopy(report)
            changed['cases'][0]['runs'][0]['state'] = state
            with self.subTest(state=state), self.assertRaises(ValueError):
                audit.validate_completion(changed, self.start)
        changed = copy.deepcopy(report)
        changed['cases'][2]['service_account'] = 'other'
        with self.assertRaises(ValueError):
            audit.validate_completion(changed, self.start)
        with self.assertRaises(ValueError):
            audit.validate_completion(
                dict(report, cases=report['cases'][:2]), self.start)

    def collect_with_script(self, script, progress=None):
        original = subprocess.Popen
        commands = []

        def launch(command, **kwargs):
            commands.append(command)
            return original([sys.executable, '-c', script], **kwargs)

        with mock.patch.object(audit.subprocess, 'Popen', side_effect=launch):
            result = audit.collect_logs(
                audit.CONTEXT, self.start, progress=progress)
        self.assertEqual(commands[0][:6], [
            'kubectl', '--context', audit.CONTEXT, '--request-timeout=20s',
            '--namespace', 'kubeflow'
        ])
        self.assertIn('--selector=app=ml-pipeline', commands[0])
        self.assertIn('--container=ml-pipeline-api-server', commands[0])
        self.assertIn('--tail=-1', commands[0])
        return result

    def test_streaming_collection_and_stderr_redaction(self):
        expected = line(self.start)
        self.assertEqual(
            self.collect_with_script('import sys; sys.stdout.write(' +
                                     repr(expected) + ')'), expected)
        with self.assertRaisesRegex(CollectionError,
                                    '^audit_collection_failed$'):
            self.collect_with_script(
                "import sys; sys.stderr.write('PRIVATE_ERROR'); sys.exit(1)")

    def test_collection_accepts_complete_logs_above_previous_limit(self):
        size = 1024 * 1024 + 1
        result = self.collect_with_script(
            "import sys; sys.stdout.write('x' * " + str(size) + " + '\\n')")
        self.assertEqual(len(result), size + 1)

    def test_truncated_or_oversized_logs_never_pass(self):
        with self.assertRaisesRegex(CollectionError,
                                    '^audit_collection_truncated$'):
            self.collect_with_script("print('partial', end='')")
        with mock.patch.object(audit, 'MAX_BYTES', 128), self.assertRaisesRegex(
                CollectionError, '^audit_collection_exceeded_limit$'):
            self.collect_with_script("print('x' * 1024)")

    def test_failed_collection_retains_only_bounded_counters(self):
        progress = {}
        with self.assertRaisesRegex(CollectionError,
                                    '^audit_collection_failed$'):
            self.collect_with_script(
                "import sys; print('PRIVATE'); sys.exit(7)", progress)
        self.assertEqual(progress,
                         dict(collected_bytes=8, collector_exit_code=7))
        self.assertNotIn('PRIVATE', str(progress))
        progress = {}
        with mock.patch.object(audit, 'MAX_BYTES', 128), self.assertRaisesRegex(
                CollectionError, '^audit_collection_exceeded_limit$'):
            self.collect_with_script("print('x' * 1024)", progress)
        self.assertEqual(progress['collected_bytes'], 129)

    def test_cli_failure_reason_and_stage_are_sanitized(self):
        for reason, expected in [('audit_collection_exceeded_limit',
                                  'audit_collection_exceeded_limit'),
                                 ('PRIVATE_RESPONSE',
                                  'invalid_or_incomplete_audit_evidence')]:
            output = io.StringIO()

            def fail(*args, **kwargs):
                kwargs['progress']['collected_bytes'] = 1024
                raise CollectionError(reason)

            with mock.patch.object(
                    audit, 'load',
                    return_value=completion(self.start)), mock.patch.object(
                        audit, 'collect_logs',
                        side_effect=fail), contextlib.redirect_stdout(output):
                code = audit.main([
                    '--context', audit.CONTEXT, '--not-before',
                    self.start.isoformat(), '--completion-report', 'report.json'
                ])
            report = json.loads(output.getvalue())
            self.assertEqual(code, 1)
            self.assertEqual(report['outcome'], 'inconclusive')
            self.assertEqual(report['reason'], expected)
            self.assertEqual(report['diagnostics']['stage'], 'log_collection')
            self.assertEqual(report['diagnostics']['collected_bytes'], 1024)
            self.assertGreaterEqual(report['diagnostics']['elapsed_seconds'], 0)
            self.assertNotIn('PRIVATE', output.getvalue())

    def test_collection_timeout_is_bounded(self):
        began = time.monotonic()
        with mock.patch.object(audit, 'TIMEOUT', .1), self.assertRaisesRegex(
                CollectionError, '^audit_collection_timed_out$'):
            self.collect_with_script('import time; time.sleep(30)')
        self.assertLess(time.monotonic() - began, 2)

    def invoke(self, report, logs, context=audit.CONTEXT, start=None):
        output = io.StringIO()
        with mock.patch.object(
                audit, 'load', return_value=report), mock.patch.object(
                    audit, 'collect_logs',
                    return_value=logs) as collect, contextlib.redirect_stdout(
                        output):
            code = audit.main([
                '--context', context, '--not-before', (start or
                                                       self.start).isoformat(),
                '--completion-report', 'completion.json'
            ])
        return code, json.loads(output.getvalue()), collect

    def test_cli_returns_only_counts_and_exact_evidence_scope(self):
        code, report, _ = self.invoke(
            completion(self.start),
            'private-token-and-payload\n' + line(self.start))
        self.assertEqual(code, 0)
        self.assertEqual(report['scope'],
                         'isolated_namespace_account_activation_window')
        self.assertEqual(report['matching_records'], 1)
        self.assertNotIn('private', json.dumps(report))
        self.assertNotIn('resource_manager', json.dumps(report))

    def test_wrong_context_stale_activation_or_failed_fixture_prevents_log_read(
            self):
        for context, start, result in [
            ('production', self.start, completion(self.start)),
            (audit.CONTEXT, self.start - timedelta(hours=1),
             completion(self.start)),
            (audit.CONTEXT, self.start,
             dict(completion(self.start), outcome='failed'))
        ]:
            with self.subTest(context=context, start=start):
                code, report, collect = self.invoke(result, line(start),
                                                    context, start)
                self.assertEqual(code, 1)
                self.assertEqual(report['outcome'], 'inconclusive')
                collect.assert_not_called()
        with mock.patch.object(audit.subprocess,
                               'Popen') as popen, self.assertRaises(ValueError):
            audit.collect_logs('production', self.start)
        popen.assert_not_called()

    def test_missing_audit_record_blocks_acceptance(self):
        code, report, _ = self.invoke(
            completion(self.start), 'unrelated private logs\n')
        self.assertEqual(code, 1)
        self.assertEqual(report['matching_records'], 0)
        self.assertEqual(report['outcome'], 'inconclusive')


if __name__ == '__main__':
    unittest.main()

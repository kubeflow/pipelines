# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Check portable, private configuration and actionable gateway startup
failures."""

from io import BytesIO
import json
import os
from pathlib import Path
import stat
import subprocess
import tempfile
import unittest
from unittest.mock import patch
from urllib.error import URLError

import external_storage_fixture as fixture


class StartupTest(unittest.TestCase):

    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.storage = fixture.ExternalStorage(self.root, 'fixture-image')
        self.storage.issuer = 'http://fixture-issuer'
        self.storage.endpoint = 'http://127.0.0.1:8333'

    def test_gateway_configuration_is_readable_without_exposing_private_files(
            self):
        private = self.root / 'key.pem'
        private.write_text('private fixture key')
        private.chmod(0o600)
        session = self.root / 'team-a.credentials'
        session.write_text('private fixture session')
        session.chmod(0o600)
        previous = os.umask(0o077)
        try:
            self.storage.configure()
        finally:
            os.umask(previous)
        mounted = self.storage.config_directory
        self.assertEqual(
            set(path.name for path in mounted.iterdir()),
            {'s3.json', 'iam.json'})
        self.assertEqual(stat.S_IMODE(mounted.stat().st_mode), 0o755)
        for path in mounted.iterdir():
            self.assertEqual(stat.S_IMODE(path.stat().st_mode), 0o644)
        self.assertEqual(stat.S_IMODE(self.root.stat().st_mode), 0o700)
        self.assertEqual(stat.S_IMODE(private.stat().st_mode), 0o600)
        self.assertEqual(stat.S_IMODE(session.stat().st_mode), 0o600)

    def test_exited_gateway_reports_permissions_without_waiting_or_exposing_logs(
            self):
        state = {'Status': 'exited', 'ExitCode': 255, 'OOMKilled': False}
        logs = subprocess.CompletedProcess(
            [], 0, '',
            'failed to load config /fixture/s3.json: permission denied\n'
            'secretKey=must-not-be-published\n')
        with patch.object(fixture, 'urlopen', side_effect=URLError('refused')), \
                patch.object(fixture, 'run', return_value=json.dumps(state)), \
                patch.object(fixture.subprocess, 'run', return_value=logs), \
                patch.object(fixture.time, 'sleep') as sleep:
            with self.assertRaisesRegex(RuntimeError,
                                        'permission denied') as error:
                self.storage.wait_until_ready()
        sleep.assert_not_called()
        self.assertEqual(self.storage.evidence['startupFailure'], {
            'containerState': state,
            'detectedErrors': ['permission denied']
        })
        self.assertNotIn('must-not-be-published', str(error.exception))
        self.assertNotIn('must-not-be-published',
                         json.dumps(self.storage.evidence))

    def test_transient_readiness_failure_retries_running_gateway(self):
        state = {'Status': 'running', 'ExitCode': 0, 'OOMKilled': False}
        with patch.object(fixture, 'urlopen',
                          side_effect=[ConnectionResetError(), BytesIO(b'{}')]), \
                patch.object(fixture, 'run', return_value=json.dumps(state)), \
                patch.object(fixture.time, 'sleep') as sleep:
            self.storage.wait_until_ready()
        sleep.assert_called_once_with(1)
        self.assertNotIn('startupFailure', self.storage.evidence)

    def test_readiness_timeout_records_bounded_state_and_error_classes(self):
        state = {'Status': 'running', 'ExitCode': 0, 'OOMKilled': False}
        logs = subprocess.CompletedProcess([], 0, 'upstream connection refused',
                                           '')
        with patch.object(fixture, 'urlopen', side_effect=URLError('refused')), \
                patch.object(fixture, 'run', return_value=json.dumps(state)), \
                patch.object(fixture.subprocess, 'run', return_value=logs) as log, \
                patch.object(fixture.time, 'sleep') as sleep:
            with self.assertRaisesRegex(RuntimeError, 'connection refused'):
                self.storage.wait_until_ready()
        self.assertEqual(sleep.call_count, 60)
        self.assertEqual(
            log.call_args.args[0],
            ['docker', 'logs', '--tail', '100', self.storage.container])
        self.assertEqual(
            self.storage.evidence['startupFailure']['containerState'], state)


if __name__ == '__main__':
    unittest.main()

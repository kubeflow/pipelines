"""Exercise published-image loading without a Docker daemon."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest

ROOT = Path(__file__).resolve().parents[2]


class ImageLoadingTest(unittest.TestCase):

    def run_loading(self, failures):
        workflow = (ROOT /
                    '.github/workflows/cache-rolling-upgrade.yml').read_text()
        step = workflow.split(
            '      - name: Load source, target and fixture images\n', 1)[1]
        script = textwrap.dedent(
            step.split('        run: |\n', 1)[1].split('      - name:', 1)[0])
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            executable = path / 'docker'
            executable.write_text('''#!/usr/bin/env python3
import json, os, pathlib, sys
log = pathlib.Path(os.environ['COMMAND_LOG'])
commands = json.loads(log.read_text()) if log.exists() else []
commands.append([pathlib.Path(sys.argv[0]).name] + sys.argv[1:])
log.write_text(json.dumps(commands))
if sys.argv[1:5] != ['exec', 'cache-rolling-control-plane', 'crictl', 'pull']:
    sys.exit(2)
sys.exit(1 if len(commands) <= int(os.environ['FAILURES']) else 0)
''')
            executable.chmod(0o755)
            kind = path / 'kind'
            kind.write_text('''#!/usr/bin/env python3
import json, os, pathlib, sys
log = pathlib.Path(os.environ['COMMAND_LOG'])
commands = json.loads(log.read_text()) if log.exists() else []
commands.append(['kind'] + sys.argv[1:])
log.write_text(json.dumps(commands))
''')
            kind.chmod(0o755)
            sleep = path / 'sleep'
            sleep.write_text('#!/bin/sh\nexit 0\n')
            sleep.chmod(0o755)
            log = path / 'commands.json'
            result = subprocess.run(
                ['bash', '-c', script],
                cwd=ROOT,
                capture_output=True,
                text=True,
                env={
                    **os.environ, 'PATH': directory + ':' + os.environ['PATH'],
                    'COMMAND_LOG': str(log),
                    'FAILURES': str(failures)
                })
            return result, json.loads(log.read_text())

    def test_pulls_published_images_inside_kind_and_loads_local_candidate(self):
        result, commands = self.run_loading(0)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual([command[-1] for command in commands], [
            'ghcr.io/kubeflow/kfp-cache-server:2.17.2', 'mysql:8.0',
            'busybox:1.37', 'kfp-cache-server:rolling-candidate'
        ])
        self.assertEqual(
            commands[-1][:-1],
            ['kind', 'load', 'docker-image', '--name', 'cache-rolling'])

    def test_transient_pull_failure_retries(self):
        result, commands = self.run_loading(1)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(commands[0], commands[1])
        self.assertEqual(len(commands), 5)

    def test_exhausted_pull_stops_before_candidate(self):
        result, commands = self.run_loading(3)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(commands), 3)
        self.assertTrue(all(command == commands[0] for command in commands))

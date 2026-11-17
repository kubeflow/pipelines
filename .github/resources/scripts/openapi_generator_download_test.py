#
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
"""Exercise generator download recovery without network or code generation."""

import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
DOWNLOAD = ROOT / 'backend/api/download_openapi_generator.sh'


class OpenAPIGeneratorDownloadTest(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        self.target = self.root / 'generator.jar'
        self.calls = self.root / 'calls'
        self.environment = {
            **os.environ,
            'PATH': str(self.bin),
            'CALLS': str(self.calls),
            'ARGUMENTS': str(self.root / 'arguments'),
            'TARGET': str(self.target),
            'FAILURES': '0',
        }
        for name in ('mktemp', 'rm', 'mv'):
            (self.bin / name).symlink_to(shutil.which(name))
        self.command('sleep', 'exit 0')
        self.command(
            'curl',
            r"""
printf '%s\n' "$@" > "$ARGUMENTS"
count=0
if [[ -f "$CALLS" ]]; then read -r count < "$CALLS"; fi
count=$((count + 1))
printf '%s\n' "$count" > "$CALLS"
while (( $# )); do
  if [[ "$1" == -o || "$1" == --output ]]; then
    output=$2
    shift 2
  else
    shift
  fi
done
if [[ -e "$TARGET" && -s "$TARGET" ]]; then
  echo 'download published before completion' >&2
  exit 99
fi
if [[ "${SIGNAL:-}" == TERM ]]; then
  printf partial > "$output"
  kill -TERM "$PPID"
  exit 0
fi
if (( count <= FAILURES )); then
  printf partial > "$output"
  exit 35
fi
if [[ "${EMPTY:-0}" == 1 ]]; then
  : # A successful response may produce no new output.
else
  printf complete-generator > "$output"
fi
""",
        )

    def command(self, name, body):
        script = self.bin / name
        script.write_text('#!/bin/bash\nset -eu\n' + body + '\n')
        script.chmod(0o755)

    def download(self):
        return subprocess.run(
            ['/bin/bash', str(DOWNLOAD), str(self.target)],
            env=self.environment,
            capture_output=True,
            text=True,
            timeout=10,
            check=False,
        )

    def test_tls_failures_retry_without_publishing_partial_jar(self):
        self.environment['FAILURES'] = '2'
        result = self.download()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.calls.read_text().strip(), '3')
        arguments = (self.root / 'arguments').read_text().splitlines()
        self.assertIn('--fail', arguments)
        self.assertIn('--location', arguments)
        self.assertEqual(
            arguments[arguments.index('--connect-timeout') + 1], '20'
        )
        self.assertEqual(arguments[arguments.index('--max-time') + 1], '120')
        self.assertEqual(self.target.read_text(), 'complete-generator')
        self.assertEqual(list(self.root.glob('generator.jar.*')), [])

    def test_permanent_failure_is_bounded_and_leaves_no_cached_jar(self):
        self.environment['FAILURES'] = '99'
        result = self.download()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.calls.read_text().strip(), '4')
        self.assertFalse(self.target.exists())
        self.assertEqual(list(self.root.glob('generator.jar.*')), [])

    def test_termination_removes_partial_download(self):
        self.environment['SIGNAL'] = 'TERM'
        result = self.download()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.target.exists())
        self.assertEqual(list(self.root.glob('generator.jar.*')), [])

    def test_existing_cache_does_not_download(self):
        self.target.write_text('cached-generator')
        result = self.download()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(self.calls.exists())
        self.assertEqual(self.target.read_text(), 'cached-generator')

    def test_empty_success_response_is_not_cached(self):
        self.environment['EMPTY'] = '1'
        result = self.download()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.calls.read_text().strip(), '4')
        self.assertFalse(self.target.exists())
        self.assertEqual(list(self.root.glob('generator.jar.*')), [])

    def test_empty_retry_cannot_publish_previous_partial_bytes(self):
        self.environment['FAILURES'] = '1'
        self.environment['EMPTY'] = '1'
        result = self.download()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.calls.read_text().strip(), '4')
        self.assertFalse(self.target.exists())
        self.assertEqual(list(self.root.glob('generator.jar.*')), [])

    def test_empty_cache_is_replaced(self):
        self.target.touch()
        result = self.download()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.target.read_text(), 'complete-generator')


if __name__ == '__main__':
    unittest.main()

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
"""Exercise helper pod timeout diagnostics without a Kubernetes cluster."""

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

SCRIPT = (
    Path(__file__).resolve().parents[3]
    / 'test/artifact-proxy/test-artifact-proxy.sh'
)


class ArtifactProxyReadinessTest(unittest.TestCase):
    def test_wait_failure_reports_pod_and_events_and_preserves_status(
        self,
    ) -> None:
        with tempfile.TemporaryDirectory() as directory:
            kubectl = Path(directory) / 'kubectl'
            kubectl.write_text(
                """#!/bin/sh
[ "$1" = "-n" ] && [ "$2" = "user-ns" ] || exit 91
shift 2
case "$1 $2" in
  "run kfp-proxy-curl") exit 0 ;;
  "wait --for=condition=Ready") exit 42 ;;
  "describe pod/kfp-proxy-curl") echo "pod diagnostic"; exit 7 ;;
  "get events")
    case " $* " in
      *"involvedObject.kind=Pod,involvedObject.name=kfp-proxy-curl"*)
        echo "pod event: FailedScheduling"; exit 0 ;;
    esac ;;
esac
exit 92
"""
            )
            kubectl.chmod(0o755)
            result = subprocess.run(
                ['bash', str(SCRIPT), 'user-ns'],
                env={
                    **os.environ,
                    'PATH': directory + os.pathsep + os.environ['PATH'],
                },
                capture_output=True,
                text=True,
                timeout=5,
            )
        self.assertEqual(result.returncode, 42, result.stderr)
        self.assertIn('pod diagnostic', result.stdout)
        self.assertIn('pod event: FailedScheduling', result.stdout)

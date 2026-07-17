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

import os
from pathlib import Path
import subprocess
import tempfile
import time
import unittest


SCRIPT = Path(__file__).with_name('wait-for-seaweedfs-iam.sh')
NETWORK_POLICY = (
    Path(__file__).parents[3]
    / 'manifests/kustomize/third-party/seaweedfs/base/seaweedfs/seaweedfs-networkpolicy.yaml'
)


class WaitForSeaweedfsIamTest(unittest.TestCase):

    def test_retries_until_profile_controller_can_connect(self):
        result = self._run(probe_failures=2, timeout_seconds=6)

        self.assertEqual(result.returncode, 0)
        self.assertIn(
            '3 consecutive successful probes',
            result.stdout,
        )
        self.assertNotIn('SeaweedFS Service', result.stdout)

    def test_requires_consecutive_successes_before_reporting_ready(self):
        result, probe_count = self._run(
            probe_failures=0,
            timeout_seconds=4,
            return_probe_count=True,
        )

        self.assertEqual(result.returncode, 0)
        self.assertEqual(probe_count, 3)
        self.assertIn('probe succeeded (1/3)', result.stdout)
        self.assertIn('probe succeeded (2/3)', result.stdout)

    def test_failed_probe_resets_success_streak(self):
        result, probe_count = self._run(
            probe_failures=0,
            probe_results='success,failure,success,success,success',
            timeout_seconds=7,
            return_probe_count=True,
        )

        self.assertEqual(result.returncode, 0)
        self.assertEqual(probe_count, 5)
        self.assertEqual(result.stdout.count('probe succeeded (1/3)'), 2)

    def test_failure_collects_service_backend_and_controller_state(self):
        result = self._run(probe_failures=10, timeout_seconds=1)

        self.assertEqual(result.returncode, 1)
        self.assertIn('was not reachable after 1s', result.stdout)
        self.assertIn('SeaweedFS Service', result.stdout)
        self.assertIn('seaweedfs-service-state', result.stdout)
        self.assertIn('seaweedfs-endpointslice-state', result.stdout)
        self.assertIn('seaweedfs-pod-state', result.stdout)
        self.assertIn('profile-controller-log-state', result.stdout)
        self.assertIn('metacontroller-log-state', result.stdout)
        self.assertIn('kubeflow-event-state', result.stdout)
        self.assertIn('"endpoint": {"result": "success"}', result.stdout)
        self.assertIn('"service-vip": {"result": "timeout"}', result.stdout)

    def test_reports_service_vip_and_endpoint_targets(self):
        result = self._run(probe_failures=0, timeout_seconds=4)

        self.assertEqual(result.returncode, 0)
        self.assertIn(
            'Service VIP 10.96.1.1:8111; Endpoint 10.244.0.20:8111',
            result.stdout,
        )

    def test_probe_duration_counts_toward_wall_clock_timeout(self):
        start = time.monotonic()
        result = self._run(
            probe_failures=10,
            timeout_seconds=2,
            probe_delay_seconds=2,
        )
        duration = time.monotonic() - start

        self.assertEqual(result.returncode, 1)
        self.assertIn('was not reachable after 2s', result.stdout)
        self.assertLess(duration, 3.5)

    def test_network_policy_allows_only_profile_controller_to_iam(self):
        policy = NETWORK_POLICY.read_text(encoding='utf-8')

        self.assertIn(
            '  - from:\n'
            '    - podSelector:\n'
            '        matchLabels:\n'
            '          app: kubeflow-pipelines-profile-controller\n'
            '    ports:\n'
            '    - port: 8111\n',
            policy,
        )
        self.assertEqual(policy.count('    - port: 8111\n'), 1)

    def _run(
        self,
        probe_failures: int,
        timeout_seconds: int,
        probe_delay_seconds: int = 0,
        probe_results: str = '',
        return_probe_count: bool = False,
    ):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            state_file = temporary_path / 'probe-count'
            fake_kubectl = bin_directory / 'kubectl'
            fake_kubectl.write_text(_FAKE_KUBECTL, encoding='utf-8')
            fake_kubectl.chmod(0o755)

            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'
            environment['FAKE_PROBE_FAILURES'] = str(probe_failures)
            environment['FAKE_PROBE_DELAY_SECONDS'] = str(probe_delay_seconds)
            environment['FAKE_PROBE_RESULTS'] = probe_results
            environment['FAKE_STATE_FILE'] = str(state_file)
            environment['SEAWEEDFS_IAM_WAIT_TIMEOUT_SECONDS'] = str(
                timeout_seconds
            )
            environment['SEAWEEDFS_IAM_WAIT_INTERVAL_SECONDS'] = '1'
            result = subprocess.run(
                ['bash', str(SCRIPT)],
                capture_output=True,
                text=True,
                env=environment,
            )
            if return_probe_count:
                return result, int(state_file.read_text(encoding='utf-8'))
            return result


_FAKE_KUBECTL = r'''#!/usr/bin/env bash
args="$*"

if [[ "$args" == *"get service seaweedfs -o json"* ]]; then
  echo '{"spec":{"clusterIP":"10.96.1.1","ports":[{"name":"http-iam","port":8111}]}}'
  exit
fi

if [[ "$args" == *"get endpointslice"* && "$args" == *"-o json"* ]]; then
  echo '{"items":[{"metadata":{"name":"seaweedfs-a"},"ports":[{"name":"http-iam","port":8111}],"endpoints":[{"addresses":["10.244.0.20"],"conditions":{"ready":true,"serving":true,"terminating":false}}]}]}'
  exit
fi

if [[ "$args" == *"exec deploy/kubeflow-pipelines-profile-controller"* ]]; then
  sleep "${FAKE_PROBE_DELAY_SECONDS:-0}"
  count=0
  [[ ! -f "$FAKE_STATE_FILE" ]] || count=$(<"$FAKE_STATE_FILE")
  count=$((count + 1))
  echo "$count" > "$FAKE_STATE_FILE"
  if [[ -n "${FAKE_PROBE_RESULTS:-}" ]]; then
    IFS=',' read -ra probe_results <<< "$FAKE_PROBE_RESULTS"
    if [[ "${probe_results[$((count - 1))]:-success}" == "success" ]]; then
      echo '{"endpoint": {"result": "success"}, "service-dns": {"result": "success"}, "service-vip": {"result": "success"}}'
      exit 0
    fi
    echo '{"endpoint": {"result": "success"}, "service-dns": {"result": "timeout"}, "service-vip": {"result": "timeout"}}'
    exit 1
  fi
  if (( count > FAKE_PROBE_FAILURES )); then
    echo '{"endpoint": {"result": "success"}, "service-dns": {"result": "success"}, "service-vip": {"result": "success"}}'
    exit 0
  else
    echo '{"endpoint": {"result": "success"}, "service-dns": {"result": "timeout"}, "service-vip": {"result": "timeout"}}'
    exit 1
  fi
fi

case "$args" in
  *"get service seaweedfs"*) echo "seaweedfs-service-state" ;;
  *"get endpointslice"*) echo "seaweedfs-endpointslice-state" ;;
  *"get pods -l app=seaweedfs"*) echo "seaweedfs-pod-state" ;;
  *"describe pods -l app=seaweedfs"*) echo "seaweedfs-pod-description" ;;
  *"get pods -l app=kubeflow-pipelines-profile-controller"*)
    echo "profile-controller-pod-state"
    ;;
  *"logs deploy/kubeflow-pipelines-profile-controller"*)
    echo "profile-controller-log-state"
    ;;
  *"logs statefulset/metacontroller"*) echo "metacontroller-log-state" ;;
  *"get events"*) echo "kubeflow-event-state" ;;
esac
'''


if __name__ == '__main__':
    unittest.main()

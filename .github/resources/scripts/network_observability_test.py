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
import unittest


SCRIPT = Path(__file__).with_name('network-observability.sh')


class NetworkObservabilityTest(unittest.TestCase):

    def test_resolves_ready_vips_and_direct_endpoint_ports(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            kubectl = bin_directory / 'kubectl'
            kubectl.write_text(_FAKE_KUBECTL, encoding='utf-8')
            kubectl.chmod(0o755)
            output_directory = temporary_path / 'output'
            kubectl_log = temporary_path / 'kubectl.log'
            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'
            environment['FAKE_KUBECTL_LOG'] = str(kubectl_log)

            subprocess.run(
                [
                    'bash',
                    str(SCRIPT),
                    'resolve-targets',
                    'kubeflow',
                    str(output_directory),
                ],
                check=True,
                env=environment,
            )
            targets = (output_directory / 'targets.tsv').read_text(
                encoding='utf-8'
            )
            kubectl_calls = kubectl_log.read_text(encoding='utf-8').splitlines()

        self.assertIn('seaweedfs-s3\tvip\t10.96.1.1\t9000\n', targets)
        self.assertIn(
            'seaweedfs-s3\tendpoint\t10.244.0.20\t8333\n', targets
        )
        self.assertIn('kubernetes-api\tvip\t10.96.0.1\t443\n', targets)
        self.assertIn(
            'kubernetes-api\tendpoint\t172.18.0.2\t6443\n', targets
        )
        self.assertNotIn('10.244.0.21', targets)
        self.assertTrue(kubectl_calls)
        self.assertTrue(
            all('--request-timeout=10s' in call for call in kubectl_calls)
        )

    def test_packet_capture_filter_is_limited_to_resolved_targets(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            targets_path = Path(temporary_directory) / 'targets.tsv'
            targets_path.write_text(
                'seaweedfs-s3\tvip\t10.96.1.1\t9000\n'
                'seaweedfs-s3\tendpoint\t10.244.0.20\t8333\n'
                'invalid\tvip\tnot-an-address\t443\n',
                encoding='utf-8',
            )

            result = subprocess.run(
                ['bash', str(SCRIPT), 'capture-filter', str(targets_path)],
                check=True,
                capture_output=True,
                text=True,
            )

        self.assertIn('(host 10.96.1.1 and port 9000)', result.stdout)
        self.assertIn('(host 10.244.0.20 and port 8333)', result.stdout)
        self.assertNotIn('not-an-address', result.stdout)
        self.assertIn('tcp-syn|tcp-rst', result.stdout)


_FAKE_KUBECTL = r'''#!/usr/bin/env bash
args="$*"
if [[ -n "${FAKE_KUBECTL_LOG:-}" ]]; then
  printf '%s\n' "$args" >> "$FAKE_KUBECTL_LOG"
fi
case "$args" in
  *"-n kubeflow get service seaweedfs -o json"*)
    echo '{"spec":{"clusterIP":"10.96.1.1","ports":[{"name":"http-s3-compat","port":9000}]}}'
    ;;
  *"-n kubeflow get endpointslice"*)
    echo '{"items":[{"metadata":{"name":"seaweedfs-a"},"ports":[{"name":"http-s3-compat","port":8333}],"endpoints":[{"addresses":["10.244.0.21"],"conditions":{"ready":false}},{"addresses":["10.244.0.20"],"conditions":{"ready":true,"serving":true,"terminating":false}}]}]}'
    ;;
  *"-n default get service kubernetes -o json"*)
    echo '{"spec":{"clusterIP":"10.96.0.1","ports":[{"name":"https","port":443}]}}'
    ;;
  *"-n default get endpointslice"*)
    echo '{"items":[{"metadata":{"name":"kubernetes-a"},"ports":[{"name":"https","port":6443}],"endpoints":[{"addresses":["172.18.0.2"],"conditions":{"ready":true}}]}]}'
    ;;
  *)
    exit 1
    ;;
esac
'''


if __name__ == '__main__':
    unittest.main()

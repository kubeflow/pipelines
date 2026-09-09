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
        self.assertIn('kube-dns-tcp\tvip\t10.96.0.10\t53\n', targets)
        self.assertIn(
            'kube-dns-tcp\tendpoint\t10.244.0.2\t53\n', targets
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

    def test_teardown_quiesces_probe_before_final_observation_and_cleanup(self):
        script = SCRIPT.read_text(encoding='utf-8')
        stop_body = script[
            script.index('stop_observability() {'):
            script.index('report_packet_capture() {')
        ]

        quiesce = stop_body.index('quiesce_probe ')
        target_snapshot = stop_body.index('capture_target_network_state ')
        kindnet_snapshot = stop_body.index('capture_kindnet_state ')
        collect_logs = stop_body.index('collect_probe ')
        stop_sampler = stop_body.index('stop_process ')
        stop_capture = stop_body.index('stop_packet_capture ')
        cleanup = stop_body.index('cleanup_probe ')
        final_sample = stop_body.index('wait-for-following-sample')
        self.assertLess(quiesce, collect_logs)
        self.assertLess(collect_logs, target_snapshot)
        self.assertLess(target_snapshot, kindnet_snapshot)
        self.assertLess(kindnet_snapshot, final_sample)
        self.assertLess(final_sample, stop_sampler)
        self.assertLess(stop_sampler, stop_capture)
        self.assertLess(stop_capture, cleanup)

    def test_packet_capture_uses_runner_owner_and_validates_output(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            for name, contents in {
                'kubectl': _CAPTURE_KUBECTL,
                'docker': _CAPTURE_DOCKER,
                'awk': _CAPTURE_AWK,
                'sudo': _CAPTURE_SUDO,
                'nsenter': _CAPTURE_NSENTER,
                'tcpdump': _CAPTURE_TCPDUMP,
            }.items():
                executable = bin_directory / name
                executable.write_text(contents, encoding='utf-8')
                executable.chmod(0o755)
            output_directory = temporary_path / 'output'
            output_directory.mkdir()
            targets_path = temporary_path / 'targets.tsv'
            targets_path.write_text(
                'seaweedfs-s3\tvip\t10.96.1.1\t9000\n', encoding='utf-8'
            )
            tcpdump_log = temporary_path / 'tcpdump.args'
            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'
            environment['FAKE_TCPDUMP_LOG'] = str(tcpdump_log)

            subprocess.run(
                [
                    'bash',
                    str(SCRIPT),
                    'start-capture',
                    str(output_directory),
                    str(targets_path),
                ],
                check=True,
                env=environment,
            )
            status = (output_directory / 'tcpdump.status').read_text(
                encoding='utf-8'
            )
            arguments = tcpdump_log.read_text(encoding='utf-8')
            subprocess.run(
                [
                    'bash',
                    str(SCRIPT),
                    'stop-capture',
                    str(output_directory),
                ],
                check=True,
                env=environment,
            )
            capture_file_exists = (
                output_directory
                / 'connection-attempts-kind-control-plane.pcap0'
            ).is_file()

        self.assertIn('active: node kind-control-plane', status)
        self.assertIn('-C 5 -W 2', arguments)
        capture_user = subprocess.run(
            ['id', '-un'], check=True, capture_output=True, text=True
        ).stdout.strip()
        self.assertIn(
            f'-Z {capture_user}',
            arguments,
        )
        self.assertTrue(capture_file_exists)

    def test_kindnet_snapshot_retains_node_state_when_api_later_fails(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            for name, contents in {
                'kubectl': _KINDNET_KUBECTL,
                'docker': _KINDNET_DOCKER,
                'timeout': _KINDNET_TIMEOUT,
            }.items():
                executable = bin_directory / name
                executable.write_text(contents, encoding='utf-8')
                executable.chmod(0o755)
            output_directory = temporary_path / 'output'
            output_directory.mkdir()
            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'

            subprocess.run(
                [
                    'bash',
                    str(SCRIPT),
                    'capture-kindnet-state',
                    str(output_directory),
                    'baseline',
                ],
                check=True,
                env=environment,
            )
            state = (
                output_directory
                / 'kindnet-kind-control-plane-baseline.txt'
            ).read_text(encoding='utf-8')
            nodes = (output_directory / 'kind-nodes.txt').read_text(
                encoding='utf-8'
            )

        self.assertEqual(nodes, 'kind-control-plane\n')
        self.assertIn('container_id=kindnet-container', state)
        self.assertIn('__NFQUEUE__', state)
        self.assertIn('101 2538 3 2 65535 7 5 106 1', state)
        self.assertIn('__NFT__', state)
        self.assertIn('kindnet-network-policies', state)
        self.assertLess(state.index('__NFQUEUE__'), state.index('__CONTAINERS__'))
        script = SCRIPT.read_text(encoding='utf-8')
        self.assertIn('head -n 2', script)
        self.assertIn('crictl logs --tail 200', script)

    def test_kindnet_log_capture_keeps_recent_deploy_and_test_errors(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            kubectl = bin_directory / 'kubectl'
            kubectl.write_text(_KINDNET_LOG_KUBECTL, encoding='utf-8')
            kubectl.chmod(0o755)
            output_directory = temporary_path / 'output'
            output_directory.mkdir()
            kubectl_log = temporary_path / 'kubectl.log'
            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'
            environment['FAKE_KUBECTL_LOG'] = str(kubectl_log)

            subprocess.run(
                [
                    'bash',
                    str(SCRIPT),
                    'capture-kindnet-logs',
                    str(output_directory),
                ],
                check=True,
                env=environment,
            )
            recent_log = (output_directory / 'kindnet-recent.log').read_text(
                encoding='utf-8'
            )
            status = (output_directory / 'kindnet-log.status').read_text(
                encoding='utf-8'
            )
            arguments = kubectl_log.read_text(encoding='utf-8')

        self.assertIn('failed to set verdict with label', recent_log)
        self.assertIn('collected', status)
        self.assertIn('--request-timeout=10s', arguments)
        self.assertIn('--since=2h', arguments)
        self.assertIn('--tail=-1', arguments)
        self.assertIn('--timestamps=true', arguments)
        self.assertIn('--prefix=true', arguments)

    def test_packet_capture_reports_early_exit_as_unavailable(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            for name, contents in {
                'kubectl': _CAPTURE_KUBECTL,
                'docker': _CAPTURE_DOCKER,
                'awk': _CAPTURE_AWK,
                'sudo': _CAPTURE_SUDO,
                'nsenter': _CAPTURE_NSENTER,
                'tcpdump': _CAPTURE_TCPDUMP,
            }.items():
                executable = bin_directory / name
                executable.write_text(contents, encoding='utf-8')
                executable.chmod(0o755)
            output_directory = temporary_path / 'output'
            output_directory.mkdir()
            targets_path = temporary_path / 'targets.tsv'
            targets_path.write_text(
                'seaweedfs-s3\tvip\t10.96.1.1\t9000\n', encoding='utf-8'
            )
            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'
            environment['FAKE_TCPDUMP_LOG'] = str(
                temporary_path / 'tcpdump.args'
            )
            environment['FAKE_TCPDUMP_EXIT'] = '1'

            subprocess.run(
                [
                    'bash',
                    str(SCRIPT),
                    'start-capture',
                    str(output_directory),
                    str(targets_path),
                ],
                check=True,
                env=environment,
            )
            status = (output_directory / 'tcpdump.status').read_text(
                encoding='utf-8'
            )

        self.assertIn(
            'unavailable: capture exited before producing a pcap', status
        )


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
    echo '{"items":[{"metadata":{"name":"seaweedfs-a"},"ports":[{"name":"http-s3-compat","port":8333}],"endpoints":[{"addresses":["10.244.0.21"],"conditions":{"ready":false},"targetRef":{"kind":"Pod","name":"seaweedfs-old"}},{"addresses":["10.244.0.20"],"conditions":{"ready":true,"serving":true,"terminating":false},"targetRef":{"kind":"Pod","name":"seaweedfs-ready"}}]}]}'
    ;;
  *"-n default get service kubernetes -o json"*)
    echo '{"spec":{"clusterIP":"10.96.0.1","ports":[{"name":"https","port":443}]}}'
    ;;
  *"-n default get endpointslice"*)
    echo '{"items":[{"metadata":{"name":"kubernetes-a"},"ports":[{"name":"https","port":6443}],"endpoints":[{"addresses":["172.18.0.2"],"conditions":{"ready":true}}]}]}'
    ;;
  *"-n kube-system get service kube-dns -o json"*)
    echo '{"spec":{"clusterIP":"10.96.0.10","ports":[{"name":"dns-tcp","port":53}]}}'
    ;;
  *"-n kube-system get endpointslice"*)
    echo '{"items":[{"metadata":{"name":"kube-dns-a"},"ports":[{"name":"dns-tcp","port":53}],"endpoints":[{"addresses":["10.244.0.2"],"conditions":{"ready":true}}]}]}'
    ;;
  *)
    exit 1
    ;;
esac
'''

_CAPTURE_KUBECTL = r'''#!/usr/bin/env bash
echo "kind-control-plane"
'''

_CAPTURE_DOCKER = r'''#!/usr/bin/env bash
echo "1234"
'''

_CAPTURE_AWK = r'''#!/usr/bin/env bash
echo "test-start"
'''

_CAPTURE_SUDO = r'''#!/usr/bin/env bash
if [[ "${5:-}" == "kfp-network-capture-check" ]]; then
  exit 0
fi
shift
exec "$@"
'''

_CAPTURE_NSENTER = r'''#!/usr/bin/env bash
shift 3
exec "$@"
'''

_CAPTURE_TCPDUMP = r'''#!/usr/bin/env bash
printf '%s\n' "$*" > "$FAKE_TCPDUMP_LOG"
if [[ "${FAKE_TCPDUMP_EXIT:-0}" == "1" ]]; then
  exit 1
fi
output=""
while (($#)); do
  if [[ "$1" == "-w" ]]; then
    output="$2"
    break
  fi
  shift
done
: > "${output}0"
trap 'exit 0' INT TERM
while true; do sleep 1; done
'''


_KINDNET_KUBECTL = r'''#!/usr/bin/env bash
case "$*" in
  *"get nodes -o jsonpath="*)
    echo "kind-control-plane"
    ;;
  *"get pods -l k8s-app=kindnet -o wide"*)
    echo "NAME READY STATUS RESTARTS NODE"
    echo "kindnet-abc 1/1 Running 0 kind-control-plane"
    ;;
  *"get pods -l k8s-app=kindnet -o json"*)
    echo '{"items":[]}'
    ;;
  *)
    exit 1
    ;;
esac
'''


_KINDNET_DOCKER = r'''#!/usr/bin/env bash
cat <<'EOF'
__NFQUEUE__
101 2538 3 2 65535 7 5 106 1
__NFT__
{"nftables":[{"table":{"family":"inet","name":"kindnet-network-policies","handle":9}}]}
__CONTAINERS__
container_id=kindnet-container
__LOG_kindnet-container__
kindnet log line
EOF
'''


_KINDNET_TIMEOUT = r'''#!/usr/bin/env bash
shift
exec "$@"
'''


_KINDNET_LOG_KUBECTL = r'''#!/usr/bin/env bash
printf '%s\n' "$*" > "$FAKE_KUBECTL_LOG"
echo '[pod/kindnet-abc/kindnet-cni] 2026-07-17T16:19:01Z "failed to set verdict with label"'
'''


if __name__ == '__main__':
    unittest.main()

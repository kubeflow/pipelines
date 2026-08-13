#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
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
import textwrap
import unittest


SCRIPT = Path(__file__).with_name('collect-seaweedfs-diagnostics.sh')


class CollectSeaweedfsDiagnosticsTest(unittest.TestCase):

    def test_correlates_connection_failures_with_service_backends(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            failure_log = temporary_path / 'failures.log'
            failure_log.write_text(
                textwrap.dedent('''\
                    dial tcp 10.96.1.1:9000: i/o timeout
                    dial tcp 10.96.2.2:8443: connect: connection refused
                    dial tcp 10.96.2.2:8443: connect: connection refused
                    dial tcp 10.96.3.3:443: connect: connection reset by peer
                    read tcp 10.244.0.7:43120->10.96.4.4:8080: read: connection reset by peer
                    dial tcp 10.96.5.5:53: connect: connection refused
                    read udp 10.244.0.7:51234->10.96.5.5:53: i/o timeout
                    dial tcp 10.244.0.99:9090: connect: connection refused
                    connection refused without a numeric destination
                '''),
                encoding='utf-8',
            )
            self._write_executable(bin_directory / 'kubectl', _FAKE_KUBECTL)
            self._write_executable(
                bin_directory / 'docker',
                '#!/usr/bin/env bash\nexit 1\n',
            )

            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'
            result = subprocess.run(
                [
                    'bash',
                    str(SCRIPT),
                    '--namespace',
                    'kubeflow',
                    '--connection-failure-log',
                    str(failure_log),
                ],
                check=True,
                capture_output=True,
                text=True,
                env=environment,
            )

        output = result.stdout
        self.assertIn(
            'ClusterIPs correlated: tcp://10.96.1.1:9000 '
            'tcp://10.96.2.2:8443 tcp://10.96.3.3:443 '
            'tcp://10.96.4.4:8080 tcp://10.96.5.5:53 '
            'udp://10.96.5.5:53',
            output,
        )
        self.assertIn(
            'Service: opendatahub/mlflow (failed VIP 10.96.2.2:8443)',
            output,
        )
        self.assertIn('mlflow-slice  10.244.0.30', output)
        self.assertIn(
            'Backing pod: opendatahub/mlflow-0 '
            '(10.244.0.30; ready=true serving=true terminating=false)',
            output,
        )
        self.assertIn(
            'container=mlflow ready=true restarts=2 state=running '
            'lastState=terminated:Error',
            output,
        )
        self.assertIn('Warning  Unhealthy  MLflow readiness probe failed', output)
        self.assertIn(
            'Service: kubeflow/metadata-grpc '
            '(failed VIP 10.96.4.4:8080)',
            output,
        )
        self.assertIn(
            'Service: kube-system/kube-dns (failed VIP 10.96.5.5:53)', output
        )
        self.assertIn('coredns-slice  10.244.0.50', output)
        self.assertEqual(
            output.count(
                'Service: kube-system/kube-dns (failed VIP 10.96.5.5:53)'
            ),
            1,
        )
        self.assertIn(
            'Targets not in the current Service inventory (backend lookup '
            'skipped; node program probed in section 5): '
            'tcp://10.244.0.99:9090',
            output,
        )
        # Host-level capture: rendered once, regardless of docker/node state.
        self.assertIn(
            'runner-kernel softirq backlog (shared by every Kind node)', output
        )
        # Unowned targets keep their node-level probe (a deleted/re-IP'd
        # Service's stale VIP is exactly that evidence) but never get a
        # backend-object lookup: absent from section (4)'s correlated list,
        # present in section (5)'s.
        section_four_correlated = output.split('----- (4)')[1].splitlines()[1]
        section_five_correlated = output.split('----- (5)')[1].splitlines()[1]
        self.assertNotIn('10.244.0.99:9090', section_four_correlated)
        self.assertIn('10.244.0.99:9090', section_five_correlated)
        self.assertNotIn('Service:  (failed VIP 10.244.0.99:9090)', output)
        self.assertEqual(
            output.count(
                'Service: opendatahub/mlflow (failed VIP 10.96.2.2:8443)'
            ),
            1,
        )
        self.assertNotIn('without a numeric destination', output)

    def test_legacy_dial_timeout_log_option_remains_supported(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            failure_log = temporary_path / 'failures.log'
            failure_log.write_text(
                'dial tcp 10.96.2.2:8443: connect: connection refused\n',
                encoding='utf-8',
            )
            self._write_executable(bin_directory / 'kubectl', _FAKE_KUBECTL)
            self._write_executable(
                bin_directory / 'docker',
                '#!/usr/bin/env bash\nexit 1\n',
            )
            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'

            result = subprocess.run(
                [
                    'bash',
                    str(SCRIPT),
                    '--dial-timeout-log',
                    str(failure_log),
                ],
                check=True,
                capture_output=True,
                text=True,
                env=environment,
            )

        self.assertIn('10.96.2.2:8443', result.stdout)

    def test_resolves_service_dns_deadline_to_cluster_ip(self):
        result = self._run_with_fake_cluster(
            'MLflow request failed: Post '
            '"https://mlflow.opendatahub.svc.cluster.local:8443/mlflow/api/2.0/'
            'mlflow/runs/create": context deadline exceeded '
            '(Client.Timeout exceeded while awaiting headers)\n',
            {},
        )

        self.assertIn(
            'ClusterIPs correlated: tcp://10.96.1.1:9000 '
            'tcp://10.96.2.2:8443',
            result.stdout,
        )
        self.assertIn(
            'Service: opendatahub/mlflow (failed VIP 10.96.2.2:8443)',
            result.stdout,
        )

    def test_missing_seaweedfs_pod_still_probes_other_service_vips(self):
        # The generic inventory and sections 4-5 must not depend on the
        # SeaweedFS pod: an MLflow VIP failure deserves its Service/backend
        # snapshot even when SeaweedFS is absent.
        result = self._run_with_fake_cluster(
            'dial tcp 10.96.2.2:8443: connect: connection refused\n',
            {'NO_SEAWEEDFS_POD': 'true'},
        )
        output = result.stdout
        self.assertIn('skipping pod-scoped sections', output)
        self.assertIn(
            'Service: opendatahub/mlflow (failed VIP 10.96.2.2:8443)', output
        )
        self.assertIn(
            'ClusterIPs correlated: tcp://10.96.1.1:9000 '
            'tcp://10.96.2.2:8443',
            output,
        )
        self.assertIn('No SeaweedFS pod; skipping socket state.', output)

    def test_udp_timeout_selects_udp_service_rule(self):
        result = self._run_with_fake_cluster(
            'read udp 10.244.0.7:51234->10.96.5.5:53: i/o timeout\n',
            {'FAKE_KIND_NODES': 'true'},
            docker_script=_FAKE_DOCKER,
            iptables_save_script=_FAKE_IPTABLES_SAVE,
        )

        self.assertIn(
            'service program for udp://10.96.5.5:53 (iptables, then ipvs)',
            result.stdout,
        )
        self.assertIn('KUBE-SVC-UDPDNS', result.stdout)
        self.assertIn('--to-destination 10.244.0.50:53', result.stdout)
        self.assertNotIn('KUBE-SVC-TCPDNS', result.stdout)
        self.assertNotIn('--to-destination 10.244.0.51:53', result.stdout)

    def test_reports_service_inventory_api_failure_as_unavailable(self):
        result = self._run_with_fake_cluster(
            'dial tcp 10.96.2.2:8443: connect: connection refused\n',
            {'FAIL_SERVICE_LOOKUP': 'true'},
        )

        self.assertIn('Service inventory unavailable', result.stdout)
        self.assertIn(
            'Service lookup unavailable for 10.96.2.2:8443.', result.stdout
        )
        self.assertNotIn('No Kubernetes Service currently owns', result.stdout)

    def test_reports_endpointslice_api_failure_as_unavailable(self):
        result = self._run_with_fake_cluster(
            'dial tcp 10.96.2.2:8443: connect: connection refused\n',
            {'FAIL_ENDPOINTSLICE_LOOKUP': 'true'},
        )

        self.assertIn('(EndpointSlice backend lookup unavailable)', result.stdout)
        self.assertNotIn('(no EndpointSlice backends found)', result.stdout)

    def test_reports_counted_service_and_masquerade_rules(self):
        for random_fully, expected_status in (
            ('true', 'enabled'),
            ('false', 'not present'),
        ):
            with self.subTest(random_fully=random_fully):
                result = self._run_with_fake_cluster(
                    'dial tcp 10.96.1.1:9000: i/o timeout\n',
                    {
                        'FAKE_KIND_NODES': 'true',
                        'RANDOM_FULLY': random_fully,
                    },
                    docker_script=_FAKE_DOCKER,
                    iptables_save_script=_FAKE_IPTABLES_SAVE,
                )
                output = result.stdout

                self.assertIn(
                    'conntrack insertion/drop counters '
                    '(cumulative node-wide; nonzero requires correlation)',
                    output,
                )
                self.assertNotIn('nonzero == table pressure', output)
                self.assertIn(
                    'iptables counters: cumulative [packets:bytes] values',
                    output,
                )
                self.assertIn(
                    '[11:660] -A KUBE-SERVICES -d 10.96.1.1/32', output
                )
                self.assertIn(
                    '[0:0] -A KUBE-SVC-SEAWEED ! -s 10.244.0.0/16', output
                )
                self.assertIn(
                    '[0:0] -A KUBE-SEP-SEAWEED -s 10.244.0.20/32', output
                )
                self.assertIn(
                    '[11:660] -A KUBE-SEP-SEAWEED -p tcp', output
                )
                self.assertIn(
                    '[0:0] -A KUBE-POSTROUTING -m mark --mark 0x4000/0x4000',
                    output,
                )
                self.assertIn(
                    '[0:0] -A KUBE-MARK-MASQ -j MARK '
                    '--set-xmark 0x4000/0x4000',
                    output,
                )
                self.assertIn(
                    f'MASQUERADE --random-fully: {expected_status}', output
                )
                self.assertEqual(
                    output.count('node-wide SNAT / masquerade plumbing:'), 1
                )

    def test_ipvs_fallback_matches_vip_and_port_literally(self):
        result = self._run_with_fake_cluster(
            'dial tcp 10.96.1.1:9000: i/o timeout\n',
            {'FAKE_KIND_NODES': 'true'},
            docker_script=_FAKE_DOCKER,
            iptables_save_script='#!/usr/bin/env bash\nexit 1\n',
            ipvsadm_script=_FAKE_IPVSADM,
        )

        self.assertIn('10.96.1.1:9000', result.stdout)
        self.assertIn('10.244.0.20:8333', result.stdout)
        self.assertNotIn('regex-decoy', result.stdout)
        self.assertNotIn('next-service-backend', result.stdout)

    def _run_with_fake_cluster(
        self,
        failure_text: str,
        extra_environment=None,
        docker_script='#!/usr/bin/env bash\nexit 1\n',
        iptables_save_script=None,
        ipvsadm_script=None,
    ):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            failure_log = temporary_path / 'failures.log'
            failure_log.write_text(failure_text, encoding='utf-8')
            self._write_executable(bin_directory / 'kubectl', _FAKE_KUBECTL)
            self._write_executable(bin_directory / 'docker', docker_script)
            if iptables_save_script is not None:
                self._write_executable(
                    bin_directory / 'iptables-save', iptables_save_script
                )
            if ipvsadm_script is not None:
                self._write_executable(bin_directory / 'ipvsadm', ipvsadm_script)
            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'
            environment.update(extra_environment or {})
            result = subprocess.run(
                [
                    'bash',
                    str(SCRIPT),
                    '--connection-failure-log',
                    str(failure_log),
                ],
                check=True,
                capture_output=True,
                text=True,
                env=environment,
            )
            self.assertEqual('', result.stderr, result.stderr)
            return result

    @staticmethod
    def _write_executable(path: Path, contents: str):
        path.write_text(contents, encoding='utf-8')
        path.chmod(0o755)


_FAKE_KUBECTL = r'''#!/usr/bin/env bash
args="$*"

case "$args" in
  *"get pod -n kubeflow -l app=seaweedfs"*)
    [[ "${NO_SEAWEEDFS_POD:-false}" == "true" ]] || echo "seaweedfs-0"
    ;;
  *"get pod seaweedfs-0 -n kubeflow -o wide"*)
    echo "seaweedfs-0  1/1  Running  0  10m  10.244.0.20"
    ;;
  *"get pod seaweedfs-0 -n kubeflow -o jsonpath="*"restartCount="*)
    printf 'restartCount=0\nlastState={}\nqosClass=Burstable\n'
    ;;
  *"get pod seaweedfs-0 -n kubeflow -o jsonpath="*) echo "kind-control-plane" ;;
  *"describe node kind-control-plane"*) echo "Conditions:" ;;
  *"get pod -n kube-system -l k8s-app=kube-proxy"*)
    echo "kube-proxy  1/1  Running"
    ;;
  *"get endpoints seaweedfs -n kubeflow"*)
    echo "seaweedfs  10.244.0.20:8333"
    ;;
  *"get svc seaweedfs -n kubeflow"*) echo "10.96.1.1" ;;
  *"get service -A"*)
    [[ "${FAIL_SERVICE_LOOKUP:-false}" != "true" ]] || exit 1
    printf 'kubeflow\tseaweedfs\t10.96.1.1\n'
    printf 'opendatahub\tmlflow\t10.96.2.2\n'
    printf 'default\tkubernetes\t10.96.3.3\n'
    printf 'kubeflow\tmetadata-grpc\t10.96.4.4\n'
    printf 'kube-system\tkube-dns\t10.96.5.5\n'
    ;;
  *"get service mlflow -n opendatahub"*)
    echo "mlflow  ClusterIP  10.96.2.2  8443/TCP"
    ;;
  *"get service seaweedfs -n kubeflow"*)
    echo "seaweedfs  ClusterIP  10.96.1.1  9000/TCP"
    ;;
  *"get service kubernetes -n default"*)
    echo "kubernetes  ClusterIP  10.96.3.3  443/TCP"
    ;;
  *"get service metadata-grpc -n kubeflow"*)
    echo "metadata-grpc  ClusterIP  10.96.4.4  8080/TCP"
    ;;
  *"get service kube-dns -n kube-system"*)
    echo "kube-dns  ClusterIP  10.96.5.5  53/UDP,53/TCP"
    ;;
  *"get endpointslice -n opendatahub"*"-o wide"*)
    echo "mlflow-slice  10.244.0.30"
    ;;
  *"get endpointslice -n opendatahub"*"-o jsonpath="*)
    [[ "${FAIL_ENDPOINTSLICE_LOOKUP:-false}" != "true" ]] || exit 1
    echo '10.244.0.30|true|true|false|Pod|opendatahub|mlflow-0'
    ;;
  *"get endpointslice -n kubeflow"*"service-name=seaweedfs"*"-o wide"*)
    echo "seaweedfs-slice  10.244.0.20"
    ;;
  *"get endpointslice -n kubeflow"*"service-name=seaweedfs"*"-o jsonpath="*)
    echo '10.244.0.20|true|true|false|Pod|kubeflow|seaweedfs-0'
    ;;
  *"get endpointslice -n kubeflow"*"service-name=metadata-grpc"*"-o wide"*)
    echo "metadata-slice  10.244.0.40"
    ;;
  *"get endpointslice -n kubeflow"*"service-name=metadata-grpc"*"-o jsonpath="*)
    echo '10.244.0.40|true|true|false|Pod|kubeflow|metadata-0'
    ;;
  *"get endpointslice -n default"*"-o wide"*) echo "kubernetes-slice  172.18.0.2" ;;
  *"get endpointslice -n default"*"-o jsonpath="*) echo "" ;;
  *"get endpointslice -n kube-system"*"service-name=kube-dns"*"-o wide"*)
    echo "coredns-slice  10.244.0.50"
    ;;
  *"get endpointslice -n kube-system"*"service-name=kube-dns"*"-o jsonpath="*)
    echo '10.244.0.50|true|true|false|Pod|kube-system|coredns-0'
    ;;
  *"get pod mlflow-0 -n opendatahub -o wide"*)
    echo "mlflow-0  1/1  Running  2  10m  10.244.0.30"
    ;;
  *"get pod mlflow-0 -n opendatahub -o jsonpath="*)
    echo "container=mlflow ready=true restarts=2 state=running lastState=terminated:Error"
    ;;
  *"describe pod mlflow-0 -n opendatahub"*)
    printf 'Events:\nWarning  Unhealthy  MLflow readiness probe failed\n'
    ;;
  *"get pod metadata-0 -n kubeflow -o wide"*) echo "metadata-0  1/1  Running" ;;
  *"get pod metadata-0 -n kubeflow -o jsonpath="*)
    echo "container=metadata ready=true restarts=0"
    ;;
  *"describe pod metadata-0 -n kubeflow"*) echo "Events:  <none>" ;;
  *"get pod coredns-0 -n kube-system -o wide"*) echo "coredns-0  1/1  Running" ;;
  *"get pod coredns-0 -n kube-system -o jsonpath="*)
    echo "container=coredns ready=true restarts=0"
    ;;
  *"describe pod coredns-0 -n kube-system"*) echo "Events:  <none>" ;;
  *"get nodes -o jsonpath="*)
    [[ "${FAKE_KIND_NODES:-false}" == "true" ]] && echo "kind-control-plane"
    ;;
  *"exec seaweedfs-0"*) echo "socket diagnostics" ;;
  *"describe pod seaweedfs-0 -n kubeflow"*) echo "Events:  <none>" ;;
  *) true ;;
esac
'''

_FAKE_DOCKER = r'''#!/usr/bin/env bash
if [[ "$1" == "inspect" ]]; then
  exit 0
fi
if [[ "$1" != "exec" || "$3" != "sh" || "$4" != "-c" ]]; then
  exit 1
fi

script="$5"
case "$script" in
  *"nf_conntrack_count"*) echo "20/262144" ;;
  *"command -v conntrack"*)
    echo "cpu=0 insert_failed=2 drop=2 early_drop=0"
    ;;
  *"dmesg"*) echo "(no table-full event logged)" ;;
  *)
    shift 5
    sh -c "$script" "$@"
    ;;
esac
'''

_FAKE_IPTABLES_SAVE = r'''#!/usr/bin/env bash
[[ "${1:-}" == "-c" ]] || exit 1
random_fully=""
if [[ "${RANDOM_FULLY:-false}" == "true" ]]; then
  random_fully=" --random-fully"
fi
cat <<EOF
[11:660] -A KUBE-SERVICES -d 10.96.1.1/32 -p tcp --dport 9000 -j KUBE-SVC-SEAWEED
[0:0] -A KUBE-SVC-SEAWEED ! -s 10.244.0.0/16 -d 10.96.1.1/32 -p tcp --dport 9000 -j KUBE-MARK-MASQ
[11:660] -A KUBE-SVC-SEAWEED -j KUBE-SEP-SEAWEED
[0:0] -A KUBE-SEP-SEAWEED -s 10.244.0.20/32 -j KUBE-MARK-MASQ
[11:660] -A KUBE-SEP-SEAWEED -p tcp -j DNAT --to-destination 10.244.0.20:8333
[0:0] -A KUBE-MARK-MASQ -j MARK --set-xmark 0x4000/0x4000
[0:0] -A KUBE-POSTROUTING -m mark --mark 0x4000/0x4000 -j MASQUERADE${random_fully}
[7:420] -A KUBE-SERVICES -d 10.96.5.5/32 -p tcp -m tcp --dport 53 -j KUBE-SVC-TCPDNS
[9:540] -A KUBE-SERVICES -d 10.96.5.5/32 -p udp -m udp --dport 53 -j KUBE-SVC-UDPDNS
[7:420] -A KUBE-SVC-TCPDNS -j KUBE-SEP-TCPDNS
[9:540] -A KUBE-SVC-UDPDNS -j KUBE-SEP-UDPDNS
[7:420] -A KUBE-SEP-TCPDNS -p tcp -j DNAT --to-destination 10.244.0.51:53
[9:540] -A KUBE-SEP-UDPDNS -p udp -j DNAT --to-destination 10.244.0.50:53
EOF
'''

_FAKE_IPVSADM = r'''#!/usr/bin/env bash
cat <<'EOF'
TCP  10x96x1x1:9000 rr
  -> regex-decoy:8333           Masq    1      0          0
decoy-context-1
decoy-context-2
decoy-context-3
decoy-context-4
decoy-context-5
decoy-context-6
decoy-context-7
TCP  10.96.1.1:9000 rr
  -> 10.244.0.20:8333           Masq    1      0          0
UDP  10.96.1.1:9000 rr
  -> next-service-backend:9000  Masq    1      0          0
EOF
'''


if __name__ == '__main__':
    unittest.main()

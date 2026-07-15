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
            'ClusterIPs correlated: 10.96.1.1:9000 10.96.2.2:8443 '
            '10.96.3.3:443 10.96.4.4:8080',
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
            'Non-Service connection targets ignored: 10.244.0.99:9090', output
        )
        self.assertNotIn('service program for 10.244.0.99:9090', output)
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

    def _run_with_fake_cluster(self, failure_text: str, extra_environment=None):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            bin_directory = temporary_path / 'bin'
            bin_directory.mkdir()
            failure_log = temporary_path / 'failures.log'
            failure_log.write_text(failure_text, encoding='utf-8')
            self._write_executable(bin_directory / 'kubectl', _FAKE_KUBECTL)
            self._write_executable(
                bin_directory / 'docker',
                '#!/usr/bin/env bash\nexit 1\n',
            )
            environment = os.environ.copy()
            environment['PATH'] = f'{bin_directory}:{environment["PATH"]}'
            environment.update(extra_environment or {})
            return subprocess.run(
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

    @staticmethod
    def _write_executable(path: Path, contents: str):
        path.write_text(contents, encoding='utf-8')
        path.chmod(0o755)


_FAKE_KUBECTL = r'''#!/usr/bin/env bash
args="$*"

case "$args" in
  *"get pod -n kubeflow -l app=seaweedfs"*) echo "seaweedfs-0" ;;
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
  *"get nodes -o jsonpath="*) echo "" ;;
  *"exec seaweedfs-0"*) echo "socket diagnostics" ;;
  *"describe pod seaweedfs-0 -n kubeflow"*) echo "Events:  <none>" ;;
  *) true ;;
esac
'''


if __name__ == '__main__':
    unittest.main()

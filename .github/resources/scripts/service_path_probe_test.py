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

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import service_path_probe


SCRIPT = Path(__file__).with_name('service_path_probe.py')


def service(cluster_ip, port_name, port):
    return {
        'spec': {
            'clusterIP': cluster_ip,
            'ports': [{'name': port_name, 'port': port}],
        }
    }


def slices(name, port_name, port, endpoints):
    return {
        'items': [{
            'metadata': {'name': name},
            'ports': [{'name': port_name, 'port': port}],
            'endpoints': endpoints,
        }]
    }


class ServicePathProbeTest(unittest.TestCase):

    def test_resolves_service_and_endpoint_ports_separately(self):
        seaweedfs = service_path_probe.resolve_pair(
            'seaweedfs-s3',
            service('10.96.1.1', 'http-s3-compat', 9000),
            slices(
                'seaweedfs-a',
                'http-s3-compat',
                8333,
                [{
                    'addresses': ['10.244.0.20'],
                    'conditions': {
                        'ready': True,
                        'serving': True,
                        'terminating': False,
                    },
                }],
            ),
            'http-s3-compat',
        )
        kubernetes_api = service_path_probe.resolve_pair(
            'kubernetes-api',
            service('10.96.0.1', 'https', 443),
            slices(
                'kubernetes-a',
                'https',
                6443,
                [{'addresses': ['172.18.0.2'], 'conditions': {'ready': True}}],
            ),
            'https',
        )

        self.assertEqual(
            seaweedfs,
            [
                ('seaweedfs-s3', 'vip', '10.96.1.1', 9000),
                ('seaweedfs-s3', 'endpoint', '10.244.0.20', 8333),
            ],
        )
        self.assertEqual(
            kubernetes_api,
            [
                ('kubernetes-api', 'vip', '10.96.0.1', 443),
                ('kubernetes-api', 'endpoint', '172.18.0.2', 6443),
            ],
        )

    def test_skips_unready_serving_false_and_terminating_endpoints(self):
        endpoint_slices = slices(
            'seaweedfs-a',
            'http-s3-compat',
            8333,
            [
                {'addresses': ['10.244.0.1'], 'conditions': {'ready': False}},
                {'addresses': ['10.244.0.2'], 'conditions': {'serving': False}},
                {'addresses': ['10.244.0.3'], 'conditions': {'terminating': True}},
                {'addresses': ['10.244.0.4'], 'conditions': {'ready': True}},
            ],
        )

        targets = service_path_probe.resolve_pair(
            'seaweedfs-s3',
            service('10.96.1.1', 'http-s3-compat', 9000),
            endpoint_slices,
            'http-s3-compat',
        )

        self.assertEqual(targets[-1][2:], ('10.244.0.4', 8333))

    def test_probe_reports_timeout_without_raising(self):
        target = {
            'pair': 'seaweedfs-s3',
            'path': 'vip',
            'host': '10.96.1.1',
            'port': 9000,
        }
        with mock.patch(
            'service_path_probe.socket.create_connection',
            side_effect=TimeoutError(),
        ):
            result = service_path_probe.probe_target(target, 0.1)

        self.assertEqual(result['result'], 'timeout')
        self.assertEqual(result['error_type'], 'TimeoutError')

    def test_report_classifies_all_paired_outcomes_and_mirrors_summary(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_path = Path(temporary_directory)
            input_path = temporary_path / 'probe.jsonl'
            summary_path = temporary_path / 'summary.md'
            events = [{
                'type': 'meta',
                'source_pod': 'probe',
                'source_node': 'kind-control-plane',
                'source_pod_ip': '10.244.0.99',
                'network_namespace': 'net:[7]',
            }]
            outcomes = [
                ('ok', 'ok'),
                ('timeout', 'ok'),
                ('ok', 'error'),
                ('timeout', 'error'),
            ]
            for cycle, (vip_result, endpoint_result) in enumerate(outcomes):
                events.append({
                    'type': 'cycle',
                    'cycle': cycle,
                    'timestamp_ms': 1_700_000_000_000 + cycle * 5000,
                    'loop_lag_ms': cycle,
                })
                for path_name, result in (
                    ('vip', vip_result),
                    ('endpoint', endpoint_result),
                ):
                    events.append({
                        'type': 'probe',
                        'cycle': cycle,
                        'timestamp_ms': 1_700_000_000_000 + cycle * 5000,
                        'pair': 'seaweedfs-s3',
                        'path': path_name,
                        'host': '10.96.1.1' if path_name == 'vip' else '10.244.0.20',
                        'port': 9000 if path_name == 'vip' else 8333,
                        'result': result,
                        'latency_ms': 1.0 + cycle,
                    })
            events.extend([
                {
                    'type': 'metric',
                    'cycle': 0,
                    'timestamp_ms': 1_700_000_000_000,
                    'metric': 'TcpExt.TCPSynRetrans',
                    'value': 2,
                },
                {
                    'type': 'metric',
                    'cycle': 1,
                    'timestamp_ms': 1_700_000_005_000,
                    'metric': 'TcpExt.TCPSynRetrans',
                    'value': 9,
                },
            ])
            input_path.write_text(
                ''.join(json.dumps(event) + '\n' for event in events),
                encoding='utf-8',
            )
            environment = os.environ.copy()
            environment['GITHUB_STEP_SUMMARY'] = str(summary_path)
            result = subprocess.run(
                ['python3', str(SCRIPT), 'report', '--input', str(input_path)],
                check=True,
                capture_output=True,
                text=True,
                env=environment,
            )
            summary_contents = summary_path.read_text(encoding='utf-8')

        self.assertEqual(result.stdout, summary_contents)
        self.assertIn('| VIP failed / Endpoint succeeded | 1 |', result.stdout)
        self.assertIn('| both failed | 1 |', result.stdout)
        self.assertIn('| VIP succeeded / Endpoint failed | 1 |', result.stdout)
        self.assertIn('| both succeeded | 1 |', result.stdout)
        self.assertIn('| TcpExt.TCPSynRetrans | 2 | 7 | 7 |', result.stdout)


if __name__ == '__main__':
    unittest.main()

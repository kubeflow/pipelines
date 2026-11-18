# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
"""Offline regressions for asynchronous viewer pod deletion and replacement."""

import copy
import json
import os
import subprocess
import unittest
from unittest import mock

os.environ.setdefault('RUNNER_TEMP', '/tmp')
import deployed


def pod(image: str, *, deleting: bool = False, ready: bool = True) -> dict:
    return {
        'metadata': {
            'uid': image,
            **({'deletionTimestamp': 'now'} if deleting else {}),
        },
        'spec': {
            'containers': [{'image': image}],
            'securityContext': {'runAsUser': 1000},
        },
        'status': {
            'phase': 'Running',
            'containerStatuses': [{'ready': ready, 'imageID': 'sha256:test'}],
        },
    }


class ViewerLifecycleTest(unittest.TestCase):
    def create_with_pods(
        self, snapshots: list[list[dict]]
    ) -> tuple[str, str, str, str]:

        def kube(*args: str) -> str:
            if 'viewers' in args:
                return json.dumps(
                    {
                        'items': [
                            {
                                'metadata': {'name': 'viewer'},
                                'spec': {
                                    'tensorboardSpec': {'logDir': 'gs://logs'}
                                },
                            }
                        ]
                    }
                )
            if 'pods' in args:
                return json.dumps(
                    {
                        'items': snapshots.pop(0)
                        if len(snapshots) > 1
                        else snapshots[0]
                    }
                )
            return 'deployment/viewer'

        with (
            mock.patch.object(deployed, 'kube', side_effect=kube),
            mock.patch.object(
                deployed,
                'request',
                side_effect=[
                    b'proxy',
                    json.dumps(
                        {'image': 'custom', 'proxyPath': 'proxy'}
                    ).encode(),
                ],
            ),
            mock.patch.object(deployed.time, 'sleep'),
        ):
            return deployed.create('gs://logs', 'custom')

    def test_create_ignores_old_terminating_pod(self):
        result = self.create_with_pods(
            [[pod('default', deleting=True), pod('custom')]]
        )
        self.assertEqual(result[2], 'custom')

    def test_create_ignores_terminating_pod_with_same_image(self):
        old = pod('custom', deleting=True)
        old['metadata']['uid'] = 'old'
        result = self.create_with_pods([[old, pod('custom')]])
        self.assertEqual(result[2], 'custom')

    def test_create_waits_for_ready_replacement(self):
        result = self.create_with_pods(
            [[pod('custom', ready=False)], [pod('custom')]]
        )
        self.assertEqual(result[2], 'custom')

    def test_delete_waits_until_pods_are_gone(self):
        snapshots = [[pod('default', deleting=True)], []]
        observations = []

        def kube(*args: str) -> str:
            if 'pods' in args:
                observations.append(copy.deepcopy(snapshots[0]))
                return json.dumps({'items': snapshots.pop(0)})
            return ''

        with (
            mock.patch.object(deployed, 'kube', side_effect=kube),
            mock.patch.object(deployed, 'request'),
            mock.patch.object(deployed.time, 'sleep'),
        ):
            deployed.delete('gs://logs', 'viewer')
        self.assertEqual(len(observations), 2)
        self.assertEqual(observations[-1], [])

    def test_poll_recovers_from_transient_kubectl_failures(self):
        check = mock.Mock(
            side_effect=[
                RuntimeError('kubectl get pods failed'),
                subprocess.TimeoutExpired('kubectl', 30),
                'ready',
            ]
        )
        with mock.patch.object(deployed.time, 'sleep'):
            self.assertEqual(deployed.poll('viewer pod', check), 'ready')


if __name__ == '__main__':
    unittest.main()

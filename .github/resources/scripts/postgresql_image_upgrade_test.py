# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Guard the image acceptance contract without a Kubernetes cluster."""

import copy
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import postgresql_image_upgrade as upgrade


class PostgreSQLImageUpgradeTest(unittest.TestCase):
    def deployment(self, image):
        return {
            'kind': 'Deployment',
            'metadata': {'name': 'postgres'},
            'spec': {
                'template': {
                    'spec': {
                        'securityContext': {'fsGroup': 999},
                        'containers': [
                            {
                                'name': 'postgres',
                                'image': image,
                                'securityContext': {'runAsUser': 999},
                                'env': [
                                    {'name': 'PGDATA', 'value': '/data/pgdata'}
                                ],
                                'volumeMounts': [
                                    {'name': 'data', 'mountPath': '/data'}
                                ],
                            }
                        ],
                        'volumes': [
                            {
                                'name': 'data',
                                'persistentVolumeClaim': {
                                    'claimName': 'postgres-pvc'
                                },
                            }
                        ],
                    }
                }
            },
        }

    def test_candidate_comes_from_rendered_manifest_without_security_changes(
        self,
    ):
        image = 'docker.io/postgres:18.6-alpine3.23@sha256:' + 'a' * 64
        deployment = self.deployment(image)
        original = copy.deepcopy(deployment)
        resources, actual, version = upgrade.select_resources(
            [deployment], '18', 'isolated'
        )
        self.assertEqual(image, actual)
        self.assertEqual(180006, version)
        self.assertEqual(original, deployment)
        self.assertEqual(original['spec'], resources[0]['spec'])
        self.assertEqual('isolated', resources[0]['metadata']['namespace'])

    def test_wrong_major_or_unpinned_candidate_is_rejected(self):
        for image in ('postgres:18.6', 'postgres:19.0@sha256:' + 'b' * 64):
            with (
                self.subTest(image=image),
                self.assertRaisesRegex(ValueError, 'digest-pinned same-major'),
            ):
                upgrade.select_resources([self.deployment(image)], '18', 'test')

    def test_command_failure_is_not_hidden_by_capture(self):
        with self.assertRaises(subprocess.CalledProcessError):
            upgrade.run(['sh', '-c', 'exit 17'], capture_output=True)

    def test_application_failure_cannot_write_passed_receipt(self):
        with tempfile.TemporaryDirectory() as directory:
            acceptance = upgrade.Acceptance(
                '18', Path(directory), Path('/helper')
            )
            with (
                mock.patch.object(acceptance, 'forward'),
                mock.patch.object(
                    upgrade,
                    'run',
                    side_effect=subprocess.CalledProcessError(1, 'helper'),
                ),
            ):
                with self.assertRaises(subprocess.CalledProcessError):
                    acceptance.application('upgraded')
            self.assertNotIn('status', acceptance.receipt)
            self.assertFalse((Path(directory) / 'receipt.json').exists())

    def test_missing_table_fails_before_application_can_recreate_it(self):
        with tempfile.TemporaryDirectory() as directory:
            acceptance = upgrade.Acceptance(
                '18', Path(directory), Path('/helper')
            )
            with (
                mock.patch.object(
                    acceptance,
                    'sql',
                    side_effect=['digest', 'identity', 'experiments'],
                ),
                mock.patch.object(acceptance, 'application') as application,
            ):
                with self.assertRaisesRegex(RuntimeError, 'table set'):
                    acceptance.verify_retained(
                        'digest', 'identity', 'experiments,run_details'
                    )
                application.assert_not_called()

    def test_failed_restart_never_publishes_success(self):
        with tempfile.TemporaryDirectory() as directory:
            acceptance = upgrade.Acceptance(
                '18', Path(directory), Path('/helper')
            )
            with (
                mock.patch.object(
                    acceptance,
                    'deploy_old',
                    return_value=('postgres', 'candidate', 180006),
                ),
                mock.patch.object(
                    acceptance,
                    'seed',
                    return_value=('digest', 'identity', 'tables'),
                ),
                mock.patch.object(acceptance, 'upgrade'),
                mock.patch.object(acceptance, 'verify_retained'),
                mock.patch.object(
                    acceptance,
                    'restart_and_verify',
                    side_effect=RuntimeError('lost write'),
                ),
            ):
                with self.assertRaisesRegex(RuntimeError, 'lost write'):
                    acceptance.execute()
                self.assertNotIn('status', acceptance.receipt)
                self.assertFalse((Path(directory) / 'receipt.json').exists())


if __name__ == '__main__':
    unittest.main()

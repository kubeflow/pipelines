"""Verify the pinned MLMD upgrade lanes and production hold."""

import subprocess
import sys
import unittest
from pathlib import Path

import yaml


class UpgradeWorkflowTest(unittest.TestCase):
    def test_both_upgrade_modes_run_the_pinned_fixture(self):
        root = Path(__file__).resolve().parents[3]
        workflow = yaml.safe_load(
            (root / '.github/workflows/upgrade-test.yml').read_text()
        )
        jobs = workflow['jobs']
        lane = jobs['upgrade-test']
        self.assertNotIn('if', lane)
        self.assertNotIn('if', jobs['image-artifacts-ready'])
        self.assertEqual(
            lane['strategy']['matrix']['upgrade_mode'],
            ['cli-first', 'startup-only'],
        )
        steps = {step['name']: step for step in lane['steps']}
        self.assertIn(
            '555bfa60a70da88c594960bdacfd78fa7c061043',
            steps['Deploy pinned KFP 2.18.0']['run'],
        )
        self.assertIn(
            'run-mlmd-upgrade.sh',
            steps['Verify pinned upgrade and rollback']['run'],
        )
        self.assertIn(
            '--check-release-eligibility',
            steps['Load the target API server']['run'],
        )
        self.assertIn(
            '-tags=mlmd_migration_test',
            steps['Build and load the disposable migration binaries']['run'],
        )
        runner = (
            root / '.github/resources/scripts/run-mlmd-upgrade.sh'
        ).read_text()
        self.assertIn('production MLMD migration is disabled', runner)
        self.assertIn('verify-mlmd-upgrade.py', runner)

    def test_cutover_installs_overlay_config_before_the_api(self):
        root = Path(__file__).resolve().parents[3]
        manifests = root / '.github/resources/manifests/base'
        dns = yaml.safe_load((manifests / 'dns-config.yaml').read_text())
        dns['metadata']['namespace'] = 'kubeflow'
        deployment = yaml.safe_load(
            (manifests / 'apiserver-env.yaml').read_text()
        )
        deployment['metadata']['namespace'] = 'kubeflow'
        job = {'kind': 'Job', 'metadata': {'name': 'unrelated'}}
        command = [
            sys.executable,
            str(root / '.github/resources/scripts/mlmd_upgrade_deployment.py'),
            'api:test',
            'startup-only',
        ]
        result = subprocess.run(
            command,
            input=yaml.safe_dump_all([job, deployment, dns]),
            text=True,
            capture_output=True,
            check=True,
        )
        config, api = yaml.safe_load_all(result.stdout)
        self.assertEqual(dns, config)
        self.assertEqual(0, api['spec']['replicas'])
        (container,) = api['spec']['template']['spec']['containers']
        self.assertEqual('api:test', container['image'])
        self.assertIn(
            {
                'name': 'METADATA_GRPC_SERVICE_HOST',
                'value': 'metadata-grpc-service',
            },
            container['env'],
        )
        for objects in ([deployment], [deployment, dns, dns]):
            result = subprocess.run(
                command,
                input=yaml.safe_dump_all(objects),
                text=True,
                capture_output=True,
            )
            self.assertNotEqual(0, result.returncode)
            self.assertIn(
                'expected one kubeflow ConfigMap named dns-config',
                result.stderr,
            )

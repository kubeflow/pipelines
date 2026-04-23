"""Tests for DSPADeployer.output_deployment_metadata()."""

import unittest
from unittest.mock import MagicMock

from dspa_deployer import DSPADeployer


def _make_deployer(dspa_name: str = 'test-dspa') -> DSPADeployer:
    """Create a DSPADeployer with stubbed-out dependencies."""
    args = MagicMock()
    deployment_manager = MagicMock()
    return DSPADeployer(
        args=args,
        deployment_manager=deployment_manager,
        deployment_namespace='test-ns',
        dspa_name=dspa_name,
        external_db_namespace='ext-db-ns',
        operator_namespace='opendatahub',
        temp_dir='/tmp/test',
    )


class TestOutputDeploymentMetadataDirect(unittest.TestCase):
    """Direct (non-operator) mode metadata."""

    def setUp(self):
        self.metadata = _make_deployer().output_deployment_metadata(
            is_operator=False)

    def test_deployment_mode(self):
        self.assertEqual(self.metadata['DEPLOYMENT_MODE'], 'direct')

    def test_deployment_name(self):
        self.assertEqual(self.metadata['DEPLOYMENT_NAME'], 'ml-pipeline')

    def test_service_account_is_api_server_sa(self):
        self.assertEqual(self.metadata['SERVICE_ACCOUNT_NAME'], 'ml-pipeline')

    def test_pipeline_runner_service_account(self):
        self.assertEqual(
            self.metadata['PIPELINE_RUNNER_SERVICE_ACCOUNT'],
            'pipeline-runner',
        )

    def test_database_name(self):
        self.assertEqual(self.metadata['DATABASE_NAME'], 'mysql')

    def test_mlmd_service_name(self):
        self.assertEqual(self.metadata['MLMD_SERVICE_NAME'],
                         'metadata-grpc-service')

    def test_frontend_service_name(self):
        self.assertEqual(self.metadata['FRONTEND_SERVICE_NAME'],
                         'ml-pipeline-ui')


class TestOutputDeploymentMetadataOperator(unittest.TestCase):
    """Operator mode metadata."""

    def setUp(self):
        self.dspa_name = 'my-dspa'
        self.metadata = _make_deployer(
            dspa_name=self.dspa_name).output_deployment_metadata(
                is_operator=True)

    def test_deployment_mode(self):
        self.assertEqual(self.metadata['DEPLOYMENT_MODE'], 'operator')

    def test_deployment_name(self):
        self.assertEqual(self.metadata['DEPLOYMENT_NAME'],
                         f'ds-pipeline-{self.dspa_name}')

    def test_service_account_is_api_server_sa(self):
        self.assertEqual(self.metadata['SERVICE_ACCOUNT_NAME'],
                         f'ds-pipeline-{self.dspa_name}')

    def test_pipeline_runner_service_account(self):
        self.assertEqual(
            self.metadata['PIPELINE_RUNNER_SERVICE_ACCOUNT'],
            f'pipeline-runner-{self.dspa_name}',
        )

    def test_database_name(self):
        self.assertEqual(self.metadata['DATABASE_NAME'],
                         f'ds-pipeline-db-{self.dspa_name}')

    def test_mlmd_service_name(self):
        self.assertEqual(self.metadata['MLMD_SERVICE_NAME'],
                         f'ds-pipeline-metadata-grpc-{self.dspa_name}')

    def test_frontend_service_name(self):
        self.assertEqual(self.metadata['FRONTEND_SERVICE_NAME'],
                         f'ds-pipeline-ui-{self.dspa_name}')


class TestOutputDeploymentMetadataKeys(unittest.TestCase):
    """Both modes must return exactly the expected set of keys."""

    EXPECTED_KEYS = {
        'DEPLOYMENT_MODE',
        'DEPLOYMENT_NAME',
        'SERVICE_ACCOUNT_NAME',
        'PIPELINE_RUNNER_SERVICE_ACCOUNT',
        'DATABASE_NAME',
        'MLMD_SERVICE_NAME',
        'FRONTEND_SERVICE_NAME',
    }

    def test_direct_mode_keys(self):
        metadata = _make_deployer().output_deployment_metadata(
            is_operator=False)
        self.assertEqual(set(metadata.keys()), self.EXPECTED_KEYS)

    def test_operator_mode_keys(self):
        metadata = _make_deployer().output_deployment_metadata(
            is_operator=True)
        self.assertEqual(set(metadata.keys()), self.EXPECTED_KEYS)


if __name__ == '__main__':
    unittest.main()

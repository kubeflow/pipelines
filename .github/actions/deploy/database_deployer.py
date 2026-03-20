"""Database deployment (MariaDB).

Handles deploying external MariaDB and applying secrets/configmaps
required for database connectivity.
"""

import os

from deployment_manager import K8sDeploymentManager
from deployment_manager import ResourceType


class DatabaseDeployer:

    def __init__(self, args, deployment_manager: K8sDeploymentManager,
                 tls_manager, operator_repo_path: str,
                 deployment_namespace: str, external_db_namespace: str):
        self.args = args
        self.deployment_manager = deployment_manager
        self.tls_manager = tls_manager
        self.operator_repo_path = operator_repo_path
        self.deployment_namespace = deployment_namespace
        self.external_db_namespace = external_db_namespace

    def deploy_external_mariadb(self):
        """Deploy MariaDB externally to external database namespace."""
        if not self.args.deploy_external_db:
            return

        print('🗄️  Deploying external MariaDB...')

        if not self.operator_repo_path:
            raise ValueError(
                'Operator repository not cloned for external MariaDB deployment'
            )

        print(
            f'🏷️  Creating external database namespace: {self.external_db_namespace}'
        )
        self.deployment_manager.run_command(
            ['kubectl', 'create', 'namespace', self.external_db_namespace],
            check=False)

        mariadb_resources_path = os.path.join(self.operator_repo_path,
                                              '.github/resources/mariadb')

        if not os.path.exists(mariadb_resources_path):
            raise ValueError(
                f'MariaDB resources directory not found: {mariadb_resources_path}'
            )

        success = self.deployment_manager.deploy_and_wait(
            manifest_path=mariadb_resources_path,
            namespace=self.external_db_namespace,
            kustomize=True,
            resource_type=ResourceType.DEPLOYMENT,
            resource_name='mariadb',
            wait_timeout='300s',
            description='External MariaDB')

        if not success:
            self.deployment_manager.debug_deployment_failure(
                namespace=self.external_db_namespace,
                deployment_name='mariadb',
                resource_type=ResourceType.DEPLOYMENT)
            raise RuntimeError('External MariaDB deployment failed')

        print('✅ External MariaDB deployed successfully')

        # Configure MariaDB for TLS if enabled
        if self.args.pod_to_pod_tls_enabled and self.tls_manager:
            self.tls_manager.configure_mariadb_for_tls()

    def apply_mariadb_minio_secrets_configmaps(self):
        """Apply MariaDB and MinIO Secrets and ConfigMaps to external namespace."""
        if not self.args.deploy_external_db:
            return

        print(
            '🔐 Applying MariaDB and MinIO Secrets and ConfigMaps to external namespace...'
        )

        if not self.operator_repo_path:
            raise ValueError(
                'Operator repository not cloned for external pre-requisites')

        external_prereqs_path = os.path.join(
            self.operator_repo_path, '.github/resources/external-pre-reqs')

        if not os.path.exists(external_prereqs_path):
            print(
                f'⚠️  External pre-requisites directory not found: {external_prereqs_path}'
            )
            return

        self.deployment_manager.apply_resource(
            manifest_path=external_prereqs_path,
            namespace=self.external_db_namespace,
            kustomize=True,
            description='MariaDB and MinIO Secrets and ConfigMaps')

        print('✅ MariaDB and MinIO Secrets and ConfigMaps applied successfully')

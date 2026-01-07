#!/usr/bin/env python3
"""Data Science Pipelines Deployment Orchestrator.

Provides flexible deployment options for:
1. Data Science Pipelines Operator (DSPO) — operator mode
2. Direct manifest deployment — direct mode
3. Optional: PyPI Server, external Argo, external MariaDB

Note: Assumes Docker images are already built and available in the registry
via the build action.
"""

import argparse
import os
import shutil
import tempfile

from database_deployer import DatabaseDeployer
from deployment_manager import K8sDeploymentManager
from dspa_deployer import DSPADeployer
from infra_deployer import InfraDeployer
from operator_deployer import OperatorDeployer
from storage_deployer import StorageDeployer
from tls_manager import TLSCertificateManager


class DSPDeployer:

    def __init__(self, args):
        self.args = args
        self.repo_owner = None
        self.target_branch = None
        self.temp_dir = None
        self.deployment_namespace = None
        self.dspa_name = None
        self.operator_namespace = None
        self.external_db_namespace = 'test-mariadb'
        self.skip_operator_deployment = False
        self.is_operator_deployment = None
        self.tls_manager = None
        self.deployment_manager = K8sDeploymentManager()

        self._convert_args_to_booleans()

        # Sub-deployers (initialized after setup_environment)
        self.operator = None
        self.storage = None
        self.database = None
        self.dspa = None
        self.infra = None

    @staticmethod
    def str_to_bool(value) -> bool:
        """Convert string values to boolean."""
        if isinstance(value, bool):
            return value
        if isinstance(value, str):
            return value.lower() == 'true'
        return bool(value)

    def _convert_args_to_booleans(self):
        """Convert string arguments to boolean values."""
        boolean_args = [
            'deploy_pypi_server', 'deploy_external_argo', 'proxy',
            'cache_enabled', 'multi_user', 'artifact_proxy', 'forward_port',
            'pod_to_pod_tls_enabled', 'deploy_external_db',
            'skip_operator_deployment'
        ]
        for arg_name in boolean_args:
            if hasattr(self.args, arg_name):
                current_value = getattr(self.args, arg_name)
                setattr(self.args, arg_name, self.str_to_bool(current_value))

    def setup_environment(self):
        """Extract repository information, create namespaces, init sub-
        deployers."""
        print('🔧 Setting up deployment environment...')

        if self.args.skip_operator_deployment:
            self.skip_operator_deployment = self.args.skip_operator_deployment
            print('Skipping deployment via operator and using direct deployment')

        if self.args.github_repository:
            self.repo_owner = self.args.github_repository.split('/')[0]
            print(f'📂 Detected repository owner: {self.repo_owner}')
        else:
            raise ValueError('GitHub repository not provided')

        self.target_branch = self.args.github_base_ref or 'main'
        print(f'🌳 Target branch: {self.target_branch}')

        if self.repo_owner == 'red-hat-data-services':
            self.operator_namespace = 'rhods'
        else:
            self.operator_namespace = 'opendatahub'

        self.temp_dir = tempfile.mkdtemp()
        print(f'📁 Working directory: {self.temp_dir}')

        self.deployment_namespace = self.args.namespace
        print(f'🏷️  Deployment namespace: {self.deployment_namespace}')

        self.dspa_name = self.args.dspa_name or 'dspa-test'
        print(f'🏷️  Deployment Name: {self.dspa_name}')

        if self.args.deploy_external_db:
            print(f'🏷️  External DB Namespace: {self.external_db_namespace}')

        print(f'🏷️  Creating deployment namespace: {self.deployment_namespace}')
        self.deployment_manager.run_command(
            ['kubectl', 'create', 'namespace', self.deployment_namespace],
            check=False)

        if self.args.pod_to_pod_tls_enabled:
            self.tls_manager = TLSCertificateManager(self)
            print('🔐 TLS certificate manager initialized for podToPodTLS')

        # Initialize sub-deployers
        self.operator = OperatorDeployer(
            self.args, self.deployment_manager, self.repo_owner,
            self.target_branch, self.temp_dir, self.operator_namespace)

        self.storage = StorageDeployer(
            self.args, self.deployment_manager, self.deployment_namespace,
            self.temp_dir)

        self.infra = InfraDeployer(
            self.args, self.deployment_manager, None,
            self.deployment_namespace, self.dspa_name,
            self.operator_namespace, self.temp_dir)

    def _init_deployers_after_clone(self):
        """Initialize deployers that depend on operator_repo_path."""
        operator_repo_path = self.operator.operator_repo_path

        self.infra.operator_repo_path = operator_repo_path

        self.database = DatabaseDeployer(
            self.args, self.deployment_manager, self.tls_manager,
            operator_repo_path, self.deployment_namespace,
            self.external_db_namespace)

        self.dspa = DSPADeployer(
            self.args, self.deployment_manager, self.deployment_namespace,
            self.dspa_name, self.external_db_namespace,
            self.operator_namespace, self.temp_dir)

    def _should_use_operator_deployment(self) -> bool:
        """Determine whether to use DSPO (operator) or direct deployment."""
        if self.skip_operator_deployment:
            print(
                '⚠️  User selected to skip DSPO deployment, using direct deployment'
            )
            return False
        if self.args.multi_user:
            print(
                "⚠️  Multi-user mode detected: DSPO doesn't support multi-user, using direct deployment"
            )
            return False
        if self.args.proxy:
            print(
                "⚠️  Proxy mode detected: DSPO doesn't support proxy, using direct deployment"
            )
            return False
        return True

    def deploy_dsp_direct(self):
        """Deploy Data Science Pipelines using direct manifests."""
        print('🚀 Deploying Data Science Pipelines using direct manifests...')

        deploy_args = []
        if self.args.proxy:
            deploy_args.append('--proxy')
        if not self.args.cache_enabled:
            deploy_args.append('--cache-disabled')
        if self.args.pipeline_store == 'kubernetes':
            deploy_args.append('--deploy-k8s-native')
        if self.args.multi_user:
            deploy_args.append('--multi-user')
        if self.args.artifact_proxy:
            deploy_args.append('--artifact-proxy')
        if self.args.storage_backend and self.args.storage_backend != 'seaweedfs':
            deploy_args.extend(['--storage', self.args.storage_backend])
        if self.args.argo_version:
            deploy_args.extend(['--argo-version', self.args.argo_version])
        if self.args.pod_to_pod_tls_enabled:
            deploy_args.append('--tls-enabled')

        deploy_env = os.environ.copy()
        deploy_env['REGISTRY'] = self.args.image_registry

        print(f'🔧 Setting REGISTRY={self.args.image_registry}')
        print(f'🏷️  Using image tag: {self.args.image_tag}')

        deploy_script = './.github/resources/scripts/deploy-kfp.sh'
        cmd = ['bash', deploy_script] + deploy_args
        self.deployment_manager.run_command(cmd, timeout=1800, env=deploy_env)

        print('✅ Data Science Pipelines deployed directly successfully')

    def deploy(self):
        """Main deployment orchestration."""
        try:
            self.setup_environment()

            use_operator = self._should_use_operator_deployment()
            self.is_operator_deployment = use_operator

            if use_operator:
                print('🔧 Using DSPO (operator) deployment mode')

                self.operator.clone_operator_repo()
                self._init_deployers_after_clone()

                self.operator.create_operator_namespace()
                self.operator.install_crds()

                self.infra.deploy_cert_manager()

                if self.args.pod_to_pod_tls_enabled and self.tls_manager:
                    # Create external DB namespace before TLS setup so
                    # certificates and CA issuers can be written there.
                    if self.args.deploy_external_db:
                        print(
                            f'🏷️  Pre-creating external DB namespace for TLS: {self.external_db_namespace}'
                        )
                        self.deployment_manager.run_command(
                            ['kubectl', 'create', 'namespace',
                             self.external_db_namespace],
                            check=False)
                    self.tls_manager.setup_tls_for_kind()

                self.operator.deploy_operator()

                self.infra.deploy_argo_lite()
                self.infra.deploy_external_argo()

                self.infra.apply_webhooks()
                self.infra.deploy_pypi_server()

                self.database.apply_mariadb_minio_secrets_configmaps()
                self.database.deploy_external_mariadb()

                self.storage.deploy_seaweedfs()
                self.storage.apply_seaweedfs_networking_fixes()

                self.dspa.deploy_dsp_via_operator()

                self.infra.setup_service_account_rbac(is_operator=True)

            else:
                print(
                    '🔧 Using direct manifest deployment mode'
                )

                if self.args.deploy_pypi_server:
                    self.operator.clone_operator_repo()
                    self._init_deployers_after_clone()
                    self.infra.deploy_pypi_server()

                self.deploy_dsp_direct()

            self.infra.forward_port(is_operator=use_operator)

            print('🎉 Deployment completed successfully!')

        except Exception as e:
            print(f'❌ Deployment failed: {str(e)}')
            raise
        finally:
            if self.temp_dir and os.path.exists(self.temp_dir):
                shutil.rmtree(self.temp_dir)

    def output_deployment_metadata(self):
        """Output deployment metadata for GitHub Actions."""
        use_operator = self._should_use_operator_deployment()

        # Ensure dspa deployer is available
        if not self.dspa:
            self.dspa = DSPADeployer(
                self.args, self.deployment_manager,
                self.deployment_namespace or self.args.namespace,
                self.dspa_name or self.args.dspa_name or 'dspa-test',
                self.external_db_namespace,
                self.operator_namespace or 'opendatahub',
                self.temp_dir or '')

        metadata = self.dspa.output_deployment_metadata(is_operator=use_operator)

        for key, value in metadata.items():
            output_to_github_actions(key, value)


def output_to_github_actions(key: str, value: str):
    """Output key-value pair to GitHub Actions outputs."""
    github_output = os.environ.get('GITHUB_OUTPUT')
    if github_output:
        with open(github_output, 'a') as f:
            f.write(f'{key}={value}\n')
    else:
        print(f'::set-output name={key}::{value}')


def main():
    parser = argparse.ArgumentParser(
        description='Deploy Data Science Pipelines')

    # GitHub context
    parser.add_argument(
        '--github-repository', required=True,
        help='GitHub repository (owner/repo)')
    parser.add_argument(
        '--github-base-ref', help='GitHub base ref (target branch)')

    # Image configuration
    parser.add_argument('--image-tag', required=True, help='Image tag')
    parser.add_argument('--image-registry', required=True, help='Image registry')
    parser.add_argument(
        '--image-path-prefix', required=False, default='',
        help='Image path prefix to add')

    # Deployment options
    parser.add_argument(
        '--deploy-pypi-server', default='false',
        help='Deploy PyPI server and upload packages')
    parser.add_argument(
        '--deploy-external-argo', default='false',
        help='Deploy Argo Workflows externally in separate namespace')
    parser.add_argument(
        '--skip-operator-deployment', default='false',
        help='Skip deployment via operator')
    parser.add_argument(
        '--argo-namespace', default='argo',
        help='Namespace for external Argo Workflows deployment')

    # KFP options
    parser.add_argument(
        '--pipeline-store', default='database',
        choices=['database', 'kubernetes'], help='Pipeline store type')
    parser.add_argument('--proxy', default='false', help='Enable proxy')
    parser.add_argument('--cache-enabled', default='true', help='Enable cache')
    parser.add_argument(
        '--multi-user', default='false', help='Multi-user mode')
    parser.add_argument(
        '--artifact-proxy', default='false', help='Enable artifact proxy')
    parser.add_argument(
        '--storage-backend', default='seaweedfs',
        choices=['seaweedfs', 'minio'], help='Storage backend')
    parser.add_argument('--argo-version', help='Argo version')
    parser.add_argument(
        '--forward-port', default='true', help='Forward API server port')
    parser.add_argument(
        '--pod-to-pod-tls-enabled', default='false',
        help='Enable pod-to-pod TLS')
    parser.add_argument(
        '--namespace', default='kubeflow',
        help='Namespace for DSPA deployment')
    parser.add_argument(
        '--deploy-external-db', default=False,
        help='Deploy DB externally instead of via DSPO')
    parser.add_argument(
        '--dspa-name', default='dspa-test', help='Name of DSPA resource')
    parser.add_argument(
        '--operator-image-tag', default='',
        help='Image tag for DSPO operator (overrides github-base-ref)')

    args = parser.parse_args()

    deployer = DSPDeployer(args)
    deployer.deploy()
    deployer.output_deployment_metadata()


if __name__ == '__main__':
    main()

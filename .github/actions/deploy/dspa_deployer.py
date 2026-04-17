"""DSPA (DataSciencePipelinesApplication) configuration and deployment.

Handles generating the DSPA YAML manifest and deploying it via the DSPO
operator, including the two-stage wait pattern.
"""

import os
from typing import Any, Dict

from deployment_manager import K8sDeploymentManager
from deployment_manager import ResourceType
from deployment_manager import WaitCondition
import yaml


class DSPADeployer:

    def __init__(self, args, deployment_manager: K8sDeploymentManager,
                 deployment_namespace: str, dspa_name: str,
                 external_db_namespace: str, operator_namespace: str,
                 temp_dir: str):
        self.args = args
        self.deployment_manager = deployment_manager
        self.deployment_namespace = deployment_namespace
        self.dspa_name = dspa_name
        self.external_db_namespace = external_db_namespace
        self.operator_namespace = operator_namespace
        self.operator_deployment = 'data-science-pipelines-operator-controller-manager'
        self.temp_dir = temp_dir

    def generate_dspa_yaml(self) -> Dict[str, Any]:
        """Generate DataSciencePipelinesApplication YAML."""
        print('📄 Generating DSPA configuration...')

        api_server_image_name = 'apiserver'
        if 'opendatahub' in self.args.image_path_prefix:
            api_server_image_name = 'api-server'

        api_server_config = {
            'image':
                f'{self.args.image_registry}/{self.args.image_path_prefix}{api_server_image_name}:{self.args.image_tag}',
            'argoDriverImage':
                f'{self.args.image_registry}/{self.args.image_path_prefix}driver:{self.args.image_tag}',
            'argoLauncherImage':
                f'{self.args.image_registry}/{self.args.image_path_prefix}launcher:{self.args.image_tag}',
            'cacheEnabled':
                self.args.cache_enabled,
            'enableOauth':
                False
        }

        if self.args.pod_to_pod_tls_enabled:
            api_server_config.update({
                'cABundle': {
                    'configMapName': 'openshift-service-ca.crt',
                    'configMapKey': 'service-ca.crt'
                }
            })
            print('🔧 Added CA bundle configuration for TLS communication')

        if self.args.pipeline_store == 'kubernetes':
            api_server_config['pipelineStore'] = 'kubernetes'
            print('🔧 Enabling Kubernetes native pipeline storage')

        dspa_config = {
            'apiVersion': 'datasciencepipelinesapplications.opendatahub.io/v1',
            'kind': 'DataSciencePipelinesApplication',
            'metadata': {
                'name': self.dspa_name,
                'namespace': self.deployment_namespace
            },
            'spec': {
                'dspVersion': 'v2',
                'apiServer': api_server_config,
                'persistenceAgent': {
                    'image':
                        f'{self.args.image_registry}/{self.args.image_path_prefix}persistenceagent:{self.args.image_tag}'
                },
                'scheduledWorkflow': {
                    'image':
                        f'{self.args.image_registry}/{self.args.image_path_prefix}scheduledworkflow:{self.args.image_tag}'
                },
                'mlmd': {
                    'deploy': True,
                    'envoy': {
                        'image': 'quay.io/maistra/proxyv2-ubi8:2.5.0'
                    }
                },
                'podToPodTLS': self.args.pod_to_pod_tls_enabled
            }
        }

        # Storage configuration
        if self.args.storage_backend == 'minio':
            dspa_config['spec']['objectStorage'] = {
                'minio': {
                    'deploy':
                        True,
                    'image':
                        'quay.io/opendatahub/minio:RELEASE.2019-08-14T20-37-41Z-license-compliance'
                }
            }
        else:  # seaweedfs (default)
            dspa_config['spec']['objectStorage'] = {
                'externalStorage': {
                    'host':
                        f'seaweedfs.{self.deployment_namespace}.svc.cluster.local',
                    'port':
                        '9000',
                    'bucket':
                        'mlpipeline',
                    'region':
                        'us-east-1',
                    'scheme':
                        'http',
                    's3CredentialsSecret': {
                        'accessKey': 'accesskey',
                        'secretKey': 'secretkey',
                        'secretName': 'mlpipeline-minio-artifact'
                    }
                }
            }

        # Database configuration
        if self.args.deploy_external_db:
            dspa_config['spec']['database'] = {
                'customExtraParams': '{"tls":"true"}',
                'externalDB': {
                    'host':
                        f'mariadb.{self.external_db_namespace}.svc.cluster.local',
                    'port':
                        '3306',
                    'username':
                        'mlpipeline',
                    'pipelineDBName':
                        'mlpipeline',
                    'passwordSecret': {
                        'name': 'ds-pipeline-db-test',
                        'key': 'password'
                    }
                }
            }
        else:
            dspa_config['spec']['database'] = {
                'mariaDB': {
                    'deploy': True,
                    'image': 'quay.io/sclorg/mariadb-105-c9s:latest',
                }
            }
        return dspa_config

    def deploy_dsp_via_operator(self):
        """Deploy Data Science Pipelines via operator using DSPA CR."""
        print('🚀 Deploying Data Science Pipelines via operator...')

        dspa_config = self.generate_dspa_yaml()

        dspa_file = os.path.join(self.temp_dir, 'dspa.yaml')
        with open(dspa_file, 'w') as f:
            yaml.dump(dspa_config, f, default_flow_style=False)

        print(f'📝 DSPA configuration written to: {dspa_file}')
        print(
            f'📄 DSPA Content:\n{yaml.dump(dspa_config, default_flow_style=False)}'
        )

        print('Creating DSPA deployment')
        self.deployment_manager.apply_resource(
            manifest_path=dspa_file,
            namespace=self.deployment_namespace,
            description='DSPA (Data Science Pipelines Application)')

        deployment_name = f'ds-pipeline-{self.dspa_name}'

        # Two-stage wait: 1) deployment exists, 2) deployment available
        print('⏳ Step 1: Waiting for DSPA operator to create deployment...')
        exists_success = self.deployment_manager.wait_for_resource_to_exist(
            resource_type=ResourceType.DEPLOYMENT,
            resource_name=deployment_name,
            namespace=self.deployment_namespace,
            timeout_seconds=120,
            description=f'DSPA deployment {deployment_name}')

        if not exists_success:
            print('❌ DSPA operator did not create deployment within timeout')
            print(f'🔍 Investigating deployment: {deployment_name}')
            self.deployment_manager.debug_deployment_failure(
                namespace=self.deployment_namespace,
                deployment_name=deployment_name,
                resource_type=ResourceType.DEPLOYMENT,
                include_events=True,
                include_logs=True,
                log_tail_lines=100)
            print(f'🔍 Investigating deployment: {self.operator_deployment}')
            self.deployment_manager.debug_deployment_failure(
                namespace=self.operator_namespace,
                deployment_name=self.operator_deployment,
                resource_type=ResourceType.DEPLOYMENT,
                include_events=False,
                include_logs=True,
                log_tail_lines=100)
            raise RuntimeError(
                'DSPA operator did not create deployment within timeout')

        print(
            f'⏳ Step 2: Waiting for deployment {deployment_name} to be available...'
        )
        wait_success = self.deployment_manager.wait_for_resource(
            resource_type=ResourceType.DEPLOYMENT,
            resource_name=deployment_name,
            namespace=self.deployment_namespace,
            condition=WaitCondition.AVAILABLE,
            timeout='600s',
            description=f'DSPA deployment {deployment_name}')

        if wait_success:
            print('✅ Data Science Pipelines deployed via operator successfully')
        else:
            print('❌ DSPA deployment did not become ready within timeout')
            print(f'🔍 Investigating DSPA deployment: {deployment_name}')
            self.deployment_manager.debug_deployment_failure(
                namespace=self.deployment_namespace,
                deployment_name=deployment_name,
                resource_type=ResourceType.DEPLOYMENT)
            print(
                f'🔍 Investigating operator deployment: {self.operator_deployment}'
            )
            self.deployment_manager.debug_deployment_failure(
                namespace=self.operator_namespace,
                deployment_name=self.operator_deployment,
                resource_type=ResourceType.DEPLOYMENT,
                include_events=False,
                include_logs=True,
                log_tail_lines=100)
            raise RuntimeError('DSPA did not become ready within timeout')

    def output_deployment_metadata(self, is_operator: bool):
        """Output deployment metadata for GitHub Actions."""
        deployment_mode = 'operator' if is_operator else 'direct'

        if is_operator:
            deployment_name = f'ds-pipeline-{self.dspa_name}'
            service_account_name = f'ds-pipeline-{self.dspa_name}'
            database_name = f'ds-pipeline-db-{self.dspa_name}'
            mlmd_service_name = f'ds-pipeline-metadata-grpc-{self.dspa_name}'
            frontend_service_name = f'ds-pipeline-ui-{self.dspa_name}'
        else:
            deployment_name = 'ml-pipeline'
            service_account_name = 'pipeline-runner'
            database_name = 'mysql'
            mlmd_service_name = 'metadata-grpc-service'
            frontend_service_name = 'ml-pipeline-ui'

        print('📤 Outputting deployment metadata:')
        print(f'   DEPLOYMENT_MODE={deployment_mode}')
        print(f'   DEPLOYMENT_NAME={deployment_name}')
        print(f'   SERVICE_ACCOUNT_NAME={service_account_name}')
        print(f'   DATABASE_NAME={database_name}')
        print(f'   MLMD_SERVICE_NAME={mlmd_service_name}')
        print(f'   FRONTEND_SERVICE_NAME={frontend_service_name}')

        metadata = {
            'DEPLOYMENT_MODE': deployment_mode,
            'DEPLOYMENT_NAME': deployment_name,
            'SERVICE_ACCOUNT_NAME': service_account_name,
            'DATABASE_NAME': database_name,
            'MLMD_SERVICE_NAME': mlmd_service_name,
            'FRONTEND_SERVICE_NAME': frontend_service_name,
        }
        return metadata

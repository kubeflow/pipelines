"""Storage backend deployment (SeaweedFS / MinIO).

Handles deploying SeaweedFS from local manifests, waiting for the
init job, and updating network policies for DSPO operator access.
"""

import os

import yaml

from deployment_manager import K8sDeploymentManager
from deployment_manager import ResourceType
from deployment_manager import WaitCondition


class StorageDeployer:

    def __init__(self, args, deployment_manager: K8sDeploymentManager,
                 deployment_namespace: str, temp_dir: str):
        self.args = args
        self.deployment_manager = deployment_manager
        self.deployment_namespace = deployment_namespace
        self.temp_dir = temp_dir

    def deploy_seaweedfs(self):
        """Deploy SeaweedFS using local manifests."""
        if self.args.storage_backend != 'seaweedfs':
            return

        seaweedfs_path = './manifests/kustomize/third-party/seaweedfs/base/seaweedfs'

        if not os.path.exists(
                os.path.join(seaweedfs_path, 'kustomization.yaml')):
            raise ValueError(
                f'SeaweedFS kustomization.yaml not found at {seaweedfs_path}')

        # Clean up existing SeaweedFS resources to avoid immutable field
        # errors on spec.selector and stale PVC permission issues during
        # upgrades. The apply will recreate everything.
        for resource in ['deployment/seaweedfs', 'job/init-seaweedfs',
                         'pvc/seaweedfs-pvc']:
            self.deployment_manager.run_command([
                'kubectl', 'delete', resource,
                '-n', self.deployment_namespace, '--ignore-not-found'
            ], check=False)

        success = self.deployment_manager.deploy_and_wait(
            manifest_path=seaweedfs_path,
            namespace=self.deployment_namespace,
            kustomize=True,
            resource_type=ResourceType.DEPLOYMENT,
            resource_name='seaweedfs',
            wait_timeout='300s',
            description='SeaweedFS')

        if not success:
            self.deployment_manager.debug_deployment_failure(
                namespace=self.deployment_namespace,
                deployment_name='seaweedfs',
                resource_type=ResourceType.DEPLOYMENT)
            raise RuntimeError('SeaweedFS deployment failed')

        # Wait for init job to complete
        job_success = self.deployment_manager.wait_for_resource(
            resource_type=ResourceType.JOB,
            resource_name='init-seaweedfs',
            namespace=self.deployment_namespace,
            condition=WaitCondition.COMPLETE,
            timeout='300s',
            description='SeaweedFS init job')

        if not job_success:
            print(
                '⚠️  Init job did not complete within timeout, investigating...'
            )
            self.deployment_manager.debug_deployment_failure(
                namespace=self.deployment_namespace,
                deployment_name='init-seaweedfs',
                resource_type=ResourceType.JOB,
                selector='job-name=init-seaweedfs')
            print('⚠️  Continuing without waiting for init job completion...')

        print('✅ SeaweedFS deployed successfully from local manifests')

    def update_seaweedfs_network_policy(self):
        """Update SeaweedFS network policy to allow DSPO operator access.

        Adds ingress rules for the opendatahub namespace and pod-to-pod
        access on ports 8333 (S3), 8888 (filer), and 9333 (master).
        """
        print('🔒 Updating SeaweedFS network policy for DSPO operator access...')

        updated_policy = {
            'apiVersion': 'networking.k8s.io/v1',
            'kind': 'NetworkPolicy',
            'metadata': {
                'name': 'seaweedfs',
                'namespace': self.deployment_namespace
            },
            'spec': {
                'ingress': [
                    {
                        'from': [{
                            'namespaceSelector': {
                                'matchExpressions': [{
                                    'key': 'app.kubernetes.io/part-of',
                                    'operator': 'In',
                                    'values': ['kubeflow-profile']
                                }]
                            }
                        }],
                        'ports': [
                            {'port': 8333}
                        ]
                    },
                    {
                        'from': [{
                            'namespaceSelector': {
                                'matchExpressions': [{
                                    'key': 'kubernetes.io/metadata.name',
                                    'operator': 'In',
                                    'values': ['opendatahub']
                                }]
                            }
                        }],
                        'ports': [
                            {'port': 8333}
                        ]
                    },
                    {
                        'from': [{'podSelector': {}}],
                        'ports': [
                            {'port': 8333},
                            {'port': 8888},
                            {'port': 9333}
                        ]
                    }
                ],
                'podSelector': {
                    'matchExpressions': [{
                        'key': 'app',
                        'operator': 'In',
                        'values': ['seaweedfs']
                    }]
                },
                'policyTypes': ['Ingress']
            }
        }

        policy_file = os.path.join(self.temp_dir,
                                   'seaweedfs-networkpolicy-updated.yaml')
        with open(policy_file, 'w') as f:
            yaml.dump(updated_policy, f, default_flow_style=False)

        self.deployment_manager.apply_resource(
            manifest_path=policy_file,
            namespace=self.deployment_namespace,
            description='Updated SeaweedFS network policy with opendatahub access')

        print('✅ SeaweedFS network policy updated successfully')

    def apply_seaweedfs_networking_fixes(self):
        """Apply SeaweedFS networking fixes for DSPO operator compatibility."""
        if self.args.storage_backend != 'seaweedfs':
            return

        print('🔧 Applying SeaweedFS networking fixes for DSPO compatibility...')
        self.update_seaweedfs_network_policy()
        print('✅ SeaweedFS networking fixes applied successfully')

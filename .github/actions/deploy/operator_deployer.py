"""DSPO operator lifecycle management.

Handles cloning the operator repository, installing CRDs, deploying the
operator, and configuring it for external Argo.
"""

import os
import subprocess

from deployment_manager import K8sDeploymentManager
from deployment_manager import ResourceType
from deployment_manager import WaitCondition


class OperatorDeployer:

    OPERATOR_DEPLOYMENT_NAME = 'data-science-pipelines-operator-controller-manager'

    def __init__(self, args, deployment_manager: K8sDeploymentManager,
                 repo_owner: str, target_branch: str, temp_dir: str,
                 operator_namespace: str):
        self.args = args
        self.deployment_manager = deployment_manager
        self.repo_owner = repo_owner
        self.target_branch = target_branch
        self.temp_dir = temp_dir
        self.operator_namespace = operator_namespace
        self.operator_repo_path = None

    def clone_operator_repo(self) -> str:
        """Clone data-science-pipelines-operator repository."""
        operator_repo_url = f'https://github.com/{self.repo_owner}/data-science-pipelines-operator'
        operator_path = os.path.join(self.temp_dir,
                                     'data-science-pipelines-operator')

        print(f'📥 Cloning operator repository: {operator_repo_url}')
        self.deployment_manager.run_command(
            ['git', 'clone', operator_repo_url, operator_path])

        # Map target branch to operator branch (master -> main for operator repo)
        operator_branch = 'main' if self.target_branch == 'master' else self.target_branch

        print(f'🔄 Checking out branch: {operator_branch}')
        try:
            self.deployment_manager.run_command(
                ['git', 'checkout', operator_branch], cwd=operator_path)
        except subprocess.CalledProcessError:
            print(
                f'⚠️  Branch {operator_branch} not found, using default branch')

        # Fix Makefile permissions if it exists
        makefile_path = os.path.join(operator_path, 'Makefile')
        if os.path.exists(makefile_path):
            print('🔧 Fixing Makefile permissions...')
            self.deployment_manager.run_command(['chmod', '644', makefile_path],
                                                cwd=operator_path,
                                                check=False)

        self.operator_repo_path = operator_path
        return operator_path

    def create_operator_namespace(self):
        """Create operator namespace if it doesn't exist."""
        print(f'🏷️  Creating operator namespace: {self.operator_namespace}')
        self.deployment_manager.run_command(
            ['kubectl', 'create', 'namespace', self.operator_namespace],
            check=False)

    def install_crds(self):
        """Install operator CRDs and OpenShift route CRD."""
        print('🔧 Installing operator CRDs...')

        # Apply additional CRDs from resources directory
        print('🔧 Installing additional CRDs from resources directory...')
        additional_crds_path = os.path.join(self.operator_repo_path, '.github',
                                            'resources', 'crds')
        if os.path.exists(additional_crds_path):
            self.deployment_manager.apply_resource(
                manifest_path=additional_crds_path,
                description='Additional CRDs from resources directory')

        # Apply external route CRD (OpenShift specific)
        print('🔧 Installing OpenShift route CRD...')
        route_crd_path = os.path.join(self.operator_repo_path, 'config', 'crd',
                                      'external',
                                      'route.openshift.io_routes.yaml')
        if os.path.exists(route_crd_path):
            try:
                self.deployment_manager.apply_resource(
                    manifest_path=route_crd_path,
                    description='OpenShift route CRD')
            except Exception as e:
                print(
                    f'⚠️  OpenShift route CRD apply failed (expected on kind): {e}'
                )

    def _patch_params_for_kind(self):
        """Replace Red Hat registry images in params.env with publicly
        accessible alternatives.

        The operator's default params.env references registry.redhat.io
        images that require authentication. On Kind clusters there are
        no RH registry credentials, so these images must be swapped for
        public equivalents.
        """
        params_env_path = os.path.join(self.operator_repo_path, 'config',
                                       'base', 'params.env')
        if not os.path.exists(params_env_path):
            print(f'⚠️  params.env not found at {params_env_path}, skipping image patching')
            return

        print('🔧 Patching params.env to use publicly accessible images for Kind...')

        replacements = {
            'registry.redhat.io/openshift4/ose-kube-rbac-proxy-rhel9:latest':
                'registry.k8s.io/kubebuilder/kube-rbac-proxy:v0.16.0',
            'registry.redhat.io/rhel9/mariadb-105:latest':
                'quay.io/sclorg/mariadb-105-c9s:latest',
            'registry.redhat.io/openshift-service-mesh/proxyv2-rhel9:2.6':
                'quay.io/maistra/proxyv2-ubi8:2.5.0',
        }

        with open(params_env_path, 'r') as f:
            content = f.read()

        for old_image, new_image in replacements.items():
            if old_image in content:
                content = content.replace(old_image, new_image)
                print(f'  🔄 {old_image} -> {new_image}')

        with open(params_env_path, 'w') as f:
            f.write(content)

        print('✅ params.env patched for Kind cluster compatibility')

    def deploy_operator(self):
        """Deploy Data Science Pipelines Operator."""
        print('🔧 Deploying Data Science Pipelines Operator...')

        if not self.operator_repo_path:
            raise ValueError('Operator repository not cloned')

        # Patch params.env to replace Red Hat registry images before deploying
        self._patch_params_for_kind()

        operator_image_tag = getattr(self.args, 'operator_image_tag', '') or ''
        if operator_image_tag:
            dspo_tag = operator_image_tag
        elif self.target_branch == 'stable':
            dspo_tag = 'odh-stable'
        elif self.target_branch == 'master':
            dspo_tag = 'main'
        else:
            dspo_tag = self.target_branch
        repo = 'opendatahub' if self.repo_owner == 'opendatahub-io' else 'rhoai'
        operator_image = f'quay.io/{repo}/data-science-pipelines-operator:{dspo_tag}'

        print(f'🏷️  Using operator image: {operator_image}')

        deploy_env = os.environ.copy()
        deploy_env.update({
            'IMAGES_DSPO': operator_image,
            'IMG': operator_image,
            'OPERATOR_NS': self.operator_namespace,
        })

        print(f'🔧 Setting IMAGES_DSPO={operator_image}')
        self.deployment_manager.run_command(
            ['make', 'deploy-kind', f'IMG={operator_image}'],
            cwd=self.operator_repo_path,
            env=deploy_env)

        # Verify ConfigMap creation
        print('🔍 Checking created ConfigMaps...')
        self.deployment_manager.run_command(
            ['kubectl', 'get', 'configmaps', '-n', self.operator_namespace],
            check=False)

        print('🔧 Verifying DSPO ConfigMap creation...')
        configmap_names = [
            'data-science-pipelines-operator-dspo-config',
            'dspo-config'
        ]

        configmap_found = False
        for cm_name in configmap_names:
            result = self.deployment_manager.run_command([
                'kubectl', 'get', 'configmap', cm_name, '-n',
                self.operator_namespace
            ], check=False)

            if result.returncode == 0:
                print(f'✅ Found required ConfigMap: {cm_name}')
                configmap_found = True
                break

        if not configmap_found:
            print('⚠️  Required ConfigMaps not found. Available ConfigMaps:')
            self.deployment_manager.run_command([
                'kubectl', 'get', 'configmaps', '-n', self.operator_namespace,
                '--no-headers', '-o', 'custom-columns=NAME:.metadata.name'
            ], check=False)

        # Wait for operator readiness
        print(
            f'⏳ Waiting for operator to be ready in namespace: {self.operator_namespace}...'
        )
        wait_success = self.deployment_manager.wait_for_resource(
            resource_type=ResourceType.DEPLOYMENT,
            namespace=self.operator_namespace,
            condition=WaitCondition.AVAILABLE,
            timeout='300s',
            all_resources=True,
            description='Data Science Pipelines Operator deployment')

        if not wait_success:
            print(
                '⚠️  Operator did not become ready within timeout, investigating...'
            )
            self.deployment_manager.debug_deployment_failure(
                namespace=self.operator_namespace,
                deployment_name=self.OPERATOR_DEPLOYMENT_NAME,
                resource_type=ResourceType.DEPLOYMENT,
                selector='app.kubernetes.io/name=data-science-pipelines-operator'
            )
            raise RuntimeError('Operator did not become ready within timeout')

        # Configure for external Argo if requested
        if self.args.deploy_external_argo:
            self._configure_operator_for_external_argo()

        print('✅ Data Science Pipelines Operator deployed successfully')

    def _configure_operator_for_external_argo(self):
        """Configure the deployed operator to use external Argo Workflows."""
        print('🔧 Configuring operator to use external Argo Workflows...')

        patch_json = '[{"op": "add", "path": "/spec/template/spec/containers/0/env/-", "value": {"name": "DSPO_ARGOWORKFLOWSCONTROLLERS", "value": "{\\"managementState\\": \\"Removed\\"}"}}]'

        self.deployment_manager.run_command([
            'kubectl', 'patch', 'deployment', self.OPERATOR_DEPLOYMENT_NAME,
            '-n', self.operator_namespace, '--type=json', '-p', patch_json
        ])

        print(
            '⏳ Waiting for operator to restart with external Argo configuration...'
        )
        self.deployment_manager.run_command([
            'kubectl', 'rollout', 'status',
            f'deployment/{self.OPERATOR_DEPLOYMENT_NAME}', '-n',
            self.operator_namespace, '--timeout=300s'
        ])

        print('✅ Operator configured for external Argo successfully')

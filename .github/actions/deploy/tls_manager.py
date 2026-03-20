#!/usr/bin/env python3
"""TLS Certificate Management for Data Science Pipelines on Kind Clusters.

This module handles the creation and management of TLS certificates
required for podToPodTLS communication in DSPA deployments on kind
clusters.

On OpenShift, certificates are automatically created by the service-ca
controller. For kind clusters, we need to manually create them using
cert-manager.
"""

import base64
import json
import os
import subprocess
import tempfile
from typing import Any, Dict, List

import yaml


class TLSCertificateManager:
    """Manages TLS certificates for DSPA deployments on kind clusters."""

    def __init__(self, deployer):
        """Initialize TLS manager with reference to main deployer."""
        self.deployer = deployer
        self.dspa_name = deployer.dspa_name
        self.deployment_namespace = deployer.deployment_namespace
        self.operator_namespace = deployer.operator_namespace

    def setup_tls_for_kind(self):
        """Set up complete TLS infrastructure for kind cluster with podToPodTLS
        enabled.

        This creates all certificates and ConfigMaps that would normally
        be automatically created by OpenShift's service-ca controller.
        """
        if not self.deployer.args.pod_to_pod_tls_enabled:
            return

        print(
            '🔐 Setting up TLS infrastructure for kind cluster with podToPodTLS enabled...'
        )

        try:
            # Step 1: Create self-signed ClusterIssuer for service certificates
            self._create_service_ca_issuer()

            # Step 2: Create the service CA certificate (root CA)
            self._create_service_ca_certificate()

            # Step 3: Create certificates for each DSPA component
            self._create_dspa_component_certificates()

            # Step 4: Create OpenShift service-ca ConfigMap equivalent for kind
            self._create_service_ca_configmap()

            # Step 5: Verify all certificates are ready
            self._verify_certificates()

            # Step 6: Pre-create MariaDB TLS ConfigMap for internal MariaDB (if not using external DB)
            if not self.deployer.args.deploy_external_db:
                print(
                    '🔧 Pre-creating MariaDB TLS ConfigMap for operator template...'
                )
                self._configure_internal_mariadb_tls()

            print(
                '✅ TLS infrastructure setup completed successfully for kind cluster'
            )

        except Exception as e:
            print(f'❌ Failed to set up TLS infrastructure: {e}')
            raise

    def _create_service_ca_issuer(self):
        """Create a self-signed ClusterIssuer to act as the service CA for kind
        clusters."""
        print('📜 Creating service-ca ClusterIssuer...')

        issuer_yaml = """
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: service-ca-issuer
spec:
  selfSigned: {}
"""

        self._apply_yaml_content(issuer_yaml, 'service-ca-issuer')
        print('✅ Service CA issuer created successfully')

    def _create_service_ca_certificate(self):
        """Create the root service CA certificate."""
        print('📜 Creating service CA root certificate...')

        ca_cert_yaml = """
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: service-ca-root
  namespace: cert-manager
spec:
  secretName: service-ca-root
  isCA: true
  commonName: service-ca
  subject:
    organizationalUnits:
      - "OpenShift Service CA"
  issuerRef:
    name: service-ca-issuer
    kind: ClusterIssuer
    group: cert-manager.io
  duration: 8760h # 1 year
  renewBefore: 720h # 1 month
"""

        self._apply_yaml_content(ca_cert_yaml, 'service-ca-root-certificate')

        # Wait for certificate to be ready
        print('⏳ Waiting for service CA certificate to be ready...')
        self.deployer.deployment_manager.run_command([
            'kubectl', 'wait', '--for=condition=Ready',
            'certificate/service-ca-root', '-n', 'cert-manager',
            '--timeout=300s'
        ])
        print('✅ Service CA root certificate created successfully')

    def _create_dspa_component_certificates(self):
        """Create certificates for all DSPA components that need TLS."""
        print('📜 Creating certificates for DSPA components...')

        # Create CA Issuer in all required namespaces
        self._create_ca_issuer_from_service_ca()

        # Define all required certificates based on DSPO expectations
        certificates = self._get_required_certificates()

        # Create each certificate in its appropriate namespace
        for cert in certificates:
            cert_namespace = cert.get('namespace', self.deployment_namespace)
            self._create_component_certificate(cert['name'], cert['dns_names'],
                                               cert_namespace)

        # Wait for all certificates to be ready
        print('⏳ Waiting for all component certificates to be ready...')
        failed_certs = []
        for cert in certificates:
            try:
                cert_namespace = cert.get('namespace',
                                          self.deployment_namespace)
                self.deployer.deployment_manager.run_command([
                    'kubectl', 'wait', '--for=condition=Ready',
                    f'certificate/{cert["name"]}', '-n', cert_namespace,
                    '--timeout=300s'
                ])
                print(
                    f'✅ Certificate {cert["name"]} is ready in namespace {cert_namespace}'
                )
            except subprocess.CalledProcessError as e:
                print(
                    f'⚠️  Certificate {cert["name"]} not ready within timeout: {e}'
                )
                failed_certs.append((cert['name'], e))

        if failed_certs:
            names = ', '.join(name for name, _ in failed_certs)
            raise RuntimeError(
                f'The following certificates failed to become ready: {names}'
            )

        print('✅ All component certificates processing completed')

    def _get_required_certificates(self) -> List[Dict[str, Any]]:
        """Get list of all required certificates for DSPA components.

        These certificate names and DNS names match exactly what the
        data-science-pipelines-operator expects when podToPodTLS is
        enabled.
        """
        certificates = []

        # MariaDB certificate - handle both internal and external deployment
        if self.deployer.args.deploy_external_db:
            # External MariaDB in external_db_namespace
            mariadb_dns_names = [
                'mariadb', f'mariadb.{self.deployer.external_db_namespace}',
                f'mariadb.{self.deployer.external_db_namespace}.svc.cluster.local'
            ]
        else:
            # Internal MariaDB deployed by DSPA - must match operator service naming convention
            mariadb_dns_names = [
                f'mariadb-{self.dspa_name}',
                f'mariadb-{self.dspa_name}.{self.deployment_namespace}',
                f'mariadb-{self.dspa_name}.{self.deployment_namespace}.svc.cluster.local'
            ]

        certificates.append({
            'name':
                f'ds-pipelines-mariadb-tls-{self.dspa_name}',
            'description':
                'MariaDB TLS certificate',
            'dns_names':
                mariadb_dns_names,
            'namespace':
                self.deployer.external_db_namespace
                if self.deployer.args.deploy_external_db else
                self.deployment_namespace
        })

        # Other DSPA component certificates (always in deployment namespace)
        certificates.extend([{
            'name': f'ds-pipelines-proxy-tls-{self.dspa_name}',
            'description': 'API Server proxy TLS certificate',
            'dns_names': [
                f'ds-pipeline-{self.dspa_name}',
                f'ds-pipeline-{self.dspa_name}.{self.deployment_namespace}',
                f'ds-pipeline-{self.dspa_name}.{self.deployment_namespace}.svc.cluster.local',
                'localhost'
            ],
            'namespace': self.deployment_namespace
        }, {
            'name': f'ds-pipeline-metadata-grpc-tls-certs-{self.dspa_name}',
            'description': 'MLMD gRPC server TLS certificate',
            'dns_names': [
                f'ds-pipeline-metadata-grpc-{self.dspa_name}',
                f'ds-pipeline-metadata-grpc-{self.dspa_name}.{self.deployment_namespace}',
                f'ds-pipeline-metadata-grpc-{self.dspa_name}.{self.deployment_namespace}.svc.cluster.local'
            ],
            'namespace': self.deployment_namespace
        }, {
            'name': f'ds-pipelines-envoy-proxy-tls-{self.dspa_name}',
            'description': 'Envoy proxy TLS certificate',
            'dns_names': [
                f'ds-pipeline-metadata-envoy-{self.dspa_name}',
                f'ds-pipeline-metadata-envoy-{self.dspa_name}.{self.deployment_namespace}',
                f'ds-pipeline-metadata-envoy-{self.dspa_name}.{self.deployment_namespace}.svc.cluster.local'
            ],
            'namespace': self.deployment_namespace
        }])

        return certificates

    def _create_ca_issuer_from_service_ca(self):
        """Create a CA issuer that uses the service CA root certificate."""
        print('📜 Creating CA issuer from service CA...')

        # Get list of namespaces where we need CA issuers
        namespaces = [self.deployment_namespace]
        if self.deployer.args.deploy_external_db and self.deployer.external_db_namespace not in namespaces:
            namespaces.append(self.deployer.external_db_namespace)

        # Copy the service CA secret to all required namespaces and create issuers
        for namespace in namespaces:
            print(f'📜 Setting up CA issuer in namespace: {namespace}')

            # Copy the service CA secret to this namespace
            self._copy_secret_between_namespaces('service-ca-root',
                                                 'cert-manager', namespace)

            issuer_yaml = f"""
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: service-ca-issuer
  namespace: {namespace}
spec:
  ca:
    secretName: service-ca-root
"""

            self._apply_yaml_content(issuer_yaml,
                                     f'ca-issuer-from-service-ca-{namespace}')
            print(f'✅ CA issuer created successfully in namespace: {namespace}')

    def _copy_secret_between_namespaces(self, secret_name: str, source_ns: str,
                                        target_ns: str):
        """Copy a secret from one namespace to another."""
        print(
            f'🔄 Copying secret {secret_name} from {source_ns} to {target_ns}...'
        )

        # Check if secret already exists in target namespace
        result = self.deployer.deployment_manager.run_command(
            ['kubectl', 'get', 'secret', secret_name, '-n', target_ns],
            check=False)

        if result.returncode == 0:
            print(f'⏭️  Secret {secret_name} already exists in {target_ns}, skipping copy')
            return

        # Get the secret using argument list (no shell interpolation)
        result = self.deployer.deployment_manager.run_command(
            ['kubectl', 'get', 'secret', secret_name, '-n', source_ns,
             '-o', 'yaml'])

        # Parse YAML in Python and update namespace safely
        secret_data = yaml.safe_load(result.stdout)
        secret_data['metadata']['namespace'] = target_ns
        for field in ('resourceVersion', 'uid', 'creationTimestamp'):
            secret_data['metadata'].pop(field, None)

        # Apply via temp file using existing helper
        self._apply_yaml_content(
            yaml.dump(secret_data, default_flow_style=False),
            f'copy-secret-{secret_name}-to-{target_ns}')
        print(f'✅ Secret {secret_name} copied successfully')

    def _create_component_certificate(self,
                                      name: str,
                                      dns_names: List[str],
                                      namespace: str = None):
        """Create a certificate for a specific component."""
        if namespace is None:
            namespace = self.deployment_namespace

        print(f'📜 Creating certificate: {name} in namespace: {namespace}...')

        dns_names_yaml = '\n'.join([f'    - "{dns}"' for dns in dns_names])

        cert_yaml = f"""
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {name}
  namespace: {namespace}
spec:
  secretName: {name}
  commonName: {dns_names[0]}
  dnsNames:
{dns_names_yaml}
  issuerRef:
    name: service-ca-issuer
    kind: Issuer
    group: cert-manager.io
  duration: 8760h # 1 year
  renewBefore: 720h # 1 month
  usages:
    - key encipherment
    - digital signature
    - server auth
"""

        self._apply_yaml_content(cert_yaml,
                                 f'component-certificate-{name}-{namespace}')

    def _create_service_ca_configmap(self):
        """Create openshift-service-ca.crt ConfigMap equivalent for kind.

        This ConfigMap is expected by DSPA components to verify TLS
        connections.
        """
        print('📜 Creating openshift-service-ca.crt ConfigMap...')

        try:
            # Extract CA certificate from the service-ca-root secret
            ca_cert_result = self.deployer.deployment_manager.run_command(
                [
                    'kubectl', 'get', 'secret', 'service-ca-root', '-n',
                    self.deployment_namespace, '-o', 'jsonpath={.data.ca\\.crt}'
                ],
                check=True)

            if not ca_cert_result.stdout.strip():
                raise RuntimeError(
                    'Failed to extract CA certificate from service-ca-root secret'
                )

            # Decode base64 certificate
            ca_cert = base64.b64decode(
                ca_cert_result.stdout.strip()).decode('utf-8')

            # Create ConfigMap with proper indentation
            ca_cert_indented = '\n'.join(
                [f'    {line}' for line in ca_cert.split('\n')])

            # Get list of namespaces where we need ConfigMaps
            namespaces = [self.deployment_namespace]
            if self.deployer.args.deploy_external_db and self.deployer.external_db_namespace not in namespaces:
                namespaces.append(self.deployer.external_db_namespace)

            # Create ConfigMap in all required namespaces
            for namespace in namespaces:
                configmap_yaml = f"""
apiVersion: v1
kind: ConfigMap
metadata:
  name: openshift-service-ca.crt
  namespace: {namespace}
data:
  service-ca.crt: |
{ca_cert_indented}
"""

                self._apply_yaml_content(
                    configmap_yaml,
                    f'openshift-service-ca-configmap-{namespace}')
                print(
                    f'✅ openshift-service-ca.crt ConfigMap created successfully in namespace: {namespace}'
                )

        except Exception as e:
            print(f'❌ Failed to create openshift-service-ca.crt ConfigMap: {e}')
            raise

    def _verify_certificates(self):
        """Verify that all required certificates are ready and valid."""
        print('🔍 Verifying certificate status...')

        certificates = self._get_required_certificates()

        for cert in certificates:
            try:
                cert_namespace = cert.get('namespace',
                                          self.deployment_namespace)

                # Check if certificate exists and is ready
                result = self.deployer.deployment_manager.run_command([
                    'kubectl', 'get', 'certificate', cert['name'], '-n',
                    cert_namespace, '-o',
                    'jsonpath={.status.conditions[?(@.type=="Ready")].status}'
                ],
                                                                      check=False
                                                                     )

                if result.returncode == 0 and result.stdout.strip() == 'True':
                    print(
                        f'✅ Certificate {cert["name"]} is ready in namespace {cert_namespace}'
                    )
                else:
                    print(
                        f'⚠️  Certificate {cert["name"]} is not ready in namespace {cert_namespace}'
                    )
                    # Get more details
                    self.deployer.deployment_manager.run_command([
                        'kubectl', 'describe', 'certificate', cert['name'],
                        '-n', cert_namespace
                    ],
                                                                 check=False)

                # Check if corresponding secret exists
                secret_result = self.deployer.deployment_manager.run_command(
                    [
                        'kubectl', 'get', 'secret', cert['name'], '-n',
                        cert_namespace
                    ],
                    check=False)

                if secret_result.returncode == 0:
                    print(
                        f'✅ Secret {cert["name"]} exists in namespace {cert_namespace}'
                    )
                else:
                    print(
                        f'❌ Secret {cert["name"]} does not exist in namespace {cert_namespace}'
                    )

            except Exception as e:
                print(f'⚠️  Error verifying certificate {cert["name"]}: {e}')

        print('🔍 Certificate verification completed')

    def configure_mariadb_for_tls(self):
        """Configure MariaDB deployment to use TLS certificates.

        Note: For internal MariaDB (DSPA-managed), TLS ConfigMap is created
        during setup_tls_for_kind() before DSPA deployment. This method
        only handles external MariaDB post-deployment configuration.
        """
        if self.deployer.args.deploy_external_db:
            self._configure_external_mariadb_tls()
        else:
            # Internal MariaDB TLS is handled during setup_tls_for_kind()
            print(
                '🔐 Internal MariaDB TLS already configured during TLS setup phase'
            )

    def _configure_external_mariadb_tls(self):
        """Configure external MariaDB deployment for TLS."""
        print('🔐 Configuring external MariaDB for TLS connections...')

        try:
            # Get the MariaDB TLS certificate name
            mariadb_cert_name = f'ds-pipelines-mariadb-tls-{self.dspa_name}'

            # Create a MySQL configuration ConfigMap for TLS
            tls_config = f"""
apiVersion: v1
kind: ConfigMap
metadata:
  name: mariadb-tls-config
  namespace: {self.deployer.external_db_namespace}
data:
  50-server-tls.cnf: |
    [mariadb]
    ssl-cert=/etc/mysql/tls/tls.crt
    ssl-key=/etc/mysql/tls/tls.key
    ssl-ca=/etc/mysql/ca/service-ca.crt
    require_secure_transport=ON
    bind-address=0.0.0.0
"""

            self.deployer.deployment_manager.apply_resource(
                manifest_content=tls_config,
                description='MariaDB TLS configuration')

            # Patch MariaDB deployment using strategic-merge so that
            # volumes/volumeMounts are merged by name, making the
            # operation idempotent on retries.
            patch_obj = {
                'spec': {
                    'template': {
                        'spec': {
                            'volumes': [
                                {
                                    'name': 'tls-certs',
                                    'secret': {
                                        'secretName': mariadb_cert_name
                                    }
                                },
                                {
                                    'name': 'tls-config',
                                    'configMap': {
                                        'name': 'mariadb-tls-config'
                                    }
                                },
                                {
                                    'name': 'ca-certs',
                                    'configMap': {
                                        'name': 'openshift-service-ca.crt'
                                    }
                                },
                            ],
                            'containers': [{
                                'name': 'mariadb',
                                'volumeMounts': [
                                    {
                                        'name': 'tls-certs',
                                        'mountPath': '/etc/mysql/tls',
                                        'readOnly': True
                                    },
                                    {
                                        'name': 'tls-config',
                                        'mountPath': '/etc/mysql/conf.d',
                                        'readOnly': True
                                    },
                                    {
                                        'name': 'ca-certs',
                                        'mountPath': '/etc/mysql/ca',
                                        'readOnly': True
                                    },
                                ]
                            }]
                        }
                    }
                }
            }

            print('🔧 Patching MariaDB deployment to enable TLS...')
            self.deployer.deployment_manager.run_command([
                'kubectl', 'patch', 'deployment', 'mariadb', '-n',
                self.deployer.external_db_namespace, '--type=strategic',
                '-p', json.dumps(patch_obj)
            ])

            # Wait for MariaDB to restart with new configuration
            print('⏳ Waiting for MariaDB to restart with TLS configuration...')
            self.deployer.deployment_manager.run_command([
                'kubectl', 'rollout', 'status', 'deployment/mariadb', '-n',
                self.deployer.external_db_namespace, '--timeout=300s'
            ])

            print('✅ MariaDB TLS configuration applied successfully')

        except Exception as e:
            print(f'❌ Failed to configure external MariaDB for TLS: {e}')
            raise

    def _configure_internal_mariadb_tls(self):
        """Pre-create MariaDB TLS ConfigMap that operator template expects.

        The operator template automatically mounts TLS certificates and
        config when PodToPodTLS=true, but only if the resources already
        exist.
        """
        print('🔐 Pre-creating MariaDB TLS ConfigMap for operator template...')

        try:
            configmap_name = f'ds-pipelines-mariadb-tls-config-{self.dspa_name}'

            # Create TLS configuration that matches operator expectations
            # The operator template mounts this to /etc/my.cnf.d/mariadb-tls-config.cnf
            tls_config_content = """[mariadb]
ssl_cert = /.mariadb/certs/tls.crt
ssl_key = /.mariadb/certs/tls.key
require_secure_transport = ON
bind-address = 0.0.0.0

[mysql]
ssl-cert = /.mariadb/certs/tls.crt
ssl-key = /.mariadb/certs/tls.key"""

            complete_tls_config = f"""
apiVersion: v1
kind: ConfigMap
metadata:
  name: {configmap_name}
  namespace: {self.deployment_namespace}
  labels:
    app: mariadb-{self.dspa_name}
    component: data-science-pipelines
    dsp-version: v2
data:
  mariadb-tls-config.cnf: |
{self._indent_content(tls_config_content, 4)}
"""

            print(f'🔧 Creating MariaDB TLS ConfigMap: {configmap_name}')
            print(
                '📋 This ConfigMap will be automatically mounted by the operator template'
            )
            self.deployer.deployment_manager.apply_resource(
                manifest_content=complete_tls_config,
                description=f'MariaDB TLS configuration for operator template')

            print(
                '✅ MariaDB TLS ConfigMap created - operator will mount it automatically'
            )

        except Exception as e:
            print(f'❌ Failed to create MariaDB TLS ConfigMap: {e}')
            raise

    def _indent_content(self, content: str, spaces: int) -> str:
        """Indent content by specified number of spaces."""
        indent = ' ' * spaces
        return '\n'.join(indent + line for line in content.split('\n'))

    def _apply_yaml_content(self, yaml_content: str, description: str):
        """Apply YAML content to the cluster."""
        with tempfile.NamedTemporaryFile(
                mode='w', suffix='.yaml', delete=False) as f:
            f.write(yaml_content)
            temp_file = f.name

        try:
            self.deployer.deployment_manager.run_command(
                ['kubectl', 'apply', '-f', temp_file])
        except Exception as e:
            print(f'❌ Failed to apply {description}: {e}')
            raise
        finally:
            os.unlink(temp_file)

    def cleanup_certificates(self):
        """Clean up all created certificates and related resources."""
        print('🧹 Cleaning up TLS certificates...')

        try:
            # Delete component certificates using each cert's own namespace
            certificates = self._get_required_certificates()
            for cert in certificates:
                cert_ns = cert.get('namespace', self.deployment_namespace)
                self.deployer.deployment_manager.run_command([
                    'kubectl', 'delete', 'certificate', cert['name'], '-n',
                    cert_ns
                ],
                                                             check=False)

                self.deployer.deployment_manager.run_command([
                    'kubectl', 'delete', 'secret', cert['name'], '-n',
                    cert_ns
                ],
                                                             check=False)

            # Collect all namespaces that need issuer/CA cleanup
            cleanup_namespaces = [self.deployment_namespace]
            if self.deployer.args.deploy_external_db and \
                    self.deployer.external_db_namespace not in cleanup_namespaces:
                cleanup_namespaces.append(self.deployer.external_db_namespace)

            for ns in cleanup_namespaces:
                # Delete issuers
                self.deployer.deployment_manager.run_command([
                    'kubectl', 'delete', 'issuer', 'service-ca-issuer', '-n',
                    ns
                ],
                                                             check=False)

                # Delete copied service-ca-root secret
                self.deployer.deployment_manager.run_command([
                    'kubectl', 'delete', 'secret', 'service-ca-root', '-n',
                    ns
                ],
                                                             check=False)

                # Delete ConfigMaps
                self.deployer.deployment_manager.run_command([
                    'kubectl', 'delete', 'configmap',
                    'openshift-service-ca.crt', '-n', ns
                ],
                                                             check=False)

            self.deployer.deployment_manager.run_command(
                ['kubectl', 'delete', 'clusterissuer', 'service-ca-issuer'],
                check=False)

            # Delete service CA certificate and secret from cert-manager
            self.deployer.deployment_manager.run_command([
                'kubectl', 'delete', 'certificate', 'service-ca-root', '-n',
                'cert-manager'
            ],
                                                         check=False)

            self.deployer.deployment_manager.run_command([
                'kubectl', 'delete', 'secret', 'service-ca-root', '-n',
                'cert-manager'
            ],
                                                         check=False)

            print('✅ TLS certificate cleanup completed')

        except Exception as e:
            print(f'⚠️  Error during cleanup: {e}')

    def get_ca_bundle_config(self) -> Dict[str, str]:
        """Get the CA bundle configuration for DSPA spec.

        Returns:
            Dictionary with configMapName and configMapKey for DSPA cABundle configuration
        """
        return {
            'configMapName': 'openshift-service-ca.crt',
            'configMapKey': 'service-ca.crt'
        }

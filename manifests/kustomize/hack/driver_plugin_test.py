# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Validate embedded driver sidecars after Kustomize has rendered each variant."""

import base64
import copy
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

from tensorboard_signing_key_test import render, resource, REPO_ROOT


class DriverPluginTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.variants = {}
        for name, path in {
                'standalone':
                    'manifests/kustomize/env/platform-agnostic',
                'tls':
                    'manifests/kustomize/env/cert-manager/platform-agnostic-standalone-tls',
                'dev':
                    'manifests/kustomize/env/dev',
                'openshift':
                    'manifests/kustomize/env/openshift/base',
                'ci':
                    '.github/resources/manifests/standalone/default',
                'ci-tls':
                    '.github/resources/manifests/standalone/tls-enabled',
        }.items():
            resources = render(REPO_ROOT / path, allow_external_patches=True)
            config = resource(resources, 'ConfigMap',
                              'ml-pipeline-driver-agent')
            sidecar = json.loads(
                subprocess.check_output(
                    ['yq', 'r', '-j', '-'],
                    input=config['data']['sidecar.container'],
                    text=True))
            cls.variants[name] = resources, config, sidecar

    def test_released_images_match_the_launcher(self):
        for name in ('standalone', 'tls', 'openshift'):
            with self.subTest(variant=name):
                resources, _, sidecar = self.variants[name]
                deployment = resource(resources, 'Deployment', 'ml-pipeline')
                env = deployment['spec']['template']['spec']['containers'][0][
                    'env']
                launcher = next(item['value']
                                for item in env
                                if item['name'] == 'V2_LAUNCHER_IMAGE')
                self.assertEqual(
                    sidecar['image'],
                    launcher.replace('kfp-launcher:', 'kfp-driver:'))
                self.assertNotIn(':dummy', sidecar['image'])

    def test_development_and_ci_images(self):
        self.assertEqual(self.variants['dev'][2]['image'],
                         'ghcr.io/kubeflow/kfp-driver:master')
        for name in ('ci', 'ci-tls'):
            with self.subTest(variant=name):
                self.assertEqual(self.variants[name][2]['image'],
                                 'kind-registry:5000/driver:ci')

    def test_sidecars_share_runtime_configuration(self):
        # ConfigMap string data cannot be strategically merged by Kustomize.
        # Permit only the intentional image/TLS differences across full-string patches.
        expected = self.normalized_sidecar(self.variants['standalone'][2])
        for name, (_, config, sidecar) in self.variants.items():
            with self.subTest(variant=name):
                self.assertEqual(
                    config['data']['sidecar.automountServiceAccountToken'],
                    'true')
                actual = self.normalized_sidecar(sidecar)
                if name == 'openshift':
                    self.assertNotIn('runAsUser', actual['securityContext'])
                    actual['securityContext']['runAsUser'] = 8737
                self.assertEqual(actual, expected)
                env = {item['name']: item for item in sidecar['env']}
                self.assertNotIn('LOG_ACCESS_KEY', env)
                self.assertNotIn('LOG_SECRET_KEY', env)
                mounts = {
                    item['name']: item for item in sidecar['volumeMounts']
                }
                self.assertEqual(mounts['var-run-argo']['mountPath'],
                                 '/kfp/log')
                self.assertEqual(mounts['var-run-argo']['subPath'],
                                 sidecar['name'])
                self.assertFalse(mounts['var-run-argo']['readOnly'])
                if name in ('tls', 'ci-tls'):
                    self.assertEqual(env['CA_CERT_PATH']['value'],
                                     '/kfp/certs/ca.crt')
                    self.assertEqual(
                        mounts['argo-workflows-agent-ca-certificates']
                        ['mountPath'], '/kfp/certs')
                    self.assertTrue(
                        mounts['argo-workflows-agent-ca-certificates']
                        ['readOnly'])
                else:
                    self.assertNotIn('CA_CERT_PATH', env)
                    self.assertNotIn('argo-workflows-agent-ca-certificates',
                                     mounts)

    def test_agent_controller_patches(self):
        for name in ('tls', 'ci-tls'):
            with self.subTest(variant=name):
                config = resource(self.variants[name][0], 'ConfigMap',
                                  'workflow-controller-configmap')
                defaults = json.loads(
                    subprocess.check_output(
                        ['yq', 'r', '-j', '-'],
                        input=config['data']['workflowDefaults'],
                        text=True))
                patch = json.loads(
                    subprocess.check_output(
                        ['yq', 'r', '-j', '-'],
                        input=defaults['spec']['podSpecPatch'],
                        text=True))
                secret = patch['volumes'][0]['secret']
                self.assertEqual(secret['items'], [{
                    'key': 'ca.crt',
                    'path': 'ca.crt'
                }])
                self.assertEqual(secret['secretName'],
                                 'argo-workflows-agent-ca-certificates')
        config = resource(self.variants['openshift'][0], 'ConfigMap',
                          'workflow-controller-configmap')
        self.assertIn('enabled: false', config['data']['initlessPod'])
        executor = json.loads(
            subprocess.check_output(['yq', 'r', '-j', '-'],
                                    input=config['data']['executor'],
                                    text=True))
        self.assertNotIn('runAsUser', executor['securityContext'])

    def test_metacontroller_binds_role_without_driver_permissions(self):
        resources = render(
            REPO_ROOT / 'manifests/kustomize/third-party/metacontroller/base')
        role = resource(resources, 'ClusterRole', 'kubeflow-metacontroller')
        for rule in role['rules']:
            if '' in rule['apiGroups']:
                self.assertNotIn('pods', rule['resources'])
                self.assertNotIn('persistentvolumeclaims', rule['resources'])
            self.assertNotIn('mlflow.kubeflow.org', rule['apiGroups'])
        grants = [rule for rule in role['rules'] if 'bind' in rule['verbs']]
        self.assertEqual(grants, [{
            'apiGroups': ['rbac.authorization.k8s.io'],
            'resources': ['clusterroles'],
            'resourceNames':
                ['ml-pipeline-driver-agent-executor-plugin-cluster-role'],
            'verbs': ['bind'],
        }])

    def test_documented_custom_ca_without_cert_manager(self):
        guide = (REPO_ROOT / 'manifests/kustomize/README.md').read_text()
        section = guide.split('## Driver custom CA migration', 1)[1]
        overlay_yaml = section.split('```yaml\n', 1)[1].split('```', 1)[0]
        # A distinctive bundle checks data routing, not certificate validation.
        bundle = 'custom-company-ca-test-bundle\n'
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / 'kustomize'
            shutil.copytree(REPO_ROOT / 'manifests/kustomize', root)
            overlay = root / 'env/custom-ca'
            overlay.mkdir()
            (overlay / 'company-ca.crt').write_text(bundle)
            (overlay / 'kustomization.yaml').write_text(overlay_yaml)
            resources = render(overlay, allow_external_patches=True)
        secret = resource(resources, 'Secret',
                          'argo-workflows-agent-ca-certificates')
        self.assertEqual(secret['metadata']['namespace'], 'kubeflow')
        self.assertEqual(
            base64.b64decode(secret['data']['ca.crt']).decode(), bundle)
        config = resource(resources, 'ConfigMap', 'company-ca')
        self.assertEqual(config['data']['ca.crt'], bundle)
        deployment = resource(resources, 'Deployment', 'ml-pipeline')
        containers = deployment['spec']['template']['spec']['containers']
        api = next(
            c for c in containers if c['name'] == 'ml-pipeline-api-server')
        env = {item['name']: item for item in api['env']}
        self.assertEqual(env['CABUNDLE_CONFIGMAP_NAME']['value'], 'company-ca')
        config = resource(resources, 'ConfigMap', 'ml-pipeline-driver-agent')
        sidecar = json.loads(
            subprocess.check_output(['yq', 'r', '-j', '-'],
                                    input=config['data']['sidecar.container'],
                                    text=True))
        env = {item['name']: item for item in sidecar['env']}
        self.assertEqual(env['CA_CERT_PATH']['value'], '/kfp/certs/ca.crt')
        mounts = {item['name']: item for item in sidecar['volumeMounts']}
        mount = mounts['argo-workflows-agent-ca-certificates']
        self.assertEqual(mount['mountPath'], '/kfp/certs')
        self.assertTrue(mount['readOnly'])
        self.assertFalse(
            any(item['apiVersion'].startswith('cert-manager.io/')
                for item in resources))

    def test_documented_custom_artifact_secret_authorization_contract(self):
        guide = (REPO_ROOT / 'manifests/kustomize/README.md').read_text()
        section = guide.split('## Driver log artifact credentials', 1)[1]
        grant = section.split('```yaml\n', 1)[1].split('```', 1)[0]
        with tempfile.TemporaryDirectory() as directory:
            overlay = Path(directory)
            (overlay / 'grant.yaml').write_text(grant)
            (overlay /
             'kustomization.yaml').write_text('resources:\n  - grant.yaml\n')
            resources = render(overlay)
        role = resource(resources, 'Role', 'driver-custom-artifact-credentials')
        self.assertEqual(role['metadata']['namespace'],
                         'your-pipeline-namespace')
        self.assertEqual(role['rules'], [{
            'apiGroups': [''],
            'resources': ['secrets'],
            'resourceNames': ['artifact-store-credentials'],
            'verbs': ['get']
        }])
        binding = resource(resources, 'RoleBinding',
                           'driver-custom-artifact-credentials')
        self.assertEqual(binding['metadata']['namespace'],
                         role['metadata']['namespace'])
        self.assertEqual(
            binding['roleRef'], {
                'apiGroup': 'rbac.authorization.k8s.io',
                'kind': 'Role',
                'name': role['metadata']['name']
            })
        self.assertEqual(binding['subjects'], [{
            'kind': 'ServiceAccount',
            'name': 'ml-pipeline-driver-agent-executor-plugin',
            'namespace': role['metadata']['namespace']
        }])

    @staticmethod
    def normalized_sidecar(sidecar):
        sidecar = copy.deepcopy(sidecar)
        sidecar.pop('image')
        sidecar.setdefault('imagePullPolicy', 'IfNotPresent')
        sidecar['env'] = [
            item for item in sidecar['env'] if item['name'] != 'CA_CERT_PATH'
        ]
        sidecar['volumeMounts'] = [
            item for item in sidecar['volumeMounts']
            if item['name'] != 'argo-workflows-agent-ca-certificates'
        ]
        return sidecar

    def test_attempt_resolver_has_only_taskset_get_permission(self):
        standalone = self.variants['standalone'][0]
        multiuser = render(REPO_ROOT /
                           'manifests/kustomize/base/installs/multi-user')
        for resources, kind, name in (
            (standalone, 'Role', 'ml-pipeline-executor-plugin-role'),
            (multiuser, 'ClusterRole',
             'ml-pipeline-driver-agent-executor-plugin-cluster-role')):
            role = resource(resources, kind, name)
            rules = [
                rule for rule in role['rules']
                if 'workflowtasksets' in rule['resources']
            ]
            self.assertEqual(len(rules), 1)
            self.assertEqual(rules[0]['apiGroups'], ['argoproj.io'])
            self.assertEqual(rules[0]['verbs'], ['get'])
            self.assertNotIn('workflowtasksets/status', rules[0]['resources'])

    def test_application_controller_requests_fit_limits(self):
        resources = render(REPO_ROOT /
                           'manifests/kustomize/third-party/application')
        deployment = resource(resources, 'Deployment', 'controller-manager')
        container = deployment['spec']['template']['spec']['containers'][0]
        self.assertEqual(container['resources']['requests']['memory'], '30Mi')
        self.assertEqual(container['resources']['limits']['memory'], '30Mi')

    def test_runtime_account_token_secret(self):
        resources, _, _ = self.variants['standalone']
        secret = resource(resources, 'Secret',
                          'pipeline-runner.service-account-token')
        self.assertEqual(secret['type'], 'kubernetes.io/service-account-token')
        self.assertEqual(
            secret['metadata']['annotations']
            ['kubernetes.io/service-account.name'], 'pipeline-runner')


if __name__ == '__main__':
    unittest.main()

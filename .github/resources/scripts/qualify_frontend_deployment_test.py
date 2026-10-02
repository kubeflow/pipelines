# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
"""Browser/cluster-free regression checks for deployment evidence
boundaries."""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import qualify_frontend_deployment as qualification


class QualificationTest(unittest.TestCase):

    def test_rejects_workstation_before_commands(self):
        for env in [{}, {
                'CI': 'true',
                'GITHUB_ACTIONS': 'true',
                'RUNNER_ENVIRONMENT': 'self-hosted',
                'RUNNER_TEMP': '/tmp'
        }]:
            with self.assertRaises(RuntimeError):
                qualification.require_hosted(env)

    def test_accepts_disposable_hosted_environment(self):
        qualification.require_hosted({
            'CI': 'true',
            'GITHUB_ACTIONS': 'true',
            'RUNNER_ENVIRONMENT': 'github-hosted',
            'RUNNER_TEMP': '/tmp'
        })

    def test_every_backend_and_credential_invariant_is_required(self):
        baseline = {
            'resources': {
                'Deployment/kubeflow/api': {
                    'uid': 'one',
                    'contentSha256': 'hash'
                }
            },
            'pods': {
                'api-pod': 'pod-one'
            },
            'signingUid': 'secret-one',
            'signingHash': 'a',
            'authenticationSecrets': {
                'auth/dex': {
                    'uid': 'dex-one',
                    'dataHash': 'b'
                }
            }
        }
        qualification.assert_preserved(baseline, copy.deepcopy(baseline))
        for key in baseline:
            modified = copy.deepcopy(baseline)
            modified[key] = 'changed'
            with self.subTest(key=key), self.assertRaisesRegex(
                    AssertionError, key):
                qualification.assert_preserved(baseline, modified)

    def test_resource_normalization_keeps_rbac_and_drops_volatile_status(self):
        role = {
            'apiVersion': 'rbac.authorization.k8s.io/v1',
            'kind': 'ClusterRole',
            'metadata': {
                'name': 'ml-pipeline',
                'uid': 'role-one',
                'resourceVersion': '10'
            },
            'rules': [{
                'verbs': ['get'],
                'resources': ['pods']
            }]
        }
        first = qualification.normalize_resources([role])
        role['metadata']['resourceVersion'] = '20'
        self.assertEqual(first, qualification.normalize_resources([role]))
        role['rules'][0]['verbs'].append('delete')
        self.assertNotEqual(first, qualification.normalize_resources([role]))

    def test_ui_exemption_does_not_exempt_backend(self):

        def deployment(name):
            return {
                'kind': 'Deployment',
                'metadata': {
                    'name': name,
                    'namespace': 'kubeflow',
                    'uid': name
                },
                'spec': {
                    'template': {
                        'spec': {
                            'containers': [{
                                'image': 'test'
                            }]
                        }
                    }
                }
            }

        value = qualification.normalize_resources(
            [deployment('ml-pipeline-ui'),
             deployment('ml-pipeline')])
        self.assertEqual(list(value), ['Deployment/kubeflow/ml-pipeline'])

    def test_input_bearing_command_never_exposes_stderr(self):
        result = argparse.Namespace(
            returncode=1, stdout='', stderr='secret-value')
        with patch.object(qualification.subprocess, 'run', return_value=result):
            with self.assertRaises(RuntimeError) as error:
                qualification.command(
                    'kubectl', 'apply', payload='credential manifest')
        self.assertNotIn('secret-value', str(error.exception))
        self.assertNotIn('credential manifest', str(error.exception))

    def test_image_setup_creates_output_before_extracting_assets(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'not-yet-created' / 'report'
            args = argparse.Namespace(
                output=str(output),
                legacy_archive='legacy.tar',
                candidate_archive='candidate.tar')

            def record(*unused):
                self.assertTrue(output.is_dir())
                return {'reference': 'image@sha256:fixture'}

            with patch.object(
                    qualification, 'command', return_value=''), patch.object(
                        qualification, 'image_record', side_effect=record):
                qualification.setup_images(args)
            self.assertTrue((output / 'images.json').is_file())

    def test_image_identity_resolves_alias_only_to_exact_qualified_config(self):
        expected = {
            'reference': 'registry/ui@sha256:manifest',
            'configId': 'sha256:config'
        }
        manifest = {
            'schemaVersion': 2,
            'mediaType': 'application/vnd.oci.image.manifest.v1+json',
            'config': {
                'digest': expected['configId']
            },
            'layers': []
        }

        def verify(value, raw_override=None):
            raw = json.dumps(value).encode()
            digest = 'sha256:' + hashlib.sha256(raw).hexdigest()
            status = {
                'ready': True,
                'imageID': 'docker.io/library/import-old@' + digest
            }

            def read(actual):
                self.assertEqual(actual, digest)
                return raw if raw_override is None else raw_override

            return qualification.verify_image_identity(status, expected, read)

        result = verify(manifest)
        self.assertEqual(result['method'],
                         'resolved-runtime-manifest-config-digest')
        self.assertEqual(result['resolvedConfigId'], expected['configId'])
        for invalid in [
            {
                **manifest, 'schemaVersion': 1
            },
            {
                **manifest, 'mediaType':
                    'application/vnd.oci.image.index.v1+json'
            },
            {
                **manifest, 'config': {
                    'digest': 'sha256:wrong'
                }
            },
            {
                **manifest, 'config': {}
            },
        ]:
            with self.subTest(
                    manifest=invalid), self.assertRaises(AssertionError):
                verify(invalid)
        with self.assertRaises(AssertionError):
            verify(manifest, b'altered bytes')
        for invalid in [
            {
                'ready': False,
                'imageID': 'sha256:config'
            },
            {
                'ready': True,
                'imageID': 'docker.io/library/import-old:latest'
            },
        ]:
            with self.assertRaises(AssertionError):
                qualification.verify_image_identity(
                    invalid, expected, lambda _: self.fail(
                        'Invalid identity must not read content'))

    def test_runtime_manifest_retains_failed_proof_bytes_before_validation(
            self):
        expected = {
            'reference': 'registry/ui@sha256:other',
            'configId': 'sha256:config'
        }
        address = 'sha256:' + 'a' * 64
        raw = b'{\r\n"schemaVersion": 1\r\n}\n'
        with tempfile.TemporaryDirectory() as directory, patch.object(
                qualification, 'command', return_value=raw) as command:
            with self.assertRaises(AssertionError):
                qualification.verify_image_identity(
                    {
                        'ready': True,
                        'imageID': 'import-old@' + address
                    }, expected, lambda digest: qualification.runtime_manifest(
                        'node', digest, directory))
            command.assert_called_once_with(
                'docker',
                'exec',
                'node',
                'ctr',
                '-n',
                'k8s.io',
                'content',
                'get',
                address,
                raw=True)
            self.assertEqual((Path(directory) / 'runtime-manifests' /
                              ('a' * 64 + '.json')).read_bytes(), raw)

    def test_content_command_preserves_manifest_bytes(self):
        content = b'{\r\n"schemaVersion": 2\r\n}\n'
        result = subprocess.CompletedProcess([], 0, stdout=content, stderr=b'')
        with patch.object(
                qualification.subprocess, 'run', return_value=result) as run:
            self.assertEqual(
                qualification.command(
                    'ctr', 'content', 'get', 'sha256:fixture', raw=True),
                content)
        self.assertIs(run.call_args.kwargs['text'], False)

    def test_exact_image_digest_does_not_need_runtime_alias_resolution(self):
        expected = {
            'reference': 'registry/ui@sha256:manifest',
            'configId': 'sha256:config'
        }

        def unexpected(_):
            self.fail('Exact image identity should not require alias lookup')

        for image in [
                'registry/ui@sha256:manifest', 'sha256:config',
                'containerd://sha256:config'
        ]:
            qualification.verify_image_identity(
                {
                    'ready': True,
                    'imageID': image
                }, expected, unexpected)

    def test_mesh_readiness_requires_native_sidecar_before_network_wait(self):
        pod = {
            'metadata': {
                'uid': 'pod'
            },
            'spec': {
                'serviceAccountName':
                    'ml-pipeline',
                'initContainers': [
                    {
                        'name': 'istio-validation'
                    },
                    {
                        'name': 'istio-proxy',
                        'restartPolicy': 'Always'
                    },
                    {
                        'name': 'wait-for-database'
                    },
                ]
            },
            'status': {
                'containerStatuses': [{
                    'name': 'application',
                    'ready': True
                }],
                'initContainerStatuses': [{
                    'name': 'istio-proxy',
                    'ready': True,
                    'started': True,
                    'imageID': 'sha256:proxy'
                }]
            },
        }
        self.assertTrue(qualification.mesh_pod_evidence(pod)['proxyReady'])
        for change in ('missing', 'not-native', 'not-ready', 'wrong-order'):
            invalid = copy.deepcopy(pod)
            if change == 'missing':
                invalid['spec']['initContainers'].pop(1)
            elif change == 'not-native':
                invalid['spec']['initContainers'][1].pop('restartPolicy')
            elif change == 'not-ready':
                invalid['status']['initContainerStatuses'][0]['ready'] = False
            else:
                invalid['spec']['initContainers'].reverse()
            with self.subTest(change=change), self.assertRaises(AssertionError):
                qualification.mesh_pod_evidence(invalid)

    def test_mesh_setup_includes_mutual_tls_storage_and_excludes_generated_jobs(
            self):
        self.assertEqual(
            set(qualification.MESH_DEPLOYMENTS), {
                'mysql',
                'seaweedfs',
                'ml-pipeline',
                'ml-pipeline-ui',
                'ml-pipeline-persistenceagent',
                'ml-pipeline-scheduledworkflow',
                'ml-pipeline-viewer-crd',
            })

    def test_mesh_image_must_remain_the_initial_qualified_image(self):
        with patch.object(
                qualification,
                'mesh_pod_evidence',
                return_value={'proxyImageId': 'sha256:original'}):
            qualification.verify_mesh_image({}, [{
                'proxyImageId': 'sha256:original'
            }])
            with self.assertRaises(AssertionError):
                qualification.verify_mesh_image({}, [{
                    'proxyImageId': 'sha256:different'
                }])

    def test_gateway_allowance_is_limited_to_ui_ingress_from_gateway(self):
        policy = qualification.ingress_ui_policy()['spec']
        self.assertEqual(policy['podSelector'],
                         {'matchLabels': {
                             'app': 'ml-pipeline-ui'
                         }})
        self.assertEqual(policy['policyTypes'], ['Ingress'])
        rule = policy['ingress'][0]
        self.assertEqual(rule['ports'], [{'protocol': 'TCP', 'port': 3000}])
        source = rule['from'][0]
        self.assertEqual(source['namespaceSelector']['matchLabels'],
                         {'kubernetes.io/metadata.name': 'istio-system'})
        self.assertEqual(source['podSelector']['matchLabels'],
                         {'app': 'istio-ingressgateway'})
        self.assertNotIn('egress', policy)


if __name__ == '__main__':
    unittest.main()

#!/usr/bin/env python3
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
"""Deployment and generation wiring for the supported KFP API."""

from pathlib import Path
import re
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[3]


class V2DeploymentContractTest(unittest.TestCase):

    def test_api_and_ui_probes_use_v2_healthz(self):
        for filename, count in (('ml-pipeline-apiserver-deployment.yaml', 3),
                                ('ml-pipeline-ui-deployment.yaml', 2)):
            with self.subTest(deployment=filename):
                manifest = (ROOT / 'manifests/kustomize/base/pipeline' /
                            filename).read_text()
                self.assertEqual(
                    manifest.count('path: /apis/v2beta1/healthz'), count)
                self.assertNotIn('/apis/v1beta1', manifest)

    def test_generation_and_release_only_generate_v2_clients(self):
        for path in ('Makefile', 'release/kfpr/core.py',
                     '.github/workflows/validate-generated-files.yml'):
            with self.subTest(path=path):
                text = (ROOT / path).read_text()
                self.assertIn('v2beta1', text)
                self.assertNotIn('v1beta1', text)

    def test_backend_dev_targets_submit_ir_from_existing_sources(self):
        makefile = (ROOT / 'backend/src/v2/Makefile').read_text()
        self.assertIn('kfp run submit -f "$${tmp}/$*-spec.yaml"', makefile)
        self.assertIn('build/compiler --spec "$${tmp}/$*-spec.yaml"', makefile)
        self.assertIn('argo lint "$${tmp}/$*.yaml"', makefile)
        targets = re.findall(r'^pipeline/([\w/-]+):$', makefile, re.MULTILINE)
        self.assertTrue(targets)
        for target in targets:
            with self.subTest(target=target):
                source = ROOT / 'test_data/sdk_compiled_pipelines/valid' / (
                    target + '.py')
                self.assertTrue(source.is_file(), source)

    def test_v2_integration_forwards_the_api_before_running_tests(self):
        workflow = (ROOT / '.github/workflows/'
                    'legacy-v2-api-integration-tests.yml').read_text()
        forwarding = './.github/resources/scripts/forward-port.sh kubeflow ml-pipeline 8888 8888'
        deploy = workflow.split('- name: Deploy\n', 1)[1].split(
            '- name: Deploy pipeline URL test fixtures', 1)[0]
        self.assertIn("forward_port: 'false'", deploy)
        self.assertEqual(workflow.count(forwarding), 1)
        self.assertLess(
            workflow.index('kubectl rollout status deployment/ml-pipeline'),
            workflow.index(forwarding))
        self.assertLess(
            workflow.index(forwarding),
            workflow.index('- name: API integration tests v2'))

    def test_visualization_service_is_not_deployed_or_generated(self):
        for root in ('manifests/kustomize', '.github/resources/manifests'):
            for path in (ROOT / root).rglob('kustomization.yaml'):
                with self.subTest(path=path):
                    self.assertNotIn('visualization', path.read_text())
        for path in ('backend/api/v2beta1/visualization.proto',
                     'backend/Dockerfile.visualization',
                     'backend/src/apiserver/visualization/server.py',
                     'frontend/src/apisv2beta1/visualization'):
            with self.subTest(path=path):
                self.assertFalse((ROOT / path).exists())
        for path in (
                'backend/api/v2beta1/swagger/kfp_api_single_file.swagger.json',
                'docs/_static/kfp_api_single_file.swagger.json'):
            with self.subTest(path=path):
                self.assertNotIn('visualization',
                                 (ROOT / path).read_text().lower())
        pipeline = (
            ROOT /
            'manifests/kustomize/base/pipeline/kustomization.yaml').read_text()
        self.assertIn('ml-pipeline-viewer-crd-deployment.yaml', pipeline)
        self.assertIn('ml-pipeline-ui-deployment.yaml', pipeline)

    def test_cloudbuild_cache_does_not_require_visualization_image(self):
        script = (ROOT / 'test/build-images.sh').read_text()
        function = re.search(r'function has_batch_images_been_built \{.*?\n\}',
                             script, re.DOTALL).group(0)
        images = ('frontend', 'scheduledworkflow', 'persistenceagent',
                  'viewer-crd-controller', 'inverse-proxy-agent',
                  'metadata-writer')
        for inventory, expected in ((images, 0), (images[:-1], 1)):
            with self.subTest(inventory=inventory):
                result = subprocess.run([
                    'bash', '-c',
                    'gcloud() { printf "%s\\n" ' + ' '.join(inventory) +
                    '; }\n' + function + '\nhas_batch_images_been_built'
                ],
                                        capture_output=True,
                                        text=True,
                                        check=False)
                self.assertEqual(result.returncode, expected, result.stderr)

    def test_v2_caching_is_configured_without_admission_webhook(self):
        self.assertFalse(
            (ROOT /
             'manifests/kustomize/base/cache/kustomization.yaml').exists())
        self.assertFalse(
            (ROOT / 'manifests/kustomize/base/cache-deployer/kustomization.yaml'
            ).exists())
        overlay = (ROOT / '.github/resources/manifests/standalone/'
                   'cache-disabled/cache-env.yaml').read_text()
        self.assertIn('name: CACHEENABLED', overlay)
        self.assertIn('value: "false"', overlay)
        self.assertTrue((ROOT / '.github/workflows/'
                         'legacy-v2-api-integration-tests.yml').is_file())


if __name__ == '__main__':
    unittest.main()

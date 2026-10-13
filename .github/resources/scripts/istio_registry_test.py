# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Regression tests for CI's rendered Istio registry migration."""

from pathlib import Path
import subprocess
import sys
import unittest

from migrate_istio_registry import migrate

SCRIPT = Path(__file__).with_name('migrate_istio_registry.py')


class IstioRegistryTest(unittest.TestCase):

    def test_images_and_embedded_injector_hubs_preserve_versions(self):
        manifest = """image: registry.istio.io/release/pilot:1.30.0
image: registry.istio.io/release/proxyv2@sha256:abc
image: gcr.io/istio-release/install-cni:1.30.0
values: |
  {"global": {"hub": "registry.istio.io/release", "tag": "1.30.0"}}
template: |
  hub: gcr.io/istio-release
image: ghcr.io/kubeflow/other:1
"""
        expected = manifest.replace('registry.istio.io/release',
                                    'docker.io/istio')
        expected = expected.replace('gcr.io/istio-release', 'docker.io/istio')
        self.assertEqual(migrate(manifest), expected)
        self.assertEqual(migrate(expected), expected)

    def test_unrelated_manifest_bytes_are_unchanged(self):
        text = 'image: docker.io/istio/pilot:1.30.0\n# keep spacing  \n'
        self.assertEqual(migrate(text), text)

    def test_empty_and_unknown_retired_references_fail_closed(self):
        for text in [
                '', '  ', 'image: registry.istio.io/testing/pilot:dev',
                'hub: registry.istio.io/release-other',
                'url: https://istio-release.storage.googleapis.com/charts'
        ]:
            with self.subTest(text=text), self.assertRaises(ValueError):
                migrate(text)

    def test_cli_failure_emits_no_partial_manifest(self):
        result = subprocess.run(
            [sys.executable, str(SCRIPT)],
            input='image: registry.istio.io/testing/pilot:dev',
            capture_output=True,
            text=True,
            check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, '')
        self.assertIn('inspect the pinned manifest', result.stderr)

    def test_render_and_transform_failures_abort_even_if_apply_succeeds(self):
        script = SCRIPT.with_name('deploy-kfp.sh').read_text()
        install = script.split('  # Include embedded injector hubs,', 1)[1]
        block = install[install.index('  ('):install.index('  echo "Waiting')]
        for render in [
                'return 1',
                "printf '%s' 'image: registry.istio.io/testing/pilot:dev'"
        ]:
            command = ('kubectl() { if [ "$1" = kustomize ]; then ' + render +
                       '; else cat >/dev/null; return 0; fi; }; '
                       'C_DIR="$1"; KUBEFLOW_MANIFESTS_SHA=pinned;\n' + block)
            result = subprocess.run(
                ['bash', '-c', command, 'test',
                 str(SCRIPT.parent)],
                capture_output=True,
                text=True,
                check=False)
            self.assertNotEqual(result.returncode, 0)

    def test_deployment_renders_pinned_source_with_pipeline_failure_propagation(
            self):
        script = SCRIPT.with_name('deploy-kfp.sh').read_text()
        install = script.split('  # Include embedded injector hubs,', 1)[1]
        install = install.split('  echo "Waiting for all Istio', 1)[0]
        self.assertIn('set -o pipefail', install)
        self.assertIn('istio-install/base?ref=${KUBEFLOW_MANIFESTS_SHA}',
                      install)
        self.assertLess(
            install.index('kubectl kustomize'),
            install.index('migrate_istio_registry.py'))
        self.assertLess(
            install.index('migrate_istio_registry.py'),
            install.index('kubectl apply -f -'))


if __name__ == '__main__':
    unittest.main()

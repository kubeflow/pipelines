# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Keep the retired execution stack out of this native-only fork."""

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class NativeCleanupTest(unittest.TestCase):
    def test_retired_services_and_clients_are_absent(self):
        for path in (
            'backend',
            'frontend/server',
            'frontend/mock-backend',
            'frontend/src/apisv2',
            'frontend/src/lib/Apis.ts',
            'sdk/python/kfp/client',
            'sdk/python/kfp/server_api',
            'sdk/python/kfp/kubeflow_client',
            'sdk/python/kfp/local',
            'sdk/python/kfp/compiler/pipeline_spec_builder.py',
            'manifests/kustomize',
            'manifests/helm',
            'proxy',
            'go.mod',
            'release',
            'components',
            'test_data',
            '.github/actions',
            'samples/argo-native/install-demo.py',
        ):
            with self.subTest(path=path):
                self.assertFalse((ROOT / path).exists(), path)

    def test_live_native_entrypoints_and_schema_sources_remain(self):
        for path in (
            'sdk/python/kfp/compiler/argo_compiler.py',
            'sdk/python/kfp/dsl/argo_runtime.py',
            'frontend/src/argo.tsx',
            'frontend/src/standalone/Viewer.tsx',
            'api/v2alpha1/pipeline_spec.proto',
            'kubernetes_platform/proto/kubernetes_executor_config.proto',
            'samples/argo-native/deploy.py',
        ):
            with self.subTest(path=path):
                self.assertTrue((ROOT / path).is_file(), path)

    def test_sdk_has_no_kfp_server_client_export(self):
        import kfp

        self.assertFalse(hasattr(kfp, 'Client'))
        from kfp.cli.cli import cli

        self.assertEqual(set(cli.commands), {'component', 'dsl', 'visualize'})

    def test_native_frontend_does_not_import_retired_models(self):
        for path in (ROOT / 'frontend/src').rglob('*.ts*'):
            if path.suffix not in {'.ts', '.tsx'}:
                continue
            with self.subTest(path=str(path)):
                text = path.read_text()
                self.assertNotIn('src/apisv2', text)
                self.assertNotIn('src/generated/pipeline_spec', text)
                self.assertNotIn('src/lib/Apis', text)


if __name__ == '__main__':
    unittest.main()

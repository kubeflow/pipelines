# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0.
"""Old Python imports resolve to canonical v2 models and issue v2 requests."""

import importlib
from pathlib import Path
import re
import unittest
from unittest import mock

from kfp import server_api


class TestLegacyAliases(unittest.TestCase):

    def test_all_legacy_imports_are_canonical_classes(self):
        model_root = Path(server_api.__file__).parent / 'models'
        count = 0
        for module_path in sorted(model_root.glob('v2_*.py')):
            canonical_module = importlib.import_module(
                f'kfp.server_api.models.{module_path.stem}')
            canonical_name = re.search(r'^class (V2\w+)\(',
                                       module_path.read_text(), re.MULTILINE)[1]
            canonical = getattr(canonical_module, canonical_name)
            legacy_name = canonical_name.replace('V2', 'V2beta1', 1)
            legacy_module_name = module_path.stem.replace('v2_', 'v2beta1_', 1)
            with self.subTest(model=canonical_name):
                legacy_module = importlib.import_module(
                    f'kfp.server_api.models.{legacy_module_name}')
                self.assertIs(getattr(legacy_module, legacy_name), canonical)
                self.assertIs(getattr(server_api, legacy_name), canonical)
                self.assertIs(
                    getattr(server_api.models, legacy_name), canonical)
            count += 1
        self.assertGreater(count, 40)

    def test_legacy_model_uses_canonical_endpoint_and_serialization(self):
        from kfp.server_api.models.v2beta1_experiment import V2beta1Experiment

        model = V2beta1Experiment(
            display_name='existing caller', namespace='tenant')
        client = server_api.ApiClient()
        self.assertEqual(
            client.sanitize_for_serialization(model), {
                'display_name': 'existing caller',
                'namespace': 'tenant',
            })
        api = server_api.ExperimentServiceApi(client)
        with mock.patch.object(client, 'call_api', return_value=model) as call:
            self.assertIs(
                api.experiment_service_create_experiment(experiment=model),
                model)
        self.assertEqual(call.call_args.args[:2],
                         ('/apis/v2/experiments', 'POST'))
        self.assertIs(call.call_args.kwargs['body'], model)


if __name__ == '__main__':
    unittest.main()

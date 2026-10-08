# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0.
"""Old Python imports resolve to canonical v2 models and issue v2 requests."""

import importlib
import json
from pathlib import Path
import unittest
from unittest import mock

from kfp import server_api


class TestLegacyAliases(unittest.TestCase):

    def test_all_legacy_imports_are_canonical_classes(self):
        # Independent historical export inventory, not the generator's naming
        # algorithm. Missing or incorrectly renamed aliases must fail here.
        exports = json.loads(
            Path(__file__).with_name('legacy_exports.json').read_text())
        self.assertEqual(len(exports), 50)
        for export in exports:
            with self.subTest(model=export['name']):
                legacy_module = importlib.import_module(export['module'])
                canonical = getattr(legacy_module, export['name'])
                self.assertTrue(
                    canonical.__module__.startswith(
                        'kfp.server_api.models.v2_'))
                self.assertIs(
                    getattr(server_api, canonical.__name__), canonical)
                self.assertIs(getattr(server_api, export['name']), canonical)
                self.assertIs(
                    getattr(server_api.models, export['name']), canonical)

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

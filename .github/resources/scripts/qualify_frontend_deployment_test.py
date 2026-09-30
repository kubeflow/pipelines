# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
"""Browser/cluster-free regression checks for deployment evidence
boundaries."""
import argparse
import copy
from pathlib import Path
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


if __name__ == '__main__':
    unittest.main()

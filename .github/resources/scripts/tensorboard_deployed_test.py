# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
"""Guard deployed acceptance receipts without a Kubernetes cluster."""
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import yaml

ROOT = Path(__file__).resolve().parents[3]


class TensorBoardDeployedTest(unittest.TestCase):

    def test_only_viewer_callers_opt_into_image_loading_and_waiting(self):
        action = yaml.safe_load(
            (ROOT / '.github/actions/deploy/action.yml').read_text())
        self.assertEqual(action['inputs']['load_tensorboard_image']['default'],
                         'false')
        steps = {step.get('name'): step for step in action['runs']['steps']}
        self.assertEqual(
            steps['Wait for branch image artifacts']['env']
            ['LOAD_TENSORBOARD_IMAGE'], '${{ inputs.load_tensorboard_image }}')
        self.assertEqual(
            steps['Load TensorBoard Default Image Into Kind']['if'],
            "${{ inputs.load_tensorboard_image == 'true' }}")
        opted_in = set()
        for path in (ROOT / '.github/workflows').glob('*.y*ml'):
            workflow = yaml.safe_load(path.read_text())
            for job in workflow.get('jobs', {}).values():
                for step in job.get('steps', []):
                    if step.get(
                            'uses') == './.github/actions/deploy' and step.get(
                                'with',
                                {}).get('load_tensorboard_image') == 'true':
                        opted_in.add(path.name)
        self.assertEqual(
            opted_in, {'e2e-test-frontend.yml', 'tensorboard-acceptance.yml'})
        acceptance = yaml.safe_load(
            (ROOT / '.github/workflows/tensorboard-acceptance.yml').read_text())
        self.assertEqual(acceptance['jobs']['build']['uses'],
                         './.github/workflows/image-builds.yml')

    def test_second_storage_failure_preserves_first_case_without_passing(self):
        with tempfile.TemporaryDirectory() as directory:
            with mock.patch.dict(os.environ, {'RUNNER_TEMP': directory}):
                spec = importlib.util.spec_from_file_location(
                    'tensorboard_deployed',
                    ROOT / 'test/tensorboard/deployed.py')
                deployed = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(deployed)
            deployed.STATE.mkdir()
            process = mock.Mock()
            with mock.patch.object(deployed, 'setup'), mock.patch.object(
                    deployed, 'poll'), mock.patch.object(
                        deployed.subprocess, 'Popen',
                        return_value=process), mock.patch.object(
                            deployed.subprocess,
                            'check_output',
                            return_value='source'), mock.patch.object(
                                deployed,
                                'exercise_scheme',
                                side_effect=[{
                                    'scheme': 's3',
                                    'passed': True
                                },
                                             AssertionError('GCS reload failed')
                                            ]):
                with self.assertRaisesRegex(AssertionError,
                                            'GCS reload failed'):
                    deployed.run()
            report = json.loads((deployed.STATE / 'report.json').read_text())
            self.assertFalse(report['passed'])
            self.assertEqual(report['cases'], [{
                'scheme': 's3',
                'passed': True
            }])
            process.terminate.assert_called_once()
            process.wait.assert_called_once_with(timeout=15)


if __name__ == '__main__':
    unittest.main()

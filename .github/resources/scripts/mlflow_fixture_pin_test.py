#!/usr/bin/env python3

# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import unittest
from pathlib import Path

import yaml


class MLflowFixturePinTest(unittest.TestCase):
    def test_public_operator_fixture_uses_immutable_server_and_operator_images(
        self,
    ):
        root = Path(__file__).resolve().parents[3]
        workflow = yaml.safe_load(
            (root / '.github/workflows/e2e-test.yml').read_text()
        )
        deploy = next(
            step
            for job in workflow['jobs'].values()
            for step in job.get('steps', [])
            if step.get('id') == 'deploy-mlflow'
        )
        self.assertRegex(deploy['uses'], r'@[0-9a-f]{40}$')
        for field, image in [
            ('mlflow_image', 'mlflow'),
            ('mlflow_operator_image', 'mlflow-operator'),
        ]:
            expression = deploy['with'][field]
            self.assertRegex(
                expression,
                rf"quay.io/opendatahub/{image}@sha256:[0-9a-f]{{64}}'",
            )
            self.assertNotIn(f'opendatahub/{image}:', expression)
            self.assertIn(
                "github.repository_owner == 'red-hat-data-services'", expression
            )


if __name__ == '__main__':
    unittest.main()

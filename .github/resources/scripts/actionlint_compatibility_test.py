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
"""Keep temporary gh-aw syntax exceptions narrow and validate their values."""

from pathlib import Path
import re
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = '.github/workflows/ai-analyzer.lock.yml'


class ActionlintCompatibilityTest(unittest.TestCase):

    def setUp(self):
        self.config = yaml.safe_load(
            (ROOT / '.github/actionlint.yaml').read_text())
        self.patterns = self.config['paths'][WORKFLOW]['ignore']

    def test_exceptions_only_match_known_diagnostics(self):
        self.assertEqual(len(self.patterns), 4)
        job_context = (
            '{check_run_id: number; container: {id: string; network: string}; '
            'services: {string => {id: string; network: string; ports: '
            '{string => string}}}; status: string}')
        cache_diagnostic = 'unexpected key "cache-mode" for "job" section. expected one of "concurrency", "container", "continue-on-error", "defaults", "env", "environment", "if", "name", "needs", "outputs", "permissions", "runs-on", "secrets", "services", "snapshot", "steps", "strategy", "timeout-minutes", "uses", "with"'
        known = (
            'unexpected key "queue" for "concurrency" section. expected one of '
            '"cancel-in-progress", "group"',
            'unknown permission scope "copilot-requests". all available '
            'permission scopes are "actions", "contents"',
            'property "workflow_ref" is not defined in object type ' +
            job_context,
            cache_diagnostic,
        )
        for diagnostic in known:
            self.assertTrue(
                any(re.search(p, diagnostic) for p in self.patterns))
        for diagnostic in (
                cache_diagnostic.replace("cache-mode", "cache-modes"),
                cache_diagnostic.replace('"job"', '"workflow"'),
                known[0].replace('"queue"', '"queues"'),
                known[0].replace('"concurrency"', '"jobs"'),
                known[1].replace('copilot-requests', 'unknown-scope'),
                known[2].replace('workflow_ref', 'workflow_refs'),
                known[2].replace('check_run_id: number; ', ''),
                'property "admission-result" is not defined in object type '
                '{aw_context: string}',
                'undefined variable "missing". available variables are "github"',
        ):
            self.assertFalse(
                any(re.search(p, diagnostic) for p in self.patterns))

    def test_exceptions_are_not_global_or_shared(self):
        self.assertNotIn('ignore', self.config)
        for path, settings in self.config['paths'].items():
            if path != WORKFLOW:
                self.assertFalse(set(self.patterns) & set(settings['ignore']))

    def test_generated_fields_have_supported_values(self):
        workflow = yaml.safe_load((ROOT / WORKFLOW).read_text())
        self.assertEqual(
            {
                name: job["cache-mode"]
                for name, job in workflow["jobs"].items()
                if "cache-mode" in job
            }, {"agent": "read"})
        queues = []
        permissions = []
        for scope in (workflow, *workflow['jobs'].values()):
            concurrency = scope.get('concurrency', {})
            if isinstance(concurrency, dict) and 'queue' in concurrency:
                queues.append(concurrency['queue'])
            permission = scope.get('permissions', {})
            if (isinstance(permission, dict) and
                    'copilot-requests' in permission):
                permissions.append(permission['copilot-requests'])
        self.assertTrue(queues)
        self.assertEqual(set(queues), {'max'})
        self.assertTrue(permissions)
        self.assertEqual(set(permissions), {'write'})


if __name__ == '__main__':
    unittest.main()

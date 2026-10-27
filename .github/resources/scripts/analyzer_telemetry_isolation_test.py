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
"""Keep collector credentials on the trusted side of the analyzer sandbox."""

import re
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/ai-analyzer.lock.yml'


class AnalyzerTelemetryIsolationTest(unittest.TestCase):
    def test_every_agent_sandbox_excludes_telemetry_credentials(self):
        workflow = yaml.safe_load(WORKFLOW.read_text())
        sandboxes = []
        for job_name, job in workflow['jobs'].items():
            for step in job.get('steps', []):
                for line in step.get('run', '').splitlines():
                    if not line.strip().startswith('awf '):
                        continue
                    sandboxes.append(job_name)
                    with self.subTest(job=job_name, step=step['name']):
                        excluded = set(re.findall(r'--exclude-env (\S+)', line))
                        self.assertTrue(
                            {
                                'GH_AW_OTLP_ENDPOINTS',
                                'OTEL_EXPORTER_OTLP_HEADERS',
                                'OTEL_EXPORTER_OTLP_ENDPOINT',
                            }
                            <= excluded
                        )
        self.assertCountEqual(sandboxes, ['agent', 'detection'])

    def test_trusted_host_retains_configured_telemetry(self):
        workflow = yaml.safe_load(WORKFLOW.read_text())
        env = workflow['env']
        self.assertIn(
            'secrets.GH_AW_DEFAULT_OTLP_HEADERS', env['GH_AW_OTLP_ENDPOINTS']
        )
        self.assertIn(
            'secrets.GH_AW_DEFAULT_OTLP_HEADERS',
            env['OTEL_EXPORTER_OTLP_HEADERS'],
        )
        self.assertIn(
            'GH_AW_DEFAULT_OTLP_ENDPOINT', env['OTEL_EXPORTER_OTLP_ENDPOINT']
        )


if __name__ == '__main__':
    unittest.main()

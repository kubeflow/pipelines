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
"""Keep the analyzer lockfile's runtime action on its compiler's version.

The gh-aw setup action only trusts the tool releases (firewall, Copilot CLI)
that the matching compiler pins into the lockfile. Bumping the setup action
without recompiling leaves tool versions it has no reviewed checksums for, and
every analyzer run fails before the agent starts.
"""

import json
import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/ai-analyzer.lock.yml'
SETUP_USE = re.compile(
    r'uses: github/gh-aw-actions/setup@[0-9a-f]{40} # (v\d+\.\d+\.\d+)'
)


class AnalyzerLockVersionsTest(unittest.TestCase):
    def test_setup_action_matches_compiler_version(self):
        text = WORKFLOW.read_text()
        metadata_line = text.splitlines()[0]
        self.assertTrue(metadata_line.startswith('# gh-aw-metadata: '))
        metadata = json.loads(metadata_line.removeprefix('# gh-aw-metadata: '))
        compiler_version = metadata['compiler_version']

        setup_versions = set(SETUP_USE.findall(text))
        self.assertTrue(setup_versions, 'no gh-aw setup action found')
        self.assertEqual(
            setup_versions,
            {compiler_version},
            'gh-aw-actions/setup was bumped without recompiling: run '
            f'`gh aw compile .github/workflows/ai-analyzer.md` with gh-aw '
            f'{sorted(setup_versions)} instead of {compiler_version}',
        )


if __name__ == '__main__':
    unittest.main()

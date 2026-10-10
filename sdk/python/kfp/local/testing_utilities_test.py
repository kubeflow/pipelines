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
"""Regression coverage for local-runner test environment isolation."""

from importlib import metadata
import json
from pathlib import Path
import sys
import unittest

from kfp import dsl
from kfp import local
from kfp.local import testing_utilities


class TestLocalRunnerIsolation(testing_utilities.LocalRunnerEnvironmentTestCase
                              ):

    def test_no_venv_installs_outside_the_worker(self) -> None:
        """Exercise real SDK installs without replacing the worker's SDK."""
        original_source = metadata.distribution('kfp').read_text(
            'direct_url.json')
        local.init(local.SubprocessRunner(use_venv=False))

        @dsl.component
        def installation_paths() -> str:
            from importlib import metadata
            import json
            import sys

            return json.dumps({
                'prefix':
                    sys.prefix,
                'package_root':
                    str(metadata.distribution('kfp').locate_file('')),
            })

        first = json.loads(installation_paths().output)
        second = json.loads(installation_paths().output)

        self.assertNotEqual(first['prefix'], sys.prefix)
        self.assertEqual(first, second)
        prefix = Path(first['prefix']).resolve()
        self.assertTrue(
            prefix.is_relative_to(Path(self.isolated_kfp_dir).resolve()))
        self.assertTrue(
            Path(first['package_root']).resolve().is_relative_to(prefix))
        self.assertEqual(
            metadata.distribution('kfp').read_text('direct_url.json'),
            original_source)


if __name__ == '__main__':
    unittest.main()

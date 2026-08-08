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
"""Tests for import order robustness of kfp.dsl.graph_component."""

import os
import subprocess
import sys
import unittest


class GraphComponentImportTest(unittest.TestCase):
    """Regression tests for https://github.com/kubeflow/pipelines/issues/12522.

    Under ``_KFP_RUNTIME=true``, importing ``kfp.dsl.graph_component`` or
    ``kfp.dsl.pipeline_context`` used to raise ``AttributeError: partially
    initialized module 'kfp.dsl.pipeline_context' has no attribute 'Pipeline'``
    because of a circular import between the kfp.dsl modules and the kfp.compiler
    package.
    """

    def _run_in_subprocess(self, code: str) -> None:
        env = os.environ.copy()
        env['_KFP_RUNTIME'] = 'true'
        result = subprocess.run(
            [sys.executable, '-c', code],
            env=env,
            capture_output=True,
            text=True,
        )
        self.assertEqual(
            result.returncode,
            0,
            msg=f'Import failed.\nstdout:\n{result.stdout}\nstderr:\n{result.stderr}',
        )

    def test_import_graph_component_in_runtime_mode(self) -> None:
        self._run_in_subprocess('import kfp.dsl.graph_component\n'
                                'assert kfp.dsl.graph_component.GraphComponent')

    def test_import_pipeline_context_in_runtime_mode(self) -> None:
        self._run_in_subprocess('import kfp.dsl.pipeline_context\n'
                                'assert kfp.dsl.pipeline_context.Pipeline')

    def test_import_pipeline_context_inside_function_in_runtime_mode(
            self) -> None:
        self._run_in_subprocess('def task():\n'
                                '    from kfp.dsl import pipeline_context\n'
                                '    return pipeline_context.Pipeline\n'
                                'task()')

    def test_import_dsl_then_graph_component_in_runtime_mode(self) -> None:
        self._run_in_subprocess('import kfp.dsl\n'
                                'import kfp.dsl.graph_component\n'
                                'assert kfp.dsl.graph_component.GraphComponent')


if __name__ == '__main__':
    unittest.main()

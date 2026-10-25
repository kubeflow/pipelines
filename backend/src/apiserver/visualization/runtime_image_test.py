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

import unittest

import exporter
from nbformat.v4 import new_notebook


class RuntimeImageTest(unittest.TestCase):

    def test_notebook_execution_and_html_export(self):
        notebook = new_notebook()
        notebook.cells.append(
            exporter.create_cell_from_custom_code([
                "from PIL import Image",
                "import pyarrow as pa",
                "assert Image.new('RGB', (1, 1)).size == (1, 1)",
                "assert pa.array([42]).to_pylist() == [42]",
                "print('runtime-image-result-' + str(6 * 7))",
            ]))
        html = exporter.Exporter(
            100,
            exporter.TemplateType.BASIC).generate_html_from_notebook(notebook)
        self.assertIn('runtime-image-result-42', html)


if __name__ == '__main__':
    unittest.main()

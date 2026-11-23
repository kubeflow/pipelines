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
"""Final-image smoke test without the snapshottest dependency."""

import unittest

import exporter
from nbconvert.filters.markdown import markdown2html_mistune
from nbformat.v4 import new_notebook


class RuntimeImageTest(unittest.TestCase):
    """Verify notebook execution with the installed runtime dependencies."""

    def test_nbconvert_markdown_links_and_images_with_patched_parser(
            self) -> None:
        html = markdown2html_mistune(
            '# Markdown compatibility\n\n'
            '[documentation](https://example.test/docs)\n\n'
            '![diagram](https://example.test/diagram.png)\n\n' + '[](' * 64 +
            '\n\n' + '![' * 64 + 'text' + ']' * 64)
        self.assertIn('Markdown compatibility', html)
        self.assertIn('href="https://example.test/docs"', html)
        self.assertIn('src="https://example.test/diagram.png"', html)

    def test_notebook_execution_and_html_export(self) -> None:
        """Execute a kernel and export its output after bootstrap removal."""
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

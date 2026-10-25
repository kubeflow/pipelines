# Copyright 2019 The Kubeflow Authors
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

import base64
import importlib
import struct
import unittest

from nbformat.v4 import new_notebook
import snapshottest

exporter = importlib.import_module("exporter")


class TestExporterMethods(snapshottest.TestCase):

    def setUp(self):
        self.maxDiff = None
        self.exporter = exporter.Exporter(100, exporter.TemplateType.BASIC)

    def test_create_cell_from_args_with_no_args(self):
        args = {}
        cell = exporter.create_cell_from_args(args)
        self.assertMatchSnapshot(cell.source)

    def test_create_cell_from_args_with_one_arg(self):
        args = {"source": "gs://ml-pipeline/data.csv"}
        cell = exporter.create_cell_from_args(args)
        self.assertMatchSnapshot(cell.source)

    # Test generates html to avoid issues with Python 3.5 where dict objects
    # do not retain order upon object creation. Due to this, we test that the
    # provided arguments exist and equal the provided value.
    def test_create_cell_from_args_with_multiple_args(self):
        nb = new_notebook()
        args = {
            "source": "gs://ml-pipeline/data.csv",
            "target_lambda": "lambda x: (x['target'] > x['fare'] * 0.2)"
        }
        code = [
            "print(variables.get('source'))",
            "print(variables.get('target_lambda'))"
        ]
        nb.cells.append(exporter.create_cell_from_args(args))
        nb.cells.append(exporter.create_cell_from_custom_code(code))
        html = self.exporter.generate_html_from_notebook(nb)
        self.assertMatchSnapshot(html)

    def test_create_cell_from_file(self):
        cell = exporter.create_cell_from_file("types/test.py")
        self.assertMatchSnapshot(cell.source)

    def test_create_cell_from_custom_code(self):
        code = ["x = 2", "print(x)"]
        cell = exporter.create_cell_from_custom_code(code)
        self.assertMatchSnapshot(cell.source)

    # Tests to ensure output is generated for custom visualizations.
    def test_generate_custom_visualization_html_from_notebook(self):
        nb = new_notebook()
        args = {"x": 2}
        code = ["print(variables.get('x'))"]
        nb.cells.append(exporter.create_cell_from_args(args))
        nb.cells.append(exporter.create_cell_from_custom_code(code))
        html = self.exporter.generate_html_from_notebook(nb)
        self.assertMatchSnapshot(html)

    def test_custom_arrow_table_preserves_values_in_html(self):
        nb = new_notebook()
        nb.cells.append(
            exporter.create_cell_from_custom_code([
                "import pandas as pd",
                "import pyarrow as pa",
                "import pyarrow.parquet as pq",
                "from IPython.display import display",
                "original = pd.DataFrame({'label': ['positive', 'missing'], "
                "'score': [0.75, None]})",
                "buffer = pa.BufferOutputStream()",
                "pq.write_table(pa.Table.from_pandas(original), buffer)",
                "restored = pq.read_table(pa.BufferReader(buffer.getvalue())).to_pandas()",
                "pd.testing.assert_frame_equal(restored, original)",
                "display(restored)",
            ]))

        html = self.exporter.generate_html_from_notebook(nb)

        outputs = nb.cells[0].outputs
        self.assertFalse(
            any(output.output_type == 'error' for output in outputs))
        tables = [
            output.get('data', {}).get('text/html', '') for output in outputs
        ]
        self.assertTrue(any('<table' in table for table in tables))
        for value in ('positive', 'missing', '0.75', 'NaN'):
            self.assertIn(value, ''.join(tables))
            self.assertIn(value, html)

    def test_custom_pillow_image_is_embedded_as_png(self):
        nb = new_notebook()
        nb.cells.append(
            exporter.create_cell_from_custom_code([
                "from PIL import Image",
                "from IPython.display import display",
                "display(Image.new('RGB', (2, 3), color=(255, 0, 0)))",
            ]))

        html = self.exporter.generate_html_from_notebook(nb)

        outputs = nb.cells[0].outputs
        self.assertFalse(
            any(output.output_type == 'error' for output in outputs))
        images = [
            output['data']['image/png']
            for output in outputs
            if 'image/png' in output.get('data', {})
        ]
        self.assertEqual(len(images), 1)
        png = base64.b64decode(images[0])
        self.assertEqual(png[:8], b'\x89PNG\r\n\x1a\n')
        self.assertEqual(struct.unpack('>II', png[16:24]), (2, 3))
        self.assertIn('data:image/png;base64,', html)


if __name__ == "__main__":
    unittest.main()

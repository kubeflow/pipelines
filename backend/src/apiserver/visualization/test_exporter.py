# Copyright 2019 Google LLC
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

import importlib
import unittest
from nbformat.v4 import new_code_cell
from nbformat.v4 import new_notebook
import snapshottest

exporter = importlib.import_module("exporter")


class TestExporterMethods(snapshottest.TestCase):

    def setUp(self):
        self.exporter = exporter.Exporter(100, exporter.TemplateType.BASIC)

    def test_create_cell_from_args_with_no_args(self):
        self.maxDiff = None
        nb = new_notebook()
        args = {}
        nb.cells.append(self.exporter.create_cell_from_args(args))
        nb.cells.append(new_code_cell("print(variables)"))
        html = self.exporter.generate_html_from_notebook(nb)
        self.assertMatchSnapshot(html)

    def test_create_cell_from_args_with_one_arg(self):
        self.maxDiff = None
        nb = new_notebook()
        args = {"source": "gs://ml-pipeline/data.csv"}
        nb.cells.append(self.exporter.create_cell_from_args(args))
        nb.cells.append(new_code_cell("print([variables[key] for key in sorted(variables.keys())])"))
        html = self.exporter.generate_html_from_notebook(nb)
        self.assertMatchSnapshot(html)

    def test_create_cell_from_args_with_multiple_args(self):
        self.maxDiff = None
        nb = new_notebook()
        args = {
            "source": "gs://ml-pipeline/data.csv",
            "target_lambda": "lambda x: (x['target'] > x['fare'] * 0.2)"
        }
        nb.cells.append(self.exporter.create_cell_from_args(args))
        nb.cells.append(new_code_cell("print([variables[key] for key in sorted(variables.keys())])"))
        html = self.exporter.generate_html_from_notebook(nb)
        self.assertMatchSnapshot(html)

    def test_create_cell_from_file(self):
        self.maxDiff = None
        cell = self.exporter.create_cell_from_file("test.py")
        self.assertMatchSnapshot(cell.source)

    def test_generate_html_from_notebook(self):
        self.maxDiff = None
        nb = new_notebook()
        args = {"x": 2}
        nb.cells.append(self.exporter.create_cell_from_args(args))
        nb.cells.append(new_code_cell("print(variables['x'])"))
        html = self.exporter.generate_html_from_notebook(nb)
        self.assertMatchSnapshot(html)


if __name__ == "__main__":
    unittest.main()

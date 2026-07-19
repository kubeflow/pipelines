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

import importlib
from typing import Text
import unittest
import tornado.testing
import tornado.web
from nbformat.v4 import new_code_cell

server = importlib.import_module("server")


def wrap_error_in_html(error: Text) -> bytes:
    html = "<html><title>{}</title><body>{}</body></html>".format(error, error)
    return bytes(html, "utf-8")


class TestServerEndpoints(tornado.testing.AsyncHTTPTestCase):
    def get_app(self):
        return tornado.web.Application([
            (r"/", server.VisualizationHandler),
        ])

    def test_healthcheck(self):
        response = self.fetch("/")
        self.assertEqual(200, response.code)
        self.assertEqual(b"alive", response.body)

    def test_create_visualization_fails_when_nothing_is_provided(self):
        response = self.fetch(
            "/",
            method="POST",
            body="")
        self.assertEqual(400, response.code)
        self.assertEqual(
            wrap_error_in_html("400: No type provided."),
            response.body
        )

    def test_create_visualization_fails_when_missing_type(self):
        response = self.fetch(
            "/",
            method="POST",
            body="source=gs://ml-pipeline/data.csv")
        self.assertEqual(400, response.code)
        self.assertEqual(
            wrap_error_in_html("400: No type provided."),
            response.body
        )

    def test_create_visualization_fails_when_missing_source(self):
        response = self.fetch(
            "/",
            method="POST",
            body='type=test')
        self.assertEqual(400, response.code)
        self.assertEqual(
            wrap_error_in_html("400: No source provided."),
            response.body
        )

    def test_create_visualization_passes_when_missing_source_and_type_is_custom(self):
        response = self.fetch(
            "/",
            method="POST",
            body='type=custom')
        self.assertEqual(200, response.code)

    def test_create_visualization_fails_when_invalid_json_is_provided(self):
        response = self.fetch(
            "/",
            method="POST",
            body='type=test&source=gs://ml-pipeline/data.csv&arguments={')
        self.assertEqual(400, response.code)
        self.assertEqual(
            wrap_error_in_html("400: Invalid JSON provided as arguments: Expecting property name enclosed in double quotes: line 1 column 2 (char 1)"),
            response.body
        )

    def test_create_visualization(self):
        response = self.fetch(
            "/",
            method="POST",
            body='type=test&source=gs://ml-pipeline/data.csv')
        self.assertEqual(200, response.code)


class TestSourceInjection(unittest.TestCase):
    """Regression test for the `source` string-literal injection sink.

    `server.py` builds a notebook cell that assigns the request `source`
    parameter to a Python variable. The cell must use repr() (or another
    escaping mechanism) so a `source` value containing a double-quote cannot
    break out of the string literal and inject arbitrary statements.
    """

    # The exact cell construction performed in VisualizationHandler.generate_notebook.
    # Kept in sync with server.py; if the construction changes, update both.
    @staticmethod
    def _source_cell(source: str):
        return new_code_cell('source = {}'.format(repr(source)))

    def test_malicious_source_does_not_break_out_of_literal(self):
        malicious = 'x";__import__("os").system("echo PWNED");x="'
        cell = self._source_cell(malicious)
        # The cell must compile as a single statement (an assignment), not as
        # multiple statements — i.e. the injected code must NOT be reachable.
        import ast
        tree = ast.parse(cell.source)
        self.assertEqual(
            1, len(tree.body),
            "source cell must be a single statement; injection broke out: "
            + cell.source,
        )
        self.assertIsInstance(tree.body[0], ast.Assign)
        # And executing it must leave `source` equal to the raw input — proving
        # the payload became inert string data, not executed code.
        namespace = {}
        exec(cell.source, namespace)
        self.assertEqual(malicious, namespace["source"])

    def test_legit_source_round_trips(self):
        legit = "gs://ml-pipeline/data.csv"
        cell = self._source_cell(legit)
        namespace = {}
        exec(cell.source, namespace)
        self.assertEqual(legit, namespace["source"])


if __name__ == "__main__":
    unittest.main()

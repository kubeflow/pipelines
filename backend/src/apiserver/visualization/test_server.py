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

import ast
import importlib
import re
from typing import Text
import unittest
from unittest import mock

import tornado.httputil
import tornado.testing
import tornado.web

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
        response = self.fetch("/", method="POST", body="")
        self.assertEqual(400, response.code)
        self.assertEqual(
            wrap_error_in_html("400: No type provided."), response.body)

    def test_create_visualization_fails_when_missing_type(self):
        response = self.fetch(
            "/", method="POST", body="source=gs://ml-pipeline/data.csv")
        self.assertEqual(400, response.code)
        self.assertEqual(
            wrap_error_in_html("400: No type provided."), response.body)

    def test_create_visualization_fails_when_missing_source(self):
        response = self.fetch("/", method="POST", body='type=test')
        self.assertEqual(400, response.code)
        self.assertEqual(
            wrap_error_in_html("400: No source provided."), response.body)

    def test_create_visualization_passes_when_missing_source_and_type_is_custom(
            self):
        response = self.fetch("/", method="POST", body='type=custom')
        self.assertEqual(200, response.code)

    def test_create_visualization_fails_when_invalid_json_is_provided(self):
        response = self.fetch(
            "/",
            method="POST",
            body='type=test&source=gs://ml-pipeline/data.csv&arguments={')
        self.assertEqual(400, response.code)
        self.assertEqual(
            wrap_error_in_html(
                "400: Invalid JSON provided as arguments: Expecting property name enclosed in double quotes: line 1 column 2 (char 1)"
            ), response.body)

    def test_create_visualization(self):
        response = self.fetch(
            "/",
            method="POST",
            body='type=test&source=gs://ml-pipeline/data.csv')
        self.assertEqual(200, response.code)


class TestSourceInjection(unittest.TestCase):
    """Regression test for the `source` string-literal injection sink.

    `generate_notebook_from_arguments` builds a notebook cell that assigns the
    request `source` parameter to a Python variable. That cell must escape the
    value (repr()) so a `source` containing a double-quote cannot terminate the
    literal and have the remainder parsed as further statements, which the
    kernel would then execute.

    These tests drive the production builder, not a copy of it: reverting the
    escaping in server.py makes them fail.
    """

    PAYLOAD = 'x";injected_marker = 1;x="'

    def setUp(self):
        connection = mock.Mock(spec=["set_close_callback"])
        request = tornado.httputil.HTTPServerRequest(
            method="POST", uri="/", connection=connection)
        self.handler = server.VisualizationHandler(tornado.web.Application(),
                                                   request)

    def generated_source_cell(self, source: Text, visualization_type: Text):
        """Returns the `source = ...` cell the production builder generated."""
        nb = self.handler.generate_notebook_from_arguments({"code": ["pass"]},
                                                           source,
                                                           visualization_type)
        cells = [
            cell for cell in nb.cells if re.match(r"^source\s*=", cell.source)
        ]
        self.assertEqual(1, len(cells),
                         "expected exactly one source assignment cell")
        return cells[0].source

    def assert_source_is_inert_data(self, source: Text,
                                    visualization_type: Text):
        cell = self.generated_source_cell(source, visualization_type)
        statements = ast.parse(cell).body
        self.assertEqual(
            1, len(statements),
            "source cell must be a single statement; injection broke out of "
            "the literal: " + cell)
        self.assertIsInstance(statements[0], ast.Assign)
        namespace = {}
        exec(cell, namespace)
        # The value round-trips exactly, and nothing in it ran.
        self.assertEqual(source, namespace["source"])
        self.assertNotIn("injected_marker", namespace)

    def test_payload_is_inert_for_custom_visualization(self):
        self.assert_source_is_inert_data(self.PAYLOAD, "custom")

    def test_payload_is_inert_for_predefined_visualization(self):
        # "test" is a predefined type, so the notebook also loads types/test.py.
        self.assert_source_is_inert_data(self.PAYLOAD, "test")

    def test_quotes_and_escapes_are_inert(self):
        for source in [
                'a"b',
                "a'b",
                'a\\";injected_marker = 1;x="',
                'a\n injected_marker = 1',
                'a\\x27;injected_marker = 1;x=\\x27',
                '"""\ninjected_marker = 1\n"""',
        ]:
            with self.subTest(source=source):
                self.assert_source_is_inert_data(source, "custom")

    def test_object_store_paths_round_trip(self):
        for source in [
                "gs://ml-pipeline/data.csv",
                "s3://bucket/prefix/*.csv",
                "minio://ml-pipeline/data with spaces.csv",
        ]:
            with self.subTest(source=source):
                self.assert_source_is_inert_data(source, "test")


if __name__ == "__main__":
    unittest.main()

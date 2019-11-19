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
from typing import Text
import unittest
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


if __name__ == "__main__":
    unittest.main()

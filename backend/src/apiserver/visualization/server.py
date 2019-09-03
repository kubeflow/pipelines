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

import argparse
import importlib
import json
import os
from pathlib import Path
from typing import Text

from nbformat import NotebookNode
from nbformat.v4 import new_notebook, new_code_cell
import tornado.ioloop
import tornado.web

exporter = importlib.import_module("exporter")

parser = argparse.ArgumentParser(description="Server Arguments")
parser.add_argument(
    "--timeout",
    type=int,
    default=os.getenv('KERNEL_TIMEOUT', 100),
    help="Amount of time in seconds that a visualization can run for before " +
         "being stopped."
)

args = parser.parse_args()
_exporter = exporter.Exporter(args.timeout)


class VisualizationHandler(tornado.web.RequestHandler):
    """Custom RequestHandler that generates visualizations via post requests.
    """

    def validate_and_get_arguments_from_body(self) -> dict:
        """Validates and converts arguments from post request to dict.

        Returns:
            Arguments provided from post request as a dict.
        """
        try:
            arguments = {
                "arguments": "{}",
                "type": self.get_body_argument("type")
            }
        except tornado.web.MissingArgumentError:
            raise Exception("No type provided.")

        try:
            arguments["arguments"] = self.get_body_argument("arguments")
        except tornado.web.MissingArgumentError:
            # If no arguments are provided, ignore error as arguments has been
            # set to a stringified JSON object by default.
            pass

        try:
            arguments["arguments"] = json.loads(arguments.get("arguments"))
        except json.decoder.JSONDecodeError as e:
            raise Exception("Invalid JSON provided as arguments: {}".format(str(e)))

        # If invalid JSON is provided that is incorretly escaped
        # arguments.get("arguments") can be a string. This Ensure that
        # json.loads properly converts stringified JSON to dict.
        if type(arguments.get("arguments")) != dict:
            raise Exception("Invalid JSON provided as arguments!")

        try:
            arguments["source"] = self.get_body_argument("source")
        except tornado.web.MissingArgumentError:
            arguments["source"] = ""

        if arguments.get("type") != "custom":
            if len(arguments.get("source")) == 0:
                raise Exception("No source provided.")

        return arguments

    def generate_notebook_from_arguments(
        self,
        arguments: dict,
        source: Text,
        visualization_type: Text
    ) -> NotebookNode:
        """Generates a NotebookNode from provided arguments.

        Args:
            arguments: JSON object containing provided arguments.
            source: Path or path pattern to be used as data reference for
            visualization.
            visualization_type: Name of visualization to be generated.

        Returns:
                NotebookNode that contains all parameters from a post request.
        """
        nb = new_notebook()
        nb.cells.append(exporter.create_cell_from_args(arguments))
        nb.cells.append(new_code_cell('source = "{}"'.format(source)))
        if visualization_type == "custom":
            code = arguments.get("code", [])
            nb.cells.append(exporter.create_cell_from_custom_code(code))
        else:
            visualization_file = str(Path.cwd() / "types/{}.py".format(visualization_type))
            nb.cells.append(exporter.create_cell_from_file(visualization_file))
        
        return nb

    def get(self):
        """Health check.
        """
        self.write("alive")

    def post(self):
        """Generates visualization based on provided arguments.
        """
        # Validate arguments from request and return them as a dictionary.
        try:
            request_arguments = self.validate_and_get_arguments_from_body()
        except Exception as e:
            return self.send_error(400, reason=str(e))

        # Create notebook with arguments from request.
        nb = self.generate_notebook_from_arguments(
            request_arguments.get("arguments"),
            request_arguments.get("source"),
            request_arguments.get("type")
        )

        # Generate visualization (output for notebook).
        html = _exporter.generate_html_from_notebook(nb)
        self.write(html)


if __name__ == "__main__":
    application = tornado.web.Application([
        (r"/", VisualizationHandler),
    ])
    application.listen(8888)
    tornado.ioloop.IOLoop.current().start()

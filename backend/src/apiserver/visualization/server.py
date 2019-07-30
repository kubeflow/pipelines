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
from pathlib import Path
import shlex
from nbformat.v4 import new_notebook, new_code_cell
import tornado.ioloop
import tornado.web
import exporter


# All necessary arguments required to generate visualizations.
parser = argparse.ArgumentParser(description='Visualization Generator')
# Type of visualization to be generated.
parser.add_argument('--type', type=str,
                    help='Type of visualization to be generated.')
# Path of data to be used to generate visualization.
parser.add_argument('--input_path', type=str,
                    help='Path of data to be used for generating ' +
                         'visualization.')
# Additional arguments to be used when generating a visualization (provided as a
# string representation of JSON).
parser.add_argument('--arguments', type=str, default='{}',
                    help='JSON string of arguments to be provided to ' +
                         'visualizations.')


class VisualizationHandler(tornado.web.RequestHandler):
    def get(self):
        self.write("alive")

    def post(self):
        # Parse arguments from request.
        args = parser.parse_args(shlex.split(self.get_body_argument("arguments")))
        # Validate arguments from request.
        if args.type is None:
            return self.send_error(400, reason="No type specified.")
        if args.input_path is None:
            return self.send_error(400, reason="No input_path specified.")
        # Create notebook with arguments from request.
        nb = new_notebook()
        nb.cells.append(exporter.create_cell_from_args(args.arguments))
        nb.cells.append(new_code_cell(f'input_path = "{args.input_path}"'))
        visualization_file = str(Path.cwd() / f"{args.type}.py")
        nb.cells.append(exporter.create_cell_from_file(visualization_file))
        # Generate visualization (output for notebook).
        html = exporter.generate_html_from_notebook(nb)
        self.write(html)


if __name__ == "__main__":
    application = tornado.web.Application([
        (r"/", VisualizationHandler),
    ])
    application.listen(8888)
    tornado.ioloop.IOLoop.current().start()

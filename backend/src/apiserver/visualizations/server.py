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
import shlex

import tornado.ioloop
import tornado.web
from .exporter import *
from nbformat.v4 import new_notebook


dirname = os.path.dirname(__file__)

parser = argparse.ArgumentParser(description='Visualization Generator')
parser.add_argument('--type', type=str, default='roc',
                    help='Type of visualization to be generated.')
parser.add_argument('--arguments', type=str, default='{}',
                    help='JSON string of arguments to be provided to ' +
                         'visualizations.')


class VisualizationHandler(tornado.web.RequestHandler):
    def get(self):
        self.write("alive")

    def post(self):
        args = parser.parse_args(shlex.split(
            self.get_body_argument("arguments")))
        nb = new_notebook()
        nb.cells.append(create_cell_from_args(args.arguments))
        nb.cells.append(create_cell_from_file(
            os.path.join(dirname, '{}.py'.format(args.type))))
        html = generate_html_from_notebook(nb)
        self.write(html)


if __name__ == "__main__":
    application = tornado.web.Application([
        (r"/", VisualizationHandler),
    ])
    application.listen(8888)
    tornado.ioloop.IOLoop.current().start()

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

import os
from nbconvert import HTMLExporter
from nbconvert.preprocessors import ExecutePreprocessor
from nbformat.v4 import new_notebook, new_code_cell


# Reads a python file, then creates a Notebook object with the
# lines of code from the python file.
#
# Returns the generated Notebook
def code_to_notebook(filepath):
    nb = new_notebook()
    with open(filepath, 'r') as f:
        code = f.read()

    nb.cells.append(new_code_cell(code))
    return nb


# Exports a notebook to HTML and generates any required outputs.
#
# Returns the generated HTML as a string
def generate_html_from_notebook(nb, template_file='full'):
    # HTML generator and exporter object
    html_exporter = HTMLExporter()
    html_exporter.template_file = template_file

    # Output generator object
    ep = ExecutePreprocessor(timeout=300, kernel_name='python')
    ep.preprocess(nb, {'metadata': {'path': os.getcwd()}})

    # Export all html and outputs
    (body, _) = html_exporter.from_notebook_node(nb)
    return body

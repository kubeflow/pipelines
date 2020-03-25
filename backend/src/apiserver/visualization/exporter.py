"""
exporter.py provides utility functions for generating NotebookNode objects and
converting those objects to HTML.
"""

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

from enum import Enum
from pathlib import Path
from typing import Text
from jupyter_client import KernelManager
from nbconvert import HTMLExporter
from nbconvert.preprocessors import ExecutePreprocessor
from nbformat import NotebookNode
from nbformat.v4 import new_code_cell


# Visualization Template types:
# - Basic: Uses the basic.tpl file within the templates directory to generate
# a visualization that contains no styling and minimal HTML tags. This is ideal
# for testing as it reduces size of generated visualization. However, for usage
# with actual visualizations it is not ideal due to its lack of javascript and
# styling which can limit usability of a visualization.
# - Full: Uses the full.tpl file within the template directory to generate a
# visualization that can be viewed as a standalone web page. The full.tpl file
# utilizes the basic.tpl file for visualizations then wraps that output with
# additional tags for javascript and style support. This is ideal for generating
# visualizations that will be displayed via the frontend.
class TemplateType(Enum):
    BASIC = 'basic'
    FULL = 'full'


def create_cell_from_args(variables: dict) -> NotebookNode:
    """Creates NotebookNode object containing dict of provided variables.

    Args:
        variables: Arguments that need to be injected into a NotebookNode.

    Returns:
        NotebookNode with provided arguments as variables.

    """
    return new_code_cell("variables = {}".format(repr(variables)))


def create_cell_from_file(filepath: Text) -> NotebookNode:
    """Creates a NotebookNode object with provided file as code in node.

    Args:
        filepath: Path to file that should be used.

    Returns:
        NotebookNode with specified file as code within node.

    """
    with open(filepath, 'r') as f:
        code = f.read()	

    return new_code_cell(code)


def create_cell_from_custom_code(code: list) -> NotebookNode:
    """Creates a NotebookNode object with provided list as code in node.

    Args:
        code: list representing lines of code to be run.

    Returns:
        NotebookNode with specified file as code within node.

    """
    cell = new_code_cell("\n".join(code))
    cell.get("metadata")["hide_logging"] = False
    return cell


class Exporter:
    """Handler for interaction with NotebookNodes, including output generation.

    Attributes:
        timeout (int): Amount of time in seconds that a visualization can run
        for before being stopped.
        template_type (TemplateType): Type of template to use when generating
        visualization output.
        km (KernelManager): Custom KernelManager that stays alive between
        visualizations.
        ep (ExecutePreprocessor): Process that is responsible for generating
        outputs from NotebookNodes.

    """

    def __init__(
        self,
        timeout: int = 100,
        template_type: TemplateType = TemplateType.FULL
    ):
        """
        Initializes Exporter with default timeout (100 seconds) and template
        (FULL) and handles instantiation of km and ep variables for usage when
        generating NotebookNodes and their outputs.

        Args:
            timeout (int): Amount of time in seconds that a visualization can
            run for before being stopped.
            template_type (TemplateType): Type of template to use when
            generating visualization output.
        """
        self.timeout = timeout
        self.template_type = template_type
        # Create custom KernelManager.
        # This will circumvent issues where kernel is shutdown after
        # preprocessing. Due to the shutdown, latency would be introduced
        # because a kernel must be started per visualization.
        self.km = KernelManager()
        self.km.start_kernel()
        self.ep = ExecutePreprocessor(
            timeout=self.timeout,
            kernel_name='python3',
            allow_errors=True
        )

    def generate_html_from_notebook(self, nb: NotebookNode) -> Text:
        """Converts a provided NotebookNode to HTML.

        Args:
            nb: NotebookNode that should be converted to HTML.

        Returns:
            HTML from converted NotebookNode as a string.

        """
        # HTML generator and exporter object
        html_exporter = HTMLExporter()
        template_file = "templates/{}.tpl".format(self.template_type.value)
        html_exporter.template_file = str(Path.cwd() / template_file)
        # Output generator
        self.ep.preprocess(nb, {"metadata": {"path": Path.cwd()}}, self.km)
        # Export all html and outputs
        body, _ = html_exporter.from_notebook_node(nb, resources={})
        return body

# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import inspect
import os
import re
import sys
import textwrap
from typing import List

import commonmark
import docstring_parser
from google_cloud_pipeline_components import utils
from kfp import components
from kfp import dsl
import yaml

# setting this enables the .rst files to use the paths v1.bigquery.Component (etc.) rather than google_cloud_pipeline_components.v1.biquery.Component for shorter, readable representation in docs
gcpc_root_dir = os.path.abspath(
    os.path.join(
        os.path.dirname(__file__),
        '..',
        '..',
        'google_cloud_pipeline_components',
    )
)

# keep as append not insert, otherwise there is an issue with other package discovery
sys.path.append(gcpc_root_dir)

# preserve function docstrings for components by setting component decorators to passthrough decorators
# also enables autodoc to document the components as functions without using the autodata directive (https://www.sphinx-doc.org/en/master/usage/extensions/autodoc.html#directive-autodata)
def first_order_passthrough_decorator(func):
  func._is_component = True
  return func


def second_order_passthrough_decorator(*args, **kwargs):

  def decorator(func):
    func._is_component = True
    return func

  return decorator


def second_order_passthrough_decorator_for_pipeline(*args, **kwargs):
  def decorator(func):
    func._is_pipeline = True
    return func

  return decorator


def load_from_file(path: str):
  with open(path) as f:
    contents = f.read()
    component_dict = yaml.safe_load(contents)
  comp = components.load_component_from_text(contents)
  description = component_dict.get('description', '')
  comp.__doc__ = description
  return comp


utils.gcpc_output_name_converter = second_order_passthrough_decorator
dsl.component = second_order_passthrough_decorator
dsl.container_component = first_order_passthrough_decorator
dsl.pipeline = second_order_passthrough_decorator_for_pipeline
components.load_component_from_file = load_from_file


class OutputPath(dsl.OutputPath):

  def __repr__(self) -> str:
    type_string = getattr(self.type, '__name__', '')
    return f'dsl.OutputPath({type_string})'


dsl.OutputPath = OutputPath


class InputClass:

  def __getitem__(self, type_) -> str:
    type_string = getattr(type_, 'schema_title', getattr(type_, '__name__', ''))
    return f'dsl.Input[{type_string}]'


Input = InputClass()

dsl.Input = Input


class OutputClass:

  def __getitem__(self, type_) -> str:
    type_string = getattr(type_, 'schema_title', getattr(type_, '__name__', ''))
    return f'dsl.Output[{type_string}]'


Output = OutputClass()

dsl.Output = Output

# -- General configuration ---------------------------------------------------

# If your documentation needs a minimal Sphinx version, state it here.
# needs_sphinx = '1.0'

# Add any Sphinx extension module names here, as strings. They can be
# extensions coming with Sphinx (named 'sphinx.ext.*') or your custom
# ones.
extensions = [
    'sphinx.ext.autodoc',
    'sphinx.ext.viewcode',
    'sphinx.ext.napoleon',
    'm2r2',
    'sphinx_immaterial',
    'autodocsumm',
    'notfound.extension',
]
autodoc_default_options = {
    'members': True,
    'member-order': 'alphabetical',
    'imported-members': True,
    'undoc-members': True,
    'show-inheritance': False,
    'inherited-members': False,
    'autosummary': False,
}

# notfound.extension: https://sphinx-notfound-page.readthedocs.io/en/latest/configuration.html#confval-notfound_context
notfound_context = {
    'title': 'Page not found',
    'body': textwrap.dedent("""
            <head>
            <title>Page not found</title>
            </head>
            <body>
            <div class="container">
                <h1>404: Page not found</h1>
                <p>
                It's likely the object or page you're looking for doesn't exist in this version of Google Cloud Pipeline Components. Please ensure you have the correct version selected.
                </p>
                <a href="https://google-cloud-pipeline-components.readthedocs.io/">Back to homepage</a>
            </div>
            </body>
            """),
}

html_theme = 'sphinx_immaterial'
html_title = 'Google Cloud Pipeline Components Reference Documentation'
html_static_path = ['_static']
html_css_files = ['custom.css']
html_theme_options = {
    'icon': {
        'repo': 'fontawesome/brands/github',
    },
    'repo_url': 'https://github.com/kubeflow/pipelines/tree/master/components/google-cloud',
    'repo_name': 'pipelines',
    'repo_type': 'github',
    'edit_uri': 'https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/docs/source',
    'globaltoc_collapse': True,
    'features': [
        'navigation.expand',
        # "navigation.tabs",
        # "toc.integrate",
        'navigation.sections',
        # "navigation.instant",
        # "header.autohide",
        'navigation.top',
        # "navigation.tracking",
        'search.highlight',
        'search.share',
        'toc.follow',
        'toc.sticky',
    ],
    'palette': [{
        'media': '(prefers-color-scheme: light)',
        'scheme': 'default',
        'primary': 'googleblue',
    }],
    'font': {'text': 'Open Sans'},
    'version_dropdown': True,
    'version_json': 'https://raw.githubusercontent.com/kubeflow/pipelines/test-gcpc-dropdown/versions.json',
    # "toc_title_is_page_title": True,
}
# Add any paths that contain templates here, relative to this directory.
templates_path = ['_templates']

# The suffix(es) of source filenames.
# You can specify multiple suffix as a list of string:
source_suffix = '.rst'

# The master toctree document.
master_doc = 'index'

# The language for content autogenerated by Sphinx. Refer to documentation
# for a list of supported languages.
#
# This is also used if you do content translation via gettext catalogs.
# Usually you set "language" from the command line for these cases.
language = 'en'

# List of patterns, relative to source directory, that match files and
# directories to ignore when looking for source files.
# This pattern also affects html_static_path and html_extra_path.
exclude_patterns = ['_build', 'Thumbs.db', '.DS_Store']

# The name of the Pygments (syntax highlighting) style to use.
pygments_style = None

# The default sidebars (for documents that don't match any pattern) are
# defined by theme itself.  Builtin themes are using these templates by
# default: ``['localtoc.html', 'relations.html', 'sourcelink.html',
# 'searchbox.html']``.
#
# html_sidebars = {}

# -- Options for HTMLHelp output ---------------------------------------------

# Output file base name for HTML help builder.
htmlhelp_basename = 'GoogleCloudPipelineComponentsDocs'


def component_grouper(app, what, name, obj, section, parent):
  if getattr(obj, '_is_component', False):
    return 'Components'


def pipeline_grouper(app, what, name, obj, section, parent):
  if getattr(obj, '_is_pipeline', False):
    return 'Pipelines'


def autodoc_skip_member(app, what, name, obj, skip, options):
  skip = True
  if name == 'create_custom_training_job_op_from_component':
    return skip


def make_docstring_lines_for_param(
    param_name: str,
    type_string: str,
    description: str,
) -> List[str]:
  WHITESPACE = '     '

  return [
      f'{WHITESPACE * 2}``{param_name}: {type_string}``',
      f'{WHITESPACE * 4}{description}',
  ]


def get_return_section(component) -> List[str]:
  """Modifies docstring so that a return section can be treated as an

  args section, then parses the docstring.
  """
  docstring = inspect.getdoc(component)
  type_hints = component.__annotations__
  if docstring is None:
    return []

  # Returns and Return are the only two keywords docstring_parser uses for returns
  # use newline to avoid replacements that aren't in the return section header
  return_keywords = ['Returns:\n', 'Returns\n', 'Return:\n', 'Return\n']
  for keyword in return_keywords:
    if keyword in docstring:
      modified_docstring = docstring.replace(keyword.strip(), 'Args:')
      returns_docstring = docstring_parser.parse(modified_docstring)
      new_docstring_lines = []
      for param in returns_docstring.params:
        type_string = (
            repr(
                type_hints.get(
                    param.arg_name,
                    type_hints.get(
                        param.arg_name,
                        'Unknown',
                    ),
                )
            )
            .lstrip("'")
            .rstrip("'")
        )
        new_docstring_lines.extend(
            make_docstring_lines_for_param(
                param_name=param.arg_name.lstrip(
                    utils.DOCS_INTEGRATED_OUTPUT_RENAMING_PREFIX
                ),
                type_string=type_string,
                description=param.description,
            )
        )
      return new_docstring_lines
  return []


def remove__output_prefix_from_signature(
    app, what, name, obj, options, signature, return_annotation
):
  if signature is not None:
    signature = re.sub(
        rf'{utils.DOCS_INTEGRATED_OUTPUT_RENAMING_PREFIX}(\w+):',
        r'\1:',
        signature,
    )
  return signature, return_annotation


def remove_after_returns_in_place(lines: List[str]) -> bool:
  for i in range(len(lines)):
    if lines[i].startswith(':returns:'):
      del lines[i:]
      return True
  return False

def process_named_docstring_returns(app, what, name, obj, options, lines):
  if getattr(obj, '_is_component', False):
    has_returns_section = remove_after_returns_in_place(lines)
    if has_returns_section:
      returns_section = get_return_section(obj)
      lines.extend([':returns:', ''])
      lines.extend(returns_section)

  markdown_to_rst(app, what, name, obj, options, lines)


def markdown_to_rst(app, what, name, obj, options, lines):
  md = '\n'.join(lines)
  ast = commonmark.Parser().parse(md)
  rst = commonmark.ReStructuredTextRenderer().render(ast)
  lines.clear()
  lines += rst.splitlines()


def setup(app):
  app.connect('autodoc-process-docstring', process_named_docstring_returns)
  app.connect(
      'autodoc-process-signature',
      remove__output_prefix_from_signature,
  )
  app.connect('autodocsumm-grouper', component_grouper)
  app.connect('autodocsumm-grouper', pipeline_grouper)
  app.connect('autodoc-skip-member', autodoc_skip_member)

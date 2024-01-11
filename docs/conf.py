# Copyright 2019-2022 The Kubeflow Authors
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

# Configuration file for the Sphinx documentation builder.
#
# This file does only contain a selection of the most common options. For a
# full list see the documentation:
# http://www.sphinx-doc.org/en/master/config

import functools
import os
import sys
from typing import List, Optional

import sphinx
from sphinx import application  # noqa

# -- Path setup --------------------------------------------------------------

# If extensions (or modules to document with autodoc) are in another directory,
# add these directories to sys.path here. If the directory is relative to the
# documentation root, use os.path.abspath to make it absolute, like shown here.
sys.path.insert(0, os.path.abspath('../sdk/python'))

# -- Project information -----------------------------------------------------
project = 'Kubeflow Pipelines'
copyright = '2022, The Kubeflow Authors'
author = 'The Kubeflow Authors'

# The short X.Y version
version = ''
# The full version, including alpha/beta/rc tags
release = ''

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
    'sphinx_click',
    'm2r2',
    'sphinx_immaterial',
    'autodocsumm',
]
autodoc_member_order = 'bysource'
autodoc_default_options = {
    'members': True,
    'imported-members': True,
    'undoc-members': True,
    'show-inheritance': False,
    'autosummary': True,
}

html_theme = 'sphinx_immaterial'
html_title = 'KFP SDK API Reference'
html_static_path = ['_static']
html_css_files = ['custom.css']
html_logo = '_static/kubeflow.png'
html_favicon = '_static/favicon.ico'
html_theme_options = {
    'icon': {
        'repo': 'fontawesome/brands/github',
    },
    'repo_url':
        'https://github.com/kubeflow/pipelines/',
    'repo_name':
        'pipelines',
    'repo_type':
        'github',
    'edit_uri':
        'blob/master/docs',
    'globaltoc_collapse':
        False,
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
    'palette': [
        {
            'media': '(prefers-color-scheme: dark)',
            'scheme': 'slate',
            'primary': 'kfpblue',
            # "accent": "lime",
            'toggle': {
                'icon': 'material/lightbulb',
                'name': 'Switch to light mode',
            },
        },
        {
            'media': '(prefers-color-scheme: light)',
            'scheme': 'default',
            'primary': 'kfpblue',
            # "accent": "light-blue",
            'toggle': {
                'icon': 'material/lightbulb-outline',
                'name': 'Switch to dark mode',
            },
        },
    ],
    'font': {
        'text': 'Open Sans'
    },
    'version_dropdown':
        True,
    'version_info': [
        # need to use the sdk- prefix to avoid conflict with the BE's GitHub release tags
        {
            'version':
                'https://kubeflow-pipelines.readthedocs.io/en/sdk-2.6.0/',
            'title':
                '2.6.0',
            'aliases': ['stable'],
        },
        {
            'version':
                'https://kubeflow-pipelines.readthedocs.io/en/sdk-2.5.0/',
            'title':
                '2.5.0',
            'aliases': [],
        },
        {
            'version':
                'https://kubeflow-pipelines.readthedocs.io/en/sdk-2.4.0/',
            'title':
                '2.4.0',
            'aliases': [],
        },
        {
            'version':
                'https://kubeflow-pipelines.readthedocs.io/en/sdk-2.3.0/',
            'title':
                '2.3.0',
            'aliases': [],
        },
        {
            'version':
                'https://kubeflow-pipelines.readthedocs.io/en/sdk-2.2.0/',
            'title':
                '2.2.0',
            'aliases': [],
        },
        {
            'version':
                'https://kubeflow-pipelines.readthedocs.io/en/sdk-2.0.1/',
            'title':
                '2.0.1',
            'aliases': [],
        },
        {
            'version':
                'https://kubeflow-pipelines.readthedocs.io/en/sdk-2.0.0/',
            'title':
                '2.0.0',
            'aliases': [],
        },
        {
            'version':
                'https://kubeflow-pipelines.readthedocs.io/en/sdk-2.0.0-rc.2/',
            'title':
                'v2.0.0rc2',
            'aliases': [],
        },
        {
            'version':
                'https://kubeflow-pipelines.readthedocs.io/en/sdk-2.0.0-rc.1/',
            'title':
                'v2.0.0rc1',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b17/',
            'title': 'v2.0.0b17',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b16/',
            'title': 'v2.0.0b16',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b15/',
            'title': 'v2.0.0b15',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b14/',
            'title': 'v2.0.0b14',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b13/',
            'title': 'v2.0.0b13',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b12/',
            'title': 'v2.0.0b12',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b11/',
            'title': 'v2.0.0b11',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b9/',
            'title': 'v2.0.0b9',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b8/',
            'title': 'v2.0.0b8',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b6/',
            'title': 'v2.0.0b6',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b5/',
            'title': 'v2.0.0b5',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/2.0.0b4/',
            'title': 'v2.0.0b4',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/1.8.22/',
            'title': 'v1.8.22',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/1.8.21/',
            'title': 'v1.8.21',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/1.8.20/',
            'title': 'v1.8.20',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/1.8.19/',
            'title': 'v1.8.19',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/1.8.18/',
            'title': 'v1.8.18',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/1.8.17/',
            'title': 'v1.8.17',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/1.8.16/',
            'title': 'v1.8.16',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/1.8.15/',
            'title': 'v1.8.15',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/1.8.14/',
            'title': 'v1.8.14',
            'aliases': [],
        },
        {
            'version': 'https://kubeflow-pipelines.readthedocs.io/en/1.8.13/',
            'title': 'v1.8.13',
            'aliases': [],
        },
    ],
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
htmlhelp_basename = 'KubeflowPipelinesdoc'

# -- Options for LaTeX output ------------------------------------------------

latex_elements = {
    # The paper size ('letterpaper' or 'a4paper').
    #
    # 'papersize': 'letterpaper',

    # The font size ('10pt', '11pt' or '12pt').
    #
    # 'pointsize': '10pt',

    # Additional stuff for the LaTeX preamble.
    #
    # 'preamble': '',

    # Latex figure (float) alignment
    #
    # 'figure_align': 'htbp',
}

# Grouping the document tree into LaTeX files. List of tuples
# (source start file, target name, title,
#  author, documentclass [howto, manual, or own class]).
latex_documents = [
    (master_doc, 'KubeflowPipelines.tex', 'Kubeflow Pipelines Documentation',
     'Google', 'manual'),
]

# -- Options for manual page output ------------------------------------------

# One entry per manual page. List of tuples
# (source start file, name, description, authors, manual section).
man_pages = [(master_doc, 'kubeflowpipelines',
              'Kubeflow Pipelines Documentation', [author], 1)]

# -- Options for Texinfo output ----------------------------------------------

# Grouping the document tree into Texinfo files. List of tuples
# (source start file, target name, title, author,
#  dir menu entry, description, category)
texinfo_documents = [
    (master_doc, 'KubeflowPipelines', 'Kubeflow Pipelines Documentation',
     author, 'KubeflowPipelines',
     'Kubeflow Pipelines is a platform for building and deploying portable, scalable machine learning workflows based on Docker containers within the Kubeflow project.',
     ''),
]

# -- Options for Epub output -------------------------------------------------

# Bibliographic Dublin Core info.
epub_title = project

# The unique identifier of the text. This can be a ISBN number
# or the project homepage.
#
# epub_identifier = ''

# A unique identification for the text.
#
# epub_uid = ''

# A list of files that should not be packed into the epub file.
epub_exclude_files = ['search.html']

# # -- Extension configuration -------------------------------------------------
readme_path = os.path.join(
    os.path.abspath(os.path.dirname(os.path.dirname(__file__))), 'sdk',
    'python', 'README.md')


def trim_header_of_readme(path: str) -> List[str]:
    with open(path, 'r') as f:
        contents = f.readlines()
    with open(path, 'w') as f:
        f.writelines(contents[1:])
    return contents


def re_attach_header_of_readme(path: str, contents: List[str]) -> None:
    with open(path, 'w') as f:
        f.writelines(contents)


original_readme_contents = trim_header_of_readme(readme_path)

re_attach_header_of_readme_closure = functools.partial(
    re_attach_header_of_readme,
    path=readme_path,
    contents=original_readme_contents)


def re_attach_header_of_readme_hook(app: sphinx.application.Sphinx,
                                    exception: Optional[Exception]) -> None:
    re_attach_header_of_readme_closure()


def setup(app: sphinx.application.Sphinx) -> None:
    app.connect('build-finished', re_attach_header_of_readme_hook)

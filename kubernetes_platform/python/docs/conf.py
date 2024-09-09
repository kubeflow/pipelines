# Copyright 2023 The Kubeflow Authors
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

import re

from kfp import dsl


# preserve function docstrings for components by setting component decorators to passthrough decorators
# also enables autodoc to document the components as functions without using the autodata directive (https://www.sphinx-doc.org/en/master/usage/extensions/autodoc.html#directive-autodata)
def container_component_decorator(func):
    return func


def component_decorator(*args, **kwargs):

    def decorator(func):
        return func

    return decorator


dsl.component = component_decorator
dsl.container_component = container_component_decorator

# -- Project information -----------------------------------------------------
project = 'Kubeflow Pipelines'
copyright = '2023, The Kubeflow Authors'
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
html_title = 'kfp-kubernetes Reference Documentation'
html_static_path = ['_static']
html_css_files = ['custom.css']
html_logo = '_static/kubeflow.png'
html_favicon = '_static/favicon.ico'
html_theme_options = {
    'icon': {
        'repo': 'fontawesome/brands/github',
    },
    'repo_url':
        'https://github.com/kubeflow/pipelines/tree/master/kubernetes_platform',
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
        {
            'version':
                'https://kfp-kubernetes.readthedocs.io/en/kfp-kubernetes-1.3.0/',
            'title':
                '1.3.0',
            'aliases': ['stable'],
        },
        {
            'version':
                'https://kfp-kubernetes.readthedocs.io/en/kfp-kubernetes-1.2.0/',
            'title':
                '1.2.0',
            'aliases': [],
        },
        {
            'version':
                'https://kfp-kubernetes.readthedocs.io/en/kfp-kubernetes-1.1.0/',
            'title':
                '1.1.0',
            'aliases': [],
        },
        {
            'version':
                'https://kfp-kubernetes.readthedocs.io/en/kfp-kubernetes-1.0.0/',
            'title':
                '1.0.0',
            'aliases': [],
        },
        {
            'version':
                'https://kfp-kubernetes.readthedocs.io/en/kfp-kubernetes-0.0.1/',
            'title':
                '0.0.1',
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
htmlhelp_basename = 'KfpKubernetesdoc'


# TODO: align with GCPC representation of components (in particular, OutputPath and Output[])
def strip_outputs_from_signature(app, what, name, obj, options, signature,
                                 return_annotation):
    if signature is not None:
        signature = re.sub(
            r'[0-9a-zA-Z]+: <kfp\.components\.types\.type_annotations\.OutputPath object at 0x[0-9a-fA-F]+>?,?\s',
            '', signature)
    return signature, return_annotation


def setup(app):
    app.connect('autodoc-process-signature', strip_outputs_from_signature)

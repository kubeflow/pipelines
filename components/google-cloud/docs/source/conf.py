# Configuration file for the Sphinx documentation builder.
#
# This file only contains a selection of the most common options. For a full
# list see the documentation:
# https://www.sphinx-doc.org/en/master/usage/configuration.html

import textwrap
# -- Path setup --------------------------------------------------------------

# If extensions (or modules to document with autodoc) are in another directory,
# add these directories to sys.path here. If the directory is relative to the
# documentation root, use os.path.abspath to make it absolute, like shown here.

# import os
# import sys
# sys.path.insert(0, os.path.abspath('..'))

# -- Project information -----------------------------------------------------

project = 'google_cloud_pipeline_components'
copyright = '2022, Google Inc'
author = 'Google Inc'

# The full version, including alpha/beta/rc tags
release = '0.0.1'

# -- General configuration ---------------------------------------------------

# Add any Sphinx extension module names here, as strings. They can be
# extensions coming with Sphinx (named 'sphinx.ext.*') or your custom
# ones.
extensions = [
    'sphinx.ext.autodoc',
    'notfound.extension',
]

# notfound.extension: https://sphinx-notfound-page.readthedocs.io/en/latest/configuration.html#confval-notfound_context
notfound_context = {
    'title':
        'Page not found',
    'body':
        textwrap.dedent("""
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
# Add any paths that contain templates here, relative to this directory.
templates_path = ['_templates']

# List of patterns, relative to source directory, that match files and
# directories to ignore when looking for source files.
# This pattern also affects html_static_path and html_extra_path.
exclude_patterns = []

# -- Options for HTML output -------------------------------------------------

# The theme to use for HTML and HTML Help pages.  See the documentation for
# a list of builtin themes.
#
html_theme = 'sphinx_rtd_theme'

# Add any paths that contain custom static files (such as style sheets) here,
# relative to this directory. They are copied after the builtin static files,
# so a file named "default.css" will overwrite the builtin "default.css".
html_static_path = []

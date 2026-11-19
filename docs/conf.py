# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
project = 'Native Kubeflow Pipelines'
copyright = '2026, The Kubeflow Authors'
extensions = ['sphinx.ext.autodoc', 'sphinx.ext.autosummary', 'autodocsumm', 'sphinx_click', 'myst_parser']
autosummary_generate = True
html_theme = 'furo'
exclude_patterns = ['_build', 'agents', 'README.md']

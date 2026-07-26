# Copyright The Kubeflow Authors.
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

# Configuration file for the Sphinx documentation builder.
# Kubeflow Pipelines documentation website.

import os
import sys
from pathlib import Path

DOCS_DIR = Path(__file__).parent
REPO_ROOT = DOCS_DIR.parent

# Local Sphinx extensions live in docs/_ext/.
sys.path.insert(0, str(DOCS_DIR / "_ext"))
# Allow autodoc to import the SDK from source when it is not pip-installed.
sys.path.insert(0, str(REPO_ROOT / "sdk" / "python"))

# -- Project information -----------------------------------------------------
project = "Kubeflow Pipelines"
copyright = "Kubeflow Authors"
author = "Kubeflow Authors"

# ReadTheDocs sets READTHEDOCS_VERSION automatically.
version = os.getenv("READTHEDOCS_VERSION", "latest")
release = version

# -- General configuration ---------------------------------------------------
extensions = [
    "myst_parser",  # Markdown support via MyST
    "sphinxcontrib.mermaid",  # Mermaid diagram rendering
    "sphinx_copybutton",  # Copy button on code blocks
    "sphinx_design",  # Grid layouts and card components
    "kubeflow_topnav",  # Top navigation bar (see docs/_ext/kubeflow_topnav/)
    "sphinx.ext.autodoc",  # SDK API reference from docstrings
    "sphinx.ext.napoleon",  # Google-style docstring support
    "sphinx.ext.viewcode",  # "View source" links on API pages
    "sphinx_click",  # kfp CLI reference from Click commands
    "autodocsumm",  # Summary tables on API pages
]

templates_path = ["_templates"]

exclude_patterns = [
    "_build",
    "Thumbs.db",
    ".DS_Store",
    "*.egg-info",
    "__pycache__",
    # Pre-existing docs/ content that is not part of the website.
    "diagram",
    "sdk/Architecture.md",
    "sdk/README.md",
    # Superseded by the top-level docs/index.rst.
    "sdk/index.rst",
    # Duplicate of the migrated overview.md; the platform Overview is canonical.
    "sdk/source/overview.rst",
]

# -- Options for HTML output -------------------------------------------------
html_theme = "furo"
html_title = "Kubeflow Pipelines"
html_static_path = ["_static"]
html_favicon = "images/pipelines-icon.svg"
html_baseurl = os.getenv(
    "READTHEDOCS_CANONICAL_URL", "https://kubeflow-pipelines.readthedocs.io/"
)
html_css_files = [
    # Loaded as <link> rather than an @import so ordering cannot silently
    # invalidate it, and so the fonts fetch in parallel with the stylesheets.
    (
        "https://fonts.googleapis.com/css2"
        "?family=Space+Grotesk:wght@500;700"
        "&family=IBM+Plex+Sans:wght@400;500;600"
        "&display=swap"
    ),
    "css/custom.css",
    "css/landing.css",
]
html_js_files = [
    "js/topnav.js",
    "js/landing-page.js",
    "js/external-links.js",
    "js/sidebar-toggle.js",
]

html_theme_options = {
    "light_css_variables": {
        "color-brand-primary": "#4299e1",
        "color-brand-content": "#3182ce",
        "color-foreground-primary": "#1a202c",
        "color-foreground-secondary": "#2d3748",
        "color-foreground-muted": "#4a5568",
        "color-foreground-border": "#e2e8f0",
        # Keep visited links the same blue as unvisited ones.
        "color-link--visited": "#3182ce",
        "color-link--visited--hover": "#2c5282",
    },
    "dark_css_variables": {
        "color-brand-primary": "#63b3ed",
        "color-brand-content": "#63b3ed",
        "color-foreground-primary": "#e2e8f0",
        "color-foreground-secondary": "#cbd5e0",
        "color-foreground-muted": "#a0aec0",
        "color-foreground-border": "#4a5568",
        # Keep visited links the same blue as unvisited ones.
        "color-link--visited": "#63b3ed",
        "color-link--visited--hover": "#90cdf4",
    },
    "sidebar_hide_name": False,
    "navigation_with_keys": True,
    "top_of_page_buttons": ["view", "edit"],
    "source_repository": "https://github.com/kubeflow/pipelines",
    "source_branch": "master",
    "source_directory": "docs/",
}

html_context = {
    "display_github": True,
    "github_user": "kubeflow",
    "github_repo": "pipelines",
    "github_version": "master",
    "conf_py_path": "/docs/",
}

# -- Autodoc configuration ---------------------------------------------------
autodoc_member_order = "bysource"
autodoc_default_options = {
    "members": True,
    "imported-members": True,
    "undoc-members": True,
    "show-inheritance": False,
    "autosummary": True,
}

# -- MyST Parser configuration -----------------------------------------------
myst_enable_extensions = [
    "colon_fence",  # ::: fence syntax for directives
    "deflist",  # Definition lists
    "fieldlist",  # Field lists
    "substitution",  # Variable substitution
    "tasklist",  # Task lists [ ] [x]
]
myst_links_external_new_tab = True
myst_heading_anchors = 4
# Render ```mermaid fenced code blocks as diagrams via sphinxcontrib.mermaid
# rather than treating them as an unknown code lexer.
myst_fence_as_directive = ["mermaid"]

# Many migrated pages open a section with an H3 directly under the page H1,
# as authored on kubeflow.org. The migration copies pages over rather than
# restructuring them, so the non-consecutive-heading notice is expected.
suppress_warnings = ["myst.header"]

# -- Mermaid configuration ---------------------------------------------------
mermaid_version = "10.9.1"
mermaid_d3_zoom = False

# -- Link checking configuration ---------------------------------------------
linkcheck_ignore = [
    r"http://localhost:\d+/",
    r"https://github\.com/.*/pulls/.*",
    r"https://medium\.com/.*",
]
# These hosts render anchors client-side; linkcheck can't verify them.
linkcheck_anchors_ignore_for_url = [
    r"https://github\.com/.*",
    r"https://kubernetes\.io/.*",
]

# -- Copy button configuration -----------------------------------------------
copybutton_prompt_text = r">>> |\.\.\. |\$ |In \[\d*\]: | {2,5}\.\.\.: | {5,8}: "
copybutton_prompt_is_regexp = True
copybutton_line_continuation_character = "\\"

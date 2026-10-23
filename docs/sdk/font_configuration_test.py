#!/usr/bin/env python3

# Copyright 2026 The Kubeflow Authors
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

import ast
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


def font_option(filename):
    tree = ast.parse(filename.read_text())
    options = next(
        node.value
        for node in tree.body
        if isinstance(node, ast.Assign)
        and any(
            isinstance(target, ast.Name) and target.id == 'html_theme_options'
            for target in node.targets
        )
    )
    return ast.literal_eval(
        next(
            value
            for key, value in zip(options.keys, options.values)
            if ast.literal_eval(key) == 'font'
        )
    )


BUILD_WITHOUT_NETWORK = r"""
import io
from pathlib import Path
import socket
import sys
from unittest import mock
from sphinx.application import Sphinx

original_connect = socket.socket.connect

def reject_network(sock, address):
    if sock.family in (socket.AF_INET, socket.AF_INET6):
        raise AssertionError('Documentation must not fetch network resources')
    return original_connect(sock, address)

source = Path(sys.argv[1])
with mock.patch('socket.socket.connect', reject_network), mock.patch(
        'sphinx_immaterial.google_fonts.get_url',
        side_effect=AssertionError('Google Fonts fetch attempted')):
    app = Sphinx(str(source), str(source), str(source / '_html'),
                 str(source / '_doctrees'), 'html',
                 status=io.StringIO(), warning=sys.stderr,
                 freshenv=True, warningiserror=True)
    app.build(force_all=True)
sys.exit(app.statuscode)
"""


class SDKFontConfigurationTest(unittest.TestCase):
    def test_actual_font_options_build_without_network(self):
        for config in [
            ROOT / 'docs/sdk/conf.py',
            ROOT / 'kubernetes_platform/python/docs/conf.py',
        ]:
            with (
                self.subTest(config=str(config)),
                tempfile.TemporaryDirectory() as directory,
            ):
                source = Path(directory)
                source.joinpath('conf.py').write_text(
                    "project = 'Font regression'\n"
                    "html_theme = 'sphinx_immaterial'\n"
                    "extensions = ['sphinx_immaterial']\n"
                    f"html_theme_options = {{'font': {font_option(config)!r}}}\n"
                )
                source.joinpath('index.rst').write_text(
                    'Font regression\n===============\n\nText and ``code``.\n'
                )
                result = subprocess.run(
                    [sys.executable, '-c', BUILD_WITHOUT_NETWORK, str(source)],
                    capture_output=True,
                    text=True,
                    timeout=60,
                    check=False,
                )
                self.assertEqual(
                    result.returncode, 0, result.stdout + result.stderr
                )
                self.assertTrue(source.joinpath('_html/index.html').is_file())


if __name__ == '__main__':
    unittest.main()

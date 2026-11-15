# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Smoke test for the installed GCPC documentation extras.

Run with the selected setup.py docs dependencies installed. The small
site loads our real extensions/theme without importing every cloud
component or fetching fonts.
"""

import ast
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


class DocumentationDependenciesTest(unittest.TestCase):

    def test_markdown_include_with_documentation_extensions(self):
        config = ast.parse(
            (Path(__file__).parent / 'source/conf.py').read_text())
        settings = {}
        for node in config.body:
            if isinstance(node, ast.Assign):
                for target in node.targets:
                    if isinstance(target,
                                  ast.Name) and target.id in ('extensions',
                                                              'html_theme'):
                        settings[target.id] = ast.literal_eval(node.value)
        self.assertIn('m2r2', settings['extensions'])
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / 'source'
            output = Path(directory) / 'html'
            source.mkdir()
            (source / 'conf.py'
            ).write_text('\n'.join(
                f'{key} = {value!r}' for key, value in settings.items()) +
                         "\nproject = 'Documentation dependency smoke'\n"
                         "source_suffix = '.rst'\n"
                         "exclude_patterns = ['fragment.md']\n"
                         "html_theme_options = {'font': False}\n")
            title = 'Documentation dependencies'
            (source / 'index.rst').write_text(
                f'{title}\n{"=" * len(title)}\n\n.. mdinclude:: fragment.md\n')
            (source / 'fragment.md').write_text('A **documented component**.\n')
            result = subprocess.run([
                sys.executable, '-m', 'sphinx', '-W', '--keep-going', '-b',
                'html',
                str(source),
                str(output)
            ],
                                    capture_output=True,
                                    text=True,
                                    check=False)
            self.assertEqual(result.returncode, 0,
                             result.stdout + result.stderr)
            self.assertIn('<strong>documented component</strong>',
                          (output / 'index.html').read_text())


if __name__ == '__main__':
    unittest.main()

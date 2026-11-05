#!/usr/bin/env python3
"""Tests for the boundaries between kfpr release modules."""

import ast
import importlib
import subprocess
import sys
import unittest
from pathlib import Path

from kfpr import core

PACKAGE_DIR = Path(__file__).resolve().parent
RELEASE_DIR = PACKAGE_DIR.parent
FOCUSED_MODULES = (
    'metadata',
    'state',
    'runner',
    'prompts',
    'github_api',
    'commands',
    'validation',
    'preflight',
    'context',
)
# Modules that only build, validate or format values must not run commands.
COMMAND_FREE_MODULES = ('metadata', 'prompts', 'commands', 'validation')
# Release workflows run these with only PyYAML installed.
WORKFLOW_MODULES = (
    'kfpr.promotion',
    'kfpr.ci_images_client',
    'kfpr.ci_qualification_client',
)


def defined_names(module_name):
    tree = ast.parse((PACKAGE_DIR / f'{module_name}.py').read_text())
    names = []
    for node in tree.body:
        if isinstance(node, (ast.FunctionDef, ast.ClassDef)):
            names.append(node.name)
        elif isinstance(node, ast.Assign):
            names.extend(t.id for t in node.targets)
    return names


def run_python(code):
    return subprocess.run(
        [sys.executable, '-c', code],
        cwd=RELEASE_DIR,
        capture_output=True,
        text=True,
        check=False,
    )


class ModuleBoundaryTest(unittest.TestCase):
    def test_core_reexports_every_focused_name(self):
        expected = []
        for module_name in FOCUSED_MODULES:
            module = importlib.import_module(f'kfpr.{module_name}')
            for name in defined_names(module_name):
                with self.subTest(module=module_name, name=name):
                    self.assertIs(getattr(core, name), getattr(module, name))
                expected.append(name)
        self.assertEqual(sorted(core.__all__), sorted(expected))

    def test_each_module_imports_on_its_own(self):
        for module_name in (*FOCUSED_MODULES, 'core'):
            with self.subTest(module=module_name):
                result = run_python(f'import kfpr.{module_name}')
                self.assertEqual(result.returncode, 0, result.stderr)

    def test_value_and_prompt_modules_do_not_run_commands(self):
        for module_name in COMMAND_FREE_MODULES:
            with self.subTest(module=module_name):
                module = importlib.import_module(f'kfpr.{module_name}')
                self.assertNotIn('subprocess', vars(module))

    def test_workflow_modules_import_without_cli_dependencies(self):
        blocked = ('typer', 'click', 'rich', 'build')
        for module_name in WORKFLOW_MODULES:
            with self.subTest(module=module_name):
                result = run_python(
                    'import sys\n'
                    f'for name in {blocked!r}:\n'
                    '    sys.modules[name] = None\n'
                    f'import {module_name}\n'
                )
                self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == '__main__':
    unittest.main()

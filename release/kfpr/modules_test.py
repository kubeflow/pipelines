#!/usr/bin/env python3
"""Tests for the boundaries between kfpr release modules."""

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
# Names kfpr.core exported before the split, mapped to their new module.
# Frozen for compatibility: do not add new helpers here.
CORE_EXPORTS = {
    '_release_git_mounts': 'commands',
    'ANSI_BOLD': 'prompts',
    'ANSI_RESET': 'prompts',
    'ANSI_UNDERLINE': 'prompts',
    'collect_context': 'context',
    'CommandRunner': 'runner',
    'confirm': 'prompts',
    'default_branch': 'github_api',
    'doctor_errors': 'preflight',
    'emphasize_prompt': 'prompts',
    'image_workflow_command': 'commands',
    'is_pending_merge_queue_check': 'github_api',
    'kfp_kubernetes_docs_build_command': 'commands',
    'kfp_requirements_command': 'commands',
    'kubernetes_requirements_command': 'commands',
    'normalize_fork_remote': 'validation',
    'parse_github_owner': 'validation',
    'prompt_choice': 'prompts',
    'prompt_numbered_choice': 'prompts',
    'prompt_required': 'prompts',
    'prompt_validated': 'prompts',
    'release_version_bump_command': 'commands',
    'ReleaseContext': 'context',
    'ReleaseMetadata': 'metadata',
    'ReleaseState': 'state',
    'REPO': 'github_api',
    'sdk_workflow_command': 'commands',
    'STATE_FILE': 'state',
    'underline_links': 'prompts',
    'upstream_default_branch': 'github_api',
    'URL_RE': 'prompts',
    'validate_answers': 'validation',
    'validate_state': 'validation',
    'VERSION_RE': 'metadata',
    'wait_for_pr_merge': 'github_api',
    'watch_latest_workflow_run': 'github_api',
    'watch_pr_ci': 'github_api',
}
# Modules that only build, validate or format values must not run commands.
COMMAND_FREE_MODULES = ('metadata', 'prompts', 'commands', 'validation')
# Release workflows run these with only PyYAML installed.
WORKFLOW_MODULES = (
    'kfpr.promotion',
    'kfpr.ci_images_client',
    'kfpr.ci_qualification_client',
)


def run_python(code):
    return subprocess.run(
        [sys.executable, '-c', code],
        cwd=RELEASE_DIR,
        capture_output=True,
        text=True,
        check=False,
    )


class ModuleBoundaryTest(unittest.TestCase):
    def test_core_reexports_its_original_names(self):
        self.assertEqual(sorted(core.__all__), sorted(CORE_EXPORTS))
        for name, module_name in CORE_EXPORTS.items():
            with self.subTest(name=name):
                module = importlib.import_module(f'kfpr.{module_name}')
                self.assertIs(getattr(core, name), getattr(module, name))

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

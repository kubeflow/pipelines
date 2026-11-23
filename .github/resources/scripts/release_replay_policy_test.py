"""Exercise release replay selection against a fixed trusted-base workflow
tree."""

import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from kfpr import landed_ci

SCRIPTS = Path(__file__).resolve().parent
BASE = 'a' * 40
RELEASE_ONLY = 'pull_request: {branches: [release-2.18]}'
# These definitions are returned by the mocked release-parent GraphQL query.
# The real inventory generator parses their GitHub trigger and job conditions.
WORKFLOWS = {
    'pre-commit.yml': RELEASE_ONLY,
    'build-tools-images.yml': 'pull_request: {branches: [release-2.18], paths: ["release/**"]}',
    'postgresql-image-upgrade.yml': 'pull_request: {branches: [release-2.18], paths: ["manifests/**"]}',
    'backend.yml': 'pull_request: {paths-ignore: ["docs/**"]}',
    'main.yml': 'pull_request: {branches: [main]}',
    'manual.yml': 'workflow_dispatch:',
    'upgrade-test.yml': RELEASE_ONLY,
}


def tree_response(workflows, pause_upgrade):
    entries = []
    for name, trigger in workflows.items():
        content = f'name: {name}\non:\n  {trigger}\njobs:\n  test:\n'
        if name == 'upgrade-test.yml' and pause_upgrade:
            content += '    if: ${{ false }}\n'
        content += '    runs-on: ubuntu-latest\n'
        entries.append(
            {
                'name': name,
                'mode': 33188,
                'type': 'blob',
                'object': {
                    '__typename': 'Blob',
                    'text': content,
                    'byteSize': len(content.encode()),
                    'isBinary': False,
                    'isTruncated': False,
                },
            }
        )
    return {
        'data': {
            'repository': {
                'nameWithOwner': 'kubeflow/pipelines',
                'object': {'__typename': 'Tree', 'entries': entries},
            }
        }
    }


class ReleaseReplayPolicyTest(unittest.TestCase):
    def setUp(self):
        directory = Path(self.enterContext(tempfile.TemporaryDirectory()))
        self.fixture = directory / 'tree.json'
        self.request = directory / 'graphql-request.json'
        # expected_workflows puts its interpreter's directory first on PATH.
        # Place both executables here so no GitHub connection is possible.
        interpreter = directory / 'python3'
        original_interpreter = sys.executable
        interpreter.write_text('#!/bin/sh\nexec "$REPLAY_TEST_PYTHON" "$@"\n')
        interpreter.chmod(0o755)
        gh = directory / 'gh'
        gh.write_text(
            '#!/bin/sh\n'
            '[ "$*" = "api graphql --input -" ] || exit 2\n'
            'cat > "$REPLAY_TEST_REQUEST"\n'
            'cat "$REPLAY_TEST_TREE"\n'
        )
        gh.chmod(0o755)
        self.enterContext(
            mock.patch.object(landed_ci.sys, 'executable', str(interpreter))
        )
        self.enterContext(mock.patch.object(landed_ci, 'SCRIPTS', SCRIPTS))
        self.enterContext(
            mock.patch.dict(
                os.environ,
                {
                    'REPLAY_TEST_PYTHON': original_interpreter,
                    'REPLAY_TEST_TREE': str(self.fixture),
                    'REPLAY_TEST_REQUEST': str(self.request),
                },
            )
        )

    def select(self, files, *, changed_files=None, workflows=None, paused=True):
        workflows = WORKFLOWS if workflows is None else workflows
        self.fixture.write_text(json.dumps(tree_response(workflows, paused)))
        replay = {
            'number': 123,
            'changed_files': len(files)
            if changed_files is None
            else changed_files,
        }
        with mock.patch.object(landed_ci, 'api', return_value=files) as api:
            try:
                return landed_ci.expected_workflows(BASE, replay)
            finally:
                api.assert_called_once_with('pulls/123/files?per_page=100')

    def test_release_lanes_and_path_filters_use_only_the_immutable_base(self):
        selected = self.select([{'filename': 'docs/guide.md'}])
        self.assertEqual(selected, ['.github/workflows/pre-commit.yml'])
        self.assertEqual(
            json.loads(self.request.read_text())['variables'],
            {
                'owner': 'kubeflow',
                'repo': 'pipelines',
                'expression': f'{BASE}:.github/workflows',
            },
        )
        # A rename outside the policy directory must still match lanes for either path.
        selected = self.select(
            [{'previous_filename': 'release/old', 'filename': 'manifests/new'}]
        )
        self.assertEqual(
            selected,
            [
                '.github/workflows/backend.yml',
                '.github/workflows/build-tools-images.yml',
                '.github/workflows/postgresql-image-upgrade.yml',
                '.github/workflows/pre-commit.yml',
            ],
        )

    def test_active_upgrade_suite_stays_required(self):
        self.assertEqual(
            self.select([{'filename': 'docs/guide.md'}], paused=False),
            [
                '.github/workflows/pre-commit.yml',
                '.github/workflows/upgrade-test.yml',
            ],
        )

    def test_precommit_must_apply_to_release(self):
        workflows = {
            **WORKFLOWS,
            'pre-commit.yml': 'pull_request: {branches: [main]}',
        }
        run_command = subprocess.check_output
        with (
            mock.patch.object(
                subprocess,
                'check_output',
                side_effect=lambda *a, **kw: run_command(
                    *a, **kw, stderr=subprocess.PIPE
                ),
            ),
            self.assertRaises(subprocess.CalledProcessError) as error,
        ):
            self.select([{'filename': 'release/tools.py'}], workflows=workflows)
        self.assertIn(
            'Incomplete release PR workflow coverage', error.exception.stderr
        )

    def test_truncated_changes_and_github_renames_fail_before_selection(self):
        for files, count, reason in (
            (
                [{'filename': 'release/version'}],
                2,
                'Incomplete replay change list',
            ),
            (
                [
                    {
                        'filename': 'docs/new',
                        'previous_filename': '.github/actions/old',
                    }
                ],
                1,
                'configuration changes',
            ),
            (
                [
                    {
                        'filename': '.github/workflows/new.yml',
                        'previous_filename': 'docs/new',
                    }
                ],
                1,
                'configuration changes',
            ),
        ):
            with (
                self.subTest(files=files),
                self.assertRaisesRegex(RuntimeError, reason),
            ):
                self.select(files, changed_files=count)
            self.assertFalse(self.request.exists())

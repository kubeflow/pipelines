from __future__ import annotations

import contextlib
import io
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import ready_command

REPO_ROOT = Path(__file__).resolve().parents[2]
ENV = {
    'COMMENT_BODY': '/ready',
    'COMMENT_ID': '7',
    'COMMENTER': 'Approver',
    'GH_REPO': 'kubeflow/pipelines',
    'ISSUE_NUMBER': '42',
}
LABEL_CALL = [
    'gh',
    'api',
    '--method',
    'POST',
    'repos/kubeflow/pipelines/issues/42/labels',
    '-f',
    'labels[]=ready',
]
REACTION_CALL = [
    'gh',
    'api',
    '--method',
    'POST',
    'repos/kubeflow/pipelines/issues/comments/7/reactions',
    '-f',
    'content=+1',
]
DENIAL_CALL = [
    'gh',
    'api',
    '--method',
    'POST',
    'repos/kubeflow/pipelines/issues/comments/7/reactions',
    '-f',
    'content=-1',
]


def write_owners(root: Path, files: dict[str, str]) -> None:
    """Create OWNERS files under root from relative path to YAML text."""
    for relative_path, text in files.items():
        path = root / relative_path
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding='utf-8')


class HasReadyCommandTest(unittest.TestCase):
    def test_matches_command_on_its_own_line(self):
        for body in ('/ready', '  /ready \r\n', 'LGTM\r\n/ready\r\nThanks'):
            with self.subTest(body=body):
                self.assertTrue(ready_command.has_ready_command(body))

    def test_ignores_command_inside_prose_or_longer_words(self):
        for body in ('is this /ready?', 'not /ready', '/ready-ish', '/readyy'):
            with self.subTest(body=body):
                self.assertFalse(ready_command.has_ready_command(body))


class LoadOwnersApproversTest(unittest.TestCase):
    def setUp(self):
        temp_dir = tempfile.TemporaryDirectory()
        self.addCleanup(temp_dir.cleanup)
        self.root = Path(temp_dir.name)

    def test_collects_approvers_from_every_owners_file_but_not_reviewers(self):
        write_owners(
            self.root,
            {
                'OWNERS': 'approvers:\n  - RootMaintainer\n',
                'sdk/OWNERS': (
                    'approvers:\n  - SdkApprover\nreviewers:\n  - SdkReviewer\n'
                ),
                'docs/OWNERS': 'reviewers:\n  - DocsReviewer\n',
            },
        )
        self.assertEqual(
            ready_command.load_owners_approvers(self.root),
            {'rootmaintainer', 'sdkapprover'},
        )

    def test_rejects_malformed_owners_files(self):
        cases = {
            'invalid YAML': 'approvers: [unterminated\n',
            'duplicate key': 'approvers:\n  - a\napprovers:\n  - b\n',
            'non-list approvers': 'approvers: someone\n',
            'invalid login': 'approvers:\n  - "not a login"\n',
        }
        for name, text in cases.items():
            with self.subTest(case=name):
                write_owners(self.root, {'OWNERS': text})
                with self.assertRaises(RuntimeError):
                    ready_command.load_owners_approvers(self.root)

    def test_rejects_tree_without_approvers(self):
        write_owners(self.root, {'OWNERS': 'reviewers:\n  - someone\n'})
        with self.assertRaises(RuntimeError):
            ready_command.load_owners_approvers(self.root)

    def test_repository_owners_include_root_and_subdirectory_approvers(self):
        tracked = subprocess.run(
            ['git', 'ls-files', 'OWNERS', '*/OWNERS'],
            cwd=REPO_ROOT,
            check=True,
            capture_output=True,
            text=True,
        ).stdout.split()
        # Mirror the workflow's sparse checkout of tracked OWNERS files.
        for relative_path in tracked:
            destination = self.root / relative_path
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(REPO_ROOT / relative_path, destination)

        approvers = ready_command.load_owners_approvers(self.root)

        root_approvers = ready_command.load_owners_approvers(
            self._only(REPO_ROOT / 'OWNERS')
        )
        self.assertLessEqual(root_approvers, approvers)
        self.assertGreater(len(approvers), len(root_approvers))

    def _only(self, owners_path: Path) -> Path:
        """Return a directory containing just the given OWNERS file."""
        directory = self.root / '_single'
        directory.mkdir()
        shutil.copyfile(owners_path, directory / 'OWNERS')
        return directory


class MainTest(unittest.TestCase):
    def setUp(self):
        temp_dir = tempfile.TemporaryDirectory()
        self.addCleanup(temp_dir.cleanup)
        write_owners(
            Path(temp_dir.name),
            {
                'OWNERS': 'approvers:\n  - approver\n',
                'sdk/OWNERS': 'reviewers:\n  - reviewer\n',
            },
        )
        self.env = {**ENV, 'OWNERS_ROOT': temp_dir.name}

    def run_main(self, **overrides: str) -> list[list[str]]:
        """Run main with a stubbed gh CLI and return the gh calls made."""
        with (
            mock.patch.dict(os.environ, {**self.env, **overrides}),
            mock.patch('ready_command.subprocess.run') as run,
        ):
            self.assertEqual(ready_command.main(), 0)
        return [call.args[0] for call in run.call_args_list]

    def test_approver_applies_label_and_reacts_case_insensitively(self):
        self.assertEqual(self.run_main(), [LABEL_CALL, REACTION_CALL])

    def test_reviewer_and_other_users_are_denied_with_reaction_only(self):
        for commenter in ('reviewer', 'outsider'):
            with self.subTest(commenter=commenter):
                self.assertEqual(
                    self.run_main(COMMENTER=commenter), [DENIAL_CALL]
                )

    def test_hostile_comment_text_is_never_executed_forwarded_or_logged(self):
        payload = (
            '$(touch /tmp/pwned) `id` ; rm -rf / && echo ${{ secrets.X }}\n'
            '::stop-commands::token\n::add-mask::x'
        )
        stdout = io.StringIO()
        with (
            mock.patch.dict(
                os.environ, {**self.env, 'COMMENT_BODY': f'/ready\n{payload}'}
            ),
            mock.patch('ready_command.subprocess.run') as run,
            contextlib.redirect_stdout(stdout),
        ):
            ready_command.main()

        self.assertEqual(
            [call.args[0] for call in run.call_args_list],
            [LABEL_CALL, REACTION_CALL],
        )
        for call in run.call_args_list:
            self.assertNotIn('shell', call.kwargs)
        # Workflow commands are parsed from stdout, so never echo the body.
        self.assertNotIn('::', stdout.getvalue())
        self.assertNotIn('pwned', stdout.getvalue())

    def test_non_numeric_event_ids_fail_before_any_api_call(self):
        for name in ('ISSUE_NUMBER', 'COMMENT_ID'):
            with (
                self.subTest(name=name),
                mock.patch.dict(os.environ, {**self.env, name: '1/../../evil'}),
                mock.patch('ready_command.subprocess.run') as run,
            ):
                with self.assertRaises(ValueError):
                    ready_command.main()
                run.assert_not_called()

    def test_comment_without_command_makes_no_api_calls(self):
        self.assertEqual(self.run_main(COMMENT_BODY='is this /ready?'), [])

    def test_owners_errors_fail_before_any_api_call(self):
        write_owners(
            Path(self.env['OWNERS_ROOT']), {'OWNERS': 'approvers: x\n'}
        )
        with (
            mock.patch.dict(os.environ, self.env),
            mock.patch('ready_command.subprocess.run') as run,
        ):
            with self.assertRaises(RuntimeError):
                ready_command.main()
        run.assert_not_called()

    def test_label_api_failure_propagates(self):
        with (
            mock.patch.dict(os.environ, self.env),
            mock.patch(
                'ready_command.subprocess.run',
                side_effect=subprocess.CalledProcessError(1, 'gh'),
            ),
        ):
            with self.assertRaises(subprocess.CalledProcessError):
                ready_command.main()


if __name__ == '__main__':
    unittest.main()

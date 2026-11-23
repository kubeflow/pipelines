from __future__ import annotations

import json
import subprocess
import unittest
from unittest import mock

import bot_branch_cleanup

REPO = 'kubeflow/pipelines'
SHA = 'a' * 40
BRANCH = 'dependabot/npm_and_yarn/frontend/lodash-4.17.21'
EVENT = {
    'repo': REPO,
    'pr_number': '42',
    'author': 'dependabot[bot]',
    'branch': BRANCH,
    'head_sha': SHA,
    'head_repo': REPO,
    'merged': False,
}
PULL_PATH = f'repos/{REPO}/pulls/42'
REF_PATH = f'repos/{REPO}/git/ref/heads/{BRANCH}'
DELETE_PATH = f'repos/{REPO}/git/refs/heads/{BRANCH}'


class FakeGh:
    """Answer `gh api` calls from a table of path -> response."""

    def __init__(self, **overrides):
        self.responses = {
            PULL_PATH: {'state': 'closed', 'merged': False},
            REF_PATH: {
                'ref': f'refs/heads/{BRANCH}',
                'object': {'sha': SHA},
            },
            f'head=kubeflow:{BRANCH}': [],
            f'base={BRANCH}': [],
            DELETE_PATH: None,
        }
        self.responses.update(overrides)
        self.calls = []

    def __call__(self, command, **kwargs):
        self.assert_gh_api(command, kwargs)
        args = tuple(command[2:])
        self.calls.append(args)
        key = next(
            (arg for arg in args if arg.startswith(('head=', 'base='))),
            args[-1],
        )
        response = self.responses[key]
        if isinstance(response, tuple):
            returncode, stderr = response
            return subprocess.CompletedProcess(command, returncode, '', stderr)
        stdout = '' if response is None else json.dumps(response)
        return subprocess.CompletedProcess(command, 0, stdout, '')

    @staticmethod
    def assert_gh_api(command, kwargs):
        # No shell, bounded runtime, captured output.
        assert command[:2] == ['gh', 'api'], command
        assert 'shell' not in kwargs
        assert kwargs['timeout'] == 30
        assert kwargs['capture_output'] is True

    @property
    def deleted(self):
        return any(call[:2] == ('--method', 'DELETE') for call in self.calls)


def run_cleanup(fake, **event):
    with mock.patch.object(bot_branch_cleanup.subprocess, 'run', fake):
        return bot_branch_cleanup.cleanup(**{**EVENT, **event})


class IsCleanupCandidateTest(unittest.TestCase):
    def test_accepts_unmerged_bot_branches_in_this_repository(self):
        for author, branch in (
            ('dependabot[bot]', 'dependabot/go_modules/x-1.2.3'),
            ('copybara-service[bot]', 'test_971831486'),
        ):
            with self.subTest(author=author):
                self.assertTrue(
                    bot_branch_cleanup.is_cleanup_candidate(
                        author, branch, REPO, REPO, False
                    )
                )

    def test_rejects_everything_else(self):
        cases = {
            'merged': ('dependabot[bot]', 'dependabot/x', REPO, True),
            'fork head': (
                'dependabot[bot]',
                'dependabot/x',
                'a/pipelines',
                False,
            ),
            'human author': ('someone', 'dependabot/x', REPO, False),
            'wrong prefix for bot': ('dependabot[bot]', 'test_1', REPO, False),
            'copybara non-test branch': (
                'copybara-service[bot]',
                'main',
                REPO,
                False,
            ),
            'lookalike login': ('dependabot', 'dependabot/x', REPO, False),
        }
        for name, (author, branch, head_repo, merged) in cases.items():
            with self.subTest(name):
                self.assertFalse(
                    bot_branch_cleanup.is_cleanup_candidate(
                        author, branch, head_repo, REPO, merged
                    )
                )


class IsSafeBranchNameTest(unittest.TestCase):
    def test_accepts_bot_branch_names(self):
        for name in (
            BRANCH,
            'dependabot/github_actions/dot-github/actions/setup-go/'
            'release-2.18/github_actions-a9a40029ed',
            'dependabot/pip/sdk/python/protobuf-gte-4.21.1-and-lt-7.0',
            'test_971831486',
        ):
            with self.subTest(name=name):
                self.assertTrue(bot_branch_cleanup.is_safe_branch_name(name))

    def test_rejects_names_that_could_escape_the_path_or_log(self):
        for name in (
            '',
            '../main',
            'dependabot/../main',
            'a//b',
            'a/',
            'a.lock',
            '::error::x',
            'a b',
            'a?b',
            'a#b',
            'a%2Fb',
            '-x',
            'a\nb',
        ):
            with self.subTest(name=name):
                self.assertFalse(bot_branch_cleanup.is_safe_branch_name(name))


class CleanupTest(unittest.TestCase):
    def test_deletes_branch_left_by_closed_bot_pull_request(self):
        fake = FakeGh()
        message = run_cleanup(fake)
        self.assertTrue(fake.deleted)
        self.assertIn(f'Deleted {BRANCH}', message)
        self.assertIn('Restore', message)
        self.assertEqual(fake.calls[-1], ('--method', 'DELETE', DELETE_PATH))

    def test_deletes_copybara_branch(self):
        branch = 'test_971831486'
        fake = FakeGh(
            **{
                f'repos/{REPO}/git/ref/heads/{branch}': {
                    'ref': f'refs/heads/{branch}',
                    'object': {'sha': SHA},
                },
                f'head=kubeflow:{branch}': [],
                f'base={branch}': [],
                f'repos/{REPO}/git/refs/heads/{branch}': None,
            }
        )
        run_cleanup(fake, author='copybara-service[bot]', branch=branch)
        self.assertTrue(fake.deleted)

    def test_ignores_non_candidates_without_calling_the_api(self):
        fake = FakeGh()
        for event in (
            {'merged': True},
            {'author': 'someone'},
            {'head_repo': 'fork/pipelines'},
        ):
            with self.subTest(event=event):
                run_cleanup(fake, **event)
        self.assertEqual(fake.calls, [])

    def test_keeps_branch_when_pull_request_was_reopened_or_merged(self):
        for pull in (
            {'state': 'open', 'merged': False},
            {'state': 'closed', 'merged': True},
        ):
            with self.subTest(pull=pull):
                fake = FakeGh(**{PULL_PATH: pull})
                self.assertIn('kept', run_cleanup(fake))
                self.assertFalse(fake.deleted)

    def test_keeps_branch_that_moved_after_the_pull_request_closed(self):
        fake = FakeGh(
            **{
                REF_PATH: {
                    'ref': f'refs/heads/{BRANCH}',
                    'object': {'sha': 'b' * 40},
                },
            }
        )
        self.assertIn('moved', run_cleanup(fake))
        self.assertFalse(fake.deleted)

    def test_keeps_branch_used_by_an_open_pull_request(self):
        for key in (f'head=kubeflow:{BRANCH}', f'base={BRANCH}'):
            with self.subTest(key=key):
                fake = FakeGh(**{key: [{'number': 7}]})
                self.assertIn('open pull request', run_cleanup(fake))
                self.assertFalse(fake.deleted)

    def test_open_pull_request_queries_are_exact_and_bounded(self):
        fake = FakeGh()
        run_cleanup(fake)
        queries = [
            call
            for call in fake.calls
            if call[-1].endswith('/pulls') or 'state=open' in call
        ]
        self.assertEqual(len(queries), 2)
        for call in queries:
            self.assertEqual(
                call[:3], ('--method', 'GET', f'repos/{REPO}/pulls')
            )
            self.assertIn('state=open', call)
            self.assertIn('per_page=1', call)

    def test_already_deleted_branch_is_success(self):
        fake = FakeGh(**{REF_PATH: (1, 'gh: Not Found (HTTP 404)')})
        self.assertIn('already deleted', run_cleanup(fake))
        self.assertFalse(fake.deleted)

    def test_concurrent_deletion_is_success(self):
        for response in (
            (1, 'gh: Not Found (HTTP 404)'),
            (1, 'gh: Reference does not exist (HTTP 422)'),
        ):
            with self.subTest(response=response):
                fake = FakeGh(**{DELETE_PATH: response})
                self.assertIn('Deleted', run_cleanup(fake))

    def test_other_api_errors_fail_the_job(self):
        fake = FakeGh(**{DELETE_PATH: (1, 'gh: Forbidden (HTTP 403)')})
        with self.assertRaisesRegex(RuntimeError, 'HTTP 403'):
            run_cleanup(fake)

    def test_rejects_malformed_event_data_before_any_api_call(self):
        for event in (
            {'branch': 'dependabot/../main'},
            {'pr_number': '42/../1'},
            {'head_sha': 'not-a-sha'},
        ):
            with self.subTest(event=event):
                fake = FakeGh()
                with self.assertRaises(ValueError):
                    run_cleanup(fake, **event)
                self.assertEqual(fake.calls, [])

    def test_branch_name_is_url_encoded_in_api_paths(self):
        branch = 'dependabot/pip/a+b-1.0'
        encoded = 'dependabot/pip/a%2Bb-1.0'
        self.assertEqual(
            bot_branch_cleanup.branch_path(REPO, branch, lookup=True),
            f'repos/{REPO}/git/ref/heads/{encoded}',
        )
        self.assertEqual(
            bot_branch_cleanup.branch_path(REPO, branch),
            f'repos/{REPO}/git/refs/heads/{encoded}',
        )


class MainTest(unittest.TestCase):
    def test_reads_event_from_environment(self):
        env = {
            'GH_REPO': REPO,
            'PR_NUMBER': '42',
            'PR_AUTHOR': 'dependabot[bot]',
            'HEAD_REF': BRANCH,
            'HEAD_SHA': SHA,
            'HEAD_REPO': REPO,
            'PR_MERGED': 'false',
        }
        fake = FakeGh()
        with (
            mock.patch.dict(bot_branch_cleanup.os.environ, env, clear=True),
            mock.patch.object(bot_branch_cleanup.subprocess, 'run', fake),
            mock.patch('builtins.print'),
        ):
            self.assertEqual(bot_branch_cleanup.main(), 0)
        self.assertTrue(fake.deleted)

    def test_merged_flag_must_be_exactly_false_to_delete(self):
        for merged in ('true', '', 'False', 'null'):
            with self.subTest(merged=merged):
                env = {
                    'GH_REPO': REPO,
                    'PR_NUMBER': '42',
                    'PR_AUTHOR': 'dependabot[bot]',
                    'HEAD_REF': BRANCH,
                    'HEAD_SHA': SHA,
                    'HEAD_REPO': REPO,
                    'PR_MERGED': merged,
                }
                fake = FakeGh()
                with (
                    mock.patch.dict(
                        bot_branch_cleanup.os.environ, env, clear=True
                    ),
                    mock.patch.object(
                        bot_branch_cleanup.subprocess, 'run', fake
                    ),
                    mock.patch('builtins.print'),
                ):
                    bot_branch_cleanup.main()
                self.assertEqual(fake.calls, [])


if __name__ == '__main__':
    unittest.main()

from __future__ import annotations

import json
import subprocess
import unittest
from unittest import mock

import bot_branch_cleanup

REPO = 'kubeflow/pipelines'
REPO_ID = 'R_kgDOB-6BOA'
SHA = 'a' * 40
OTHER_SHA = 'b' * 40
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


class FakeGitHub:
    """Simulate the GitHub API calls the cleanup makes through `gh api`.

    Branch tips are state, and updateRefs enforces beforeOid the way
    GitHub does, so tests can move or delete a branch between the
    cleanup's checks and its deletion.
    """

    def __init__(self, branch=BRANCH, pull=None, open_heads=(), open_bases=()):
        self.branch = branch
        self.tips = {branch: SHA}
        self.pull = pull or {
            'state': 'closed',
            'merged': False,
            'base': {'repo': {'node_id': REPO_ID}},
        }
        self.open_heads = set(open_heads)
        self.open_bases = set(open_bases)
        self.after_lookup = None
        self.graphql_error = None
        self.calls = []

    def __call__(self, command, **kwargs):
        # No shell, bounded runtime, captured output.
        assert command[:2] == ['gh', 'api'], command
        assert 'shell' not in kwargs
        assert kwargs['timeout'] == 30
        assert kwargs['capture_output'] is True
        args = tuple(command[2:])
        self.calls.append(args)
        if args[0] == 'graphql':
            return self.update_refs(command, args)
        path = args[2] if args[0] == '--method' else args[-1]
        if path == f'repos/{REPO}/pulls/42':
            return self.ok(command, self.pull)
        if path.startswith(f'repos/{REPO}/git/ref/heads/'):
            tip = self.tips.get(self.branch)
            if self.after_lookup:
                self.after_lookup(self)
            if tip is None:
                return self.fail(command, 'gh: Not Found (HTTP 404)')
            return self.ok(command, {'object': {'sha': tip}})
        if path == f'repos/{REPO}/pulls':
            fields = dict(arg.split('=', 1) for arg in args if '=' in arg)
            head = fields.get('head', '').removeprefix('kubeflow:')
            used = (
                head in self.open_heads or fields.get('base') in self.open_bases
            )
            return self.ok(command, [{'number': 7}] if used else [])
        raise AssertionError(f'unexpected gh api call: {args}')

    def update_refs(self, command, args):
        fields = dict(arg.split('=', 1) for arg in args[2::2])
        assert 'updateRefs' in fields['query']
        assert fields['repositoryId'] == REPO_ID
        if self.graphql_error:
            return self.fail(command, self.graphql_error)
        branch = fields['name'].removeprefix('refs/heads/')
        if self.tips.get(branch) != fields['beforeOid']:
            # Real GitHub returns this same generic error for a stale or
            # missing ref.
            return self.fail(
                command,
                'gh: Something went wrong while executing your query.',
            )
        assert fields['afterOid'] == '0' * 40
        del self.tips[branch]
        return self.ok(command, {'data': {'updateRefs': {}}})

    @staticmethod
    def ok(command, body):
        return subprocess.CompletedProcess(command, 0, json.dumps(body), '')

    @staticmethod
    def fail(command, stderr):
        return subprocess.CompletedProcess(command, 1, '', stderr)

    @property
    def update_requests(self):
        return [call for call in self.calls if call[0] == 'graphql']

    @property
    def deleted(self):
        return self.branch not in self.tips


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
        fake = FakeGitHub()
        message = run_cleanup(fake)
        self.assertTrue(fake.deleted)
        self.assertIn(f'Deleted {BRANCH}', message)
        self.assertIn('Restore', message)

    def test_deletion_is_conditional_on_the_closed_head_sha(self):
        fake = FakeGitHub()
        run_cleanup(fake)
        (request,) = fake.update_requests
        self.assertIn(f'name=refs/heads/{BRANCH}', request)
        self.assertIn(f'beforeOid={SHA}', request)
        self.assertIn('afterOid=' + '0' * 40, request)
        self.assertIn(f'repositoryId={REPO_ID}', request)

    def test_push_after_the_tip_check_survives(self):
        # Regression: a push landing after the checks used to be deleted
        # along with the branch, and the log named the old SHA.
        fake = FakeGitHub()

        def push(github):
            github.tips[BRANCH] = OTHER_SHA
            github.after_lookup = None

        fake.after_lookup = push
        message = run_cleanup(fake)
        self.assertEqual(fake.tips[BRANCH], OTHER_SHA)
        self.assertEqual(len(fake.update_requests), 1)
        self.assertIn('moved or was deleted during cleanup', message)
        self.assertNotIn('Deleted', message)

    def test_deletion_by_someone_else_during_cleanup_is_success(self):
        fake = FakeGitHub()

        def delete(github):
            del github.tips[BRANCH]
            github.after_lookup = None

        fake.after_lookup = delete
        self.assertIn('moved or was deleted', run_cleanup(fake))

    def test_rejected_update_on_unchanged_branch_fails_the_job(self):
        fake = FakeGitHub()
        fake.graphql_error = 'gh: GraphQL: Resource not accessible'
        with self.assertRaisesRegex(RuntimeError, 'not accessible'):
            run_cleanup(fake)
        self.assertEqual(fake.tips[BRANCH], SHA)

    def test_deletes_copybara_branch(self):
        branch = 'test_971831486'
        fake = FakeGitHub(branch=branch)
        run_cleanup(fake, author='copybara-service[bot]', branch=branch)
        self.assertTrue(fake.deleted)

    def test_ignores_non_candidates_without_calling_the_api(self):
        fake = FakeGitHub()
        for event in (
            {'merged': True},
            {'author': 'someone'},
            {'head_repo': 'fork/pipelines'},
        ):
            with self.subTest(event=event):
                run_cleanup(fake, **event)
        self.assertEqual(fake.calls, [])

    def test_keeps_branch_when_pull_request_was_reopened_or_merged(self):
        for state, merged in (('open', False), ('closed', True)):
            with self.subTest(state=state, merged=merged):
                fake = FakeGitHub(pull={'state': state, 'merged': merged})
                self.assertIn('kept', run_cleanup(fake))
                self.assertFalse(fake.deleted)
                self.assertEqual(fake.update_requests, [])

    def test_keeps_branch_that_moved_after_the_pull_request_closed(self):
        fake = FakeGitHub()
        fake.tips[BRANCH] = OTHER_SHA
        self.assertIn('moved', run_cleanup(fake))
        self.assertEqual(fake.tips[BRANCH], OTHER_SHA)
        self.assertEqual(fake.update_requests, [])

    def test_keeps_branch_used_by_an_open_pull_request(self):
        for usage in ({'open_heads': [BRANCH]}, {'open_bases': [BRANCH]}):
            with self.subTest(usage=usage):
                fake = FakeGitHub(**usage)
                self.assertIn('open pull request', run_cleanup(fake))
                self.assertFalse(fake.deleted)
                self.assertEqual(fake.update_requests, [])

    def test_open_pull_request_queries_are_exact_and_bounded(self):
        fake = FakeGitHub()
        run_cleanup(fake)
        queries = [call for call in fake.calls if 'state=open' in call]
        self.assertEqual(len(queries), 2)
        for call in queries:
            self.assertEqual(
                call[:3], ('--method', 'GET', f'repos/{REPO}/pulls')
            )
            self.assertIn('per_page=1', call)
        self.assertIn(f'head=kubeflow:{BRANCH}', queries[0])
        self.assertIn(f'base={BRANCH}', queries[1])

    def test_already_deleted_branch_is_success(self):
        fake = FakeGitHub()
        del fake.tips[BRANCH]
        self.assertIn('already deleted', run_cleanup(fake))
        self.assertEqual(fake.update_requests, [])

    def test_rejects_malformed_event_data_before_any_api_call(self):
        for event in (
            {'branch': 'dependabot/../main'},
            {'pr_number': '42/../1'},
            {'head_sha': 'not-a-sha'},
        ):
            with self.subTest(event=event):
                fake = FakeGitHub()
                with self.assertRaises(ValueError):
                    run_cleanup(fake, **event)
                self.assertEqual(fake.calls, [])

    def test_branch_name_is_url_encoded_in_the_lookup_path(self):
        self.assertEqual(
            bot_branch_cleanup.branch_path(REPO, 'dependabot/pip/a+b-1.0'),
            f'repos/{REPO}/git/ref/heads/dependabot/pip/a%2Bb-1.0',
        )


class MainTest(unittest.TestCase):
    def env(self, merged):
        return {
            'GH_REPO': REPO,
            'PR_NUMBER': '42',
            'PR_AUTHOR': 'dependabot[bot]',
            'HEAD_REF': BRANCH,
            'HEAD_SHA': SHA,
            'HEAD_REPO': REPO,
            'PR_MERGED': merged,
        }

    def run_main(self, fake, merged):
        with (
            mock.patch.dict(
                bot_branch_cleanup.os.environ, self.env(merged), clear=True
            ),
            mock.patch.object(bot_branch_cleanup.subprocess, 'run', fake),
            mock.patch('builtins.print'),
        ):
            return bot_branch_cleanup.main()

    def test_reads_event_from_environment(self):
        fake = FakeGitHub()
        self.assertEqual(self.run_main(fake, 'false'), 0)
        self.assertTrue(fake.deleted)

    def test_merged_flag_must_be_exactly_false_to_delete(self):
        for merged in ('true', '', 'False', 'null'):
            with self.subTest(merged=merged):
                fake = FakeGitHub()
                self.run_main(fake, merged)
                self.assertEqual(fake.calls, [])


if __name__ == '__main__':
    unittest.main()

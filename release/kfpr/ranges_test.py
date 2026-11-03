"""Real-history tests for explicit release ranges."""

import os
import subprocess
import unittest
from dataclasses import FrozenInstanceError
from pathlib import Path
from tempfile import TemporaryDirectory

from kfpr import ranges


class ReleaseRangeTest(unittest.TestCase):
    def setUp(self):
        source = Path(__file__).resolve().parents[2]
        identity = {}
        for field in ('name', 'email'):
            result = subprocess.run(
                ['git', 'config', f'user.{field}'],
                cwd=source,
                capture_output=True,
                text=True,
            )
            if result.returncode or not result.stdout.strip():
                self.skipTest(
                    f'Git user.{field} must be configured for fixtures'
                )
            identity[field] = result.stdout.strip()
        temporary = TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.git('init', '-b', 'main')
        for field, value in identity.items():
            self.git('config', f'user.{field}', value)
        self.git('config', 'commit.gpgsign', 'false')
        self.base = self.commit('chore: base')
        self.git('tag', 'sdk-1.0.0')
        self.first = self.commit('fix(sdk): first fix')
        self.target = self.commit(
            'feat(api): new API\n\nDetails with multiple lines.\n'
            '\nBREAKING CHANGE: Upgrade clients before deploying.'
        )
        self.git('tag', 'sdk-1.1.0')

    def git(self, *args):
        return subprocess.run(
            ['git', *args],
            cwd=self.root,
            check=True,
            capture_output=True,
            text=True,
            env={
                **os.environ,
                'GIT_AUTHOR_DATE': '2025-01-02T03:04:05+00:00',
                'GIT_COMMITTER_DATE': '2025-01-03T03:04:05+00:00',
            },
        ).stdout.strip()

    def commit(self, message):
        self.git('commit', '--allow-empty', '-s', '-m', message)
        return self.git('rev-parse', 'HEAD')

    def test_resolves_immutable_boundaries_and_complete_messages(self):
        selected = ranges.resolve_range(self.root, 'sdk-1.0.0', 'sdk-1.1.0')
        self.assertEqual(
            selected.as_dict(),
            {
                'base_ref': 'sdk-1.0.0',
                'target_ref': 'sdk-1.1.0',
                'base_sha': self.base,
                'target_sha': self.target,
            },
        )
        self.assertTrue(
            selected.compare_url.endswith(f'/{self.base}...{self.target}')
        )
        with self.assertRaises(FrozenInstanceError):
            selected.target_sha = self.base
        commits = ranges.read_commits(self.root, selected)
        self.assertEqual(
            [commit.sha for commit in commits], [self.first, self.target]
        )
        self.assertIn('BREAKING CHANGE: Upgrade clients', commits[1].message)
        self.assertIn('\n\nDetails with multiple lines.\n', commits[1].message)

    def test_moving_head_and_tag_does_not_change_resolved_history(self):
        selected = ranges.resolve_range(self.root, 'sdk-1.0.0', 'HEAD')
        before = ranges.read_commits(self.root, selected)
        self.commit('fix: later change')
        self.git('tag', '--force', 'sdk-1.0.0')
        self.assertEqual(before, ranges.read_commits(self.root, selected))

    def test_divergent_release_branches_share_history(self):
        self.git('checkout', '-b', 'previous-release', self.first)
        previous_release = self.commit('chore(release): previous version bump')
        self.git('tag', 'sdk-1.0.1')
        selected = ranges.resolve_range(self.root, 'sdk-1.0.1', 'sdk-1.1.0')
        self.assertEqual(selected.base_sha, previous_release)
        self.assertEqual(selected.target_sha, self.target)
        self.assertEqual(
            [commit.sha for commit in ranges.read_commits(self.root, selected)],
            [self.target],
        )

    def test_reversed_range_fails(self):
        with self.assertRaisesRegex(ValueError, 'precedes base'):
            ranges.resolve_range(self.root, self.target, self.base)

    def test_unrelated_histories_fail(self):
        self.git('checkout', '--orphan', 'unrelated')
        unrelated = self.commit('chore: unrelated history')
        with self.assertRaisesRegex(ValueError, 'must share history'):
            ranges.resolve_range(self.root, unrelated, self.target)

    def test_missing_or_invalid_refs_fail_actionably(self):
        for ref in (
            'missing-tag',
            '',
            ' ',
            '--all',
            '--output=owned',
            'HEAD\x00',
            'HEAD; touch owned',
            '$(touch owned)',
        ):
            with self.subTest(ref=ref):
                with self.assertRaises(ValueError):
                    ranges.resolve_range(self.root, ref, self.target)
                with self.assertRaises(ValueError):
                    ranges.resolve_range(self.root, self.base, ref)
        self.assertFalse((self.root / 'owned').exists())

    def test_empty_range_is_explicit(self):
        selected = ranges.resolve_range(self.root, self.base, self.base)
        self.assertEqual(ranges.read_commits(self.root, selected), [])

    def test_empty_commit_message_preserves_record_boundaries(self):
        self.git('commit', '--allow-empty', '--allow-empty-message', '-m', '')
        selected = ranges.resolve_range(self.root, self.target, 'HEAD')
        commits = ranges.read_commits(self.root, selected)
        self.assertEqual(len(commits), 1)
        self.assertEqual(commits[0].message, '')

    def test_commit_date_uses_commit_timestamp(self):
        self.assertEqual(
            ranges.commit_date(self.root, self.target), '2025-01-03'
        )

    def test_invalid_saved_sha_cannot_be_used_as_git_option(self):
        selected = ranges.ReleaseRange('base', 'target', '--all', self.target)
        with self.assertRaisesRegex(ValueError, 'resolved full commit SHA'):
            ranges.read_commits(self.root, selected)
        with self.assertRaisesRegex(ValueError, 'resolved full commit SHA'):
            ranges.commit_date(self.root, '--all')

    def test_unavailable_history_fails_instead_of_returning_empty_notes(self):
        selected = ranges.ReleaseRange('base', 'target', '0' * 40, self.target)
        with self.assertRaisesRegex(ValueError, 'fetch'):
            ranges.read_commits(self.root, selected)


if __name__ == '__main__':
    unittest.main()

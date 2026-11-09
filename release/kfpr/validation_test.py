#!/usr/bin/env python3
"""Tests for release answer and state validation."""

import unittest

from kfpr import core


class ParseGithubOwnerTest(unittest.TestCase):
    def test_normalize_fork_remote_expands_bare_username(self):
        self.assertEqual(
            core.normalize_fork_remote('droctothorpe'),
            'https://github.com/droctothorpe/pipelines.git',
        )

    def test_normalize_fork_remote_keeps_ssh_remote(self):
        self.assertEqual(
            core.normalize_fork_remote('git@github.com:user123/pipelines.git'),
            'git@github.com:user123/pipelines.git',
        )

    def test_normalize_fork_remote_keeps_https_remote(self):
        self.assertEqual(
            core.normalize_fork_remote(
                'https://github.com/user123/pipelines.git'
            ),
            'https://github.com/user123/pipelines.git',
        )

    def test_normalize_fork_remote_rejects_at_username(self):
        with self.assertRaises(ValueError):
            core.normalize_fork_remote('@droctothorpe')

    def test_normalize_fork_remote_rejects_upstream_repository(self):
        for fork_remote in (
            'kubeflow',
            'Kubeflow',
            'git@github.com:kubeflow/pipelines.git',
            'https://github.com/Kubeflow/pipelines.git',
        ):
            with self.subTest(fork_remote=fork_remote):
                with self.assertRaises(ValueError):
                    core.normalize_fork_remote(fork_remote)

    def test_parse_ssh_format(self):
        self.assertEqual(
            core.parse_github_owner('git@github.com:user123/pipelines.git'),
            'user123',
        )

    def test_parse_https_format(self):
        self.assertEqual(
            core.parse_github_owner('https://github.com/user456/pipelines.git'),
            'user456',
        )

    def test_parse_invalid_format_raises_value_error(self):
        with self.assertRaises(ValueError) as cm:
            core.parse_github_owner('invalid-remote-url')

        error_msg = str(cm.exception)
        self.assertIn('Cannot parse GitHub owner', error_msg)
        self.assertIn('invalid-remote-url', error_msg)
        self.assertIn('Expected format', error_msg)


if __name__ == '__main__':
    unittest.main()

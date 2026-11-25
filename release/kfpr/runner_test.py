#!/usr/bin/env python3
"""Tests for release command execution."""

import unittest

from kfpr import core


class CommandRunnerTest(unittest.TestCase):
    def test_dry_run_records_command_without_running(self):
        runner = core.CommandRunner(dry_run=True)
        runner.run(['git', 'status'])
        self.assertEqual(runner.commands, [['git', 'status']])


if __name__ == '__main__':
    unittest.main()

# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Verify the maintained viewer's loader default and argument forwarding."""

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
WRAPPER = ROOT / 'backend/src/tensorboard/tensorboard.sh'


class TensorboardEntrypointTest(unittest.TestCase):
    def test_default_and_forwarded_arguments(self):
        with tempfile.TemporaryDirectory() as directory:
            cli = Path(directory) / 'tensorboard-cli'
            cli.write_text('#!/bin/sh\nprintf "%s\\n" "$@"\n')
            cli.chmod(0o755)
            for args in [
                [],
                [
                    '--logdir=s3://bucket/path with spaces',
                    '--path_prefix=/tensorboard/viewer/',
                    '--bind_all',
                ],
                ['--load_fast=true'],
            ]:
                with self.subTest(args=args):
                    result = subprocess.run(
                        ['/bin/sh', str(WRAPPER), *args],
                        check=True,
                        text=True,
                        capture_output=True,
                        env={**os.environ, 'PATH': directory},
                    )
                    self.assertEqual(
                        result.stdout.splitlines(), ['--load_fast=false', *args]
                    )


if __name__ == '__main__':
    unittest.main()

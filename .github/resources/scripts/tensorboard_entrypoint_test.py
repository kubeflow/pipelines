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
    def test_combined_series_use_named_directories(self):
        with tempfile.TemporaryDirectory() as directory:
            cli = Path(directory) / 'tensorboard-cli'
            cli.write_text('#!/bin/sh\nprintf "%s\\n" "$@"\n')
            cli.chmod(0o755)
            paths = 'Series1:s3://bucket/path with spaces,Series2:gs://bucket/b'
            result = subprocess.check_output(
                ['/bin/sh', str(WRAPPER), '--logdir=' + paths, '--bind_all'],
                text=True,
                env={**os.environ, 'PATH': directory},
            )
            self.assertEqual(
                result.splitlines(),
                ['--load_fast=false', '--logdir_spec=' + paths, '--bind_all'],
            )

    def test_legacy_and_explicit_endpoints(self):
        with tempfile.TemporaryDirectory() as directory:
            cli = Path(directory) / 'tensorboard-cli'
            cli.write_text('#!/bin/sh\nprintf "%s" "$S3_ENDPOINT"\n')
            cli.chmod(0o755)
            for endpoint, flag, expected in [
                ('minio:9000', '0', 'http://minio:9000'),
                ('minio:9000', '', 'https://minio:9000'),
                ('https://minio:9000', '0', 'https://minio:9000'),
                ('http://minio:9000', '1', 'http://minio:9000'),
                ('', '', ''),
            ]:
                with self.subTest(endpoint=endpoint, flag=flag):
                    result = subprocess.check_output(
                        ['/bin/sh', str(WRAPPER)],
                        text=True,
                        env={
                            **os.environ,
                            'PATH': directory,
                            'S3_ENDPOINT': endpoint,
                            'S3_USE_HTTPS': flag,
                        },
                    )
                    self.assertEqual(result, expected)

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

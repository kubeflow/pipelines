# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Test uv execution with current and previously published generator images."""

import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parents[3] / 'backend/api/hack/uv.sh'


class UvCompatibilityTest(unittest.TestCase):

    def test_bundled_uv_preserves_arguments_without_bootstrap(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binary = root / 'uv'
            binary.write_text('#!/bin/sh\nprintf "%s\\n" "$@"\n')
            binary.chmod(0o755)
            result = subprocess.run(
                ['/bin/sh',
                 str(SCRIPT), 'build', '--out-dir', 'space in path'],
                env={
                    **os.environ, 'PATH': str(root),
                    'HOME': str(root)
                },
                check=True,
                text=True,
                capture_output=True)
            self.assertEqual(result.stdout, 'build\n--out-dir\nspace in path\n')

    def test_old_image_bootstraps_pinned_uv_then_preserves_arguments(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            home = root / 'home'
            target = home / '.local/bin'
            target.mkdir(parents=True)
            python = root / 'python3'
            python.write_text(
                '#!/bin/sh\n'
                'printf "%s\\n" "$@" > "$HOME/bootstrap-args"\n'
                'printf \'#!/bin/sh\\nprintf "%%s\\\\n" "$@"\\n\' '
                '> "$HOME/.local/bin/uv"\n'
                '/bin/chmod +x "$HOME/.local/bin/uv"\n')
            python.chmod(0o755)
            result = subprocess.run(
                ['/bin/sh',
                 str(SCRIPT), 'build', '--out-dir', 'space in path'],
                env={
                    **os.environ, 'PATH': str(root),
                    'HOME': str(home)
                },
                check=True,
                text=True,
                capture_output=True)
            self.assertEqual(result.stdout, 'build\n--out-dir\nspace in path\n')
            self.assertEqual(
                (home / 'bootstrap-args').read_text().splitlines(), [
                    '-m', 'pip', 'install', '--user', '--break-system-packages',
                    'uv==0.13.5'
                ])


if __name__ == '__main__':
    unittest.main()

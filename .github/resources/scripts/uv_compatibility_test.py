# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Test uv execution with current and previously published generator images."""

import os
from pathlib import Path
import shlex
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parents[3] / 'backend/api/hack/uv.sh'


class UvCompatibilityTest(unittest.TestCase):

    def test_kubernetes_make_build_passes_each_local_index_separately(self):
        for target in ('python', 'python-from-source'):
            for enabled in ('true', 'false', ''):
                with self.subTest(target=target, enabled=enabled):
                    result = subprocess.run([
                        'make', '--no-print-directory', '-n', '-o',
                        'fetch-protos', '-o', 'ensure-api-generator-image',
                        target, f'USE_FIND_LINKS={enabled}',
                        'FIND_LINKS_PATH=/first/dist /second/dist /third/dist'
                    ],
                                            cwd=SCRIPT.parents[3] /
                                            'kubernetes_platform',
                                            check=True,
                                            text=True,
                                            capture_output=True)
                    command = result.stdout.replace('\\\n', '')
                    script = shlex.split(command.split('sh -c ', 1)[1])[0]
                    arguments = shlex.split(script.rsplit('&&', 1)[1])
                    expected = [
                        str('/go/src/github.com/kubeflow/pipelines/'
                            'backend/api/hack/uv.sh'), 'build'
                    ]
                    if enabled == 'true':
                        for path in ('/first/dist', '/second/dist',
                                     '/third/dist'):
                            expected.extend(['--find-links', path])
                    expected.extend(['--out-dir', 'dist', '.'])
                    self.assertEqual(arguments, expected)
                    self.assertNotIn('UV_FIND_LINKS=', command)

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

# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Keep local module manifests available in cached Go download layers."""

import json
from pathlib import Path
import shlex
import subprocess
import unittest

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]


class GoModuleDockerfilesTest(unittest.TestCase):

    def test_download_layers_copy_local_module_manifests(self):
        module = json.loads(
            subprocess.check_output(['go', 'mod', 'edit', '-json'],
                                    cwd=REPOSITORY_ROOT,
                                    text=True))
        manifests = {
            Path(replacement['New']['Path']) / 'go.mod'
            for replacement in module['Replace']
            if not replacement['New'].get('Version')
        }
        self.assertTrue(manifests, 'expected local module replacements')
        tracked = subprocess.check_output(
            ['git', 'ls-files', '*Dockerfile*', '*dockerfile*'],
            cwd=REPOSITORY_ROOT,
            text=True).splitlines()
        checked = []
        for name in tracked:
            if not Path(name).name.lower().startswith('dockerfile'):
                continue
            contents = (REPOSITORY_ROOT / name).read_text(encoding='utf-8')
            if 'go mod download' not in contents:
                continue
            checked.append(name)
            prefix = contents.split('go mod download', 1)[0]
            copies = [
                shlex.split(line)[1:]
                for line in prefix.splitlines()
                if line.startswith('COPY ')
            ]
            if ['.', '.'] in copies:
                continue
            for manifest in manifests:
                with self.subTest(dockerfile=name, manifest=str(manifest)):
                    self.assertTrue(
                        any(
                            len(copy) == 2 and Path(copy[0]) == manifest and
                            Path(copy[1]) == manifest for copy in copies),
                        f'{name} must copy {manifest} before go mod download')
        self.assertTrue(checked, 'expected Go download Docker layers')


if __name__ == '__main__':
    unittest.main()

"""Keep native Go source identity in first-party release binaries."""

import shlex
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
BUILDS = {
    'backend/Dockerfile': (
        './backend/src/apiserver',
        './backend/src/apiserver/migration/mlmd/cmd/mlmd-migrate',
        './backend/src/apiserver/migration/mlmd/cmd/maintenance',
    ),
    'backend/Dockerfile.driver': ('./backend/src/driver',),
    'backend/Dockerfile.launcher': ('./backend/src/v2/cmd/launcher-v2',),
    'backend/Dockerfile.persistenceagent': ('./backend/src/agent/persistence',),
    'backend/Dockerfile.scheduledworkflow': (
        './backend/src/crd/controller/scheduledworkflow',
    ),
    'backend/Dockerfile.viewercontroller': (
        './backend/src/crd/controller/viewer',
    ),
}


class GoBinaryProvenanceTest(unittest.TestCase):
    def test_builds_preserve_native_package_and_vcs_identity(self) -> None:
        for dockerfile, packages in BUILDS.items():
            with self.subTest(dockerfile=dockerfile):
                builds = [
                    shlex.split(line)
                    for line in (ROOT / dockerfile).read_text().splitlines()
                    if line.startswith('RUN ') and 'go build ' in line
                ]
                self.assertCountEqual(
                    [command[-1] for command in builds], packages
                )
                for command in builds:
                    with self.subTest(package=command[-1]):
                        self.assertIn('-buildvcs=true', command)
                        self.assertIn('-mod=readonly', command)
                        self.assertTrue((ROOT / command[-1]).is_dir())
                        self.assertFalse(any('*.go' in arg for arg in command))
                        self.assertFalse(
                            any(
                                'runtime/debug.modinfo' in arg
                                for arg in command
                            )
                        )


if __name__ == '__main__':
    unittest.main()

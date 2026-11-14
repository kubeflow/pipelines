"""Keep native Go source identity in first-party release binaries."""

from pathlib import Path
import shlex
import unittest

ROOT = Path(__file__).resolve().parents[3]
BUILDS = {
    'backend/Dockerfile':
        './backend/src/apiserver',
    'backend/Dockerfile.cacheserver':
        './backend/src/cache',
    'backend/Dockerfile.driver':
        './backend/src/v2/cmd/driver',
    'backend/Dockerfile.launcher':
        './backend/src/v2/cmd/launcher-v2',
    'backend/Dockerfile.persistenceagent':
        './backend/src/agent/persistence',
    'backend/Dockerfile.scheduledworkflow':
        './backend/src/crd/controller/scheduledworkflow',
    'backend/Dockerfile.viewercontroller':
        './backend/src/crd/controller/viewer',
}


class GoBinaryProvenanceTest(unittest.TestCase):

    def test_builds_preserve_native_package_and_vcs_identity(self):
        for dockerfile, package in BUILDS.items():
            with self.subTest(dockerfile=dockerfile):
                builds = [
                    line for line in (ROOT /
                                      dockerfile).read_text().splitlines()
                    if line.startswith('RUN ') and 'go build ' in line
                ]
                self.assertEqual(len(builds), 1)
                command = shlex.split(builds[0])
                self.assertIn('-buildvcs=true', command)
                self.assertIn('-mod=readonly', command)
                self.assertEqual(command[-1], package)
                self.assertTrue((ROOT / package).is_dir())
                self.assertNotIn('*.go', builds[0])
                self.assertNotIn('runtime/debug.modinfo', builds[0])


if __name__ == '__main__':
    unittest.main()

"""Published Go binaries must carry their actual clean source identity."""

import importlib.util
import os
import re
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
VERIFIER = ROOT / '.github/resources/scripts/verify-published-go-binary.sh'
SPEC = importlib.util.spec_from_file_location(
    'publication_provenance', ROOT / 'release/kfpr/go_build_provenance.py'
)
PROVENANCE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(PROVENANCE)
BINARIES = {
    'backend/Dockerfile': '/bin/apiserver',
    'backend/Dockerfile.driver': '/bin/driver',
    'backend/Dockerfile.launcher': '/bin/launcher-v2',
    'backend/Dockerfile.persistenceagent': '/bin/persistence_agent',
    'backend/Dockerfile.scheduledworkflow': '/bin/controller',
    'backend/Dockerfile.viewercontroller': '/bin/controller',
}


class PublicationGoBinaryContractTest(unittest.TestCase):
    def test_every_go_builder_verifies_before_export(self) -> None:
        for recipe, binary in BINARIES.items():
            with self.subTest(recipe=recipe):
                text = (ROOT / recipe).read_text()
                builder = re.split(r'^FROM ', text, flags=re.MULTILINE)[1]
                self.assertRegex(builder, r'(?m)^ARG KFP_SOURCE_SHA(?:=.*)?$')
                runs = re.findall(
                    r'^RUN (.*?)(?=\n[^ \t]|\Z)',
                    builder,
                    re.MULTILINE | re.DOTALL,
                )
                checks = [
                    run
                    for run in runs
                    if 'verify-published-go-binary.sh' in run
                ]
                self.assertEqual(len(checks), 1)
                self.assertIn('if ', checks[0])
                self.assertIn('KFP_SOURCE_SHA', checks[0])
                self.assertIn(binary, checks[0])
                self.assertIn('fi', checks[0])
                self.assertLess(
                    builder.index('go build '),
                    builder.index('verify-published-go-binary.sh'),
                )


@unittest.skipUnless(shutil.which('go'), 'Requires the Go compiler')
class PublicationGoBinaryTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.temporary = tempfile.TemporaryDirectory(prefix='publication-go-')
        cls.root = Path(cls.temporary.name)
        cls.environment = dict(
            os.environ,
            GOOS='linux',
            CGO_ENABLED='0',
            GOTOOLCHAIN='local',
            GOPROXY='off',
            GOSUMDB='off',
            GOCACHE=str(cls.root / 'cache'),
            GOFLAGS='',
        )
        cls.source = cls.root / 'source'
        cls.source.mkdir()
        (cls.source / 'go.mod').write_text(
            'module github.com/kubeflow/pipelines\n\ngo 1.24\n'
        )
        (cls.source / 'main.go').write_text('package main\nfunc main() {}\n')
        (cls.source / 'Dockerfile').write_text('FROM scratch\n')
        (cls.source / '.dockerignore').write_text('.git\n')
        for args in (
            ['init', '--template='],
            ['add', '.'],
            [
                '-c',
                'user.name=Fixture',
                '-c',
                'user.email=fixture@example.invalid',
                'commit',
                '-m',
                'fixture',
            ],
        ):
            PROVENANCE._git(args, cls.source)
        cls.sha = PROVENANCE._git(['rev-parse', 'HEAD'], cls.source)
        cls.prepared = PROVENANCE.prepare_context(
            cls.source, cls.sha, 'Dockerfile', cls.root / 'prepared'
        )['context']
        cls.clean_binary = cls.build(cls.prepared, 'clean')
        cls.without_git = cls.root / 'without-git'
        shutil.copytree(
            cls.source, cls.without_git, ignore=shutil.ignore_patterns('.git')
        )
        cls.unstamped_binary = cls.build(cls.without_git, 'unstamped')

    @classmethod
    def tearDownClass(cls) -> None:
        cls.temporary.cleanup()

    @classmethod
    def build(cls, source: Path, name: str) -> Path:
        binary = cls.root / name
        subprocess.run(
            [
                'go',
                'build',
                '-buildvcs=true',
                '-mod=readonly',
                '-o',
                str(binary),
                '.',
            ],
            cwd=source,
            env=cls.environment,
            check=True,
            capture_output=True,
            text=True,
        )
        return binary

    def verify(
        self, binary: Path, sha: str | None = None
    ) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ['sh', str(VERIFIER), str(binary), sha or self.sha],
            cwd=self.prepared,
            env=self.environment,
            capture_output=True,
            text=True,
        )

    def test_buildvcs_succeeds_without_git_but_verifier_rejects_binary(
        self,
    ) -> None:
        info = subprocess.check_output(
            ['go', 'version', '-m', str(self.unstamped_binary)],
            env=self.environment,
            text=True,
        )
        self.assertNotIn('vcs.revision=', info)
        self.assertTrue(VERIFIER.is_file(), 'Publication verifier is missing')
        self.assertNotEqual(self.verify(self.unstamped_binary).returncode, 0)

    def test_sanitized_source_binary_passes(self) -> None:
        result = self.verify(self.clean_binary)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_wrong_source_sha_fails(self) -> None:
        self.assertTrue(VERIFIER.is_file(), 'Publication verifier is missing')
        self.assertNotEqual(
            self.verify(self.clean_binary, 'a' * 40).returncode, 0
        )

    def test_dirty_source_binary_fails(self) -> None:
        self.assertTrue(VERIFIER.is_file(), 'Publication verifier is missing')
        dirty = self.root / 'dirty-source'
        shutil.copytree(self.prepared, dirty)
        with (dirty / 'main.go').open('a') as stream:
            stream.write('// Uncommitted change.\n')
        binary = self.build(dirty, 'dirty')
        info = subprocess.check_output(
            ['go', 'version', '-m', str(binary)],
            env=self.environment,
            text=True,
        )
        self.assertIn('vcs.modified=true', info)
        self.assertNotEqual(self.verify(binary).returncode, 0)


if __name__ == '__main__':
    unittest.main()

# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Unit tests for the repository-wide Go version updater."""

import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

SCRIPT_PATH = Path(__file__).with_name('update_go_version.py')
SPEC = importlib.util.spec_from_file_location('update_go_version', SCRIPT_PATH)
update_go_version = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(update_go_version)

OLD_DIGEST = 'sha256:' + ('1' * 64)
DIGESTS = {
    '1.28.3': 'sha256:' + ('2' * 64),
    '1.28.3-alpine': 'sha256:' + ('3' * 64),
    '1.28.3-bookworm': 'sha256:' + ('4' * 64),
    '1.29.0': 'sha256:' + ('5' * 64),
    '1.29.0-alpine': 'sha256:' + ('6' * 64),
    '1.29.0-bookworm': 'sha256:' + ('7' * 64),
}


class UpdateGoVersionTest(unittest.TestCase):

    def setUp(self):
        self.temp_dir = tempfile.TemporaryDirectory()
        self.repo_root = Path(self.temp_dir.name)
        self.files = {
            Path('go.mod'): 'module example.com/root\n\ngo 1.28.0\n',
            Path('nested/go.mod'):
                'module example.com/nested\n\ngo 1.28.0\n\n'
                'toolchain go1.28.2\n\n'
                'require example.com/tool v1.0.0\n',
            Path('Dockerfile'):
                f'FROM golang:1.28.0@{OLD_DIGEST} AS builder\n',
            Path('service/Dockerfile'):
                f'FROM golang:1.28.0-alpine@{OLD_DIGEST} as builder\n',
            Path('another/Dockerfile.worker'):
                f'FROM golang:1.28.0-alpine@{OLD_DIGEST} AS builder\n',
            Path('bookworm/Dockerfile'):
                f'FROM golang:1.28.0-bookworm@{OLD_DIGEST} AS builder\n',
        }
        for relative_path, contents in self.files.items():
            path = self.repo_root / relative_path
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(contents, encoding='utf-8')

    def tearDown(self):
        self.temp_dir.cleanup()

    def _resolve(self, calls):

        def resolver(tag):
            calls.append(tag)
            return DIGESTS[tag]

        return resolver

    def test_patch_update_aligns_modules_and_builder_images(self):
        calls = []
        expected = update_go_version.synchronized_contents(
            self.repo_root,
            '1.28.3',
            digest_resolver=self._resolve(calls),
            repository_paths=self.files,
        )

        self.assertIn('go 1.28.0\n\ntoolchain go1.28.3',
                      expected[Path('go.mod')])
        self.assertIn('go 1.28.0\n\ntoolchain go1.28.3',
                      expected[Path('nested/go.mod')])
        self.assertIn('require example.com/tool v1.0.0',
                      expected[Path('nested/go.mod')])
        self.assertIn(
            f'golang:1.28.3-alpine@{DIGESTS["1.28.3-alpine"]}',
            expected[Path('service/Dockerfile')],
        )
        self.assertIn(
            f'golang:1.28.3-alpine@{DIGESTS["1.28.3-alpine"]}',
            expected[Path('another/Dockerfile.worker')],
        )
        self.assertEqual(calls,
                         ['1.28.3', '1.28.3-alpine', '1.28.3-bookworm'])

    def test_patch_update_preserves_existing_language_floor(self):
        root = self.repo_root / 'go.mod'
        root.write_text('module example.com/root\n\ngo 1.28.2\n',
                        encoding='utf-8')

        expected = update_go_version.synchronized_contents(
            self.repo_root,
            '1.28.3',
            digest_resolver=lambda tag: DIGESTS[tag],
            repository_paths=self.files,
        )

        self.assertIn('go 1.28.2\n\ntoolchain go1.28.3',
                      expected[Path('go.mod')])

    def test_rejects_target_below_module_language_floor(self):
        nested = self.repo_root / 'nested/go.mod'
        nested.write_text('module example.com/nested\n\ngo 1.28.4\n',
                          encoding='utf-8')

        with self.assertRaisesRegex(ValueError, 'exceeds the target compiler'):
            update_go_version.synchronized_contents(
                self.repo_root,
                '1.28.3',
                digest_resolver=lambda tag: DIGESTS[tag],
                repository_paths=self.files,
            )

    def test_dot_zero_update_removes_toolchain(self):
        expected = update_go_version.synchronized_contents(
            self.repo_root,
            '1.29.0',
            digest_resolver=lambda tag: DIGESTS[tag],
            repository_paths=self.files,
        )

        for relative_path in (Path('go.mod'), Path('nested/go.mod')):
            self.assertIn('go 1.29.0', expected[relative_path])
            self.assertNotIn('toolchain ', expected[relative_path])

    def test_sync_is_idempotent(self):
        resolver = lambda tag: DIGESTS[tag]
        first = update_go_version.sync(
            self.repo_root,
            '1.29.0',
            digest_resolver=resolver,
            repository_paths=self.files,
        )
        second = update_go_version.sync(
            self.repo_root,
            '1.29.0',
            digest_resolver=resolver,
            repository_paths=self.files,
        )

        self.assertEqual(set(first), set(self.files))
        self.assertEqual(second, [])

    def test_digest_failure_does_not_write_partial_changes(self):

        def fail(_tag):
            raise RuntimeError('registry unavailable')

        with self.assertRaisesRegex(RuntimeError, 'registry unavailable'):
            update_go_version.sync(
                self.repo_root,
                '1.28.3',
                digest_resolver=fail,
                repository_paths=self.files,
            )
        for relative_path, contents in self.files.items():
            self.assertEqual(
                (self.repo_root / relative_path).read_text(encoding='utf-8'),
                contents,
            )

    def test_rejects_invalid_versions_and_downgrades(self):
        for version in ('1.29', 'go1.29.0', 'v1.29.0', '1.29.0-rc1'):
            with self.subTest(version=version):
                with self.assertRaisesRegex(ValueError, 'exact stable form'):
                    update_go_version.synchronized_contents(
                        self.repo_root,
                        version,
                        digest_resolver=lambda _tag: OLD_DIGEST,
                        repository_paths=self.files,
                    )
        with self.assertRaisesRegex(ValueError, 'refusing to downgrade'):
            update_go_version.synchronized_contents(
                self.repo_root,
                '1.27.9',
                digest_resolver=lambda _tag: OLD_DIGEST,
                repository_paths=self.files,
            )

    def test_rejects_unmanaged_runtime_pin(self):
        relative_path = Path('bad/Dockerfile')
        path = self.repo_root / relative_path
        path.parent.mkdir(parents=True)
        path.write_text('FROM golang:1.28.0-alpine AS builder\n',
                        encoding='utf-8')
        repository_paths = set(self.files) | {relative_path}

        with self.assertRaisesRegex(ValueError, 'unsupported Go runtime pins'):
            update_go_version.synchronized_contents(
                self.repo_root,
                '1.28.3',
                digest_resolver=lambda _tag: OLD_DIGEST,
                repository_paths=repository_paths,
            )

    def test_discovers_tracked_and_untracked_nonignored_modules(self):
        subprocess.run(('git', 'init', '--quiet'),
                       cwd=self.repo_root,
                       check=True)
        subprocess.run(('git', 'add', 'go.mod', 'Dockerfile'),
                       cwd=self.repo_root,
                       check=True)
        ignored = self.repo_root / '.gitignore'
        ignored.write_text('ignored/\n', encoding='utf-8')
        ignored_module = self.repo_root / 'ignored/go.mod'
        ignored_module.parent.mkdir()
        ignored_module.write_text('module ignored\n\ngo 1.28.0\n',
                                  encoding='utf-8')

        paths = update_go_version._repository_paths(self.repo_root)

        self.assertIn(Path('go.mod'), paths)
        self.assertIn(Path('nested/go.mod'), paths)
        self.assertNotIn(Path('ignored/go.mod'), paths)

    def test_digest_resolver_reads_multi_platform_manifest_digest(self):
        digest = 'sha256:' + ('a' * 64)
        completed = subprocess.CompletedProcess(
            args=[], returncode=0, stdout=f'{{"digest": "{digest}"}}')
        with mock.patch.object(
                update_go_version.subprocess,
                'run',
                return_value=completed,
        ) as run:
            resolved = update_go_version.resolve_docker_hub_digest(
                '1.28.3-alpine')

        self.assertEqual(resolved, digest)
        run.assert_called_once_with(
            (
                'docker',
                'buildx',
                'imagetools',
                'inspect',
                'golang:1.28.3-alpine',
                '--format',
                '{{json .Manifest}}',
            ),
            check=True,
            capture_output=True,
            text=True,
        )

    def test_digest_resolver_rejects_invalid_output(self):
        for output in ('not json', '{}', '{"digest": "sha256:short"}'):
            completed = subprocess.CompletedProcess(
                args=[], returncode=0, stdout=output)
            with self.subTest(output=output):
                with mock.patch.object(
                        update_go_version.subprocess,
                        'run',
                        return_value=completed,
                ):
                    with self.assertRaisesRegex(RuntimeError,
                                                'manifest digest|invalid digest'):
                        update_go_version.resolve_docker_hub_digest('1.28.3')

    def test_digest_resolver_reports_missing_docker_and_lookup_failure(self):
        with mock.patch.object(
                update_go_version.subprocess,
                'run',
                side_effect=FileNotFoundError,
        ):
            with self.assertRaisesRegex(RuntimeError,
                                        'docker buildx is required'):
                update_go_version.resolve_docker_hub_digest('1.28.3')

        failure = subprocess.CalledProcessError(
            1, ['docker'], stderr='registry unavailable')
        with mock.patch.object(
                update_go_version.subprocess,
                'run',
                side_effect=failure,
        ):
            with self.assertRaisesRegex(RuntimeError, 'registry unavailable'):
                update_go_version.resolve_docker_hub_digest('1.28.3')

    def test_digest_resolver_falls_back_to_docker_hub_mirror(self):
        digest = 'sha256:' + ('b' * 64)
        failure = subprocess.CalledProcessError(
            1, ['docker'], stderr='rate limited')
        mirrored = subprocess.CompletedProcess(
            args=[], returncode=0, stdout=f'{{"digest": "{digest}"}}')
        with mock.patch.object(
                update_go_version.subprocess,
                'run',
                side_effect=(failure, mirrored),
        ) as run:
            resolved = update_go_version.resolve_docker_hub_digest('1.28.3')

        self.assertEqual(resolved, digest)
        self.assertIn(
            'mirror.gcr.io/library/golang:1.28.3',
            run.call_args_list[1].args[0],
        )


if __name__ == '__main__':
    unittest.main()

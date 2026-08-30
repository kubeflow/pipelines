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

    def test_rejects_higher_minor_module_language_floor(self):
        nested = self.repo_root / 'nested/go.mod'
        nested.write_text('module example.com/nested\n\ngo 1.30.0\n',
                          encoding='utf-8')

        with self.assertRaisesRegex(ValueError, 'exceeds the target compiler'):
            update_go_version.synchronized_contents(
                self.repo_root,
                '1.29.3',
                digest_resolver=lambda _tag: OLD_DIGEST,
                repository_paths=self.files,
            )

    def test_indented_module_directives_are_normalized(self):
        nested = self.repo_root / 'nested/go.mod'
        nested.write_text(
            'module example.com/nested\n\n'
            '  go 1.28.0// language floor\n\n'
            '\ttoolchain go1.28.2// compiler version\n',
            encoding='utf-8',
        )

        expected = update_go_version.synchronized_contents(
            self.repo_root,
            '1.28.3',
            digest_resolver=lambda tag: DIGESTS[tag],
            repository_paths=self.files,
        )[Path('nested/go.mod')]

        self.assertIn(
            '\ngo 1.28.0// language floor\n\n'
            'toolchain go1.28.3// compiler version\n', expected)
        self.assertEqual(expected.count('toolchain '), 1)

    def test_module_block_entries_are_not_directives(self):
        nested = self.repo_root / 'nested/go.mod'
        nested.write_text(
            'module example.com/nested\n\n'
            'require (\n'
            '  go v1.0.0\n'
            '  toolchain v1.0.0\n'
            ')\n\n'
            'go 1.28.0\n',
            encoding='utf-8',
        )

        expected = update_go_version.synchronized_contents(
            self.repo_root,
            '1.28.3',
            digest_resolver=lambda tag: DIGESTS[tag],
            repository_paths=self.files,
        )[Path('nested/go.mod')]

        self.assertIn('  go v1.0.0\n', expected)
        self.assertIn('  toolchain v1.0.0\n', expected)
        self.assertIn('go 1.28.0\n\ntoolchain go1.28.3\n', expected)

    def test_preserves_backslashes_in_go_directive_comments(self):
        root = self.repo_root / 'go.mod'
        comment = r'// paths C:\qtoolchains and D:\toolchains'
        root.write_text(
            f'module example.com/root\n\ngo 1.28.0 {comment}\n',
            encoding='utf-8',
        )

        expected = update_go_version.synchronized_contents(
            self.repo_root,
            '1.28.3',
            digest_resolver=lambda tag: DIGESTS[tag],
            repository_paths=self.files,
        )[Path('go.mod')]

        self.assertIn(f'go 1.28.0 {comment}\n', expected)

    def test_rejects_bare_toolchain_directive(self):
        nested = self.repo_root / 'nested/go.mod'
        nested.write_text('module example.com/nested\n\ngo 1.28.0\n\ntoolchain\n',
                          encoding='utf-8')

        with self.assertRaisesRegex(ValueError,
                                    'invalid toolchain directive'):
            update_go_version.synchronized_contents(
                self.repo_root,
                '1.28.3',
                digest_resolver=lambda tag: DIGESTS[tag],
                repository_paths=self.files,
            )

    def test_rejects_extra_malformed_go_directive(self):
        nested = self.repo_root / 'nested/go.mod'
        nested.write_text(
            'module example.com/nested\n\ngo 1.28.0\n\n  go\n',
            encoding='utf-8',
        )

        with self.assertRaisesRegex(ValueError, 'go directive'):
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

    def test_concurrent_edit_during_digest_resolution_is_not_overwritten(self):
        root = self.repo_root / 'go.mod'
        concurrent_contents = (
            'module example.com/root\n\n'
            'go 1.28.0\n\n'
            '// concurrent edit\n')
        edited = False

        def edit_then_resolve(tag):
            nonlocal edited
            if not edited:
                root.write_text(concurrent_contents, encoding='utf-8')
                edited = True
            return DIGESTS[tag]

        with self.assertRaisesRegex(RuntimeError,
                                    'changed during Go version update'):
            update_go_version.sync(
                self.repo_root,
                '1.28.3',
                digest_resolver=edit_then_resolve,
                repository_paths=self.files,
            )

        self.assertEqual(root.read_text(encoding='utf-8'),
                         concurrent_contents)
        for relative_path, contents in self.files.items():
            if relative_path != Path('go.mod'):
                self.assertEqual(
                    (self.repo_root /
                     relative_path).read_text(encoding='utf-8'), contents)

    def test_file_replacement_failure_restores_original_files(self):
        real_replace = update_go_version.os.replace
        replacement_count = 0

        def fail_second_replacement(source, destination):
            nonlocal replacement_count
            replacement_count += 1
            if replacement_count == 2:
                raise OSError('simulated replacement failure')
            real_replace(source, destination)

        with mock.patch.object(
                update_go_version.os,
                'replace',
                side_effect=fail_second_replacement,
        ):
            with self.assertRaisesRegex(RuntimeError,
                                        'restored original files'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        for relative_path, contents in self.files.items():
            self.assertEqual(
                (self.repo_root / relative_path).read_text(encoding='utf-8'),
                contents,
            )

    def test_keyboard_interrupt_restores_original_files_and_is_reraised(self):
        real_replace = update_go_version.os.replace
        replacement_count = 0

        def interrupt_after_second_replacement(source, destination):
            nonlocal replacement_count
            replacement_count += 1
            if replacement_count == 2:
                real_replace(source, destination)
                raise KeyboardInterrupt
            real_replace(source, destination)

        with mock.patch.object(
                update_go_version.os,
                'replace',
                side_effect=interrupt_after_second_replacement,
        ):
            with self.assertRaises(KeyboardInterrupt):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        for relative_path, contents in self.files.items():
            self.assertEqual(
                (self.repo_root / relative_path).read_text(encoding='utf-8'),
                contents,
            )

    def test_failed_rollback_backup_is_retained(self):
        real_replace = update_go_version.os.replace
        replacement_count = 0

        def fail_update_and_rollback(source, destination):
            nonlocal replacement_count
            replacement_count += 1
            if replacement_count in (2, 3):
                raise OSError(f'simulated failure {replacement_count}')
            real_replace(source, destination)

        with mock.patch.object(
                update_go_version.os,
                'replace',
                side_effect=fail_update_and_rollback,
        ):
            with self.assertRaisesRegex(
                    RuntimeError, 'original retained at') as context:
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        retained_path = Path(
            str(context.exception).split('original retained at ', 1)[1])
        self.assertTrue(retained_path.exists())
        self.assertEqual(retained_path.read_text(encoding='utf-8'),
                         self.files[Path('Dockerfile')])

    def test_rollback_does_not_overwrite_concurrent_edit(self):
        real_replace = update_go_version.os.replace
        replacement_count = 0
        concurrent_contents = '# concurrent Dockerfile edit\n'

        def edit_first_then_fail_second(source, destination):
            nonlocal replacement_count
            replacement_count += 1
            if replacement_count == 1:
                real_replace(source, destination)
                Path(destination).write_text(concurrent_contents,
                                             encoding='utf-8')
                return
            if replacement_count == 2:
                raise OSError('simulated replacement failure')
            real_replace(source, destination)

        with mock.patch.object(
                update_go_version.os,
                'replace',
                side_effect=edit_first_then_fail_second,
        ):
            with self.assertRaisesRegex(
                    RuntimeError,
                    'destination changed after replacement') as context:
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        retained_path = Path(
            str(context.exception).split('original retained at ', 1)[1])
        self.assertEqual(
            (self.repo_root / 'Dockerfile').read_text(encoding='utf-8'),
            concurrent_contents,
        )
        self.assertTrue(retained_path.exists())
        self.assertEqual(retained_path.read_text(encoding='utf-8'),
                         self.files[Path('Dockerfile')])

    def test_rejects_invalid_versions_and_downgrades(self):
        for version in ('1.29', 'go1.29.0', 'v1.29.0', '1.29.0-rc1',
                        '1.029.000', '1.٢٩.٠', '１.２９.０'):
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

    def test_rejects_noncanonical_module_versions(self):
        nested = self.repo_root / 'nested/go.mod'
        for version in ('1.028.000', '1.٢٨.٠', '１.２８.０'):
            with self.subTest(version=version):
                nested.write_text(
                    f'module example.com/nested\n\ngo {version}\n',
                    encoding='utf-8',
                )
                with self.assertRaisesRegex(ValueError,
                                            'invalid go directive'):
                    update_go_version.synchronized_contents(
                        self.repo_root,
                        '1.28.3',
                        digest_resolver=lambda tag: DIGESTS[tag],
                        repository_paths=self.files,
                    )

    def test_rejects_unmanaged_runtime_pin(self):
        relative_path = Path('bad/Dockerfile')
        path = self.repo_root / relative_path
        path.parent.mkdir(parents=True)
        repository_paths = set(self.files) | {relative_path}
        references = (
            'FROM golang:1.28.0-alpine AS builder\n',
            'FROM golang AS builder\n',
            f'FROM golang@{OLD_DIGEST} AS builder\n',
            'RUN curl -LO https://go.dev/dl/go1.28.0.linux-amd64.tar.gz\n',
        )
        for contents in references:
            with self.subTest(contents=contents):
                path.write_text(contents, encoding='utf-8')
                with self.assertRaisesRegex(ValueError,
                                            'unsupported Go runtime pins'):
                    update_go_version.synchronized_contents(
                        self.repo_root,
                        '1.28.3',
                        digest_resolver=lambda _tag: OLD_DIGEST,
                        repository_paths=repository_paths,
                    )

    def test_rejects_additional_unmanaged_golang_stage(self):
        dockerfile = self.repo_root / 'Dockerfile'
        dockerfile.write_text(
            self.files[Path('Dockerfile')] +
            'FROM golang:1.26.0 AS stale\n',
            encoding='utf-8',
        )

        with self.assertRaisesRegex(ValueError,
                                    'unsupported Go runtime pins'):
            update_go_version.synchronized_contents(
                self.repo_root,
                '1.28.3',
                digest_resolver=lambda tag: DIGESTS[tag],
                repository_paths=self.files,
            )

    def test_rejects_global_golang_repository_arg(self):
        relative_path = Path('bad/Dockerfile')
        path = self.repo_root / relative_path
        path.parent.mkdir(parents=True)
        path.write_text(
            'ARG GO_REPOSITORY=docker.io/library/golang\n'
            'FROM ${GO_REPOSITORY}:1.26.0 AS stale\n',
            encoding='utf-8',
        )

        with self.assertRaisesRegex(ValueError,
                                    'unsupported Go runtime pins'):
            update_go_version.synchronized_contents(
                self.repo_root,
                '1.28.3',
                digest_resolver=lambda tag: DIGESTS[tag],
                repository_paths=set(self.files) | {relative_path},
            )

    def test_rejects_symlink_destination(self):
        root = self.repo_root / 'go.mod'
        target = self.repo_root / 'real-go.mod'
        target.write_text(self.files[Path('go.mod')], encoding='utf-8')
        root.unlink()
        root.symlink_to(target.name)

        with self.assertRaisesRegex(ValueError, 'must be a regular file'):
            update_go_version.sync(
                self.repo_root,
                '1.28.3',
                digest_resolver=lambda tag: DIGESTS[tag],
                repository_paths=self.files,
            )

        self.assertTrue(root.is_symlink())
        self.assertEqual(target.read_text(encoding='utf-8'),
                         self.files[Path('go.mod')])

    def test_rejects_tagless_yaml_runtime(self):
        relative_path = Path('bad/component.yaml')
        path = self.repo_root / relative_path
        path.parent.mkdir(parents=True)
        repository_paths = set(self.files) | {relative_path}

        for contents in ('container: golang\n',
                         'image: docker.io/library/golang # latest\n',
                         '  - image: "golang" # builder\n',
                         'container: { image: golang }\n',
                         'container: {"image":"golang"}\n',
                         'container: {options: "", "image":"golang"}\n',
                         'container: {credentials: {user: test}, image: golang}\n'):
            with self.subTest(contents=contents):
                path.write_text(contents, encoding='utf-8')
                with self.assertRaisesRegex(ValueError,
                                            'unsupported Go runtime pins'):
                    update_go_version.synchronized_contents(
                        self.repo_root,
                        '1.28.3',
                        digest_resolver=lambda _tag: OLD_DIGEST,
                        repository_paths=repository_paths,
                    )

    def test_runtime_detection_ignores_comments_and_command_strings(self):
        for contents in (
                '# golang:latest',
                '# https://go.dev/dl/go1.28.0.linux-amd64.tar.gz',
                'run: echo \'{"image":"golang"}\'',
                'container: alpine:3.22 # not golang:latest',
        ):
            with self.subTest(contents=contents):
                self.assertFalse(
                    update_go_version.has_go_runtime_reference(
                        Path('test.yaml'), contents))

    def test_verifies_each_distinct_builder_tag_once(self):
        calls = []

        def resolver(tag):
            calls.append(tag)
            return OLD_DIGEST

        update_go_version.verify_image_digests(
            self.repo_root,
            digest_resolver=resolver,
            repository_paths=self.files,
        )

        self.assertEqual(calls,
                         ['1.28.0', '1.28.0-alpine', '1.28.0-bookworm'])

    def test_digest_retry_budget_fits_workflow_timeout(self):
        worst_case = update_go_version._digest_verification_worst_case_seconds(
            3)

        self.assertEqual(worst_case, 369)
        self.assertLessEqual(
            worst_case,
            update_go_version.DIGEST_VERIFICATION_BUDGET_SECONDS,
        )
        self.assertLess(update_go_version.DIGEST_VERIFICATION_BUDGET_SECONDS,
                        600)

    def test_rejects_builder_digest_that_does_not_match_tag(self):
        resolved_digest = 'sha256:' + ('9' * 64)
        with self.assertRaises(ValueError) as context:
            update_go_version.verify_image_digests(
                self.repo_root,
                digest_resolver=lambda _tag: resolved_digest,
                repository_paths=self.files,
            )
        message = str(context.exception)
        self.assertIn('golang:1.28.0', message)
        self.assertIn('Dockerfile', message)
        self.assertIn(OLD_DIGEST, message)
        self.assertIn(resolved_digest, message)

    def test_rejects_inconsistent_digests_for_same_tag(self):
        different_digest = 'sha256:' + ('8' * 64)
        path = self.repo_root / 'another/Dockerfile.worker'
        path.write_text(
            f'FROM golang:1.28.0-alpine@{different_digest} AS builder\n',
            encoding='utf-8',
        )

        with self.assertRaisesRegex(ValueError,
                                    'inconsistent pinned digests'):
            update_go_version.verify_image_digests(
                self.repo_root,
                digest_resolver=lambda _tag: OLD_DIGEST,
                repository_paths=self.files,
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
            timeout=update_go_version.DIGEST_LOOKUP_TIMEOUT_SECONDS,
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
        ) as run, mock.patch.object(update_go_version.time, 'sleep') as sleep:
            with self.assertRaisesRegex(RuntimeError, 'registry unavailable'):
                update_go_version.resolve_docker_hub_digest('1.28.3')
        self.assertEqual(run.call_count, 6)
        self.assertEqual(sleep.call_args_list, [mock.call(1), mock.call(2)])

    def test_digest_resolver_retries_both_sources(self):
        digest = 'sha256:' + ('c' * 64)
        failure = subprocess.CalledProcessError(
            1, ['docker'], stderr='registry unavailable')
        resolved = subprocess.CompletedProcess(
            args=[], returncode=0, stdout=f'{{"digest": "{digest}"}}')
        with mock.patch.object(
                update_go_version.subprocess,
                'run',
                side_effect=(failure, failure, resolved),
        ) as run, mock.patch.object(update_go_version.time, 'sleep') as sleep:
            self.assertEqual(
                update_go_version.resolve_docker_hub_digest('1.28.3'), digest)
        self.assertEqual(run.call_count, 3)
        sleep.assert_called_once_with(1)

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

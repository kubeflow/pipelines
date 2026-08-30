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
import json
from pathlib import Path
import stat
import subprocess
import sys
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
        self._git('init', '-q')
        self._git('add', '--', *(str(path) for path in self.files))
        self._git('-c', 'user.name=Test', '-c',
                  'user.email=test@example.com', 'commit', '-qm', 'initial')

    def tearDown(self):
        self.temp_dir.cleanup()

    def _resolve(self, calls):

        def resolver(tag):
            calls.append(tag)
            return DIGESTS[tag]

        return resolver

    def _git(self, *arguments):
        return subprocess.run(
            ('git', *arguments),
            cwd=self.repo_root,
            check=True,
            capture_output=True,
            text=True,
        )

    def _recovery_patches(self):
        recovery_dir = (self.repo_root / '.git' /
                        'go-version-update-recovery')
        return list(recovery_dir.rglob('*.patch'))

    def _recovery_bundles(self):
        recovery_dir = (self.repo_root / '.git' /
                        'go-version-update-recovery')
        return list(recovery_dir.glob('*.bundle'))

    def _restore_patch(self, relative_path):
        bundles = self._recovery_bundles()
        self.assertEqual(len(bundles), 1)
        manifest = json.loads(
            (bundles[0] / 'manifest.json').read_text(encoding='utf-8'))
        return bundles[0] / manifest['originalRestorePatches'][str(
            relative_path)]

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

    def test_rejects_malformed_go_and_toolchain_blocks(self):
        nested = self.repo_root / 'nested/go.mod'
        for contents in (
                'module example.com/nested\n\ngo (\n  1.28.0\n)\n',
                'module example.com/nested\n\ngo 1.28.0\n\n'
                'toolchain (\n  go1.28.2\n)\n'):
            with self.subTest(contents=contents):
                nested.write_text(contents, encoding='utf-8')
                with self.assertRaisesRegex(ValueError, 'invalid .* directive'):
                    update_go_version.synchronized_contents(
                        self.repo_root,
                        '1.28.3',
                        digest_resolver=lambda tag: DIGESTS[tag],
                        repository_paths=self.files,
                    )

    def test_updates_containerfile_recipes(self):
        relative_path = Path('service/Containerfile.debug')
        contents = f'FROM golang:1.28.0@{OLD_DIGEST} AS builder\n'
        (self.repo_root / relative_path).write_text(contents, encoding='utf-8')
        repository_paths = set(self.files) | {relative_path}

        expected = update_go_version.synchronized_contents(
            self.repo_root,
            '1.28.3',
            digest_resolver=lambda tag: DIGESTS[tag],
            repository_paths=repository_paths,
        )

        self.assertIn(relative_path, expected)
        self.assertIn(
            f'golang:1.28.3@{DIGESTS["1.28.3"]}',
            expected[relative_path],
        )

    def test_stage_alias_named_golang_is_conservatively_unsupported(self):
        dockerfile = self.repo_root / 'Dockerfile'
        dockerfile.write_text(
            f'FROM golang:1.28.0@{OLD_DIGEST} AS golang\n'
            'FROM golang AS final\n',
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

    def test_chained_arg_values_are_not_evaluated(self):
        dockerfile = self.repo_root / 'Dockerfile'
        dockerfile.write_text(
            'ARG GO=go\n'
            'ARG LANG=lang\n'
            'ARG IMAGE=${GO}${LANG}:1.28.0\n'
            'FROM ${IMAGE} AS builder\n',
            encoding='utf-8',
        )

        expected = update_go_version.synchronized_contents(
            self.repo_root,
            '1.28.3',
            digest_resolver=lambda tag: DIGESTS[tag],
            repository_paths=self.files,
        )

        self.assertNotIn(Path('Dockerfile'), expected)

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
                                    'must be tracked and clean'):
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

    def test_head_change_during_resolution_rejects_stale_plan(self):
        initial_head = self._git('rev-parse', 'HEAD').stdout.strip()
        root = self.repo_root / 'go.mod'
        alternate_contents = 'module example.com/root\n\ngo 1.28.1\n'
        root.write_text(alternate_contents, encoding='utf-8')
        self._git('add', 'go.mod')
        self._git('-c', 'user.name=Test', '-c',
                  'user.email=test@example.com', 'commit', '-qm', 'alternate')
        alternate_head = self._git('rev-parse', 'HEAD').stdout.strip()
        self._git('checkout', '-q', initial_head)
        switched = False

        def switch_head_then_resolve(tag):
            nonlocal switched
            if not switched:
                self._git('checkout', '-q', alternate_head)
                switched = True
            return DIGESTS[tag]

        with self.assertRaisesRegex(RuntimeError,
                                    'HEAD changed during Go version update'):
            update_go_version.sync(
                self.repo_root,
                '1.28.3',
                digest_resolver=switch_head_then_resolve,
                repository_paths=self.files,
            )

        self.assertEqual(root.read_text(encoding='utf-8'), alternate_contents)
        self.assertEqual(self._recovery_patches(), [])

    def test_managed_paths_must_be_tracked_and_clean(self):
        root = self.repo_root / 'go.mod'
        root.write_text(self.files[Path('go.mod')] + '// unstaged\n',
                        encoding='utf-8')
        with self.assertRaisesRegex(RuntimeError,
                                    'must be tracked and clean'):
            update_go_version.sync(
                self.repo_root,
                '1.28.3',
                digest_resolver=lambda tag: DIGESTS[tag],
                repository_paths=self.files,
            )

    def test_staged_managed_paths_are_rejected(self):
        dockerfile = self.repo_root / 'Dockerfile'
        dockerfile.write_text(
            self.files[Path('Dockerfile')] + '# staged edit\n',
            encoding='utf-8',
        )
        self._git('add', 'Dockerfile')

        with self.assertRaisesRegex(RuntimeError,
                                    'must be tracked and clean'):
            update_go_version.sync(
                self.repo_root,
                '1.28.3',
                digest_resolver=lambda tag: DIGESTS[tag],
                repository_paths=self.files,
            )

    def test_unrelated_dirty_files_are_preserved(self):
        notes = self.repo_root / 'notes.txt'
        notes.write_text('original\n', encoding='utf-8')
        self._git('add', 'notes.txt')
        self._git('-c', 'user.name=Test', '-c',
                  'user.email=test@example.com', 'commit', '-qm', 'notes')
        notes.write_text('concurrent unrelated edit\n', encoding='utf-8')

        changed = update_go_version.sync(
            self.repo_root,
            '1.28.3',
            digest_resolver=lambda tag: DIGESTS[tag],
            repository_paths=self.files,
        )

        self.assertEqual(set(changed), set(self.files))
        self.assertEqual(notes.read_text(encoding='utf-8'),
                         'concurrent unrelated edit\n')

    def test_worktree_verification_may_not_change_unplanned_files(self):
        notes = self.repo_root / 'notes.txt'
        notes.write_text('original\n', encoding='utf-8')
        self._git('add', 'notes.txt')
        self._git('-c', 'user.name=Test', '-c',
                  'user.email=test@example.com', 'commit', '-qm', 'notes')

        def mutate_worktree(repo_root):
            if Path(repo_root) != self.repo_root:
                (Path(repo_root) / 'notes.txt').write_text(
                    'unexpected verifier output\n', encoding='utf-8')

        with mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=mutate_worktree,
        ):
            with self.assertRaisesRegex(RuntimeError,
                                        'temporary worktree changed'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertEqual(notes.read_text(encoding='utf-8'), 'original\n')
        self.assertEqual(self._recovery_patches(), [])

    def test_apply_failure_keeps_recovery_patch_and_originals(self):
        real_git = update_go_version._git

        def fail_apply(repo_root, *arguments):
            if arguments[:2] == ('apply', '--whitespace=nowarn'):
                raise RuntimeError('simulated apply failure')
            return real_git(repo_root, *arguments)

        with mock.patch.object(update_go_version,
                               '_git', side_effect=fail_apply):
            with self.assertRaisesRegex(RuntimeError,
                                        'recovery bundle retained at'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)
        for relative_path, contents in self.files.items():
            self.assertEqual(
                (self.repo_root / relative_path).read_text(encoding='utf-8'),
                contents,
            )

    def test_complete_original_artifacts_exist_before_live_apply(self):
        real_git = update_go_version._git

        def inspect_journal_then_fail(repo_root, *arguments):
            if arguments[:2] == ('apply', '--whitespace=nowarn'):
                artifacts = self._recovery_patches()
                self.assertEqual(len(artifacts), len(self.files) + 1)
                bundle = self._recovery_bundles()[0]
                manifest = json.loads(
                    (bundle / 'manifest.json').read_text(encoding='utf-8'))
                self.assertEqual(set(manifest['originalRestorePatches']),
                                 {str(path) for path in self.files})
                raise RuntimeError('simulated crash window')
            return real_git(repo_root, *arguments)

        with mock.patch.object(
                update_go_version,
                '_git',
                side_effect=inspect_journal_then_fail,
        ):
            with self.assertRaisesRegex(RuntimeError,
                                        'simulated crash window'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)
        for relative_path, contents in self.files.items():
            self.assertEqual(
                (self.repo_root / relative_path).read_text(encoding='utf-8'),
                contents,
            )

    def test_hard_crash_after_apply_leaves_complete_recovery_bundle(self):
        child = r'''
import importlib.util
import json
import os
from pathlib import Path
import sys

script, repo, paths_json, digests_json = sys.argv[1:]
sys.path.insert(0, str(Path(script).parent))
spec = importlib.util.spec_from_file_location('crash_update_go_version', script)
updater = importlib.util.module_from_spec(spec)
spec.loader.exec_module(updater)
real_git = updater._git

def apply_then_crash(repo_root, *arguments):
    result = real_git(repo_root, *arguments)
    if arguments[:2] == ('apply', '--whitespace=nowarn'):
        os._exit(97)
    return result

updater._git = apply_then_crash
digests = json.loads(digests_json)
updater.sync(
    Path(repo),
    '1.28.3',
    digest_resolver=lambda tag: digests[tag],
    repository_paths=[Path(path) for path in json.loads(paths_json)],
)
'''
        result = subprocess.run(
            (
                sys.executable,
                '-c',
                child,
                str(SCRIPT_PATH),
                str(self.repo_root),
                json.dumps([str(path) for path in self.files]),
                json.dumps(DIGESTS),
            ),
            cwd=self.repo_root,
            check=False,
            capture_output=True,
            text=True,
        )

        self.assertEqual(result.returncode, 97, result.stderr)
        bundles = self._recovery_bundles()
        self.assertEqual(len(bundles), 1)
        manifest = json.loads(
            (bundles[0] / 'manifest.json').read_text(encoding='utf-8'))
        self.assertEqual(set(manifest['originalRestorePatches']),
                         {str(path) for path in self.files})
        for relative_path, restore_name in manifest[
                'originalRestorePatches'].items():
            path = self.repo_root / relative_path
            path.unlink()
            self._git('apply', str(bundles[0] / restore_name))
        for relative_path, contents in self.files.items():
            self.assertEqual(
                (self.repo_root / relative_path).read_text(encoding='utf-8'),
                contents,
            )

    def test_artifact_write_failure_prevents_live_apply(self):
        real_write = update_go_version._write_durable_recovery_file
        write_count = 0
        live_apply_called = False
        real_git = update_go_version._git

        def fail_during_bundle_write(*args, **kwargs):
            nonlocal write_count
            write_count += 1
            if write_count == 3:
                raise OSError('simulated recovery storage full')
            return real_write(*args, **kwargs)

        def record_live_apply(repo_root, *arguments):
            nonlocal live_apply_called
            if arguments[:2] == ('apply', '--whitespace=nowarn'):
                live_apply_called = True
            return real_git(repo_root, *arguments)

        with mock.patch.object(
                update_go_version,
                '_write_durable_recovery_file',
                side_effect=fail_during_bundle_write,
        ), mock.patch.object(
                update_go_version,
                '_git',
                side_effect=record_live_apply,
        ):
            with self.assertRaisesRegex(RuntimeError,
                                        'simulated recovery storage full'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertFalse(live_apply_called)
        self.assertEqual(self._recovery_patches(), [])
        for relative_path, contents in self.files.items():
            self.assertEqual(
                (self.repo_root / relative_path).read_text(encoding='utf-8'),
                contents,
            )

    def test_partial_apply_failure_restores_only_applied_paths(self):
        real_git = update_go_version._git

        def partially_apply_then_fail(repo_root, *arguments):
            if arguments[:2] == ('apply', '--whitespace=nowarn'):
                patch_path = arguments[-1]
                real_git(repo_root, 'apply', '--include=Dockerfile',
                         '--whitespace=nowarn', patch_path)
                self.assertIn(
                    'golang:1.28.3',
                    (self.repo_root / 'Dockerfile').read_text(encoding='utf-8'),
                )
                raise RuntimeError('simulated partial apply failure')
            return real_git(repo_root, *arguments)

        with mock.patch.object(update_go_version,
                               '_git', side_effect=partially_apply_then_fail):
            with self.assertRaisesRegex(
                    RuntimeError,
                    'simulated partial apply failure.*recovery bundle retained'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)
        for relative_path, contents in self.files.items():
            self.assertEqual(
                (self.repo_root / relative_path).read_text(encoding='utf-8'),
                contents,
            )

    def test_partial_apply_recovery_preserves_unresolved_paths(self):
        real_git = update_go_version._git
        concurrent = '// concurrent managed edit\n'

        def partially_apply_edit_then_fail(repo_root, *arguments):
            if arguments[:2] == ('apply', '--whitespace=nowarn'):
                patch_path = arguments[-1]
                real_git(repo_root, 'apply', '--include=Dockerfile',
                         '--whitespace=nowarn', patch_path)
                self.assertIn(
                    'golang:1.28.3',
                    (self.repo_root / 'Dockerfile').read_text(encoding='utf-8'),
                )
                (self.repo_root / 'go.mod').write_text(concurrent,
                                                       encoding='utf-8')
                raise RuntimeError('simulated partial apply failure')
            return real_git(repo_root, *arguments)

        with mock.patch.object(
                update_go_version,
                '_git',
                side_effect=partially_apply_edit_then_fail,
        ):
            with self.assertRaisesRegex(
                    RuntimeError,
                    'managed paths left unchanged.*go.mod.*recovery bundle retained'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertEqual(
            (self.repo_root / 'Dockerfile').read_text(encoding='utf-8'),
            self.files[Path('Dockerfile')],
        )
        self.assertEqual(
            (self.repo_root / 'go.mod').read_text(encoding='utf-8'),
            concurrent,
        )
        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)
        self.assertTrue(self._restore_patch(Path('go.mod')).exists())

    def test_missing_path_gets_self_contained_original_restore_patch(self):
        root = self.repo_root / 'go.mod'

        def remove_then_fail(repo_root):
            if Path(repo_root) == self.repo_root:
                root.unlink()
                raise RuntimeError('simulated verification failure')

        with mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=remove_then_fail,
        ):
            with self.assertRaisesRegex(
                    RuntimeError,
                    r'go.mod \(original restore patch: .*\.bundle/'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertFalse(root.exists())
        restore_patch = self._restore_patch(Path('go.mod'))
        self._git('apply', str(restore_patch))
        self.assertEqual(root.read_text(encoding='utf-8'),
                         self.files[Path('go.mod')])

    def test_original_restore_patch_preserves_executable_mode(self):
        dockerfile = self.repo_root / 'Dockerfile'
        dockerfile.chmod(0o755)
        self._git('add', 'Dockerfile')
        self._git('-c', 'user.name=Test', '-c',
                  'user.email=test@example.com', 'commit', '-qm',
                  'executable Dockerfile')

        def remove_then_fail(repo_root):
            if Path(repo_root) == self.repo_root:
                dockerfile.unlink()
                raise RuntimeError('simulated verification failure')

        with mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=remove_then_fail,
        ):
            with self.assertRaisesRegex(RuntimeError,
                                        'original restore patch'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        restore_patch = self._restore_patch(Path('Dockerfile'))
        self._git('apply', str(restore_patch))
        self.assertEqual(dockerfile.read_text(encoding='utf-8'),
                         self.files[Path('Dockerfile')])
        self.assertEqual(stat.S_IMODE(dockerfile.stat().st_mode), 0o755)

    def test_non_owner_execute_mode_is_unresolved_and_preserved(self):
        dockerfile = self.repo_root / 'Dockerfile'
        dockerfile.chmod(0o755)
        self._git('add', 'Dockerfile')
        self._git('-c', 'user.name=Test', '-c',
                  'user.email=test@example.com', 'commit', '-qm',
                  'executable Dockerfile')

        def change_mode_then_fail(repo_root):
            if Path(repo_root) == self.repo_root:
                dockerfile.chmod(0o645)
                raise RuntimeError('simulated mode-only interference')

        with mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=change_mode_then_fail,
        ):
            with self.assertRaisesRegex(
                    RuntimeError,
                    'managed paths left unchanged.*Dockerfile.*original '
                    'restore patch'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertIn('golang:1.28.3',
                      dockerfile.read_text(encoding='utf-8'))
        self.assertEqual(stat.S_IMODE(dockerfile.stat().st_mode), 0o645)
        restore_patch = self._restore_patch(Path('Dockerfile'))
        concurrent = self.repo_root / 'Dockerfile.concurrent'
        dockerfile.rename(concurrent)
        self._git('apply', str(restore_patch))
        self.assertEqual(dockerfile.read_text(encoding='utf-8'),
                         self.files[Path('Dockerfile')])
        self.assertEqual(stat.S_IMODE(dockerfile.stat().st_mode), 0o755)
        self.assertEqual(stat.S_IMODE(concurrent.stat().st_mode), 0o645)

    def test_git_compatible_permission_change_does_not_block_rollback(self):
        dockerfile = self.repo_root / 'Dockerfile'
        dockerfile.chmod(0o755)
        self._git('add', 'Dockerfile')
        self._git('-c', 'user.name=Test', '-c',
                  'user.email=test@example.com', 'commit', '-qm',
                  'executable Dockerfile')

        def change_compatible_permissions_then_fail(repo_root):
            if Path(repo_root) == self.repo_root:
                dockerfile.chmod(0o754)
                raise RuntimeError('simulated compatible mode change')

        with mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=change_compatible_permissions_then_fail,
        ):
            with self.assertRaisesRegex(
                    RuntimeError,
                    'simulated compatible mode change.*recovery bundle '
                    'retained') as context:
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertNotIn('managed paths left unchanged',
                         str(context.exception))
        self.assertNotIn('automatic rollback failed', str(context.exception))
        self.assertEqual(dockerfile.read_text(encoding='utf-8'),
                         self.files[Path('Dockerfile')])
        # Recovery promises Git's executable mode, not preservation of every
        # group/other POSIX permission bit.
        self.assertTrue(dockerfile.stat().st_mode & stat.S_IXUSR)

    def test_truncated_path_restore_patch_preserves_then_recovers(self):
        root = self.repo_root / 'go.mod'
        truncated = b'module truncated\n'

        def truncate_then_fail(repo_root):
            if Path(repo_root) == self.repo_root:
                root.write_bytes(truncated)
                raise RuntimeError('simulated verification failure')

        with mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=truncate_then_fail,
        ):
            with self.assertRaisesRegex(RuntimeError,
                                        'original restore patch'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        restore_patch = self._restore_patch(Path('go.mod'))
        with self.assertRaises(subprocess.CalledProcessError):
            self._git('apply', str(restore_patch))
        self.assertEqual(root.read_bytes(), truncated)
        concurrent = self.repo_root / 'go.mod.concurrent'
        root.rename(concurrent)
        self._git('apply', str(restore_patch))
        self.assertEqual(root.read_text(encoding='utf-8'),
                         self.files[Path('go.mod')])
        self.assertEqual(concurrent.read_bytes(), truncated)

    def test_symlink_path_restore_patch_preserves_then_recovers(self):
        root = self.repo_root / 'go.mod'

        def replace_with_symlink_then_fail(repo_root):
            if Path(repo_root) == self.repo_root:
                root.unlink()
                root.symlink_to('concurrent-go.mod')
                raise RuntimeError('simulated verification failure')

        with mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=replace_with_symlink_then_fail,
        ):
            with self.assertRaisesRegex(RuntimeError,
                                        'original restore patch'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        restore_patch = self._restore_patch(Path('go.mod'))
        with self.assertRaises(subprocess.CalledProcessError):
            self._git('apply', str(restore_patch))
        self.assertTrue(root.is_symlink())
        self.assertEqual(root.readlink(), Path('concurrent-go.mod'))
        concurrent = self.repo_root / 'go.mod.concurrent-link'
        root.rename(concurrent)
        self._git('apply', str(restore_patch))
        self.assertEqual(root.read_text(encoding='utf-8'),
                         self.files[Path('go.mod')])
        self.assertTrue(concurrent.is_symlink())

    def test_path_recovery_continues_after_one_path_fails(self):
        real_apply = update_go_version._git_apply_contents
        failed = False

        def fail_first_recovery(repo_root, patch, *, reverse, check=False):
            nonlocal failed
            if not failed:
                failed = True
                raise RuntimeError('simulated path recovery failure')
            return real_apply(repo_root, patch, reverse=reverse, check=check)

        def fail_current_repository(repo_root):
            if Path(repo_root) == self.repo_root:
                raise RuntimeError('simulated verification failure')

        with mock.patch.object(
                update_go_version,
                '_git_apply_contents',
                side_effect=fail_first_recovery,
        ), mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=fail_current_repository,
        ):
            with self.assertRaisesRegex(
                    RuntimeError,
                    'automatic rollback failed: go.mod: simulated path '
                    'recovery failure.*recovery bundle retained'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertIn(
            'toolchain go1.28.3',
            (self.repo_root / 'go.mod').read_text(encoding='utf-8'),
        )
        for relative_path, contents in self.files.items():
            if relative_path != Path('go.mod'):
                self.assertEqual(
                    (self.repo_root /
                     relative_path).read_text(encoding='utf-8'), contents)
        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)
        self.assertTrue(self._restore_patch(Path('go.mod')).exists())

    def test_path_recovery_finishes_before_reraising_interrupt(self):
        real_apply = update_go_version._git_apply_contents
        interrupted = False

        def interrupt_first_recovery(repo_root, patch, *, reverse,
                                     check=False):
            nonlocal interrupted
            if not interrupted:
                interrupted = True
                raise KeyboardInterrupt
            return real_apply(repo_root, patch, reverse=reverse, check=check)

        def fail_current_repository(repo_root):
            if Path(repo_root) == self.repo_root:
                raise RuntimeError('simulated verification failure')

        with mock.patch.object(
                update_go_version,
                '_git_apply_contents',
                side_effect=interrupt_first_recovery,
        ), mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=fail_current_repository,
        ):
            with self.assertWarnsRegex(
                    RuntimeWarning,
                    'Go update recovery interrupted.*recovery bundle retained'):
                with self.assertRaises(KeyboardInterrupt):
                    update_go_version.sync(
                        self.repo_root,
                        '1.28.3',
                        digest_resolver=lambda tag: DIGESTS[tag],
                        repository_paths=self.files,
                    )

        self.assertIn(
            'toolchain go1.28.3',
            (self.repo_root / 'go.mod').read_text(encoding='utf-8'),
        )
        for relative_path, contents in self.files.items():
            if relative_path != Path('go.mod'):
                self.assertEqual(
                    (self.repo_root /
                     relative_path).read_text(encoding='utf-8'), contents)
        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)
        self.assertTrue(self._restore_patch(Path('go.mod')).exists())

    def test_failed_post_apply_verification_rolls_back_and_keeps_patch(self):

        def fail_current_repository(repo_root):
            if Path(repo_root) == self.repo_root:
                raise RuntimeError('simulated verification failure')

        with mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=fail_current_repository,
        ):
            with self.assertRaisesRegex(RuntimeError,
                                        'recovery bundle retained at'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)
        for relative_path, contents in self.files.items():
            self.assertEqual(
                (self.repo_root / relative_path).read_text(encoding='utf-8'),
                contents,
            )

    def test_concurrent_edit_after_apply_is_not_overwritten_by_rollback(self):
        root = self.repo_root / 'go.mod'
        concurrent = '// concurrent replacement\n'

        def edit_then_fail(repo_root):
            if Path(repo_root) == self.repo_root:
                root.write_text(concurrent, encoding='utf-8')
                raise RuntimeError('simulated verification failure')

        with mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=edit_then_fail,
        ):
            with self.assertRaisesRegex(RuntimeError,
                                        'recovery bundle retained at'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertEqual(root.read_text(encoding='utf-8'), concurrent)
        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)
        self.assertTrue(self._restore_patch(Path('go.mod')).exists())

    def test_keyboard_interrupt_rolls_back_and_keeps_recovery_patch(self):

        def interrupt_current_repository(repo_root):
            if Path(repo_root) == self.repo_root:
                raise KeyboardInterrupt

        with mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=interrupt_current_repository,
        ):
            with self.assertWarnsRegex(RuntimeWarning,
                                       'recovery bundle retained at'):
                with self.assertRaises(KeyboardInterrupt):
                    update_go_version.sync(
                        self.repo_root,
                        '1.28.3',
                        digest_resolver=lambda tag: DIGESTS[tag],
                        repository_paths=self.files,
                    )

        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)
        for relative_path, contents in self.files.items():
            self.assertEqual(
                (self.repo_root / relative_path).read_text(encoding='utf-8'),
                contents,
            )

    def test_interrupt_after_git_apply_rolls_back_and_keeps_patch(self):
        real_git = update_go_version._git

        def apply_then_interrupt(repo_root, *arguments):
            if arguments[:2] == ('apply', '--whitespace=nowarn'):
                real_git(repo_root, *arguments)
                raise KeyboardInterrupt
            return real_git(repo_root, *arguments)

        with mock.patch.object(update_go_version,
                               '_git', side_effect=apply_then_interrupt):
            with self.assertWarnsRegex(RuntimeWarning,
                                       'recovery bundle retained at'):
                with self.assertRaises(KeyboardInterrupt):
                    update_go_version.sync(
                        self.repo_root,
                        '1.28.3',
                        digest_resolver=lambda tag: DIGESTS[tag],
                        repository_paths=self.files,
                    )

        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)
        for relative_path, contents in self.files.items():
            self.assertEqual(
                (self.repo_root / relative_path).read_text(encoding='utf-8'),
                contents,
            )

    def test_invalid_utf8_concurrent_edit_does_not_mask_recovery_error(self):
        root = self.repo_root / 'go.mod'

        def corrupt_then_fail(repo_root):
            if Path(repo_root) == self.repo_root:
                root.write_bytes(b'\xff')
                raise RuntimeError('simulated verification failure')

        with mock.patch.object(
                update_go_version,
                '_verify_repository_consistency',
                side_effect=corrupt_then_fail,
        ):
            with self.assertRaisesRegex(
                    RuntimeError,
                    'simulated verification failure.*recovery bundle retained'):
                update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertEqual(root.read_bytes(), b'\xff')
        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)
        self.assertTrue(self._restore_patch(Path('go.mod')).exists())

    def test_interrupt_before_recovery_patch_does_not_report_none(self):
        with mock.patch.object(update_go_version,
                               '_write_recovery_bundle',
                               side_effect=KeyboardInterrupt):
            with self.assertWarnsRegex(
                    RuntimeWarning,
                    'interrupted before application; managed files were not changed') as warning:
                with self.assertRaises(KeyboardInterrupt):
                    update_go_version.sync(
                        self.repo_root,
                        '1.28.3',
                        digest_resolver=lambda tag: DIGESTS[tag],
                        repository_paths=self.files,
                    )

        self.assertNotIn('None', str(warning.warning))
        self.assertEqual(self._recovery_patches(), [])

    def test_success_leaves_index_and_recovery_directory_clean(self):
        changed = update_go_version.sync(
            self.repo_root,
            '1.28.3',
            digest_resolver=lambda tag: DIGESTS[tag],
            repository_paths=self.files,
        )

        self.assertEqual(set(changed), set(self.files))
        self.assertEqual(self._git('diff', '--cached', '--name-only').stdout,
                         '')
        self.assertEqual(self._recovery_patches(), [])

    def test_recovery_patch_cleanup_failure_is_only_a_warning(self):
        with mock.patch.object(
                update_go_version.shutil,
                'rmtree',
                side_effect=OSError('simulated cleanup failure'),
        ):
            with self.assertWarnsRegex(RuntimeWarning,
                                       'committed and verified'):
                changed = update_go_version.sync(
                    self.repo_root,
                    '1.28.3',
                    digest_resolver=lambda tag: DIGESTS[tag],
                    repository_paths=self.files,
                )

        self.assertEqual(set(changed), set(self.files))
        self.assertIn('toolchain go1.28.3',
                      (self.repo_root / 'go.mod').read_text(encoding='utf-8'))
        self.assertEqual(len(self._recovery_patches()), len(self.files) + 1)

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
            f'FROM GOLANG:1.28.0@{OLD_DIGEST} AS builder\n',
            f'FROM golang:1.28.0-foo:bar@{OLD_DIGEST} AS builder\n',
            'FROM ${IMAGE:-golang} AS builder\n',
            'FROM golang${TAG} AS builder\n',
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

    def test_rejects_external_golang_copy_and_run_mount_sources(self):
        dockerfile = self.repo_root / 'Dockerfile'
        for instruction in (
                'COPY --from=golang:1.27.0 /go/bin/go /usr/bin/go\n',
                'RUN --mount=from=golang:1.27.0,target=/go true\n',
                'COPY --from=${IMAGE:-golang} /go/bin/go /usr/bin/go\n',
                'RUN --mount=from=golang${TAG},target=/go true\n',
                'RUN <<EOF\ncurl https://go.dev/dl/go1.27.0.linux-amd64.tar.gz\nEOF\n'):
            with self.subTest(instruction=instruction):
                dockerfile.write_text(
                    f'FROM golang:1.28.0@{OLD_DIGEST} AS builder\n' +
                    instruction,
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

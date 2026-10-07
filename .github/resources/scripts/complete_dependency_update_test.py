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
"""Exercise real update plans while replacing registry/compiler boundaries."""

import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest import mock

import complete_dependency_update as completion
import sync_argo_versions_test as argo_tests


class CompletionTest(unittest.TestCase):

    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name).resolve()

    def write(self, path, contents):
        destination = self.root / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_text(contents)

    def snapshot(self):
        return {
            str(p.relative_to(self.root)): p.read_bytes()
            for p in self.root.rglob('*')
            if p.is_file() and '.git' not in p.relative_to(self.root).parts
        }

    def git(self, *arguments):
        return subprocess.run(['git', *arguments],
                              cwd=self.root,
                              check=True,
                              capture_output=True,
                              text=True)

    def commit(self):
        self.git('add', '.')
        self.git('-c', 'user.name=Fixture', '-c',
                 'user.email=fixture@example.test', '-c',
                 'commit.gpgsign=false', 'commit', '-qm', 'fixture')

    def go_fixture(self):
        for relative in (completion.GO_OUTPUT_PATHS +
                         completion.go.MANAGED_SETUP_GO_ACTIONS):
            self.write(relative,
                       (completion.TRUSTED_ROOT / relative).read_text())
        self.git('init', '-q')
        self.commit()

    def image_versions(self, digest):
        return {'linux/amd64': '1.27.1', 'linux/arm64': '1.27.1'}

    def test_go_preserves_partial_digest_proposal_and_normalizes_newline(self):
        self.go_fixture()
        candidate = next(pin for pin in completion.go.MANAGED_DOCKERFILES
                         if pin.flavor == '-alpine')
        path = self.root / candidate.path
        metadata = completion.go._docker_metadata(path.read_text(), candidate)
        digest = 'sha256:' + 'd' * 64
        path.write_text(path.read_text().replace(metadata.digest,
                                                 digest).rstrip('\n'))
        self.commit()
        before = self.snapshot()
        with mock.patch.object(
                completion.go,
                'resolve_image_versions',
                side_effect=self.image_versions), mock.patch.object(
                    completion.go, 'resolve_image_digest') as resolve:
            changed = completion.complete(self.root, 'go', '1.27.1',
                                          [f'1.27.1-alpine@{digest}'])
            self.assertTrue(changed)
            for pin in completion.go.MANAGED_DOCKERFILES:
                contents = (self.root / pin.path).read_text()
                self.assertTrue(contents.endswith('\n'))
                self.assertIn('golang:1.27.1', contents)
                if pin.flavor == '-alpine':
                    self.assertIn(digest, contents)
                else:
                    self.assertEqual(before[str(pin.path)], contents.encode())
            self.assertEqual(
                completion.complete(self.root, 'go', '1.27.1',
                                    [f'1.27.1-alpine@{digest}']), [])
            resolve.assert_not_called()

    def test_go_registry_failure_or_wrong_architecture_does_not_write(self):
        self.go_fixture()
        before = self.snapshot()
        for response in (RuntimeError('registry unavailable'), {
                'linux/amd64': '1.27.1',
                'linux/arm64': '1.27.0'
        }):
            with self.subTest(response=response), mock.patch.object(
                    completion.go, 'resolve_image_versions') as resolve:
                if isinstance(response, Exception):
                    resolve.side_effect = response
                else:
                    resolve.return_value = response
                with self.assertRaises((RuntimeError, ValueError)):
                    completion.complete(self.root, 'go', '1.27.1')
                self.assertEqual(before, self.snapshot())

    def test_go_rejects_unknown_modules_and_absent_candidate_pins(self):
        self.go_fixture()
        before = self.snapshot()
        with mock.patch.object(completion.go,
                               'resolve_image_versions') as resolve:
            with self.assertRaisesRegex(ValueError, 'absent'):
                completion.complete(self.root, 'go', '1.27.1',
                                    ['1.27.1-alpine@sha256:' + 'e' * 64])
            self.assertEqual(before, self.snapshot())
            self.write(
                Path('unknown/go.mod'),
                'module example.test/unknown\ngo 1.27.1\n')
            self.commit()
            with self.assertRaisesRegex(ValueError, 'inventory changed'):
                completion.complete(self.root, 'go', '1.27.1')
            resolve.assert_not_called()

    def argo_fixture(self):
        fixture = argo_tests.SyncArgoVersionsTest()
        fixture.setUp()
        self.addCleanup(fixture.tearDown)
        fixture.add_runtime_files()
        self.root = fixture.repo_root
        return fixture

    def test_argo_completes_module_only_input_with_real_synchronizer(self):
        fixture = self.argo_fixture()
        module = self.root / 'go.mod'
        module.write_text(module.read_text().replace('v4.1.2', 'v4.1.4'))
        # A candidate script must never be imported or executed.
        self.write(
            Path('.github/resources/scripts/sync_argo_versions.py'),
            'raise AssertionError("candidate script executed")\n')
        with mock.patch.object(
                completion.argo,
                '_tidied_module',
                side_effect=fixture.resolved_module):
            changed = completion.complete(self.root, 'argo', 'v4.1.4')
            self.assertTrue(set(changed).issubset(completion.ARGO_OUTPUT_PATHS))
            self.assertEqual(
                completion.argo.sync(self.root, scope='all', check=True), [])
            self.assertEqual(
                completion.complete(self.root, 'argo', 'v4.1.4'), [])
        self.assertEqual(
            (self.root / 'third_party/argo/COMPATIBILITY_VERSION').read_text(),
            'v4.0.5\n')

    def test_argo_rejects_major_downgrade_and_symlink_before_go(self):
        self.argo_fixture()
        before = self.snapshot()
        with mock.patch.object(completion.argo, '_tidied_module') as tidy:
            for version in ('v5.0.0', 'v4.0.6', 'v4.1.4-rc1'):
                with self.subTest(
                        version=version), self.assertRaises(ValueError):
                    completion.complete(self.root, 'argo', version)
                self.assertEqual(before, self.snapshot())
            path = self.root / 'go.sum'
            path.rename(self.root / 'sums')
            path.symlink_to('sums')
            with self.assertRaisesRegex(ValueError, 'symlinks'):
                completion.complete(self.root, 'argo', 'v4.1.4')
            tidy.assert_not_called()

    def gh_aw_fixture(self):
        trusted = self.root / 'trusted'
        candidate = self.root / 'candidate'
        trusted.mkdir()
        for relative in completion.GH_AW_OUTPUT_PATHS + (
                completion.GH_AW_SOURCE,):
            destination = trusted / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(completion.TRUSTED_ROOT / relative, destination)
        shutil.copytree(trusted, candidate)
        setup = candidate / completion.GH_AW_SETUP
        setup.write_text(setup.read_text().replace('# v0.87.2', '# v0.89.17'))
        return trusted, candidate

    def compiler_runner(self, args, cwd):
        if args[:3] == ['gh', 'aw', '--version']:
            return subprocess.CompletedProcess(args, 0,
                                               'gh aw version v0.89.17\n', '')
        if args[:3] == ['gh', 'aw', 'compile']:
            self.assertEqual(args[3:], [
                str(completion.GH_AW_SOURCE), '--action-mode', 'action',
                '--action-tag', 'v0.89.17', '--no-check-update'
            ])
            self.assertFalse((cwd / 'candidate-marker').exists())
            self.assertEqual(
                (cwd / completion.GH_AW_SOURCE).read_text(),
                (completion.TRUSTED_ROOT / completion.GH_AW_SOURCE).read_text())
            lock = cwd / completion.GH_AW_LOCK
            lock.write_text(
                '# gh-aw-metadata: {"compiler_version":"v0.89.17"}\n'
                'jobs:\n  test:\n    steps:\n'
                '      - name: Set up gh-aw\n        uses: github/gh-aw-actions/setup@'
                + 'f' * 40 + ' # v0.89.17\n')
            (cwd / completion.GH_AW_ACTIONS_LOCK).write_text(
                json.dumps({
                    'entries': {
                        'github/gh-aw-actions/setup@v0.89.17': {
                            'sha': 'f' * 40
                        }
                    }
                }) + '\n')
        return subprocess.CompletedProcess(args, 0, '', '')

    def test_gh_aw_compiles_trusted_source_and_updates_source_version_idempotently(
            self):
        trusted, candidate = self.gh_aw_fixture()
        (candidate /
         'candidate-marker').write_text('must not enter compile workspace')
        with mock.patch.object(
                completion, '_run', side_effect=self.compiler_runner):
            changed = completion.complete_gh_aw(candidate, 'v0.89.17', trusted)
            self.assertEqual(set(changed), set(completion.GH_AW_OUTPUT_PATHS))
            self.assertIn('default: v0.89.17',
                          (candidate / completion.GH_AW_SETUP).read_text())
            self.assertEqual(
                completion.complete_gh_aw(candidate, 'v0.89.17', trusted), [])

    def test_gh_aw_rejects_changed_source_and_wrong_compiler(self):
        trusted, candidate = self.gh_aw_fixture()
        before = {
            path: (candidate / path).read_text()
            for path in completion.GH_AW_OUTPUT_PATHS
        }
        with mock.patch.object(completion, '_run') as run:
            run.return_value = subprocess.CompletedProcess(
                [], 0, 'gh aw version v0.87.2\n', '')
            with self.assertRaisesRegex(ValueError, 'compiler must be exactly'):
                completion.complete_gh_aw(candidate, 'v0.89.17', trusted)
            (candidate / completion.GH_AW_SOURCE).write_text('untrusted source')
            run.reset_mock()
            with self.assertRaisesRegex(ValueError,
                                        'unchanged trusted Markdown'):
                completion.complete_gh_aw(candidate, 'v0.89.17', trusted)
            run.assert_not_called()
        self.assertEqual(
            before, {path: (candidate / path).read_text() for path in before})

    def test_gh_aw_compile_failure_does_not_change_candidate(self):
        trusted, candidate = self.gh_aw_fixture()
        before = {
            path: (candidate / path).read_bytes()
            for path in completion.GH_AW_OUTPUT_PATHS
        }

        def fail_compile(args, cwd):
            if args[:3] == ['gh', 'aw', 'compile']:
                raise subprocess.CalledProcessError(
                    1, args, stderr='invalid workflow')
            return self.compiler_runner(args, cwd)

        with mock.patch.object(completion, '_run', side_effect=fail_compile):
            with self.assertRaises(subprocess.CalledProcessError):
                completion.complete_gh_aw(candidate, 'v0.89.17', trusted)
        self.assertEqual(
            before, {path: (candidate / path).read_bytes() for path in before})


if __name__ == '__main__':
    unittest.main()

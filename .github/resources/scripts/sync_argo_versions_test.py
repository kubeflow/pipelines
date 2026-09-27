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
"""Unit tests for synchronizing Argo CI and documentation versions."""

import importlib.util
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest import mock

SCRIPT_PATH = Path(__file__).with_name('sync_argo_versions.py')
SPEC = importlib.util.spec_from_file_location('sync_argo_versions', SCRIPT_PATH)
sync_argo_versions = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(sync_argo_versions)


class SyncArgoVersionsTest(unittest.TestCase):

    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.repo_root = Path(self.temporary_directory.name)
        files = {
            'third_party/argo/VERSION': 'v4.1.2\n',
            # Promote the previous current release into the compatibility slot
            # to verify replacements do not cascade into the new current one.
            'third_party/argo/COMPATIBILITY_VERSION': 'v4.0.5\n',
            '.github/workflows/e2e-test.yml':
                ('        argo_version: ["v3.7.14", "v4.0.5"]\n'),
            '.github/workflows/api-server-tests.yml': (
                '        argo_version: ["v3.7.14", "v4.0.5"]\n'
                "          ARGO_COMPATIBILITY_TESTS: ${{ matrix.argo_version == 'v4.0.5' }}\n"
            ),
            '.github/resources/runtime-base-images.txt':
                ('quay.io/argoproj/workflow-controller:v3.7.14\n'
                 'quay.io/argoproj/argoexec:v3.7.14\n'
                 'quay.io/argoproj/workflow-controller:v4.0.5\n'
                 'quay.io/argoproj/argoexec:v4.0.5\n'),
            'AGENTS.md': 'Argo v3.7.14 and v4.0.5\n',
        }
        for relative_path, contents in files.items():
            path = self.repo_root / relative_path
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(contents, encoding='utf-8')

    def tearDown(self):
        self.temporary_directory.cleanup()

    def test_sync_updates_compatibility_and_current_references(self):
        changed_paths = sync_argo_versions.sync(self.repo_root)

        self.assertEqual(len(changed_paths), 4)
        for relative_path in sync_argo_versions.CI_REFERENCE_PATHS:
            contents = (self.repo_root /
                        relative_path).read_text(encoding='utf-8')
            self.assertNotIn('v3.7.14', contents)
        workflow = (self.repo_root /
                    '.github/workflows/e2e-test.yml').read_text(
                        encoding='utf-8')
        self.assertIn('v4.0.5', workflow)
        self.assertIn('v4.1.2', workflow)
        self.assertIn(
            'Argo v4.0.5 and v4.1.2',
            (self.repo_root / 'AGENTS.md').read_text(encoding='utf-8'),
        )

        self.assertEqual(
            sync_argo_versions.sync(self.repo_root, check=True), [])

    def test_each_workflow_requires_both_supported_versions(self):
        self.add_runtime_files()
        sync_argo_versions.sync(self.repo_root)
        incomplete = (
            'argo_version: ["v4.0.5"]\n',
            'argo_version: ["v4.1.2"]\n',
            'argo_version: ${{ matrix.argo_version }}\n',
        )
        with mock.patch.object(sync_argo_versions, '_tidied_module') as tidy:
            for relative_path in sync_argo_versions.WORKFLOW_PATHS:
                path = self.repo_root / relative_path
                original = path.read_text()
                for contents in incomplete:
                    path.write_text(contents)
                    before = self.snapshot()
                    for check in (False, True):
                        with self.subTest(
                                path=relative_path, contents=contents,
                                check=check):
                            with self.assertRaises(ValueError) as error:
                                sync_argo_versions.sync(
                                    self.repo_root, scope='all', check=check)
                            self.assertIn(
                                str(relative_path), str(error.exception))
                            self.assertEqual(before, self.snapshot())
                path.write_text(original)
            tidy.assert_not_called()

    def test_workflows_require_matching_source_version_pairs(self):
        path = self.repo_root / sync_argo_versions.WORKFLOW_PATHS[0]
        path.write_text('argo_version: ["v3.7.14", "v4.1.2"]\n')
        before = self.snapshot()
        for check in (False, True):
            with self.subTest(check=check):
                with self.assertRaisesRegex(ValueError,
                                            'two Argo versions in CI matrices'):
                    sync_argo_versions.sync(self.repo_root, check=check)
                self.assertEqual(before, self.snapshot())

    def test_single_version_jobs_preserve_workflow_coverage(self):
        for relative_path in sync_argo_versions.WORKFLOW_PATHS:
            path = self.repo_root / relative_path
            path.write_text(
                path.read_text() + 'legacy_job:\n  argo_version: ["v3.7.14"]\n'
                'input:\n  argo_version: ${{ matrix.argo_version }}\n')
        sync_argo_versions.sync(self.repo_root)
        for relative_path in sync_argo_versions.WORKFLOW_PATHS:
            contents = (self.repo_root / relative_path).read_text()
            self.assertIn('argo_version: ["v4.0.5", "v4.1.2"]', contents)
            self.assertIn('legacy_job:\n  argo_version: ["v4.0.5"]', contents)
        self.assertEqual(
            sync_argo_versions.sync(self.repo_root, check=True), [])

    def test_check_reports_changes_without_writing(self):
        changed_paths = sync_argo_versions.sync(self.repo_root, check=True)

        self.assertEqual(len(changed_paths), 4)
        self.assertIn(
            'v4.0.5',
            (self.repo_root /
             '.github/workflows/e2e-test.yml').read_text(encoding='utf-8'),
        )

    def add_runtime_files(self):
        module = 'github.com/argoproj/argo-workflows/v4'
        files = {
            'go.mod':
                'module example.test/pipelines\n\ngo 1.27.0\n\ntoolchain go1.27.1\n\nrequire '
                + module + ' v4.1.2\n',
            'go.sum':
                self.module_sums('v4.1.2'),
            'third_party/argo/UPGRADE.md':
                'ARGO_TAG=v4.1.2\n',
            str(sync_argo_versions.MANIFEST_ROOT /
                'base/workflow-controller-deployment-patch.yaml'):
                'image: quay.io/argoproj/workflow-controller:v4.1.2\n'
                'args: ["quay.io/argoproj/argoexec:v4.1.2"]\n',
            str(sync_argo_versions.MANIFEST_ROOT /
                'base/workflow-controller-configmap-patch.yaml'):
                ''.join(
                    f'# https://github.com/argoproj/argo-workflows/blob/v4.1.2/docs/{i}.md\n'
                    for i in range(3)),
        }
        for path, count in sync_argo_versions.MANIFEST_REFS:
            files[str(path)] = ''.join(
                f'- https://github.com/argoproj/argo-workflows/manifests/{i}?ref=v4.1.2&timeout=60s\n'
                for i in range(count))
        for relative_path, contents in files.items():
            path = self.repo_root / relative_path
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(contents, encoding='utf-8')

    def module_sums(self, version):
        return ''.join(
            f'github.com/argoproj/argo-workflows/v4 {version}{suffix} h1:testhash\n'
            for suffix in ('', '/go.mod'))

    def snapshot(self):
        return {
            str(p.relative_to(self.repo_root)): p.read_bytes()
            for p in self.repo_root.rglob('*')
            if p.is_file()
        }

    def resolved_module(self, repo_root, contents, version):
        return {
            repo_root / 'go.mod': contents,
            repo_root / 'go.sum': self.module_sums(version)
        }

    def test_all_updates_every_runtime_pin_and_is_idempotent(self):
        self.add_runtime_files()
        with mock.patch.object(
                sync_argo_versions,
                '_tidied_module',
                side_effect=self.resolved_module):
            changed = sync_argo_versions.sync(
                self.repo_root, scope='all', version='v4.1.4')
            self.assertEqual(len(changed), 14)
            self.assertEqual(
                (self.repo_root / 'third_party/argo/VERSION').read_text(),
                'v4.1.4\n')
            self.assertEqual(
                (self.repo_root /
                 'third_party/argo/COMPATIBILITY_VERSION').read_text(),
                'v4.0.5\n')
            for path, count in sync_argo_versions.MANIFEST_REFS:
                self.assertEqual(
                    (self.repo_root / path).read_text().count('ref=v4.1.4'),
                    count)
            deployment = (
                self.repo_root / sync_argo_versions.MANIFEST_ROOT /
                'base/workflow-controller-deployment-patch.yaml').read_text()
            self.assertIn('workflow-controller:v4.1.4', deployment)
            self.assertIn('argoexec:v4.1.4', deployment)
            before = self.snapshot()
            self.assertEqual(
                sync_argo_versions.sync(
                    self.repo_root, scope='all', version='v4.1.4'), [])
            self.assertEqual(before, self.snapshot())
        with mock.patch.object(
                sync_argo_versions,
                '_tidied_module',
                side_effect=AssertionError('check must not run Go')):
            self.assertEqual(
                sync_argo_versions.sync(
                    self.repo_root, scope='all', check=True), [])

    def test_all_completes_a_partially_updated_dependabot_module(self):
        self.add_runtime_files()
        path = self.repo_root / 'go.mod'
        path.write_text(path.read_text().replace('v4.1.2', 'v4.1.4'))
        with mock.patch.object(
                sync_argo_versions,
                '_tidied_module',
                side_effect=self.resolved_module):
            sync_argo_versions.sync(
                self.repo_root, scope='all', version='v4.1.4')
        self.assertEqual(
            sync_argo_versions.sync(self.repo_root, scope='all', check=True),
            [])

    def test_invalid_target_or_missing_pin_does_not_write(self):
        self.add_runtime_files()
        before = self.snapshot()
        with mock.patch.object(sync_argo_versions, '_tidied_module') as tidy:
            for version in ('4.1.4', 'v4.1.4-rc1', 'v04.1.4', 'v5.0.0',
                            'v4.0.5'):
                with self.subTest(
                        version=version), self.assertRaises(ValueError):
                    sync_argo_versions.sync(
                        self.repo_root, scope='all', version=version)
                self.assertEqual(before, self.snapshot())
            path = self.repo_root / sync_argo_versions.MANIFEST_REFS[0][0]
            path.write_text('resources: []\n')
            before = self.snapshot()
            with self.assertRaisesRegex(ValueError,
                                        'expected 1 Argo version references'):
                sync_argo_versions.sync(
                    self.repo_root, scope='all', version='v4.1.4')
            self.assertEqual(before, self.snapshot())
            tidy.assert_not_called()

    def test_stable_pins_reject_prereleases_and_digest_suffixes(self):
        self.add_runtime_files()
        path = self.repo_root / sync_argo_versions.MANIFEST_ROOT / 'base/workflow-controller-deployment-patch.yaml'
        original = path.read_text()
        for suffix in ('-rc1', '@sha256:1234'):
            with self.subTest(suffix=suffix):
                path.write_text(
                    original.replace('workflow-controller:v4.1.2',
                                     'workflow-controller:v4.1.2' + suffix))
                before = self.snapshot()
                with self.assertRaisesRegex(
                        ValueError, 'expected 1 Argo version references'):
                    sync_argo_versions.sync(
                        self.repo_root, scope='all', version='v4.1.4')
                self.assertEqual(before, self.snapshot())

    def test_check_reports_missing_module_checksums_without_running_go(self):
        self.add_runtime_files()
        (self.repo_root / 'go.sum').write_text('')
        with mock.patch.object(
                sync_argo_versions,
                '_tidied_module',
                side_effect=AssertionError('check must not run Go')):
            before = self.snapshot()
            changed = sync_argo_versions.sync(
                self.repo_root, scope='all', check=True)
            self.assertIn(self.repo_root / 'go.sum', changed)
            self.assertEqual(before, self.snapshot())

    def test_ci_replacement_does_not_match_version_prefix(self):
        for path in sync_argo_versions.WORKFLOW_PATHS:
            (self.repo_root /
             path).write_text('argo_version: ["v4.1.2", "v4.1.20"]\n')
        (self.repo_root / 'third_party/argo/VERSION').write_text('v4.2.0\n')
        (self.repo_root /
         'third_party/argo/COMPATIBILITY_VERSION').write_text('v4.1.4\n')
        (self.repo_root / 'AGENTS.md').write_text('Argo v4.1.2 and v4.1.20\n')
        preload = self.repo_root / '.github/resources/runtime-base-images.txt'
        preload.write_text(preload.read_text().replace('v3.7.14',
                                                       'v4.1.2').replace(
                                                           'v4.0.5', 'v4.1.20'))
        sync_argo_versions.sync(self.repo_root)
        self.assertEqual((self.repo_root / 'AGENTS.md').read_text(),
                         'Argo v4.1.4 and v4.2.0\n')
        self.assertEqual(
            sync_argo_versions.sync(self.repo_root, check=True), [])

    def test_preload_validation_rejects_missing_duplicate_stale_or_digest_pins(
            self):
        path = self.repo_root / '.github/resources/runtime-base-images.txt'
        original = path.read_text()
        image = 'quay.io/argoproj/argoexec:v4.0.5'
        invalid = (original.replace(image + '\n', ''), original + image + '\n',
                   original.replace(image, 'quay.io/argoproj/argoexec:v4.0.8'),
                   original.replace(image, image + '@sha256:1234'))
        for contents in invalid:
            with self.subTest(contents=contents):
                path.write_text(contents)
                before = self.snapshot()
                with self.assertRaisesRegex(ValueError,
                                            'runtime-base-images.txt'):
                    sync_argo_versions.sync(self.repo_root)
                self.assertEqual(before, self.snapshot())

    def test_ci_rejects_prerelease_lanes(self):
        path = self.repo_root / sync_argo_versions.WORKFLOW_PATHS[0]
        path.write_text('argo_version: ["v3.7.14", "v4.0.5-rc1"]\n')
        before = self.snapshot()
        with self.assertRaisesRegex(ValueError, 'exact stable Argo version'):
            sync_argo_versions.sync(self.repo_root)
        self.assertEqual(before, self.snapshot())

    def test_make_can_select_compatibility_runtime_without_updating_go(self):
        self.add_runtime_files()
        (self.repo_root / 'third_party/argo/VERSION').write_text('v3.7.18\n')
        script = self.repo_root / '.github/resources/scripts/sync_argo_versions.py'
        script.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(SCRIPT_PATH, script)
        shutil.copyfile(SCRIPT_PATH.parents[3] / 'third_party/argo/Makefile',
                        self.repo_root / 'third_party/argo/Makefile')
        bin_path = self.repo_root / 'bin'
        bin_path.mkdir()
        fake_go = bin_path / 'go'
        fake_go.write_text(
            '#!/bin/sh\necho "Go must not run for runtime selection" >&2\nexit 23\n'
        )
        fake_go.chmod(0o755)
        before = self.snapshot()
        command = [
            'make', '-C',
            str(self.repo_root / 'third_party/argo'), 'update_manifests'
        ]
        environment = dict(
            os.environ, PATH=str(bin_path) + os.pathsep + os.environ['PATH'])
        result = subprocess.run(
            command, env=environment, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        after = self.snapshot()
        self.assertEqual(before.keys(), after.keys())
        changed = {path for path in before if before[path] != after[path]}
        expected = {str(path) for path, _ in sync_argo_versions.MANIFEST_REFS}
        expected.update(
            str(sync_argo_versions.MANIFEST_ROOT / 'base' / name)
            for name in ('workflow-controller-deployment-patch.yaml',
                         'workflow-controller-configmap-patch.yaml'))
        self.assertEqual(changed, expected)
        for path in changed:
            self.assertIn(b'v3.7.18', after[path])
            self.assertNotIn(b'v4.1.2', after[path])
        self.assertEqual(
            sync_argo_versions.sync(
                self.repo_root, scope='manifests', check=True), [])
        repeated = subprocess.run(
            command, env=environment, capture_output=True, text=True)
        self.assertEqual(repeated.returncode, 0, repeated.stderr)
        self.assertEqual(after, self.snapshot())

    def test_backend_update_still_rejects_a_different_module_major(self):
        self.add_runtime_files()
        (self.repo_root / 'third_party/argo/VERSION').write_text('v3.7.18\n')
        before = self.snapshot()
        with mock.patch.object(sync_argo_versions, '_tidied_module') as tidy:
            with self.assertRaisesRegex(
                    ValueError, 'major upgrades require a code migration'):
                sync_argo_versions.sync(self.repo_root, scope='backend')
            tidy.assert_not_called()
        self.assertEqual(before, self.snapshot())

    def test_make_propagates_go_failure_without_tracked_writes(self):
        self.add_runtime_files()
        script = self.repo_root / '.github/resources/scripts/sync_argo_versions.py'
        script.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(SCRIPT_PATH, script)
        shutil.copyfile(SCRIPT_PATH.parents[3] / 'third_party/argo/Makefile',
                        self.repo_root / 'third_party/argo/Makefile')
        bin_path = self.repo_root / 'bin'
        bin_path.mkdir()
        fake_go = bin_path / 'go'
        fake_go.write_text(
            '#!/bin/sh\nfor arg do\n case "$arg" in -modfile=*) printf "\\nchanged by failed Go\\n" >> "${arg#-modfile=}";; esac\ndone\nprintf "resolution failed\\n" >&2\nexit 23\n'
        )
        fake_go.chmod(0o755)
        before = self.snapshot()
        result = subprocess.run([
            'make', '-C',
            str(self.repo_root / 'third_party/argo'), 'update',
            'ARGO_VERSION=v4.1.4'
        ],
                                env=dict(
                                    os.environ,
                                    PATH=str(bin_path) + os.pathsep +
                                    os.environ['PATH']),
                                capture_output=True,
                                text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('resolution failed', result.stderr)
        self.assertEqual(before, self.snapshot())

    def test_go_result_cannot_silently_change_compiler(self):
        self.add_runtime_files()

        def rewrite_module(args, **kwargs):
            path = Path(args[-1].split('=', 1)[1])
            path.write_text(path.read_text().replace('go 1.27.0', 'go 1.28.0'))
            path.with_suffix('.sum').write_text(self.module_sums('v4.1.4'))

        before = self.snapshot()
        with mock.patch.object(
                sync_argo_versions.subprocess, 'run',
                side_effect=rewrite_module):
            with self.assertRaisesRegex(ValueError, 'changes the Go compiler'):
                sync_argo_versions.sync(
                    self.repo_root, scope='all', version='v4.1.4')
        self.assertEqual(before, self.snapshot())


if __name__ == '__main__':
    unittest.main()

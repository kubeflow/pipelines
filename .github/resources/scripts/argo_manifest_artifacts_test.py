#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Pinned Argo sources are fetched once and consumed without network access."""

import copy
import io
import json
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
import time
import unittest
from pathlib import Path
from unittest import mock

import argo_manifest_artifacts as argo
from shared_ci_images_test import FakeAPI, identity

RESOURCE_PATHS = (
    'manifests/base/workflow-controller',
    'manifests/base/crds/minimal',
    'manifests/namespace-install/workflow-controller-rbac',
    'manifests/cluster-install-no-crds/workflow-controller-rbac',
)


class FetchRetryTest(unittest.TestCase):
    def process(self, waits, pid):
        process = mock.MagicMock()
        process.__enter__.return_value = process
        process.pid = pid
        process.wait.side_effect = waits
        return process

    def test_failed_fetch_and_timeout_retry_before_success(self):
        processes = [
            self.process([128], 101),
            self.process([subprocess.TimeoutExpired(['git'], 180), -9], 102),
            self.process([0], 103),
        ]
        reset = mock.Mock()
        with mock.patch.object(
            argo.subprocess, 'Popen', side_effect=processes
        ) as popen:
            with mock.patch.object(argo.time, 'sleep') as sleep:
                with mock.patch.object(argo.os, 'killpg') as killpg:
                    argo.fetch_with_retry(
                        ['git', 'fetch'], Path('/tmp'), before_attempt=reset
                    )
        self.assertEqual(reset.call_count, 3)
        self.assertEqual(popen.call_count, 3)
        self.assertEqual(sleep.call_args_list, [mock.call(10), mock.call(10)])
        self.assertEqual(
            popen.call_args.kwargs['env']['GIT_TERMINAL_PROMPT'], '0'
        )
        self.assertTrue(popen.call_args.kwargs['start_new_session'])
        self.assertEqual(popen.call_args.kwargs['stdin'], subprocess.DEVNULL)
        killpg.assert_called_once_with(102, argo.signal.SIGKILL)
        self.assertEqual(
            processes[1].wait.call_args_list,
            [mock.call(timeout=180), mock.call()],
        )

    def test_fetch_version_resets_locks_and_partial_objects_between_attempts(
        self,
    ):
        archive_bytes = io.BytesIO()
        payload = b'resources: []\n'
        with tarfile.open(fileobj=archive_bytes, mode='w') as archive:
            member = tarfile.TarInfo(
                'manifests/base/workflow-controller/kustomization.yaml'
            )
            member.size = len(payload)
            archive.addfile(member, io.BytesIO(payload))
        resets = []

        def interrupted_fetch(command, cwd, *, before_attempt):
            self.assertEqual(command[-1], 'refs/tags/v4.1.6')
            before_attempt()
            git_dir = cwd / '.git'
            lock = git_dir / 'shallow.lock'
            partial = git_dir / 'objects/partial-pack'
            lock.write_text('left by interrupted git fetch')
            partial.write_text('incomplete objects')
            before_attempt()
            self.assertTrue(git_dir.is_dir())
            self.assertFalse(lock.exists())
            self.assertFalse(partial.exists())
            resets.append(True)

        with tempfile.TemporaryDirectory() as destination:
            with mock.patch.object(
                argo, 'fetch_with_retry', side_effect=interrupted_fetch
            ):
                with mock.patch.object(
                    argo.subprocess,
                    'check_output',
                    side_effect=['a' * 40, archive_bytes.getvalue()],
                ):
                    commit = argo.fetch_version('v4.1.6', Path(destination))
            self.assertEqual(commit, 'a' * 40)
            self.assertEqual(
                (
                    Path(destination)
                    / 'manifests/base/workflow-controller/kustomization.yaml'
                ).read_bytes(),
                payload,
            )
        self.assertEqual(resets, [True])

    def test_exhaustion_is_bounded_without_final_sleep(self):
        processes = [
            self.process([subprocess.TimeoutExpired(['git'], 180), -9], pid)
            for pid in (101, 102, 103)
        ]
        with mock.patch.object(
            argo.subprocess, 'Popen', side_effect=processes
        ) as popen:
            with mock.patch.object(argo.time, 'sleep') as sleep:
                with mock.patch.object(argo.os, 'killpg') as killpg:
                    with self.assertRaises(subprocess.TimeoutExpired):
                        argo.fetch_with_retry(['git', 'fetch'], Path('/tmp'))
        self.assertEqual(popen.call_count, 3)
        self.assertEqual(sleep.call_count, 2)
        self.assertEqual(killpg.call_count, 3)
        for process in processes:
            self.assertEqual(
                process.wait.call_args_list,
                [mock.call(timeout=180), mock.call()],
            )

    def test_timeout_terminates_fetch_descendants(self):
        with tempfile.TemporaryDirectory() as directory:
            sentinel = Path(directory) / 'late-write'
            ready = Path(directory) / 'child-started'
            child = (
                'from pathlib import Path; import sys, time; '
                'Path(sys.argv[1]).write_text("started"); time.sleep(0.6); '
                'Path(sys.argv[2]).write_text("orphan survived")'
            )
            parent = (
                'import subprocess, sys, time; '
                'subprocess.Popen([sys.executable, "-c", sys.argv[1], '
                'sys.argv[2], sys.argv[3]]); time.sleep(2)'
            )
            with self.assertRaises(subprocess.TimeoutExpired):
                argo.fetch_with_retry(
                    [
                        sys.executable,
                        '-c',
                        parent,
                        child,
                        str(ready),
                        str(sentinel),
                    ],
                    Path(directory),
                    attempts=1,
                    timeout=0.3,
                )
            self.assertTrue(
                ready.exists(), 'the test must start a real descendant'
            )
            time.sleep(0.8)
            self.assertFalse(
                sentinel.exists(), 'timed-out fetch left a live descendant'
            )

    def test_first_success_does_not_retry(self):
        process = self.process([0], 101)
        with mock.patch.object(
            argo.subprocess, 'Popen', return_value=process
        ) as popen:
            with mock.patch.object(argo.time, 'sleep') as sleep:
                with mock.patch.object(argo.os, 'killpg') as killpg:
                    argo.fetch_with_retry(['git', 'fetch'], Path('/tmp'))
        popen.assert_called_once()
        process.wait.assert_called_once_with(timeout=180)
        sleep.assert_not_called()
        killpg.assert_not_called()


class ArgoArtifactTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name) / 'repo'
        pins = self.root / 'third_party/argo'
        pins.mkdir(parents=True)
        for filename in ('VERSION', 'COMPATIBILITY_VERSION'):
            shutil.copyfile(
                argo.ROOT / 'third_party/argo' / filename, pins / filename
            )
        shutil.copytree(
            argo.ROOT / argo.pins.MANIFEST_ROOT,
            self.root / argo.pins.MANIFEST_ROOT,
            ignore=shutil.ignore_patterns('.ci-argo'),
        )
        self.output = Path(temporary.name) / 'artifact'
        self.producer = identity()
        self.consumer = identity(20)
        self.api = FakeAPI(self.producer)
        self.state = {
            'consumer': self.consumer,
            'identity': self.producer,
            'architecture': 'amd64',
            'sets': {},
            'last_kind': 'images',
        }
        self.pinned = argo.versions(self.root)
        self.commits = dict(zip(self.pinned, ['1' * 40, '2' * 40]))
        with mock.patch.object(
            argo, 'fetch_version', side_effect=self.fake_fetch
        ) as fetch:
            argo.prepare(self.output, self.root, self.producer)
        self.fetched = [call.args[0] for call in fetch.call_args_list]

    def fake_fetch(self, version, destination):
        for relative in RESOURCE_PATHS:
            path = destination / relative
            path.mkdir(parents=True)
            (path / 'kustomization.yaml').write_text('resources: []\n')
        return self.commits[version]

    def metadata(self):
        return json.loads((self.output / argo.METADATA).read_text())

    def write_metadata(self, metadata):
        (self.output / argo.METADATA).write_text(json.dumps(metadata))

    def verify(self):
        return argo.verify(
            self.output,
            self.state,
            root=self.root,
            api=self.api,
            consumer=self.consumer,
        )

    def replace_archive(self, entries):
        with tarfile.open(self.output / argo.ARCHIVE, 'w') as archive:
            for name, kind, content in entries:
                member = tarfile.TarInfo(name)
                member.type = kind
                member.size = len(content) if kind == tarfile.REGTYPE else 0
                if kind in (tarfile.SYMTYPE, tarfile.LNKTYPE):
                    member.linkname = '../../escape'
                archive.addfile(
                    member, io.BytesIO(content) if member.isfile() else None
                )
        metadata = self.metadata()
        metadata['sha256'] = argo.shared.checksum(self.output / argo.ARCHIVE)
        self.write_metadata(metadata)

    def test_prepare_fetches_both_pins_once_and_records_exact_commits(self):
        self.assertEqual(self.fetched, self.pinned)
        metadata = self.verify()
        self.assertEqual(metadata['versions'], self.commits)
        self.assertEqual(metadata['identity'], self.producer)
        self.assertEqual(
            metadata['sha256'], argo.shared.checksum(self.output / argo.ARCHIVE)
        )

    def test_producer_identity_or_attempt_mismatch_is_rejected(self):
        for key, value in [
            ('source_sha', 'f' * 40),
            ('run_attempt', 2),
            ('run_id', 99),
        ]:
            with self.subTest(key=key):
                metadata = self.metadata()
                changed = copy.deepcopy(metadata)
                changed['identity'][key] = value
                self.write_metadata(changed)
                with self.assertRaisesRegex(ValueError, 'mismatch'):
                    self.verify()
                self.write_metadata(metadata)

    def test_consumer_source_mismatch_is_rejected(self):
        self.consumer['source_sha'] = 'f' * 40
        with self.assertRaises(argo.shared.InvalidEvidence):
            self.verify()

    def test_live_producer_attempt_change_is_rejected(self):
        self.api.current_run['run_attempt'] = 2
        with self.assertRaises(argo.shared.InvalidEvidence):
            self.verify()

    def test_archive_checksum_mismatch_is_rejected(self):
        with (self.output / argo.ARCHIVE).open('ab') as archive:
            archive.write(b'corruption')
        with self.assertRaisesRegex(ValueError, 'checksum mismatch'):
            self.verify()

    def test_missing_or_changed_version_pin_is_rejected(self):
        metadata = self.metadata()
        del metadata['versions'][self.pinned[0]]
        self.write_metadata(metadata)
        with self.assertRaisesRegex(ValueError, 'version.*mismatch'):
            self.verify()

    def test_unresolved_tag_metadata_is_rejected(self):
        metadata = self.metadata()
        metadata['versions'][self.pinned[0]] = 'refs/tags/' + self.pinned[0]
        self.write_metadata(metadata)
        with self.assertRaisesRegex(ValueError, 'mismatch'):
            self.verify()

    def test_malformed_archive_is_rejected_even_with_matching_checksum(self):
        (self.output / argo.ARCHIVE).write_bytes(b'not a tar archive')
        metadata = self.metadata()
        metadata['sha256'] = argo.shared.checksum(self.output / argo.ARCHIVE)
        self.write_metadata(metadata)
        with self.assertRaises(tarfile.ReadError):
            self.verify()

    def test_archive_traversal_links_duplicates_and_wrong_version_are_rejected(
        self,
    ):
        valid = f'{self.pinned[0]}/manifests/config.yaml'
        for entries in [
            [('../escape', tarfile.REGTYPE, b'x')],
            [('/escape', tarfile.REGTYPE, b'x')],
            [(valid, tarfile.SYMTYPE, b'')],
            [(valid, tarfile.LNKTYPE, b'')],
            [(valid, tarfile.REGTYPE, b'x'), (valid, tarfile.REGTYPE, b'y')],
            [('v0.0.0/manifests/config.yaml', tarfile.REGTYPE, b'x')],
            [(f'{self.pinned[0]}/other/config.yaml', tarfile.REGTYPE, b'x')],
        ]:
            with self.subTest(entries=entries):
                self.replace_archive(entries)
                with self.assertRaises(ValueError):
                    self.verify()

    def test_metadata_and_archive_symlinks_are_rejected(self):
        for filename in (argo.METADATA, argo.ARCHIVE):
            with self.subTest(filename=filename):
                path = self.output / filename
                saved = self.output / (filename + '.saved')
                path.rename(saved)
                path.symlink_to(saved)
                try:
                    with self.assertRaises(
                        (ValueError, argo.shared.InvalidEvidence)
                    ):
                        self.verify()
                finally:
                    path.unlink()
                    saved.rename(path)

    def assert_local_resources(self, version):
        count = 0
        for relative, expected in argo.pins.MANIFEST_REFS:
            path = self.root / relative
            contents = path.read_text()
            self.assertNotIn(
                'https://github.com/argoproj/argo-workflows/', contents
            )
            resources = [
                line.removeprefix('- ').strip()
                for line in contents.splitlines()
                if '.ci-argo/' in line
            ]
            self.assertEqual(len(resources), expected)
            for resource in resources:
                self.assertIn(
                    f'/.ci-argo/{version}/manifests/',
                    str((path.parent / resource).resolve()),
                )
                self.assertTrue((path.parent / resource).is_dir())
            count += len(resources)
        self.assertEqual(count, 5)
        deployment = (
            self.root
            / argo.pins.MANIFEST_ROOT
            / 'base/workflow-controller-deployment-patch.yaml'
        ).read_text()
        for image in ('workflow-controller', 'argoexec'):
            self.assertEqual(
                re.findall(rf'quay.io/argoproj/{image}:(v[\d.]+)', deployment),
                [version],
            )

    def test_install_localizes_all_resources_and_is_retry_and_version_safe(
        self,
    ):
        with mock.patch.object(
            argo.subprocess,
            'run',
            side_effect=AssertionError('consumer must be offline'),
        ):
            with mock.patch.object(
                argo,
                'fetch_version',
                side_effect=AssertionError('consumer must not fetch'),
            ):
                for version in [
                    self.pinned[0],
                    self.pinned[0],
                    self.pinned[1],
                    self.pinned[0],
                ]:
                    argo.install(self.output, version, self.root)
                    self.assert_local_resources(version)

    def test_missing_selected_version_does_not_change_manifest_references(self):
        before = {
            p: (self.root / p).read_text() for p, _ in argo.pins.MANIFEST_REFS
        }
        with self.assertRaisesRegex(ValueError, 'missing'):
            argo.install(self.output, 'v0.0.0', self.root)
        self.assertEqual(
            before, {p: (self.root / p).read_text() for p in before}
        )


if __name__ == '__main__':
    unittest.main()

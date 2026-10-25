#!/usr/bin/env python3
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

import io
import tarfile
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from osv_manifest_source import download_manifests, extract_manifests


class ManifestSourceTest(unittest.TestCase):
    def archive(self, directory, entries):
        archive = Path(directory) / 'archive.tar'
        with tarfile.open(archive, 'w') as output:
            for name, kind in entries:
                item = tarfile.TarInfo(name)
                item.type = kind
                item.mode = 0o777
                if kind == tarfile.REGTYPE:
                    item.size = len(b'image: pinned')
                    output.addfile(item, io.BytesIO(b'image: pinned'))
                else:
                    item.linkname = '../../outside'
                    output.addfile(item)
        return archive

    def test_only_manifest_data_is_written_without_executable_permissions(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = self.archive(
                directory,
                [
                    ('repo/manifests/kustomize/base.yaml', tarfile.REGTYPE),
                    ('repo/.github/actions/evil.sh', tarfile.REGTYPE),
                ],
            )
            destination = Path(directory) / 'producer'
            extract_manifests(archive, destination)
            manifest = destination / 'manifests/kustomize/base.yaml'
            self.assertEqual(manifest.read_text(), 'image: pinned')
            self.assertEqual(manifest.stat().st_mode & 0o111, 0)
            self.assertFalse((destination / '.github').exists())

    def test_requested_runtime_source_is_data_and_other_code_is_excluded(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = self.archive(
                directory,
                [
                    ('repo/manifests/kustomize/base.yaml', tarfile.REGTYPE),
                    ('repo/backend/Makefile', tarfile.REGTYPE),
                    ('repo/backend/other.py', tarfile.REGTYPE),
                ],
            )
            destination = Path(directory) / 'producer'
            extract_manifests(
                archive, destination, extra_paths=('backend/Makefile',)
            )
            source = destination / 'backend/Makefile'
            self.assertTrue(source.is_file())
            self.assertEqual(source.stat().st_mode & 0o111, 0)
            self.assertFalse((destination / 'backend/other.py').exists())

    def test_rejects_traversal_links_and_special_files(self):
        for name, kind in [
            ('repo/manifests/../../outside', tarfile.REGTYPE),
            ('/repo/manifests/absolute', tarfile.REGTYPE),
            ('repo/manifests/link', tarfile.SYMTYPE),
            ('repo/manifests/hardlink', tarfile.LNKTYPE),
            ('repo/manifests/fifo', tarfile.FIFOTYPE),
        ]:
            with (
                self.subTest(name=name),
                tempfile.TemporaryDirectory() as directory,
            ):
                archive = self.archive(directory, [(name, kind)])
                with self.assertRaises(ValueError):
                    extract_manifests(archive, Path(directory) / 'producer')
                self.assertFalse((Path(directory) / 'outside').exists())

    def test_missing_manifests_fail(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = self.archive(
                directory, [('repo/script.py', tarfile.REGTYPE)]
            )
            with self.assertRaisesRegex(ValueError, 'no manifests'):
                extract_manifests(archive, Path(directory) / 'producer')

    @mock.patch('osv_manifest_source.extract_manifests')
    @mock.patch('osv_manifest_source.subprocess.run')
    def test_download_uses_exact_sha_and_validates_before_network(
        self, run, extract
    ):
        sha = 'a' * 40
        destination = Path('producer')
        download_manifests('kubeflow/pipelines', sha, destination)
        self.assertEqual(
            run.call_args.args[0],
            ['gh', 'api', f'repos/kubeflow/pipelines/tarball/{sha}'],
        )
        self.assertTrue(run.call_args.kwargs['check'])
        self.assertEqual(extract.call_args.args[1], destination)
        run.reset_mock()
        for repository, ref in [
            ('kubeflow/pipelines', 'master'),
            ('../outside', sha),
        ]:
            with self.assertRaises(ValueError):
                download_manifests(repository, ref, destination)
        run.assert_not_called()


if __name__ == '__main__':
    unittest.main()

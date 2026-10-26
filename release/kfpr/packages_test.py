"""Regression coverage for wheels built without generated package sources."""

import subprocess
import sys
import unittest
import zipfile
from pathlib import Path
from tempfile import TemporaryDirectory

from kfpr import packages
from kfpr.packages import LEGACY_PACKAGES, validate_legacy_wheels


class LegacyWheelTest(unittest.TestCase):
    def write_wheels(self, root, missing=None, metadata_version='2.17.3'):
        for directory, name, modules in LEGACY_PACKAGES:
            normalized = name.replace('-', '_')
            dist = root / directory / 'dist'
            dist.mkdir(parents=True, exist_ok=True)
            with zipfile.ZipFile(
                dist / f'{normalized}-2.17.3-py3-none-any.whl', 'w'
            ) as wheel:
                wheel.writestr(
                    f'{normalized}-2.17.3.dist-info/METADATA',
                    f'Name: {name}\nVersion: {metadata_version}\n',
                )
                for module in modules:
                    if module != missing:
                        wheel.writestr(module, '# generated source\n')

    def test_clean_removes_previous_release_artifacts_and_preserves_other_files(
        self,
    ):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.write_wheels(root)
            preserved = []
            for directory, name, _ in LEGACY_PACKAGES:
                dist = root / directory / 'dist'
                for version in ('2.17.1', '2.17.2'):
                    for spelling in {name, name.replace('-', '_')}:
                        for extension in ('tar.gz', 'zip', 'whl'):
                            (
                                dist / f'{spelling}-{version}.{extension}'
                            ).write_text('old artifact')
                for filename in (
                    'README.txt',
                    f'{name}-notes.zip',
                    f'{name}-extra-2.17.2.tar.gz',
                    'unrelated-1.0.whl',
                ):
                    file = dist / filename
                    file.write_text('retain')
                    preserved.append(file)
                (dist / f'{name}-2.17.0.tar.gz').mkdir()
            command = [sys.executable, packages.__file__, 'clean', str(root)]
            subprocess.run(command, check=True)
            subprocess.run(
                command, check=True
            )  # Repeated preparation is harmless.
            for file in preserved:
                self.assertEqual(file.read_text(), 'retain')
            for directory, name, _ in LEGACY_PACKAGES:
                dist = root / directory / 'dist'
                self.assertTrue((dist / f'{name}-2.17.0.tar.gz').is_dir())
                self.assertEqual(len(list(dist.iterdir())), 5)
            self.write_wheels(root)
            validate_legacy_wheels(root, '2.17.3')

    def test_clean_accepts_fresh_checkout_without_dist_directories(self):
        with TemporaryDirectory() as tmp:
            packages.clean_legacy_distributions(Path(tmp))
            self.assertEqual(list(Path(tmp).iterdir()), [])

    def test_complete_packages_pass(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.write_wheels(root)
            validate_legacy_wheels(root, '2.17.3')

    def test_build_success_without_generated_bindings_is_rejected(self):
        for _, _, modules in LEGACY_PACKAGES:
            with self.subTest(modules=modules), TemporaryDirectory() as tmp:
                root = Path(tmp)
                self.write_wheels(root, missing=modules[0])
                with self.assertRaisesRegex(ValueError, 'regenerate bindings'):
                    validate_legacy_wheels(root, '2.17.3')

    def test_wrong_version_is_rejected(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.write_wheels(root, metadata_version='2.17.0')
            with self.assertRaisesRegex(
                ValueError, 'Incorrect release metadata'
            ):
                validate_legacy_wheels(root, '2.17.3')

    def test_stale_wheels_do_not_satisfy_release(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.write_wheels(root)
            with self.assertRaisesRegex(ValueError, 'Expected one'):
                validate_legacy_wheels(root, '2.17.4')

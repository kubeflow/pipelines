"""Test complete CI scope and artifact assembly before qualification."""

import hashlib
import json
import unittest
from unittest import mock

from . import ci_images_runner_test as image_tests
from . import ci_qualification_runner as runner


class QualificationRunnerTest(unittest.TestCase):
    setUp = image_tests.CiImagesRunnerTest.setUp

    def sdk(self):
        package = self.source / 'sdk/python'
        package.mkdir(parents=True)
        (package / 'setup.py').write_text(
            "classifiers=['Programming Language :: Python :: 3.9', 'Programming Language :: Python :: 3.13']"
        )

    def security(self):
        directory = self.root / 'security'
        directory.mkdir()
        (directory / 'security.json').write_text(json.dumps(self.release))
        (directory / 'raw.json').write_text('{}')
        return directory

    def test_inventory_uses_exact_source_and_selected_scope(self):
        self.sdk()
        inventory = runner.inventory(self.source, self.release, True, True)
        self.assertEqual(len(inventory['matrix']['include']), 2)
        self.assertEqual(inventory['python_versions'], ['3.9', '3.13'])
        self.assertEqual(
            runner.inventory(self.source, self.release, False, True)[
                'python_versions'
            ],
            [],
        )
        (self.source / 'VERSION').write_text('different-backend-version')
        inventory = runner.inventory(self.source, self.release, True, False)
        self.assertEqual(inventory['matrix'], {'include': []})
        self.assertEqual(inventory['python_versions'], ['3.9', '3.13'])
        with self.assertRaisesRegex(ValueError, 'VERSION'):
            runner.inventory(self.source, self.release, False, True)
        with self.assertRaisesRegex(ValueError, 'SHA'):
            runner.inventory(
                self.source,
                {**self.release, 'source_sha': 'f' * 40},
                True,
                False,
            )
        for sdk, backend in ((False, False), ('true', True), (True, 1)):
            with self.assertRaises(ValueError):
                runner.inventory(self.source, self.release, sdk, backend)

    def test_sdk_only_aggregate_requires_security_and_binds_manifest(self):
        self.sdk()
        security = self.security()
        output = self.root / 'qualified'
        with (
            mock.patch.object(runner, 'verify_security') as verify,
            mock.patch.object(runner.images, 'aggregate') as images,
        ):
            result = runner.aggregate(
                self.source,
                self.root / 'missing-lanes',
                security,
                output,
                self.release,
                True,
                False,
            )
            verify.assert_called_once_with(
                security, self.release, ['3.9', '3.13']
            )
            images.assert_not_called()
        self.assertEqual(result['images'], [])
        self.assertEqual(result['include_sdk'], True)
        self.assertEqual(result['include_backend'], False)
        self.assertEqual(
            result['security_manifest_sha256'],
            hashlib.sha256(
                (output / 'security/security.json').read_bytes()
            ).hexdigest(),
        )
        self.assertEqual((output / 'security/raw.json').read_text(), '{}')
        self.assertEqual(
            result['scanner'], runner.images.scanner_identity('osv-json')
        )

    def test_incomplete_security_or_backend_cannot_produce_final_manifest(self):
        security = self.security()
        for sdk, backend in ((True, False), (False, True)):
            if sdk:
                self.sdk()
            output = self.root / str(sdk)
            with mock.patch.object(
                runner,
                'verify_security',
                side_effect=ValueError('missing Python evidence'),
            ):
                with self.assertRaisesRegex(ValueError, 'missing Python'):
                    runner.aggregate(
                        self.source,
                        self.root / 'lanes',
                        security,
                        output,
                        self.release,
                        sdk,
                        backend,
                    )
            self.assertFalse((output / 'manifest.json').exists())
        with mock.patch.object(runner, 'verify_security') as verify:
            with self.assertRaisesRegex(ValueError, 'Incomplete'):
                runner.aggregate(
                    self.source,
                    self.root / 'missing-lanes',
                    security,
                    self.root / 'missing',
                    self.release,
                    False,
                    True,
                )
            verify.assert_called_once_with(security, self.release, [])

    def test_security_symlinks_are_not_copied(self):
        self.sdk()
        security = self.security()
        (security / 'link').symlink_to(security / 'raw.json')
        with mock.patch.object(runner, 'verify_security'):
            with self.assertRaisesRegex(ValueError, 'Symlink'):
                runner.aggregate(
                    self.source,
                    self.root / 'lanes',
                    security,
                    self.root / 'qualified',
                    self.release,
                    True,
                    False,
                )
        self.assertFalse((self.root / 'qualified/manifest.json').exists())

    def test_backend_aggregate_requires_third_party_and_binds_raw_manifest(
        self,
    ):
        security = self.security()
        output = self.root / 'qualified'
        external = self.root / 'third-party'
        external.mkdir()
        (external / 'third-party.json').write_text('{"raw":true}')
        verified = {'inventory': {'first_party_names': ['image-a']}}

        def aggregate(*args, **kwargs):
            output.mkdir()
            return {**self.release, 'images': []}

        with (
            mock.patch.object(runner, 'verify_security'),
            mock.patch.object(
                runner.images, 'aggregate', side_effect=aggregate
            ),
            mock.patch.object(
                runner.external, 'verify', return_value=verified
            ) as verify,
        ):
            result = runner.aggregate(
                self.source,
                self.root / 'lanes',
                security,
                output,
                self.release,
                False,
                True,
                third_party=external,
            )
        verify.assert_called_once_with(external, self.release)
        self.assertEqual(
            result['third_party_manifest_sha256'],
            hashlib.sha256(
                (external / 'third-party.json').read_bytes()
            ).hexdigest(),
        )
        self.assertEqual(
            (output / 'third-party/third-party.json').read_bytes(),
            (external / 'third-party.json').read_bytes(),
        )

    def test_backend_aggregate_rejects_missing_third_party_and_wrong_ownership(
        self,
    ):
        security = self.security()
        for supplied, names, error in (
            (None, ['image-a'], 'Missing third-party'),
            (self.root / 'external', ['other'], 'ownership'),
        ):
            with (
                mock.patch.object(runner, 'verify_security'),
                mock.patch.object(runner.images, 'aggregate', return_value={}),
                mock.patch.object(
                    runner.external,
                    'verify',
                    return_value={'inventory': {'first_party_names': names}},
                ),
                self.assertRaisesRegex(ValueError, error),
            ):
                runner.aggregate(
                    self.source,
                    self.root / 'lanes',
                    security,
                    self.root / 'qualified',
                    self.release,
                    False,
                    True,
                    third_party=supplied,
                )

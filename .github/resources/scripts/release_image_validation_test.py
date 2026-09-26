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
"""Release-line boundaries and native shared-tag qualification without
pulls."""

import copy
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import arm64_smoke
import release_image_validation as release
import yaml

SHA = 'a' * 40
INDEX_DIGEST = 'sha256:' + 'b' * 64
OTHER_DIGEST = 'sha256:' + 'c' * 64
TAG = '3.0.0-rc.1'


class SourceTests(unittest.TestCase):

    def test_final_and_prerelease_use_same_source_version_core(self):
        for tag, version in (('3.0.0', '3.0.0\n'), ('3.0.0-rc.1', '3.0.0'),
                             ('3.0.0-beta-1', '3.0.0-rc.2'), ('3.0.1', '3.0.1'),
                             ('4.1.0', '4.1.0')):
            with self.subTest(tag=tag, version=version):
                release.validate_source(tag, version)

    def test_2_x_requires_legacy_release_branch_workflow(self):
        for tag in ('2.18.0', '2.18.0-rc.1', '2.18.1'):
            with self.subTest(tag=tag):
                with self.assertRaisesRegex(ValueError, 'release-2.x branch'):
                    release.validate_source(tag, '2.18.0')

    def test_cross_version_and_cross_line_sources_are_rejected(self):
        for tag, version in (('3.0.0', '2.18.0'), ('3.0.0', '3.1.0'),
                             ('3.0.1', '3.0.0'), ('4.0.0', '3.0.0')):
            with self.subTest(tag=tag, version=version):
                with self.assertRaisesRegex(ValueError,
                                            'same major.minor.patch'):
                    release.validate_source(tag, version)

    def test_malformed_versions_are_rejected(self):
        for value in ('master', '3.0', 'v3.0.0', '3.0.0-', '3.0.0;echo x'):
            for tag, version in ((value, '3.0.0'), ('3.0.0', value)):
                with self.subTest(tag=tag, version=version):
                    with self.assertRaisesRegex(ValueError, 'Expected MAJOR'):
                        release.validate_source(tag, version)


class NativeImagesTests(unittest.TestCase):

    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.directory = Path(temporary.name)
        self.configs = {
            'amd64': 'sha256:' + '1' * 64,
            'arm64': 'sha256:' + '2' * 64
        }
        self.raw_children = {
            arch:
                json.dumps({
                    'schemaVersion': 2,
                    'config': {
                        'digest': value
                    }
                }).encode() for arch, value in self.configs.items()
        }
        self.child_digests = {
            arch: 'sha256:' + hashlib.sha256(raw).hexdigest()
            for arch, raw in self.raw_children.items()
        }
        self.index = {
            'digest':
                INDEX_DIGEST,
            'schemaVersion':
                2,
            'mediaType':
                'application/vnd.oci.image.index.v1+json',
            'manifests': [{
                'mediaType': 'application/vnd.oci.image.manifest.v1+json',
                'digest': value,
                'platform': {
                    'os': 'linux',
                    'architecture': arch
                }
            } for arch, value in self.child_digests.items()],
        }
        for name in arm64_smoke.IMAGES:
            repository = f'ghcr.io/kubeflow/{name}'
            (self.directory / f'{name}.json').write_text(
                json.dumps({
                    'image': repository,
                    'source_sha': SHA,
                    'reference': f'{repository}@{INDEX_DIGEST}',
                    'platforms': ['linux/amd64', 'linux/arm64'],
                }))
        self.local_overrides = {}
        self.corrupt_child = False

    def validate(self,
                 architecture='amd64',
                 machine=None,
                 system='Linux',
                 tag=TAG,
                 source_sha=SHA,
                 inspected_indexes=None,
                 pull_error=None):
        machine = machine or {
            'amd64': 'x86_64',
            'arm64': 'aarch64'
        }[architecture]

        def output(command, **kwargs):
            if command[:4] == ['docker', 'buildx', 'imagetools', 'inspect']:
                self.assertEqual(command[-1], '--raw')
                self.assertTrue(command[-2].endswith(
                    self.child_digests[architecture]))
                self.assertNotIn('text', kwargs)
                return self.raw_children[architecture] + (
                    b' ' if self.corrupt_child else b'')
            self.assertEqual(command[:3], ['docker', 'image', 'inspect'])
            self.assertTrue(kwargs['text'])
            repository = command[-1].rsplit(':', 1)[0]
            local = {
                'Os': 'linux',
                'Architecture': architecture,
                'Id': self.configs[architecture],
                'RepoDigests': [f'{repository}@{INDEX_DIGEST}']
            }
            local.update(self.local_overrides)
            return json.dumps([local])

        with mock.patch.object(release.platform, 'machine', return_value=machine), \
                mock.patch.object(release.platform, 'system', return_value=system), \
                mock.patch.object(release, 'inspect', return_value=self.index,
                                  side_effect=inspected_indexes), \
                mock.patch.object(release.subprocess, 'run', side_effect=pull_error) as pull, \
                mock.patch.object(release.subprocess, 'check_output', side_effect=output):
            result = release.validate_native_images(self.directory, source_sha,
                                                    tag, architecture)
        return result, pull

    def test_both_native_architectures_pull_shared_tags_without_platform_override(
            self):
        for architecture in ('amd64', 'arm64'):
            with self.subTest(architecture=architecture):
                result, pull = self.validate(architecture)
                self.assertEqual(result['source_sha'], SHA)
                self.assertEqual({image['image'] for image in result['images']},
                                 arm64_smoke.IMAGES)
                self.assertEqual(pull.call_count, len(arm64_smoke.IMAGES))
                for image in result['images']:
                    self.assertEqual(image['platform'], f'linux/{architecture}')
                    self.assertEqual(image['child_digest'],
                                     self.child_digests[architecture])
                    self.assertEqual(image['config_digest'],
                                     self.configs[architecture])
                    pull.assert_any_call(['docker', 'pull', image['tag']],
                                         check=True)
                    self.assertTrue(image['tag'].endswith(f':{TAG}'))
                self.assertNotIn('--platform', str(pull.call_args_list))

    def test_containerd_image_ids_are_supported(self):
        for architecture in ('amd64', 'arm64'):
            for image_id in (INDEX_DIGEST, self.child_digests[architecture]):
                with self.subTest(architecture=architecture, image_id=image_id):
                    self.local_overrides = {'Id': image_id}
                    self.validate(architecture)

    def test_wrong_host_or_operating_system_fails_before_pull(self):
        for kwargs in ({
                'machine': 'aarch64'
        }, {
                'system': 'Darwin'
        }, {
                'machine': 'riscv64'
        }):
            with self.subTest(kwargs=kwargs):
                with self.assertRaisesRegex(ValueError, 'native Linux amd64'):
                    self.validate(**kwargs)

    def test_legacy_or_invalid_tags_cannot_enter_native_validation(self):
        for tag in ('2.18.0', 'master', '3.0'):
            with self.subTest(tag=tag):
                with self.assertRaisesRegex(ValueError, r'3.0\+ release tag'):
                    self.validate(tag=tag)

    def test_wrong_source_records_are_rejected(self):
        with self.assertRaisesRegex(ValueError, 'was not built from'):
            self.validate(source_sha='f' * 40)

    def test_incomplete_image_inventory_is_rejected(self):
        (self.directory / 'kfp-launcher.json').unlink()
        with self.assertRaisesRegex(ValueError, 'Missing images'):
            self.validate()

    def test_tag_must_reference_same_run_verified_index(self):
        self.index['digest'] = OTHER_DIGEST
        with self.assertRaisesRegex(ValueError, 'verified index'):
            self.validate()

    def test_both_platforms_must_exist_in_registry_index(self):
        self.index['manifests'].pop()
        with self.assertRaisesRegex(ValueError, 'Missing index platforms'):
            self.validate()

    def test_raw_child_manifest_hash_is_verified(self):
        self.corrupt_child = True
        with self.assertRaisesRegex(ValueError, 'different child manifest'):
            self.validate()

    def test_wrong_native_platform_or_unrelated_image_id_is_rejected(self):
        for values in ({
                'Os': 'windows'
        }, {
                'Architecture': 'arm64'
        }, {
                'Id': OTHER_DIGEST
        }, {
                'RepoDigests': []
        }, {
                'RepoDigests': [
                    f'ghcr.io/kubeflow/kfp-api-server@{OTHER_DIGEST}'
                ]
        }):
            with self.subTest(values=values):
                self.local_overrides = values
                with self.assertRaisesRegex(ValueError, 'did not select'):
                    self.validate()

    def test_tag_drift_during_pull_is_rejected(self):
        changed = copy.deepcopy(self.index)
        changed['digest'] = OTHER_DIGEST
        with self.assertRaisesRegex(ValueError, 'changed during'):
            self.validate(inspected_indexes=[self.index, changed])

    def test_pull_failure_is_not_reported_as_validation_success(self):
        with self.assertRaises(subprocess.CalledProcessError):
            self.validate(
                pull_error=subprocess.CalledProcessError(1, ['docker']))


class WorkflowTests(unittest.TestCase):

    def setUp(self):
        root = Path(__file__).resolve().parents[3]
        self.jobs = yaml.safe_load(
            (root /
             '.github/workflows/image-builds-release.yml').read_text())['jobs']

    def test_validation_runs_natively_after_publication_and_skips_dry_runs(
            self):
        job = self.jobs['validate-release-images']
        self.assertEqual(
            set(job['needs']), {'resolve-source', 'create-manifests'})
        self.assertEqual(job['if'], '${{ !inputs.dry_run }}')
        self.assertEqual(job['runs-on'], '${{ matrix.runner }}')
        self.assertFalse(job['strategy']['fail-fast'])
        self.assertEqual(job['strategy']['matrix']['include'], [
            {
                'architecture': 'amd64',
                'runner': 'ubuntu-latest'
            },
            {
                'architecture': 'arm64',
                'runner': 'ubuntu-24.04-arm'
            },
        ])
        self.assertEqual(self.jobs['create-manifests']['if'],
                         '${{ !inputs.dry_run }}')
        self.assertEqual(self.jobs['build-images-for-release']['with']['push'],
                         '${{ !inputs.dry_run }}')

    def test_validation_consumes_same_attempt_records_and_resolved_source(self):
        steps = self.jobs['validate-release-images']['steps']
        checkout = next(
            step for step in steps
            if step.get('uses', '').startswith('actions/checkout@'))
        self.assertEqual(checkout['with']['ref'],
                         '${{ needs.resolve-source.outputs.sha }}')
        download = next(
            step for step in steps if step.get('uses') ==
            './.github/actions/download-artifact-with-retry')
        self.assertEqual(download['with']['pattern'],
                         'published-index-*-${{ github.run_attempt }}')
        self.assertNotIn('run-id', download['with'])
        native = next(
            step for step in steps
            if 'release_image_validation.py native' in step.get('run', ''))
        self.assertEqual(native['env']['SOURCE_SHA'],
                         '${{ needs.resolve-source.outputs.sha }}')
        self.assertEqual(native['env']['TARGET_TAG'],
                         '${{ inputs.target_tag }}')
        self.assertIn(f"--records {download['with']['path']}", native['run'])
        smoke = next(
            step for step in steps
            if step.get('uses') == './.github/actions/arm64-smoke')
        self.assertEqual(smoke['if'], "matrix.architecture == 'arm64'")
        self.assertEqual(smoke['with']['image_records'],
                         download['with']['path'])
        self.assertEqual(smoke['with']['source_sha'],
                         '${{ needs.resolve-source.outputs.sha }}')
        self.assertLess(steps.index(native), steps.index(smoke))
        upload = next(
            step for step in steps
            if step.get('uses', '').startswith('actions/upload-artifact@'))
        self.assertEqual(upload['if'], 'always()')
        self.assertIn('${{ matrix.architecture }}-${{ github.run_attempt }}',
                      upload['with']['name'])

    def test_source_validation_precedes_builds_and_propagates_immutable_commit(
            self):
        resolve = self.jobs['resolve-source']
        validation = next(
            step for step in resolve['steps']
            if 'release_image_validation.py source' in step.get('run', ''))
        self.assertEqual(validation['env']['TARGET_TAG'],
                         '${{ inputs.target_tag }}')
        self.assertIn('--version-file release-source/VERSION',
                      validation['run'])
        commit = next(
            step for step in resolve['steps'] if step.get('id') == 'commit')
        self.assertLess(resolve['steps'].index(validation),
                        resolve['steps'].index(commit))
        self.assertIn('git -C release-source rev-parse HEAD', commit['run'])
        build = self.jobs['build-images-for-release']
        self.assertEqual(build['needs'], 'resolve-source')
        self.assertEqual(build['with']['src_branch'],
                         '${{ needs.resolve-source.outputs.sha }}')
        self.assertEqual(self.jobs['create-manifests']['with']['source_sha'],
                         '${{ needs.resolve-source.outputs.sha }}')


if __name__ == '__main__':
    unittest.main()

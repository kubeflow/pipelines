"""Deployment inventory coverage and evidence regressions."""

import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from . import deployed_images as inventory


def write_inventory_fixture(directory, image='example.org/controller:v1'):
    directory = Path(directory)
    directory.mkdir(parents=True, exist_ok=True)
    profiles = [
        'manifests/kustomize/env/platform-agnostic',
        'manifests/kustomize/env/platform-agnostic-multi-user',
    ]
    sources = {
        inventory.SOURCE_PATHS[1]: 'KUBEFLOW_MANIFESTS_SHA="'
        + 'a' * 40
        + '"\n',
        inventory.SOURCE_PATHS[2]: 'CERT_MANAGER_VERSION ?= v1.16.2\n',
        inventory.SOURCE_PATHS[3]: json.dumps({'CACHE_IMAGE': image}),
        inventory.SOURCE_PATHS[4]: 'const defaultTensorflowImage = "'
        + image
        + '"',
        inventory.SOURCE_PATHS[5]: "const DEFAULT_TF_IMAGE = '" + image + "';",
        inventory.CACHE_CONSUMER: 'os.LookupEnv("CACHE_IMAGE")',
    }
    sources[inventory.SOURCE_PATHS[1]] += '\n'.join(
        'kubectl apply -k "https://github.com/kubeflow/community-distribution/'
        + path
        + '?ref=${KUBEFLOW_MANIFESTS_SHA}"'
        for path in inventory.PREREQUISITES.values()
    )
    sources[inventory.SOURCE_PATHS[2]] += (
        'kubectl apply -f https://github.com/cert-manager/'
        'cert-manager/releases/download/$(CERT_MANAGER_VERSION)/cert-manager.yaml\n'
    )
    inputs = {
        'profiles': profiles,
        'sources': sources,
        'kustomizations': {
            profile + '/kustomization.yaml': 'resources: []'
            for profile in profiles
        },
        'prerequisites': inventory._pins(sources),
    }
    rendered = 'spec:\n  containers:\n  - image: ' + image + '\n'
    renders = {profile: rendered for profile in profiles}
    renders.update(
        {'prerequisite/' + name: rendered for name in inputs['prerequisites']}
    )
    (directory / 'inputs.json').write_text(json.dumps(inputs))
    (directory / 'rendered.json').write_text(json.dumps(renders))
    return inventory.discover(directory)


class InventoryTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = Path(self.temp.name)
        write_inventory_fixture(self.path)

    def edit(self, name, fn):
        path = self.path / name
        data = json.loads(path.read_text())
        fn(data)
        path.write_text(json.dumps(data))

    def test_missing_runtime_source_fails_closed(self):
        self.edit(
            'inputs.json',
            lambda data: data['sources'].pop(inventory.SOURCE_PATHS[4]),
        )
        with self.assertRaisesRegex(
            ValueError, 'Missing or unexpected inventory source'
        ):
            inventory.discover(self.path)

    def test_capture_missing_source_fails_before_rendering(self):
        with self.assertRaisesRegex(ValueError, 'Missing inventory source'):
            inventory.capture(
                self.path / 'absent-source',
                '/unused/kustomize',
                self.path / 'output',
            )

    def test_failed_render_preserves_partial_evidence_but_cannot_qualify(self):
        inputs = json.loads((self.path / 'inputs.json').read_text())
        source = self.path / 'source'
        for relative, content in {
            **inputs['sources'],
            **inputs['kustomizations'],
        }.items():
            path = source / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content)
        output = self.path / 'partial'
        rendered = 'kind: Deployment\n'
        with mock.patch.object(
            inventory.subprocess,
            'check_output',
            side_effect=[
                rendered,
                subprocess.CalledProcessError(1, 'kustomize'),
            ],
        ):
            with self.assertRaises(subprocess.CalledProcessError):
                inventory.capture(source, '/unused/kustomize', output)
        self.assertEqual(
            json.loads((output / 'inputs.json').read_text()), inputs
        )
        self.assertEqual(
            json.loads((output / 'rendered.json').read_text()),
            {inputs['profiles'][0]: rendered},
        )
        with self.assertRaisesRegex(ValueError, 'Incomplete'):
            inventory.discover(output)

    def test_capture_records_removed_cache_consumer_as_null(self):
        inputs = json.loads((self.path / 'inputs.json').read_text())
        source = self.path / 'source'
        for relative, content in {
            **inputs['sources'],
            **inputs['kustomizations'],
        }.items():
            if relative == inventory.CACHE_CONSUMER:
                continue
            path = source / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content)
        # The server repository prefix is combined with an explicit user tag;
        # requests without image or tfversion are rejected by the handler.
        config = source / 'frontend/server/configs.ts'
        config.parent.mkdir(parents=True, exist_ok=True)
        config.write_text(
            "VIEWER_TENSORBOARD_TF_IMAGE_NAME = 'tensorflow/tensorflow',"
        )
        output = self.path / 'captured'
        with (
            mock.patch.object(
                inventory.subprocess,
                'check_output',
                return_value='kind: Deployment',
            ),
            mock.patch.object(inventory.urllib.request, 'urlopen') as urlopen,
        ):
            urlopen.return_value.__enter__.return_value.read.return_value = (
                b'kind: Deployment'
            )
            result = inventory.capture(source, '/unused/kustomize', output)
        captured = json.loads((output / 'inputs.json').read_text())
        self.assertIn(inventory.CACHE_CONSUMER, captured['sources'])
        self.assertIsNone(captured['sources'][inventory.CACHE_CONSUMER])
        self.assertFalse(
            any(
                row['target'].startswith('runtime/cache/')
                for row in result['targets']
            )
        )
        self.assertNotIn(
            'tensorflow/tensorflow', [row['image'] for row in result['targets']]
        )

    def test_removed_cache_consumer_excludes_obsolete_config_default(self):
        self.edit(
            'inputs.json',
            lambda data: data['sources'].update(
                {inventory.CACHE_CONSUMER: None}
            ),
        )
        targets = inventory.discover(self.path)['targets']
        self.assertFalse(
            any(row['target'].startswith('runtime/cache/') for row in targets)
        )

    def test_absent_cache_evidence_key_is_not_an_absent_consumer(self):
        self.edit(
            'inputs.json',
            lambda data: data['sources'].pop(inventory.CACHE_CONSUMER),
        )
        with self.assertRaisesRegex(
            ValueError, 'Missing or unexpected inventory source'
        ):
            inventory.discover(self.path)

    def test_deployed_cache_requires_consumer_evidence(self):
        self.edit(
            'inputs.json',
            lambda data: data['sources'].update(
                {inventory.CACHE_CONSUMER: None}
            ),
        )
        self.edit(
            'rendered.json',
            lambda data: data.update(
                {
                    'manifests/kustomize/env/platform-agnostic': 'spec:\n  containers:\n  - image: ghcr.io/kubeflow/kfp-cache-server:v1\n'
                }
            ),
        )
        with self.assertRaisesRegex(
            ValueError, 'Missing deployed cache consumer'
        ):
            inventory.discover(self.path)

    def test_changed_cache_consumer_fails_closed(self):
        self.edit(
            'inputs.json',
            lambda data: data['sources'].update({inventory.CACHE_CONSUMER: ''}),
        )
        with self.assertRaisesRegex(ValueError, 'Cache image consumer changed'):
            inventory.discover(self.path)

    def test_missing_cache_default_fails_closed(self):
        self.edit(
            'inputs.json',
            lambda data: data['sources'].update(
                {inventory.SOURCE_PATHS[3]: '{}'}
            ),
        )
        with self.assertRaisesRegex(ValueError, 'Missing cache image default'):
            inventory.discover(self.path)

    def test_required_prerequisite_cannot_be_omitted(self):
        self.edit('rendered.json', lambda data: data.pop('prerequisite/istio'))
        with self.assertRaisesRegex(ValueError, 'Incomplete'):
            inventory.discover(self.path)

    def test_source_profile_cannot_be_omitted(self):
        self.edit(
            'inputs.json',
            lambda data: data['kustomizations'].update(
                {
                    'manifests/kustomize/env/aws/kustomization.yaml': 'resources: []'
                }
            ),
        )
        with self.assertRaisesRegex(ValueError, 'catalog differs'):
            inventory.discover(self.path)

    def test_runtime_defaults_are_not_all_menu_images(self):
        self.edit(
            'inputs.json',
            lambda data: data['sources'].update(
                {
                    inventory.SOURCE_PATHS[
                        3
                    ]: '{"CACHE_IMAGE":"registry/cache:v1"}',
                    inventory.SOURCE_PATHS[
                        4
                    ]: 'const defaultTensorflowImage = "tensorflow/tensorflow:1.13.2"',
                    inventory.SOURCE_PATHS[
                        5
                    ]: "const DEFAULT_TF_IMAGE = 'tensorflow/tensorflow:2.2.2';\n<MenuItem value={'tensorflow/tensorflow:1.7.1'} />",
                }
            ),
        )
        targets = inventory.discover(self.path)['targets']
        self.assertEqual(
            len([row for row in targets if row['scope'] == 'runtime']), 3
        )
        self.assertNotIn(
            'tensorflow/tensorflow:1.7.1', [row['image'] for row in targets]
        )
        self.assertEqual(len({row['target'] for row in targets}), len(targets))

    def test_legacy_repository_does_not_replace_standard(self):
        profile = inventory.LEGACY + 'db'
        self.edit(
            'inputs.json',
            lambda data: (
                data['profiles'].insert(0, profile),
                data['kustomizations'].update(
                    {profile + '/kustomization.yaml': 'resources: []'}
                ),
            ),
        )
        self.edit(
            'rendered.json',
            lambda data: data.update(
                {
                    profile: 'spec:\n  containers:\n  - image: example.org/controller:old\n'
                }
            ),
        )
        targets = inventory.discover(self.path)['targets']
        self.assertIn(
            'legacy/example.org/controller', [row['target'] for row in targets]
        )
        self.assertIn(
            'example.org/controller', [row['target'] for row in targets]
        )

    def test_runtime_category_survives_version_updates(self):
        source = inventory.SOURCE_PATHS[5]
        self.edit(
            'inputs.json',
            lambda data: data['sources'].update(
                {
                    source: "const DEFAULT_TF_IMAGE = 'tensorflow/tensorflow:2.2.2';"
                }
            ),
        )
        before = [
            row['target'] for row in inventory.discover(self.path)['targets']
        ]
        self.edit(
            'inputs.json',
            lambda data: data['sources'].update(
                {
                    source: "const DEFAULT_TF_IMAGE = 'tensorflow/tensorflow:2.20.0';"
                }
            ),
        )
        self.assertEqual(
            before,
            [row['target'] for row in inventory.discover(self.path)['targets']],
        )

    def test_empty_profiles_are_valid_but_missing_renders_are_not(self):
        profile = 'manifests/kustomize/env/cert-manager/base'
        self.edit(
            'inputs.json',
            lambda data: (
                data['profiles'].insert(0, profile),
                data['kustomizations'].update(
                    {profile + '/kustomization.yaml': 'resources: []'}
                ),
            ),
        )
        self.edit(
            'rendered.json',
            lambda data: data.update({profile: 'kind: Certificate'}),
        )
        self.assertEqual(inventory.discover(self.path)['profiles'][profile], [])

    def test_changed_prerequisite_installer_fails_closed(self):
        self.edit(
            'inputs.json',
            lambda data: data['sources'].update(
                {
                    inventory.SOURCE_PATHS[1]: data['sources'][
                        inventory.SOURCE_PATHS[1]
                    ].replace(
                        'community-distribution', 'different-distribution'
                    )
                }
            ),
        )
        with self.assertRaisesRegex(ValueError, 'installer location changed'):
            inventory.discover(self.path)

    def test_checksum_covers_inputs(self):
        before = inventory.discover(self.path)['input_sha256']
        self.edit(
            'inputs.json',
            lambda data: data['kustomizations'].update(
                {next(iter(data['kustomizations'])): 'resources: []\n# changed'}
            ),
        )
        self.assertNotEqual(
            before, inventory.discover(self.path)['input_sha256']
        )

    def test_symlink_rejected(self):
        (self.path / 'rendered.json').unlink()
        (self.path / 'rendered.json').symlink_to(self.path / 'inputs.json')
        with self.assertRaisesRegex(ValueError, 'regular files'):
            inventory.discover(self.path)

    def test_executors_cache_and_conditional_acme(self):
        images = inventory.extract_images("""
kind: ConfigMap
data:
  cacheImage: example.org/cache:v1
---
spec:
  containers:
  - image: example.org/controller:v1
    args:
    - --executor-image=example.org/executor:v1
    - --acme-http01-solver-image
    - example.org/solver:v1
""")
        self.assertEqual(
            images,
            {
                'example.org/cache:v1',
                'example.org/controller:v1',
                'example.org/executor:v1',
                'example.org/solver:v1',
            },
        )


if __name__ == '__main__':
    unittest.main()

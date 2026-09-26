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
"""Keep runtime E2E fixture images inside the shared Kind preload inventory."""

import ast
from pathlib import Path
import re
import unittest
from unittest import mock

import yaml

ROOT = Path(__file__).resolve().parents[3]
FIXTURES = ROOT / 'test_data/sdk_compiled_pipelines/valid'
# These are the directories selected by backend/test/end2end/pipeline_e2e_test.go.
# The API upload suite also reads the valid/ root, but that contains compiler-only
# fixtures with deliberately nonexistent images and is not a runtime inventory.
RUNTIME_DIRECTORIES = (
    'essential',
    'critical',
    'parallel_and_nested',
    'integration',
    'failing',
    'gpu-scheduling',
    'dra',
)
IMAGE_EXCEPTIONS = {
    'argostub/createpvc':
        'Driver-terminal Kubernetes executor, not an image pull.',
    'argostub/deletepvc':
        'Driver-terminal Kubernetes executor, not an image pull.',
    'kfp-dra-sdk:test':
        'Built and loaded into Kind by the DRA E2E job.',
}
# Ray creates its own pods outside the KFP executor archive contract. Keep its
# compatibility-pinned workload external, rather than retagging away its digest.
EXTERNAL_WORKLOAD_IMAGES = {
    ('integration/ray_integration.py',
     'quay.io/modh/ray@sha256:6d076aeb38ab3c34a6a2ef0f58dc667089aa15826fa08a73273c629333e12f1e'
    ):
        'CodeFlare integration workload; Ray pods pull this exact digest directly.',
}


def canonical_tag(image):
    """Normalize registry aliases without erasing runtime image identity."""
    if '@' in image:
        raise ValueError(
            'Digest references are only supported in the producer inventory')
    parts = image.split('/')
    if len(parts) == 1 or not ('.' in parts[0] or ':' in parts[0] or
                               parts[0] == 'localhost'):
        parts.insert(0, 'docker.io')
    if parts[0] == 'index.docker.io':
        parts[0] = 'docker.io'
    if parts[0] == 'docker.io' and len(parts) == 2:
        parts.insert(1, 'library')
    if ':' not in parts[-1]:
        parts[-1] += ':latest'
    return '/'.join(parts)


def inventory_tags(inventory):
    """Validate acquisition pins and return tags retained by docker
    save/load."""
    tags = set()
    for image in inventory:
        image = image.strip()
        if not image or image.startswith('#'):
            continue
        tag, separator, digest = image.partition('@')
        if separator and (not re.fullmatch(r'sha256:[0-9a-f]{64}', digest) or
                          not re.fullmatch(r'[^:\s]+:[\w][\w.-]{0,127}',
                                           tag.rsplit('/', 1)[-1])):
            raise ValueError(
                f'{image}: expected tag@sha256:<64 lowercase hex digits>')
        canonical = canonical_tag(tag)
        if canonical in tags:
            raise ValueError(
                f'{image}: duplicate canonical inventory tag {canonical}')
        tags.add(canonical)
    return tags


def fixture_paths(suffix):
    paths = []
    for directory in RUNTIME_DIRECTORIES:
        paths.extend((FIXTURES / directory).rglob('*' + suffix))
    paths.append(FIXTURES /
                 ('env_var.py' if suffix == '.py' else 'env-var.yaml'))
    return sorted(paths)


def yaml_images(path):
    """Return only executor image references, never image-like component
    text."""
    documents = list(yaml.safe_load_all(path.read_text()))
    policies = {}
    for document in documents:
        policies.update(
            document.get('platforms', {}).get('kubernetes',
                                              {}).get('deploymentSpec',
                                                      {}).get('executors', {}))
    images = []
    for document in documents:
        for executor, spec in document.get('deploymentSpec',
                                           {}).get('executors', {}).items():
            if 'container' in spec:
                images.append((spec['container']['image'],
                               policies.get(executor,
                                            {}).get('imagePullPolicy')))
    return images


def python_images(path):
    """Check source literals too, so regeneration cannot undo a preload fix."""
    tree = ast.parse(path.read_text())
    imports = {}
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            for alias in node.names:
                imports[alias.asname or alias.name.split('.')[0]] = (
                    alias.name if alias.asname else alias.name.split('.')[0])
        elif isinstance(node, ast.ImportFrom) and node.module:
            for alias in node.names:
                imports[alias.asname or
                        alias.name] = f'{node.module}.{alias.name}'

    def qualified_name(node):
        if isinstance(node, ast.Name):
            return imports.get(node.id, node.id)
        if isinstance(node, ast.Attribute):
            return f'{qualified_name(node.value)}.{node.attr}'
        return ''

    constants = {
        target.id: node.value for node in tree.body
        if isinstance(node, ast.Assign) for target in node.targets
        if isinstance(target, ast.Name)
    }
    images = []
    for node in ast.walk(tree):
        if not isinstance(node, ast.Call):
            continue
        if qualified_name(node.func) not in (
                'kfp.dsl.component', 'kfp.dsl.ContainerSpec',
                'codeflare_sdk.ray.cluster.ClusterConfiguration'):
            continue
        for keyword in node.keywords:
            if keyword.arg not in ('image', 'base_image'):
                continue
            value = constants.get(keyword.value.id) if isinstance(
                keyword.value, ast.Name) else keyword.value
            if not isinstance(value, ast.Constant) or not isinstance(
                    value.value, str):
                raise ValueError(
                    f'{path}: runtime image must be an inventoried literal')
            images.append(value.value)
    return images


def source_coverage_errors(path, images, inventory):
    fixture = path.relative_to(FIXTURES).as_posix()
    checked = [(image, None)
               for image in images
               if (fixture, image) not in EXTERNAL_WORKLOAD_IMAGES]
    return coverage_errors(checked, inventory, check_policy=False)


def coverage_errors(images, inventory, *, check_policy=True):
    cached = inventory_tags(inventory)
    errors = []
    for image, policy in images:
        if image in IMAGE_EXCEPTIONS:
            continue
        if '@' in image:
            errors.append(
                f'{image}: use the retained tag; docker archives do not preserve RepoDigests'
            )
            continue
        if canonical_tag(image) not in cached:
            errors.append(
                f'{image}: add its exact tag to runtime-base-images.txt')
        # A :latest/untagged image still contacts its registry even when present.
        if check_policy and (policy == 'Always' or
                             (canonical_tag(image).endswith(':latest') and
                              policy != 'IfNotPresent')):
            errors.append(
                f'{image}: use a versioned tag or explicit IfNotPresent')
    return errors


class RuntimeFixtureImagesTest(unittest.TestCase):

    def test_python_image_calls_include_imports_and_aliases(self):
        sources = [
            "from kfp.dsl import ContainerSpec\nContainerSpec(image='missing:v1')",
            "from kfp.dsl import ContainerSpec as Spec\nSpec(image='missing:v1')",
            "from kfp.dsl import component as comp\ncomp(base_image='missing:v1')",
            "from kfp import dsl as pipeline\npipeline.ContainerSpec(image='missing:v1')",
            "import kfp.dsl as pipeline\npipeline.component(base_image='missing:v1')",
            "import kfp\nkfp.dsl.ContainerSpec(image='missing:v1')",
            "from codeflare_sdk.ray.cluster import ClusterConfiguration as Config\nConfig(image='missing:v1')",
            "import codeflare_sdk.ray.cluster as cluster\ncluster.ClusterConfiguration(image='missing:v1')",
        ]
        for source in sources:
            with self.subTest(source=source):
                with mock.patch.object(Path, 'read_text', return_value=source):
                    images = python_images(Path('fixture.py'))
                self.assertEqual(images, ['missing:v1'])
                self.assertTrue(coverage_errors([(images[0], None)], []))

    def test_ray_workload_is_detected_separately_from_executor(self):
        images = python_images(FIXTURES / 'integration/ray_integration.py')
        self.assertEqual(len(images), 2)
        self.assertTrue(
            any(
                image.startswith('quay.io/modh/ray@sha256:')
                for image in images))

    def test_external_workload_exception_is_fixture_and_digest_specific(self):
        for (fixture, image), reason in EXTERNAL_WORKLOAD_IMAGES.items():
            self.assertTrue(reason)
            path = FIXTURES / fixture
            self.assertIn(image, python_images(path))
            self.assertEqual(source_coverage_errors(path, [image], []), [])
            self.assertTrue(
                source_coverage_errors(FIXTURES / 'other.py', [image], []))
            changed = image.split('@')[0] + '@sha256:' + '0' * 64
            self.assertTrue(source_coverage_errors(path, [changed], []))
            self.assertTrue(
                source_coverage_errors(path, [image.split('@')[0] + ':latest'],
                                       []))
            # The committed lightweight component contains the same workload.
            compiled = list(
                yaml.safe_load_all(
                    (FIXTURES / 'integration/ray_integration_compiled.yaml'
                    ).read_text()))[0]
            command = compiled['deploymentSpec']['executors']['exec-ray-fn'][
                'container']['command']
            self.assertTrue(any(image in argument for argument in command))

    def test_unrelated_calls_are_not_treated_as_kfp_images(self):
        source = "from another_library import ContainerSpec\nContainerSpec(image='not-a-runtime-image')"
        with mock.patch.object(Path, 'read_text', return_value=source):
            self.assertEqual(python_images(Path('fixture.py')), [])

    def test_inventory_directories_match_runtime_suite_selection(self):
        suite = (ROOT / 'backend/test/end2end/pipeline_e2e_test.go').read_text()
        selected = set(re.findall(r'var pipelineDir = "valid/([^"]+)"', suite))
        self.assertEqual(selected, set(RUNTIME_DIRECTORIES))

    def test_runtime_yaml_images_are_preloaded_without_registry_resolution(
            self):
        inventory = (ROOT / '.github/resources/runtime-base-images.txt'
                    ).read_text().splitlines()
        for path in fixture_paths('.yaml'):
            with self.subTest(fixture=str(path.relative_to(ROOT))):
                self.assertEqual(
                    coverage_errors(yaml_images(path), inventory), [])

    def test_runtime_source_images_are_preloaded(self):
        inventory = (ROOT / '.github/resources/runtime-base-images.txt'
                    ).read_text().splitlines()
        for path in fixture_paths('.py'):
            with self.subTest(fixture=str(path.relative_to(ROOT))):
                images = python_images(path)
                self.assertEqual(
                    source_coverage_errors(path, images, inventory), [])

    def test_proxy_fixture_copy_uses_the_same_preloaded_images(self):
        self.assertEqual(
            yaml_images(ROOT / 'backend/test/v2/resources/env-var.yaml'),
            yaml_images(FIXTURES / 'env-var.yaml'))

    def test_missing_image_is_rejected(self):
        errors = coverage_errors([('python:3.12', None)], ['python:3.11'])
        self.assertEqual(len(errors), 1)
        self.assertIn('add its exact tag', errors[0])

    def test_registry_aliases_match_but_versions_do_not(self):
        self.assertEqual({canonical_tag('python:3.12')},
                         inventory_tags([
                             'docker.io/library/python:3.12@sha256:' + 'a' * 64
                         ]))
        self.assertEqual(
            canonical_tag('docker.io/alpine:3.23'),
            canonical_tag('alpine:3.23'))
        self.assertNotEqual(
            canonical_tag('python:3.11'), canonical_tag('python:3.12'))
        self.assertNotEqual(
            canonical_tag('public.ecr.aws/docker/library/python:3.12'),
            canonical_tag('python:3.12'))

    def test_runtime_digest_references_are_rejected_even_when_tag_is_cached(
            self):
        pin = 'python:3.12@sha256:' + 'a' * 64
        self.assertIn('use the retained tag',
                      coverage_errors([(pin, None)], [pin])[0])
        self.assertIn(
            'use the retained tag',
            coverage_errors([('python@sha256:' + 'a' * 64, 'IfNotPresent')],
                            ['python:latest'])[0])
        with self.assertRaisesRegex(ValueError,
                                    'only supported in the producer'):
            canonical_tag(pin)

    def test_inventory_pins_require_explicit_tags_and_full_sha256_digests(self):
        invalid = [
            'python@sha256:' + 'a' * 64,
            'python:@sha256:' + 'a' * 64,
            'localhost:5000/python@sha256:' + 'a' * 64,
            'python:3.12@sha256:abc',
            'python:3.12@sha256:' + 'a' * 63,
            'python:3.12@sha256:' + 'a' * 65,
            'python:3.12@sha256:' + 'z' * 64,
            'python:3.12@sha512:' + 'a' * 64,
            'python:3.12@sha256:' + 'a' * 64 + '@extra',
        ]
        for image in invalid:
            with self.subTest(image=image):
                with self.assertRaisesRegex(ValueError, 'expected tag@sha256'):
                    inventory_tags([image])

    def test_inventory_rejects_duplicate_aliases_and_differing_digest_pins(
            self):
        for entries in (['python:3.12', 'docker.io/library/python:3.12'], [
                'alpine:3.23', 'docker.io/alpine:3.23'
        ], ['python:3.12@sha256:' + 'a' * 64,
                'python:3.12@sha256:' + 'b' * 64]):
            with self.subTest(entries=entries):
                with self.assertRaisesRegex(ValueError, 'duplicate canonical'):
                    inventory_tags(entries)

    def test_latest_images_require_explicit_if_not_present(self):
        for image in ('example/test', 'example/test:latest'):
            self.assertTrue(coverage_errors([(image, None)], [image]))
            self.assertEqual(
                coverage_errors([(image, 'IfNotPresent')], [image]), [])
        self.assertTrue(
            coverage_errors([('python:3.12', 'Always')], ['python:3.12']))

    def test_only_exact_driver_and_local_build_exceptions_are_allowed(self):
        for image in IMAGE_EXCEPTIONS:
            self.assertEqual(coverage_errors([(image, None)], []), [])
        self.assertTrue(coverage_errors([('argostub/new-image', None)], []))
        self.assertTrue(coverage_errors([('kfp-dra-sdk:other', None)], []))


if __name__ == '__main__':
    unittest.main()

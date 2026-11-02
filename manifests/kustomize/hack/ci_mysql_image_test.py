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
"""Check that CI MySQL references resolve from the shared image archive."""

import unittest

from tensorboard_signing_key_test import REPO_ROOT, render, resource

CI_MYSQL_OVERLAYS = (
    'standalone/default',
    'standalone/cache-disabled',
    'standalone/proxy',
    'standalone/cache-disabled-proxy',
    'standalone/tls-enabled',
    'kubernetes-native/default',
    'kubernetes-native/cache-disabled',
    'multiuser/default',
    'multiuser/cache-disabled',
    'multiuser/artifact-proxy',
)
CI_POSTGRESQL_OVERLAYS = ('standalone/postgresql', 'multiuser/postgresql')
PRODUCTION_OVERLAYS = ('platform-agnostic', 'platform-agnostic-multi-user')


def mysql_image_reference(image):
    return image.removeprefix('docker.io/').removeprefix('library/')


def mysql_container(resources):
    deployment = resource(resources, 'Deployment', 'mysql')
    return next(
        container
        for container in deployment['spec']['template']['spec']['containers']
        if container['name'] == 'mysql'
    )


class CiMysqlImageTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.ci_resources = {
            overlay: render(
                REPO_ROOT / '.github/resources/manifests' / overlay,
                allow_external_patches=True,
            )
            for overlay in CI_MYSQL_OVERLAYS + CI_POSTGRESQL_OVERLAYS
        }
        cls.production_resources = {
            overlay: render(REPO_ROOT / 'manifests/kustomize/env' / overlay)
            for overlay in PRODUCTION_OVERLAYS
        }
        cls.base_image = mysql_container(
            render(REPO_ROOT / 'manifests/kustomize/third-party/mysql/base')
        )['image']
        inventory = (
            REPO_ROOT / '.github/resources/runtime-base-images.txt'
        ).read_text()
        cls.mysql_acquisition_images = [
            mysql_image_reference(line.strip())
            for line in inventory.splitlines()
            if mysql_image_reference(line.strip()).startswith('mysql:')
        ]

    def test_acquisition_pin_matches_production_base(self):
        image = mysql_image_reference(self.base_image)
        self.assertRegex(image, r'^mysql:[^@\s]+@sha256:[0-9a-f]{64}$')
        self.assertEqual(self.mysql_acquisition_images, [image])

    def test_ci_uses_the_archived_tag_without_registry_resolution(self):
        self.assertEqual(len(self.mysql_acquisition_images), 1)
        archived_tag = self.mysql_acquisition_images[0].split('@', 1)[0]
        for overlay in CI_MYSQL_OVERLAYS:
            with self.subTest(overlay=overlay):
                container = mysql_container(self.ci_resources[overlay])
                self.assertEqual(
                    mysql_image_reference(container['image']), archived_tag
                )
                # A versioned tag defaults to IfNotPresent if omitted.
                self.assertEqual(
                    container.get('imagePullPolicy', 'IfNotPresent'),
                    'IfNotPresent',
                )

    def test_postgresql_overlays_do_not_deploy_mysql(self):
        for overlay in CI_POSTGRESQL_OVERLAYS:
            with self.subTest(overlay=overlay):
                mysql_deployments = [
                    item
                    for item in self.ci_resources[overlay]
                    if item['kind'] == 'Deployment'
                    and item['metadata']['name'] == 'mysql'
                ]
                self.assertEqual(mysql_deployments, [])

    def test_production_overlays_retain_the_digest_pin(self):
        for overlay in PRODUCTION_OVERLAYS:
            with self.subTest(overlay=overlay):
                container = mysql_container(self.production_resources[overlay])
                self.assertEqual(container['image'], self.base_image)


if __name__ == '__main__':
    unittest.main()

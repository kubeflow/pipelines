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

import unittest
from unittest import mock

from osv_manifest_images import create_matrix
from osv_manifest_images import extract_images
from osv_manifest_images import render_overlay


class OsvManifestImagesTest(unittest.TestCase):

    def test_extract_images_deduplicates_rendered_container_images(self):
        manifest = '''
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      initContainers:
        - image: "docker.io/alpine:3.23"
      containers:
        - image: ghcr.io/kubeflow/kfp-api-server:2.17.0
        - image: ghcr.io/kubeflow/kfp-api-server:2.17.0
      imagePullSecrets:
        - name: registry-credentials
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  versions:
    - schema:
        openAPIV3Schema:
          properties:
            image: properties:
'''
        self.assertEqual(
            extract_images(manifest), {
                'docker.io/alpine:3.23',
                'ghcr.io/kubeflow/kfp-api-server:2.17.0',
            })

    def test_create_matrix_is_sorted_and_has_stable_categories(self):
        matrix = create_matrix({'registry.example/b:2', 'registry.example/a:1'})

        self.assertEqual(
            matrix, {
                'include': [
                    {
                        'image': 'registry.example/a:1',
                        'category': '2bd75a0be1a9bf75',
                    },
                    {
                        'image': 'registry.example/b:2',
                        'category': '3080c2e5bf31546b',
                    },
                ]
            })

    @mock.patch('osv_manifest_images.subprocess.run')
    def test_render_overlay_fails_closed_and_returns_rendered_resources(
            self, run_mock):
        run_mock.return_value.stdout = 'kind: Deployment\n'

        self.assertEqual(
            render_overlay('/tools/kustomize', 'manifests/overlay'),
            'kind: Deployment\n',
        )
        run_mock.assert_called_once_with(
            [
                '/tools/kustomize',
                'build',
                '--load-restrictor',
                'LoadRestrictionsNone',
                'manifests/overlay',
            ],
            check=True,
            capture_output=True,
            text=True,
        )


if __name__ == '__main__':
    unittest.main()

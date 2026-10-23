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
import json
import unittest
from pathlib import Path
from unittest import mock

import yaml
from osv_manifest_images import (
    create_image_map,
    create_matrix,
    extract_images,
    main,
    render_overlay,
)


class OsvManifestImagesTest(unittest.TestCase):
    def test_extract_images_deduplicates_rendered_container_images(self):
        manifest = """
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
            image:
              type: string
"""
        self.assertEqual(
            extract_images(manifest),
            {
                'docker.io/alpine:3.23',
                'ghcr.io/kubeflow/kfp-api-server:2.17.0',
            },
        )

    def test_embedded_job_template_is_not_a_deployed_container(self):
        template_path = (
            Path(__file__).resolve().parents[3]
            / 'manifests/kustomize/base/pipeline/ui-signing-key/job.yaml'
        )
        template = template_path.read_text()
        configmap = {
            'apiVersion': 'v1',
            'kind': 'ConfigMap',
            'data': {'job-template': template},
        }
        job = yaml.safe_load(template)
        image = 'ghcr.io/kubeflow/kfp-frontend:2.17.1'
        job['spec']['template']['spec']['containers'][0]['image'] = image
        for style in ('|', '>', '"'):
            with self.subTest(style=style):
                manifest = (
                    yaml.safe_dump(configmap, default_style=style)
                    + '---\n'
                    + yaml.safe_dump(job)
                )
                self.assertEqual(extract_images(manifest), {image})

    def test_inline_containers_and_nested_image_metadata(self):
        manifest = """
apiVersion: batch/v1
kind: CronJob
spec:
  jobTemplate:
    spec:
      template:
        spec:
          initContainers: [{name: init, image: 'registry.example/init:1'}]
          containers:
          - name: task
            image: registry.example/task:1
            env:
            - name: EXAMPLE
              value: |
                containers:
                - image: registry.example/not-a-container:1
"""
        self.assertEqual(
            extract_images(manifest),
            {
                'registry.example/init:1',
                'registry.example/task:1',
            },
        )

    def test_real_placeholder_images_are_not_silently_filtered(self):
        manifest = """
kind: Pod
spec:
  containers:
  - image: ghcr.io/kubeflow/kfp-frontend:dummy
"""
        self.assertEqual(
            extract_images(manifest),
            {
                'ghcr.io/kubeflow/kfp-frontend:dummy',
            },
        )

    def test_invalid_yaml_fails_discovery(self):
        with self.assertRaises(yaml.YAMLError):
            extract_images('spec: [unterminated')

    def test_empty_documents_do_not_hide_later_images(self):
        manifest = """
---
# Empty document
---
kind: Pod
spec:
  containers:
  - image: registry.example/task:1
---
"""
        self.assertEqual(extract_images(manifest), {'registry.example/task:1'})

    def test_matrix_identity_survives_image_upgrades(self):
        old = create_matrix({'registry.example/task:1'})
        for image in (
            'registry.example/task',
            'registry.example/task:2',
            'registry.example/task@sha256:abc',
            'registry.example/task:2@sha256:def',
        ):
            with self.subTest(image=image):
                self.assertEqual(old, create_matrix({image}))
        self.assertEqual(set(old['include'][0]), {'category'})

    def test_matrix_preserves_repository_and_registry_port_identity(self):
        images = {
            'registry.example:5000/task:1',
            'registry.example:5001/task:1',
            'registry.example:5000/other:1',
        }
        matrix = create_matrix(images)
        self.assertEqual(len(matrix['include']), 3)
        self.assertEqual(len({row['category'] for row in matrix['include']}), 3)
        self.assertEqual(
            create_matrix({'registry.example:5000/task:1'}),
            create_matrix({'registry.example:5000/task:2'}),
        )

    def test_multiple_versions_of_one_repository_fail_closed(self):
        for other_image in (
            'registry.example/task:2',
            'registry.example/task@sha256:abc',
            'registry.example/task:1@sha256:def',
        ):
            with self.subTest(other_image=other_image):
                with self.assertRaisesRegex(
                    ValueError, 'Multiple image references'
                ):
                    create_matrix({'registry.example/task:1', other_image})

    def test_image_map_preserves_exact_references_and_sorted_order(self):
        images = {
            'registry.example/b@sha256:abc',
            'registry.example/a:1@sha256:def',
        }
        image_map = create_image_map(images)
        self.assertEqual(list(image_map.values()), sorted(images))
        self.assertEqual(
            create_matrix(images)['include'],
            [{'category': category} for category in image_map],
        )
        self.assertEqual(len(create_matrix({'alpine:1'})['include']), 1)
        self.assertEqual(
            create_matrix({'alpine:1'}), create_matrix({'alpine:2'})
        )

    @mock.patch('osv_manifest_images.render_overlay')
    @mock.patch('osv_manifest_images.parse_args')
    def test_main_emits_matrix_and_matching_image_map(
        self, args_mock, render_mock
    ):
        args_mock.return_value = mock.Mock(
            kustomize='kustomize', overlay=['a', 'b']
        )
        render_mock.return_value = 'kind: Pod\nspec:\n  containers:\n  - image: registry.example/task:1\n'
        with mock.patch('sys.stdout', new_callable=io.StringIO) as output:
            main()
        inventory = json.loads(output.getvalue())
        category = inventory['matrix']['include'][0]['category']
        self.assertEqual(
            inventory['images'], {category: 'registry.example/task:1'}
        )
        self.assertEqual(
            inventory['matrix'], {'include': [{'category': category}]}
        )

    @mock.patch('osv_manifest_images.subprocess.run')
    def test_render_overlay_fails_closed_and_returns_rendered_resources(
        self, run_mock
    ):
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

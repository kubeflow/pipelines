# Copyright 2022 The Kubeflow Authors
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
"""Tests for KFP Registry Client."""

import json
import os
import shutil
import tempfile
import unittest

from absl.testing import parameterized
from kfp import components
from kfp import compiler
from kfp import dsl
from kfp.components.types import type_utils
from kfp.dsl import PipelineTaskFinalStatus
from kfp.registry import Client



class ClientTest(parameterized.TestCase):
    @parameterized.parameters(
        {
            'host': 'https://us-central1-kfp.pkg.dev/proj/repo',
            'expected': True,
        },
        {
            'host': 'https://hub.docker.com/r/google/cloud-sdk',
            'expected': False,
        },
    )
    def test_is_ar_host(self, host, expected):
        client = Client(host=host)
        self.assertEqual(client._is_ar_host(), expected)

    def test_load_config(self):
        host = 'https://us-central1-kfp.pkg.dev/proj/repo'
        client = Client(host)
        expected_config = {
            'host': host,
            'upload_url': host,
            'download_version_url': f'{host}/{{package_name}}/sha256:{{version}}',
            'download_tag_url': f'{host}/{{package_name}}/{{tag}}',
            'get_package_url': ('https://artifactregistry.googleapis.com/v1/projects/'
                                'proj/locations/us-central1/repositories'
                                '/repo/packages/{package_name}'),
            'list_packages_url': ('https://artifactregistry.googleapis.com/v1/projects/'
                                'proj/locations/us-central1/repositories'
                                '/repo/packages'),
            'delete_package_url': ('https://artifactregistry.googleapis.com/v1/projects/'
                                   'proj/locations/us-central1/repositories'
                                   '/repo/packages/{package_name}'),
            'get_tag_url': ('https://artifactregistry.googleapis.com/v1/projects/'
                            'proj/locations/us-central1/repositories'
                            '/repo/tags/{tag}'),
            'list_tags_url': ('https://artifactregistry.googleapis.com/v1/projects/'
                              'proj/locations/us-central1/repositories'
                              '/repo/tags'),
            'delete_tag_url': ('https://artifactregistry.googleapis.com/v1/projects/'
                               'proj/locations/us-central1/repositories'
                               '/repo/tags/{tag}'),
            'create_tag_url': ('https://artifactregistry.googleapis.com/v1/projects/'
                               'proj/locations/us-central1/repositories'
                               '/repo/tags?tagId={tag}'),
            'update_tag_url': ('https://artifactregistry.googleapis.com/v1/projects/'
                               'proj/locations/us-central1/repositories'
                               '/repo/tags/{tag}?updateMask=version'),
            'get_version_url': ('https://artifactregistry.googleapis.com/v1/projects/'
                               'proj/locations/us-central1/repositories'
                               '/repo/versions/{version}'),
            'list_versions_url': ('https://artifactregistry.googleapis.com/v1/projects/'
                                  'proj/locations/us-central1/repositories'
                                  '/repo/versions'),
            'delete_version_url': ('https://artifactregistry.googleapis.com/v1/projects/'
                                   'proj/locations/us-central1/repositories'
                                   '/repo/versions/{version}'),
            'package_format': ('projects/proj/locations/us-central1/repositories'
                               '/repo/packages/{package_name}')
            'tag_format': ('projects/proj/locations/us-central1/repositories'
                           '/repo/packages/{package_name}/tags/{tag}')
            'version_format': ('projects/proj/locations/us-central1/repositories'
                           '/repo/packages/{package_name}/versions/{version}')
        }

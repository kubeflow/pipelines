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
"""Unit tests for structure-aware Go metadata discovery."""

import unittest
from pathlib import Path

from go_version_metadata import (docker_go_runtime_sources,
                                 has_go_runtime_reference, has_setup_go_use,
                                 is_container_recipe, yaml_mapping_values)


class GoVersionMetadataTest(unittest.TestCase):

    def test_malformed_flow_collection_is_rejected(self):
        with self.assertRaises(ValueError):
            yaml_mapping_values('steps: [}\n', ('uses',))

    def test_discovers_every_golang_stage(self):
        stages, arguments = docker_go_runtime_sources(
            'FROM golang:1.27.0 AS builder\n'
            'FROM golang:1.26.0 AS stale\n')

        self.assertEqual(stages,
                         ['golang:1.27.0', 'golang:1.26.0'])
        self.assertEqual(arguments, [])

    def test_resolves_global_arg_defaults_used_by_from(self):
        stages, arguments = docker_go_runtime_sources(
            'ARG GO_REPOSITORY=docker.io/library/golang\n'
            'FROM ${GO_REPOSITORY}:1.26.0 AS stale\n')

        self.assertEqual(stages, ['${GO_REPOSITORY}:1.26.0'])
        self.assertEqual(arguments, ['GO_REPOSITORY'])

    def test_yaml_block_scalars_are_structural(self):
        contents = (
            'steps:\n'
            '  - uses: >-\n'
            '      actions/setup-go@v7\n'
            'container:\n'
            '  image: |-\n'
            '    golang:1.27.0\n')
        self.assertTrue(has_setup_go_use(contents))
        self.assertTrue(has_go_runtime_reference(Path('test.yaml'), contents))

        command = (
            'run: |\n'
            '  uses: actions/setup-go@v7\n'
            '  image: golang:1.27.0\n')
        self.assertFalse(has_setup_go_use(command))
        self.assertFalse(
            has_go_runtime_reference(Path('test.yaml'), command))

    def test_yaml_decodes_escapes_and_aliases(self):
        contents = (
            'go-image: &go-image "gol\\u0061ng"\n'
            'setup: &setup "actions/setup\\u002dgo@v7"\n'
            'container: {image: *go-image}\n'
            'steps: [{uses: *setup}]\n')
        self.assertTrue(has_setup_go_use(contents))
        self.assertTrue(has_go_runtime_reference(Path('test.yaml'), contents))

    def test_yaml_colon_requires_mapping_separation(self):
        contents = (
            'image:golang\n'
            'uses:actions/setup-go@v7\n'
            'container:golang\n')
        self.assertEqual(
            yaml_mapping_values(contents, ('image', 'uses', 'container')),
            {'image': [], 'uses': [], 'container': []},
        )
        self.assertFalse(has_setup_go_use(contents))
        self.assertFalse(
            has_go_runtime_reference(Path('test.yaml'), contents))

    def test_docker_parser_handles_arg_chains_escape_and_aliases(self):
        contents = (
            '# escape=`\n'
            'ARG GO=go\n'
            'ARG LANG=lang\n'
            'ARG IMAGE=${GO}${LANG}:1.27.0\n'
            'FROM alpine AS golang\n'
            'FROM golang AS alias\n'
            'FROM `\n'
            '  ${IMAGE} AS builder\n')

        stages, arguments = docker_go_runtime_sources(contents)

        self.assertEqual(stages, ['${IMAGE}'])
        self.assertEqual(arguments, ['IMAGE'])

    def test_docker_arg_fallback_is_resolved(self):
        stages, _ = docker_go_runtime_sources(
            'ARG IMAGE\n'
            'FROM ${IMAGE:-golang:1.27.0} AS builder\n')
        self.assertEqual(stages, ['${IMAGE:-golang:1.27.0}'])

    def test_prior_golang_stage_alias_is_not_an_external_image(self):
        stages, arguments = docker_go_runtime_sources(
            'FROM alpine AS golang\n'
            'FROM GoLaNg AS final\n')
        self.assertEqual(stages, [])
        self.assertEqual(arguments, [])

    def test_container_recipe_names_are_discovered(self):
        for name in ('Dockerfile', 'Dockerfile.dev', 'Containerfile',
                     'Containerfile.dev'):
            with self.subTest(name=name):
                self.assertTrue(is_container_recipe(Path('nested') / name))
        self.assertFalse(is_container_recipe(Path('NotAContainerfile')))


if __name__ == '__main__':
    unittest.main()
